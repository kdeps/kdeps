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
const mkimageRawBIOSImage = "alpine:3.19"

// buildRawBIOS builds a raw-bios disk image using a two-step process:
//  1. linuxkit produces kernel+initrd files (no Docker mkimage container needed)
//  2. We assemble the raw-bios disk ourselves with proper FAT headroom
//
// This bypasses a bug in linuxkit's mkimage-raw-bios which allocates only 1 MB
// of FAT headroom, causing "Disk full" for images larger than ~1 GB.
func buildRawBIOS(ctx context.Context, runner LinuxKitRunner, assembler RawBIOSAssembleFunc, configPath, arch, buildDir string) (string, error) {
	return buildRawBIOSWithImage(ctx, runner, assembler, configPath, arch, buildDir, "", "")
}

// buildRawBIOSWithImage is the internal implementation that supports thin builds.
func buildRawBIOSWithImage(ctx context.Context, runner LinuxKitRunner, assembler RawBIOSAssembleFunc, configPath, arch, buildDir, imageName, bootScript string) (string, error) {
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

	if err := assembler(ctx, kernelPath, initrdPath, cmdlinePath, outputFile, imageName, bootScript); err != nil {
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
// Files are copied to a temp dir under ~/.cache/kdeps (on macOS Docker Desktop,
// /Users is the only reliably shared path for volume mounts).
// The assembly script is written to a file and executed inside the container
// to avoid BusyBox ash issues with heredocs in -c mode.
func assembleRawBIOS(ctx context.Context, kernelPath, initrdPath, cmdlinePath, outputPath, imageName, bootScript string) error {
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

	// For thin builds, export the Docker image to a tarball in the work directory
	if imageName != "" {
		fmt.Fprintf(os.Stdout, "Exporting app image %s to data partition...\n", imageName)
		imageTar := filepath.Join(workDir, "image.tar")
		saveCmd := exec.CommandContext(ctx, "docker", "save", "-o", imageTar, imageName)
		if saveErr := saveCmd.Run(); saveErr != nil {
			return fmt.Errorf("failed to export docker image: %w", saveErr)
		}

		// Also write the boot script
		if bootScript != "" {
			bootPath := filepath.Join(workDir, "boot.sh")
			if err := os.WriteFile(bootPath, []byte(bootScript), 0755); err != nil {
				return fmt.Errorf("failed to write boot script: %w", err)
			}
		}
	}

	// Write the assembly script to a file (avoids BusyBox ash -c heredoc bugs)
	scriptPath := filepath.Join(workDir, "assemble.sh")
	if err := writeAssembleScript(scriptPath, workDir, imageName != ""); err != nil {
		return fmt.Errorf("failed to write assembly script: %w", err)
	}

	args := []string{
		"run", "--rm",
		"-v", workDir + ":/work",
		"--tmpfs", "/scratch:exec,size=20g",
		"--entrypoint", "sh",
		mkimageRawBIOSImage,
		"/work/assemble.sh",
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

// writeAssembleScript writes the raw-bios assembly shell script to a file.

// The script reads kernel+initrd from /work (volume mount), does all disk

// assembly in /scratch (tmpfs), and copies the final image back to /work.

// Key fix vs upstream linuxkit: headroom = 5% of content + 10 MB (vs flat 1 MB).

func writeAssembleScript(path, _ string, thin bool) error {

	script := `#!/bin/sh

set -ex



# Install necessary tools

apk add --no-cache syslinux mtools dosfstools e2fsprogs util-linux e2tools



KERNEL="/work/kernel"

INITRD="/work/initrd.img"

CMDLINE_FILE="/work/cmdline"

IMAGE_TAR="/work/image.tar"

BOOT_SH="/work/boot.sh"



# Read cmdline content robustly

CMDLINE=$(cat "$CMDLINE_FILE" | tr -d '\n\r' | sed 's/^ *//;s/ *$//')



echo "CMDLINE: $CMDLINE"



mkdir -p /scratch/bios

cd /scratch/bios



# Write syslinux config (using leading slashes for paths)

echo "DEFAULT linux" > syslinux.cfg

echo "LABEL linux" >> syslinux.cfg

echo "    KERNEL /kernel" >> syslinux.cfg

echo "    INITRD /initrd.img" >> syslinux.cfg

echo "    APPEND ${CMDLINE}" >> syslinux.cfg



cat syslinux.cfg



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



echo "ESP file size: ${ESP_FILE_SIZE_KB} KB (${ESP_FILE_SIZE_SECTORS} sectors)"



ESP_FILE=/scratch/bios/boot.img

IMGFILE=/scratch/bios/disk.img



# Create FAT filesystem (use -F 32 for large images)

FAT_TYPE="-F 16"

if [ "$ESP_FILE_SIZE_KB" -gt 524288 ]; then

    FAT_TYPE="-F 32"

fi



mkfs.vfat $FAT_TYPE -v -C "$ESP_FILE" "$ESP_FILE_SIZE_KB" > /dev/null



# Use local mtools config to avoid interference

export MTOOLSRC=/scratch/bios/mtoolsrc

echo "mtools_skip_check=1" > "$MTOOLSRC"



echo "Copying files to ESP..."

mcopy -i "$ESP_FILE" syslinux.cfg ::/syslinux.cfg

mcopy -i "$ESP_FILE" "$KERNEL" ::/kernel

mcopy -i "$ESP_FILE" "$INITRD" ::/initrd.img



echo "Installing syslinux..."

# -i = ignore fs checks

syslinux --install -i "$ESP_FILE"



# Data partition size logic

DATA_PART_SIZE_SECTORS=0

if [ -f "$IMAGE_TAR" ]; then

    IMAGE_SIZE=$(stat -c %s "$IMAGE_TAR")

    # 2GB minimum for data partition, or image size * 1.5 + 512MB

    DATA_SIZE=$(( IMAGE_SIZE * 3 / 2 + 512 * 1024 * 1024 ))

    MIN_DATA=$(( 2048 * 1024 * 1024 ))

    if [ "$DATA_SIZE" -lt "$MIN_DATA" ]; then

        DATA_SIZE=$MIN_DATA

    fi

    DATA_PART_SIZE_SECTORS=$(( DATA_SIZE / 512 ))

fi



ONEMB=1048576

ESP_ACTUAL_SIZE=$(stat -c %s "$ESP_FILE")

SIZE_IN_BYTES=$(( ESP_ACTUAL_SIZE + (DATA_PART_SIZE_SECTORS * 512) + 10 * ONEMB ))

MB_BLOCKS=$(( SIZE_IN_BYTES / ONEMB ))



echo "Creating disk image: ${MB_BLOCKS} MB"

dd if=/dev/zero of="$IMGFILE" bs=1M count="$MB_BLOCKS" 2>&1



# Explicit partition table creation

PART_TYPE="0e" # FAT16 LBA

if [ "$FAT_TYPE" = "-F 32" ]; then

    PART_TYPE="0c" # FAT32 LBA

fi



# Use sfdisk to create partitions

if [ "$DATA_PART_SIZE_SECTORS" -gt 0 ]; then

    # Two partitions: Boot (FAT) and Data (Linux)

    printf "label: dos\n2048,%s,%s,*\n%s,%s,83,\n" "$ESP_FILE_SIZE_SECTORS" "$PART_TYPE" "$(( 2048 + ESP_FILE_SIZE_SECTORS ))" "$DATA_PART_SIZE_SECTORS" | sfdisk "$IMGFILE" 2>&1

else

    # Single partition: Boot (FAT)

    echo "2048,$ESP_FILE_SIZE_SECTORS,$PART_TYPE,*" | sfdisk "$IMGFILE" 2>&1

fi



# Write ESP volume to the first partition

ESP_SECTOR_START=2048

dd if="$ESP_FILE" of="$IMGFILE" bs=512 count="$ESP_FILE_SIZE_SECTORS" conv=notrunc seek="$ESP_SECTOR_START" 2>&1



# Handle data partition if needed

if [ "$DATA_PART_SIZE_SECTORS" -gt 0 ]; then

    echo "Preparing data partition..."

    DATA_FILE=/scratch/bios/data.img

    truncate -s "$(( DATA_PART_SIZE_SECTORS * 512 ))" "$DATA_FILE"

    mkfs.ext4 -F "$DATA_FILE"

    

    # Store the image and the bootstrapper using e2tools (no mount needed)

    echo "Copying files to data partition..."

    e2cp "$IMAGE_TAR" "$DATA_FILE:/image.tar"

    if [ -f "$BOOT_SH" ]; then

        e2cp "$BOOT_SH" "$DATA_FILE:/boot.sh"

    fi

    

    # Write data volume to the second partition

    DATA_SECTOR_START=$(( 2048 + ESP_FILE_SIZE_SECTORS ))

    dd if="$DATA_FILE" of="$IMGFILE" bs=512 count="$DATA_PART_SIZE_SECTORS" conv=notrunc seek="$DATA_SECTOR_START" 2>&1

fi



# Write altmbr.bin and configure it to boot the first partition

if [ -f /usr/share/syslinux/altmbr.bin ]; then

    dd if=/usr/share/syslinux/altmbr.bin of="$IMGFILE" bs=439 count=1 conv=notrunc 2>&1

    printf '\001' | dd of="$IMGFILE" bs=1 count=1 seek=439 conv=notrunc 2>&1

fi



sync



echo "Copying disk image to output..."

cp /scratch/bios/disk.img /work/disk.img



echo "Raw BIOS disk image assembled successfully"

`



	return os.WriteFile(path, []byte(script), 0755)

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
