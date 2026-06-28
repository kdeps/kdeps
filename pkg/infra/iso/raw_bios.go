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
	"path/filepath"
	"strings"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

	"github.com/spf13/afero"
)

// buildRawBIOSWithImage is the internal implementation that supports thin builds.
func buildRawBIOSWithImage(
	ctx context.Context,
	runner LinuxKitRunner,
	assembler RawBIOSAssembleFunc,
	configPath, arch, buildDir, imageName string,
) (string, error) {
	kdeps_debug.Log("enter: buildRawBIOSWithImage")
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
		"",
	); assembleErr != nil {
		return "", assembleErr
	}

	return outputFile, nil
}

// findKernelInitrd locates the kernel, initrd, and cmdline files in the build directory.
func findKernelInitrd(dir string) (string, string, string, error) {
	kdeps_debug.Log("enter: findKernelInitrd")
	entries, err := afero.ReadDir(AppFS, dir)
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

func createRawBIOSWorkDir() (string, error) {
	home, err := osUserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDir := filepath.Join(home, ".cache", "kdeps")
	if mkdirErr := AppFS.MkdirAll(cacheDir, 0750); mkdirErr != nil {
		return "", fmt.Errorf("failed to create cache directory: %w", mkdirErr)
	}

	workDir, err := os.MkdirTemp(cacheDir, "rawbios-*")
	if err != nil {
		return "", fmt.Errorf("failed to create work directory: %w", err)
	}
	return workDir, nil
}

func copyKernelArtifacts(workDir, kernelPath, initrdPath, cmdlinePath string) error {
	for _, src := range []struct{ from, to string }{
		{kernelPath, filepath.Join(workDir, "kernel")},
		{initrdPath, filepath.Join(workDir, "initrd.img")},
		{cmdlinePath, filepath.Join(workDir, "cmdline")},
	} {
		if copyErr := copyFile(src.from, src.to); copyErr != nil {
			return fmt.Errorf("failed to copy %s: %w", filepath.Base(src.from), copyErr)
		}
	}
	return nil
}

func exportDockerImageToWorkDir(
	ctx context.Context,
	workDir, imageName, bootScript string,
) error {
	fmt.Fprintf(os.Stdout, "Exporting app image %s to data partition...\n", imageName)
	imageTar := filepath.Join(workDir, "image.tar")
	saveCmd := execCommandContext(ctx, "docker", "save", "-o", imageTar, imageName)
	if saveErr := saveCmd.Run(); saveErr != nil {
		return fmt.Errorf("failed to export docker image: %w", saveErr)
	}

	if bootScript == "" {
		return nil
	}
	bootPath := filepath.Join(workDir, "boot.sh")
	if writeErr := afero.WriteFile(AppFS, bootPath, []byte(bootScript), 0600); writeErr != nil {
		return fmt.Errorf("failed to write boot script: %w", writeErr)
	}
	return nil
}

func runDockerAssembleScript(ctx context.Context, workDir string) error {
	args := []string{
		"run", "--rm",
		"-v", workDir + ":/work",
		"--tmpfs", "/scratch:exec,size=20g",
		"--entrypoint", "sh",
		mkimageRawBIOSImage,
		"/work/assemble.sh",
	}

	cmd := execCommandContext(ctx, "docker", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Fprintln(os.Stdout, "Assembling raw disk image...")
	if runErr := cmd.Run(); runErr != nil {
		return fmt.Errorf("raw disk assembly failed: %w", runErr)
	}
	return nil
}

// assembleRawBIOS creates a raw disk image (BIOS or EFI) from kernel+initrd files.
func assembleRawBIOS(
	ctx context.Context,
	kernelPath, initrdPath, cmdlinePath, outputPath, imageName, bootScript string,
) error {
	kdeps_debug.Log("enter: assembleRawBIOS")
	workDir, err := createRawBIOSWorkDir()
	if err != nil {
		return err
	}
	defer AppFS.RemoveAll(workDir) //nolint:errcheck // cleanup-only deferred call

	if copyErr := copyKernelArtifacts(workDir, kernelPath, initrdPath, cmdlinePath); copyErr != nil {
		return copyErr
	}

	if imageName != "" {
		if exportErr := exportDockerImageToWorkDir(ctx, workDir, imageName, bootScript); exportErr != nil {
			return exportErr
		}
	}

	scriptPath := filepath.Join(workDir, "assemble.sh")
	if writeScriptErr := writeAssembleScript(scriptPath, workDir, imageName != ""); writeScriptErr != nil {
		return fmt.Errorf("failed to write assembly script: %w", writeScriptErr)
	}

	if runErr := runDockerAssembleScript(ctx, workDir); runErr != nil {
		return runErr
	}

	produced := filepath.Join(workDir, "disk.img")
	if copyDiskErr := copyFile(produced, outputPath); copyDiskErr != nil {
		return fmt.Errorf("failed to copy disk image to output: %w", copyDiskErr)
	}

	return nil
}

func copyFile(src, dst string) error {
	kdeps_debug.Log("enter: copyFile")
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
