// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

//go:build !js

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

// mkimageRawBIOSImage is the Docker image used for assembling raw disk images.
const mkimageRawBIOSImage = "alpine:3.19"

// buildRawBIOSWithImage is the internal implementation that supports thin builds.
func buildRawBIOSWithImage(
	ctx context.Context,
	runner LinuxKitRunner,
	assembler RawBIOSAssembleFunc,
	configPath, arch, buildDir, imageName, bootScript string,
) (string, error) {
	// Step 1: Build kernel+initrd with linuxkit
	if err := runner.Build(ctx, configPath, "kernel+initrd", arch, buildDir, ""); err != nil {
		return "", fmt.Errorf("linuxkit kernel+initrd build failed: %w", err)
	}

	// Find the kernel and initrd files produced by linuxkit
	kernelPath, initrdPath, cmdlinePath, err := findKernelInitrd(buildDir)
	if err != nil {
		return "", err
	}

	// Step 2: Create raw disk image
	outputFile := filepath.Join(buildDir, "disk.img")

	if assembleErr := assembler(
		ctx,
		kernelPath,
		initrdPath,
		cmdlinePath,
		outputFile,
		imageName,
		bootScript,
	); assembleErr != nil {
		return "", assembleErr
	}

	return outputFile, nil
}

// findKernelInitrd locates the kernel, initrd, and cmdline files in the build directory.
func findKernelInitrd(dir string) (string, string, string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read build directory: %w", err)
	}

	var kernel, initrd, cmdline string
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

// assembleRawBIOS creates a raw disk image (BIOS or EFI) from kernel+initrd files.
func assembleRawBIOS(
	ctx context.Context,
	kernelPath, initrdPath, cmdlinePath, outputPath, imageName, bootScript string,
) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(home, ".cache", "kdeps")
	if mkdirErr := os.MkdirAll(cacheDir, 0750); mkdirErr != nil {
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
		if copyErr := copyFile(src.from, src.to); copyErr != nil {
			return fmt.Errorf("failed to copy %s: %w", filepath.Base(src.from), copyErr)
		}
	}

	// For thin builds, export the Docker image to a tarball
	if imageName != "" {
		fmt.Fprintf(os.Stdout, "Exporting app image %s to data partition...\n", imageName)
		imageTar := filepath.Join(workDir, "image.tar")
		saveCmd := exec.CommandContext(ctx, "docker", "save", "-o", imageTar, imageName)
		if saveErr := saveCmd.Run(); saveErr != nil {
			return fmt.Errorf("failed to export docker image: %w", saveErr)
		}

		if bootScript != "" {
			bootPath := filepath.Join(workDir, "boot.sh")
			if writeErr := os.WriteFile(bootPath, []byte(bootScript), 0600); writeErr != nil {
				return fmt.Errorf("failed to write boot script: %w", writeErr)
			}
		}
	}

	scriptPath := filepath.Join(workDir, "assemble.sh")
	if writeScriptErr := writeAssembleScript(scriptPath, workDir, imageName != ""); writeScriptErr != nil {
		return fmt.Errorf("failed to write assembly script: %w", writeScriptErr)
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

	fmt.Fprintln(os.Stdout, "Assembling raw disk image...")

	if runErr := cmd.Run(); runErr != nil {
		return fmt.Errorf("raw disk assembly failed: %w", runErr)
	}

	produced := filepath.Join(workDir, "disk.img")
	if copyDiskErr := copyFile(produced, outputPath); copyDiskErr != nil {
		return fmt.Errorf("failed to copy disk image to output: %w", copyDiskErr)
	}

	return nil
}

//nolint:funlen // generating a large shell script
func writeAssembleScript(path, _ string, _ bool) error {
	const scriptHeader = `#!/bin/sh
set -ex

# Install necessary tools
apk add --no-cache ncurses syslinux mtools dosfstools e2fsprogs util-linux e2tools

KERNEL="/work/kernel"
INITRD="/work/initrd.img"
CMDLINE_FILE="/work/cmdline"
IMAGE_TAR="/work/image.tar"
BOOT_SH="/work/boot.sh"

# Read cmdline content robustly
CMDLINE=$(cat "$CMDLINE_FILE" | tr -d '\n\r' | sed 's/^ *//;s/ *$//')

mkdir -p /scratch/bios
cd /scratch/bios

# Write syslinux config
cat > syslinux.cfg <<EOF
SERIAL 0 115200
PROMPT 0
TIMEOUT 0
DEFAULT linux

LABEL linux
    LINUX /kernel
    APPEND initrd=/initrd.img ${CMDLINE}
EOF

KERNEL_FILE_SIZE=$(stat -c %s "$KERNEL")
INITRD_FILE_SIZE=$(stat -c %s "$INITRD")

# 5% of content + 20 MB headroom
CONTENT_SIZE=$(( KERNEL_FILE_SIZE + INITRD_FILE_SIZE ))
ESP_HEADROOM=$(( CONTENT_SIZE / 20 + 20 * 1024 * 1024 ))
ESP_FILE_SIZE=$(( CONTENT_SIZE + ESP_HEADROOM ))
ESP_FILE_SIZE_KB=$(( (ESP_FILE_SIZE + 1023) / 1024 ))
ESP_FILE_SIZE_SECTORS=$(( ESP_FILE_SIZE_KB * 2 ))
`

	const scriptDataPart = `
# Data partition size logic
DATA_PART_SIZE_SECTORS=0
if [ -f "$IMAGE_TAR" ]; then
    IMAGE_SIZE=$(stat -c %s "$IMAGE_TAR")
    DATA_SIZE=$(( IMAGE_SIZE * 3 / 2 + 512 * 1024 * 1024 ))
    MIN_DATA=$(( 2048 * 1024 * 1024 ))
    if [ "$DATA_SIZE" -lt "$MIN_DATA" ]; then
        DATA_SIZE=$MIN_DATA
    fi
    DATA_PART_SIZE_SECTORS=$(( DATA_SIZE / 512 ))
fi

IMGFILE=/work/disk.img
# Create an empty sparse file for the disk image to save memory in tmpfs
TOTAL_SECTORS=$(( 2048 + ESP_FILE_SIZE_SECTORS + DATA_PART_SIZE_SECTORS + 20480 ))
truncate -s $(( TOTAL_SECTORS * 512 )) "$IMGFILE"

# Create GPT partition table using sfdisk (available in util-linux)
if [ "$DATA_PART_SIZE_SECTORS" -gt 0 ]; then
    printf "label: gpt\n2048,%s,C12A7328-F81F-11D2-BA4B-00A0C93EC93B,*\n%s,%s,0FC63DAF-8483-4772-8E79-3D69D8477DE4,\n" \
        "$ESP_FILE_SIZE_SECTORS" "$((2048 + ESP_FILE_SIZE_SECTORS))" "$DATA_PART_SIZE_SECTORS" | sfdisk "$IMGFILE"
else
    printf "label: gpt\n2048,%s,C12A7328-F81F-11D2-BA4B-00A0C93EC93B,*\n" \
        "$ESP_FILE_SIZE_SECTORS" | sfdisk "$IMGFILE"
fi
`

	const scriptESP = `
# Create FAT filesystem directly into the partition using mtools (memory efficient)
ESP_TMP=/scratch/esp.img
truncate -s $(( ESP_FILE_SIZE_SECTORS * 512 )) "$ESP_TMP"
mkfs.vfat -F 32 "$ESP_TMP"

export MTOOLSRC=/scratch/mtoolsrc
echo "mtools_skip_check=1" > "$MTOOLSRC"

# UEFI structure
mmd -i "$ESP_TMP" ::/EFI
mmd -i "$ESP_TMP" ::/EFI/BOOT

mcopy -i "$ESP_TMP" syslinux.cfg ::/EFI/BOOT/syslinux.cfg
mcopy -i "$ESP_TMP" syslinux.cfg ::/syslinux.cfg
mcopy -i "$ESP_TMP" "$KERNEL" ::/kernel
mcopy -i "$ESP_TMP" "$INITRD" ::/initrd.img

# Copy SYSLINUX modules
if [ -d /usr/share/syslinux ]; then
    for f in ldlinux.c32 libcom32.c32 libutil.c32 mboot.c32 linux.c32 menu.c32; do
        if [ -f "/usr/share/syslinux/$f" ]; then
            mcopy -i "$ESP_TMP" "/usr/share/syslinux/$f" ::/EFI/BOOT/$f || true
            mcopy -i "$ESP_TMP" "/usr/share/syslinux/$f" ::/$f || true
        fi
    done
fi

# Copy EFI loader from main syslinux package
if [ -f /usr/share/syslinux/efi64/syslinux.efi ]; then
    mcopy -i "$ESP_TMP" /usr/share/syslinux/efi64/syslinux.efi ::/EFI/BOOT/BOOTX64.EFI
    mcopy -i "$ESP_TMP" /usr/share/syslinux/efi64/ldlinux.e64 ::/EFI/BOOT/ldlinux.e64
elif [ -f /usr/lib/syslinux/efi64/syslinux.efi ]; then
    mcopy -i "$ESP_TMP" /usr/lib/syslinux/efi64/syslinux.efi ::/EFI/BOOT/BOOTX64.EFI
    mcopy -i "$ESP_TMP" /usr/lib/syslinux/efi64/ldlinux.e64 ::/EFI/BOOT/ldlinux.e64
fi

# Write the ESP partition back to the disk image
dd if="$ESP_TMP" of="$IMGFILE" bs=512 seek=2048 count="$ESP_FILE_SIZE_SECTORS" conv=notrunc
rm "$ESP_TMP"
`

	const scriptFooter = `
# Handle data partition
if [ "$DATA_PART_SIZE_SECTORS" -gt 0 ]; then
    DATA_TMP=/scratch/data.img
    truncate -s $(( DATA_PART_SIZE_SECTORS * 512 )) "$DATA_TMP"
    mkfs.ext4 -F "$DATA_TMP"
    e2cp "$IMAGE_TAR" "$DATA_TMP:/image.tar"
    if [ -f "$BOOT_SH" ]; then
        e2cp "$BOOT_SH" "$DATA_TMP:/boot.sh"
    fi
    dd if="$DATA_TMP" of="$IMGFILE" bs=512 seek=$((2048 + ESP_FILE_SIZE_SECTORS)) count="$DATA_PART_SIZE_SECTORS" conv=notrunc
    rm "$DATA_TMP"
fi

# Make it bootable on BIOS too
if [ -f /usr/share/syslinux/gptmbr.bin ]; then
    dd if=/usr/share/syslinux/gptmbr.bin of="$IMGFILE" bs=440 count=1 conv=notrunc
fi

sync
echo "Disk image assembled successfully"
`

	script := scriptHeader + scriptDataPart + scriptESP + scriptFooter
	return os.WriteFile(path, []byte(script), 0600)
}

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
