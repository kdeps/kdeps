// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package iso

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// mkimageRawBIOSImage is the Docker image used for assembling raw-bios disk images.
// This is the same image linuxkit uses, giving us access to syslinux, sfdisk, etc.
const mkimageRawBIOSImage = "linuxkit/mkimage-raw-bios:42706053b66824d430e59edd7c195ee9b66493f4"

// buildRawBIOS builds a raw-bios disk image using a two-step process:
//  1. linuxkit produces kernel+initrd files (no Docker mkimage container needed)
//  2. We assemble the raw-bios disk ourselves with proper FAT headroom
//
// This bypasses a bug in linuxkit's mkimage-raw-bios which allocates only 1 MB
// of FAT headroom, causing "Disk full" for images larger than ~1 GB.
func buildRawBIOS(ctx context.Context, runner LinuxKitRunner, assembler RawBIOSAssembleFunc, configPath, arch, buildDir string) (string, error) {
	// Step 1: Build kernel+initrd with linuxkit (no mkimage Docker container involved)
	if err := runner.Build(ctx, configPath, "kernel+initrd", arch, buildDir, ""); err != nil {
		return "", fmt.Errorf("linuxkit kernel+initrd build failed: %w", err)
	}

	// Find the kernel and initrd files produced by linuxkit
	kernelPath, initrdPath, cmdlinePath, err := findKernelInitrd(buildDir)
	if err != nil {
		return "", err
	}

	// Step 2: Create raw-bios disk image with fixed FAT headroom
	outputFile := filepath.Join(buildDir, "disk.img")

	if err := assembler(ctx, kernelPath, initrdPath, cmdlinePath, outputFile); err != nil {
		return "", err
	}

	return outputFile, nil
}

// findKernelInitrd locates the kernel, initrd, and cmdline files in the build directory.
func findKernelInitrd(dir string) (kernel, initrd, cmdline string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read build directory: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		switch {
		case strings.HasSuffix(name, "-kernel"):
			kernel = filepath.Join(dir, name)
		case strings.HasSuffix(name, "-initrd.img"):
			initrd = filepath.Join(dir, name)
		case strings.HasSuffix(name, "-cmdline"):
			cmdline = filepath.Join(dir, name)
		}
	}

	if kernel == "" {
		return "", "", "", fmt.Errorf("kernel file not found in %s", dir)
	}

	if initrd == "" {
		return "", "", "", fmt.Errorf("initrd file not found in %s", dir)
	}

	if cmdline == "" {
		return "", "", "", fmt.Errorf("cmdline file not found in %s", dir)
	}

	return kernel, initrd, cmdline, nil
}

// assembleRawBIOS creates a raw-bios disk image from kernel+initrd files.
// Files are copied to a temp dir under /tmp (which Docker Desktop always shares),
// then a Docker container assembles the disk with proper FAT headroom.
func assembleRawBIOS(ctx context.Context, kernelPath, initrdPath, cmdlinePath, outputPath string) error {
	// Use a temp directory under HOME — on macOS Docker Desktop, /Users is the
	// only reliably shared path (/tmp and /var/folders are listed as shared but
	// volume mounts silently fail due to symlink resolution issues).
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(home, ".cache", "kdeps")
	if mkdirErr := os.MkdirAll(cacheDir, 0755); mkdirErr != nil {
		return fmt.Errorf("failed to create cache directory: %w", mkdirErr)
	}

	workDir, err := os.MkdirTemp(cacheDir, "rawbios-*")
	if err != nil {
		return fmt.Errorf("failed to create work directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// Copy kernel, initrd, cmdline to the work directory
	for _, src := range []struct{ from, to string }{
		{kernelPath, filepath.Join(workDir, "kernel")},
		{initrdPath, filepath.Join(workDir, "initrd.img")},
		{cmdlinePath, filepath.Join(workDir, "cmdline")},
	} {
		if err := copyFile(src.from, src.to); err != nil {
			return fmt.Errorf("failed to copy %s: %w", filepath.Base(src.from), err)
		}
	}

	args := []string{
		"run", "--rm",
		"-v", workDir + ":/work",
		"--tmpfs", "/scratch:exec,size=20g",
		"--entrypoint", "sh",
		mkimageRawBIOSImage,
		"-c", fixedMakeBIOSScript(),
	}

	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Fprintln(os.Stdout, "Assembling raw BIOS disk image...")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("raw-bios disk assembly failed: %w", err)
	}

	// Move the produced disk image to the final output path
	produced := filepath.Join(workDir, "disk.img")
	if err := copyFile(produced, outputPath); err != nil {
		return fmt.Errorf("failed to copy disk image to output: %w", err)
	}

	return nil
}

// copyFile copies src to dst by streaming (no full-file memory buffering).
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if closeErr := out.Close(); err == nil {
		err = closeErr
	}

	return err
}

// fixedMakeBIOSScript returns a shell script that assembles a raw-bios disk image.
// Input files are read from /work (Docker volume mount — sequential reads work fine).
// All disk assembly (dd, fdisk, sfdisk with random-access I/O) is done in /scratch
// (a large tmpfs) to avoid "bad file number" on Docker Desktop volume mounts.
// Only the final disk.img is copied back to /work.
// Key fix vs upstream: headroom = 5% of content + 10 MB (vs flat 1 MB).
func fixedMakeBIOSScript() string {
	return `set -e

CMDLINE="$(cat /work/cmdline)"

mkdir -p /scratch/bios
cd /scratch/bios

KERNEL="/work/kernel"
INITRD="/work/initrd.img"
IMGFILE="/scratch/bios/disk.img"

cat > syslinux.cfg <<SYSEOF
DEFAULT linux
LABEL linux
    KERNEL /kernel
    INITRD /initrd.img
    APPEND ${CMDLINE}
SYSEOF

KERNEL_FILE_SIZE=$(stat -c %s "$KERNEL")
INITRD_FILE_SIZE=$(stat -c %s "$INITRD")

echo "Kernel size: $KERNEL_FILE_SIZE bytes"
echo "Initrd size: $INITRD_FILE_SIZE bytes"

# 5% of content + 10 MB headroom (upstream uses flat 1 MB)
CONTENT_SIZE=$(( KERNEL_FILE_SIZE + INITRD_FILE_SIZE ))
ESP_HEADROOM=$(( CONTENT_SIZE / 20 + 10 * 1024 * 1024 ))

ESP_FILE_SIZE=$(( CONTENT_SIZE + ESP_HEADROOM ))
ESP_FILE_SIZE_KB=$(( ( ( (ESP_FILE_SIZE+1024-1) / 1024 ) + 1024-1) / 1024 * 1024 ))
ESP_FILE_SIZE_SECTORS=$(( ESP_FILE_SIZE_KB * 2 ))

echo "ESP file size: ${ESP_FILE_SIZE_KB} KB"

ESP_FILE=/scratch/bios/boot.img
mkfs.vfat -v -C "$ESP_FILE" $(( ESP_FILE_SIZE_KB )) > /dev/null
echo "mtools_skip_check=1" >> /etc/mtools.conf
mcopy -i "$ESP_FILE" syslinux.cfg ::/
mcopy -i "$ESP_FILE" "$KERNEL" ::/kernel
mcopy -i "$ESP_FILE" "$INITRD" ::/initrd.img

syslinux --install "$ESP_FILE"

ONEMB=$(( 1024 * 1024 ))
SIZE_IN_BYTES=$(( $(stat -c %s "$ESP_FILE") + 4*ONEMB ))
BLKSIZE=512
MB_BLOCKS=$(( SIZE_IN_BYTES / ONEMB ))

echo "Creating disk image: ${MB_BLOCKS} MB"
dd if=/dev/zero of="$IMGFILE" bs=1M count=$MB_BLOCKS
echo "w" | fdisk "$IMGFILE" || true
echo ","$ESP_FILE_SIZE_SECTORS",;" | sfdisk "$IMGFILE"

ESP_SECTOR_START=2048
dd if="$ESP_FILE" of="$IMGFILE" bs=$BLKSIZE count=$ESP_FILE_SIZE_SECTORS conv=notrunc seek=$ESP_SECTOR_START

dd if=/usr/share/syslinux/altmbr.bin bs=439 count=1 conv=notrunc of="$IMGFILE"
printf '\1' | dd bs=1 count=1 seek=439 conv=notrunc of="$IMGFILE"
sfdisk -A "$IMGFILE" 1

echo "Copying disk image to output..."
cp /scratch/bios/disk.img /work/disk.img

echo "Raw BIOS disk image assembled successfully"
`
}
