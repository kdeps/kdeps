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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssembleRawBIOS_DockerError(t *testing.T) {
	// Create a fake docker command that fails
	tmpDir := t.TempDir()
	fakeDocker := filepath.Join(tmpDir, "docker")

	// Create a fake docker script that always fails
	dockerScript := `#!/bin/sh
exit 1
`
	if err := os.WriteFile(fakeDocker, []byte(dockerScript), 0755); err != nil {
		t.Fatalf("failed to create fake docker: %v", err)
	}

	// Update PATH to include our fake docker
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Create fake input files
	kernelFile := filepath.Join(tmpDir, "kernel")
	initrdFile := filepath.Join(tmpDir, "initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "cmdline")
	outputFile := filepath.Join(tmpDir, "disk.img")

	for _, f := range []string{kernelFile, initrdFile, cmdlineFile} {
		if err := os.WriteFile(f, []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create fake input file: %v", err)
		}
	}

	// Test with imageName to trigger 'docker save'
	err := assembleRawBIOS(
		t.Context(),
		kernelFile,
		initrdFile,
		cmdlineFile,
		outputFile,
		"fake-image",
		"",
	)
	if err == nil {
		t.Error("expected error from docker save failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to export docker image") {
		t.Errorf("expected docker export error, got: %v", err)
	}
}

func TestAssembleRawBIOS_DockerRunError(t *testing.T) {
	// Create a fake docker command that succeeds for 'save' but fails for 'run'
	tmpDir := t.TempDir()
	fakeDocker := filepath.Join(tmpDir, "docker")

	dockerScript := `#!/bin/sh
if [ "$1" = "save" ]; then
    touch "$3" # Create the output tar file
    exit 0
fi
exit 1
`
	if err := os.WriteFile(fakeDocker, []byte(dockerScript), 0755); err != nil {
		t.Fatalf("failed to create fake docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	kernelFile := filepath.Join(tmpDir, "kernel")
	initrdFile := filepath.Join(tmpDir, "initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "cmdline")
	outputFile := filepath.Join(tmpDir, "disk.img")

	for _, f := range []string{kernelFile, initrdFile, cmdlineFile} {
		if err := os.WriteFile(f, []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create fake input file: %v", err)
		}
	}

	err := assembleRawBIOS(
		t.Context(),
		kernelFile,
		initrdFile,
		cmdlineFile,
		outputFile,
		"fake-image",
		"",
	)
	if err == nil {
		t.Error("expected error from docker run failure, got nil")
	}
	if !strings.Contains(err.Error(), "raw disk assembly failed") {
		t.Errorf("expected assembly failure error, got: %v", err)
	}
}

func TestAssembleRawBIOS_Success(t *testing.T) {
	tmpDir := t.TempDir()
	fakeDocker := filepath.Join(tmpDir, "docker")

	// fake disk.img location within workDir is /work/disk.img
	// assembleRawBIOS creates a workDir under ~/.cache/kdeps/rawbios-*
	// We need the fake docker to create a disk.img in the correct place.
	// Since we don't know the exact workDir path, we can try to find it from the arguments.

	dockerScript := `#!/bin/sh
if [ "$1" = "save" ]; then
    touch "$3"
    exit 0
fi
if [ "$1" = "run" ]; then
    # Find the workDir from the -v volume mount argument
    # Arguments: run --rm -v /path/to/work:/work ...
    WORK_DIR=""
    for arg in "$@"; do
        if echo "$arg" | grep -q ":/work"; then
            WORK_DIR=$(echo "$arg" | cut -d: -f1)
        fi
    done
    if [ -n "$WORK_DIR" ]; then
        echo "fake disk image" > "$WORK_DIR/disk.img"
    fi
    exit 0
fi
exit 0
`
	if err := os.WriteFile(fakeDocker, []byte(dockerScript), 0755); err != nil {
		t.Fatalf("failed to create fake docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	kernelFile := filepath.Join(tmpDir, "kernel")
	initrdFile := filepath.Join(tmpDir, "initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "cmdline")
	outputFile := filepath.Join(tmpDir, "disk.img")

	for _, f := range []string{kernelFile, initrdFile, cmdlineFile} {
		if err := os.WriteFile(f, []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create fake input file: %v", err)
		}
	}

	err := assembleRawBIOS(
		t.Context(),
		kernelFile,
		initrdFile,
		cmdlineFile,
		outputFile,
		"fake-image",
		"echo boot",
	)
	if err != nil {
		t.Fatalf("assembleRawBIOS failed: %v", err)
	}

	if _, statErr := os.Stat(outputFile); os.IsNotExist(statErr) {
		t.Error("expected output file to be created")
	}
}

func TestAssembleRawBIOS_CopyFileError(t *testing.T) {
	tmpDir := t.TempDir()
	fakeDocker := filepath.Join(tmpDir, "docker")
	dockerScript := `#!/bin/sh
exit 0
`
	if err := os.WriteFile(fakeDocker, []byte(dockerScript), 0755); err != nil {
		t.Fatalf("failed to create fake docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	kernelFile := filepath.Join(tmpDir, "kernel")
	initrdFile := filepath.Join(tmpDir, "initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "cmdline")

	for _, f := range []string{kernelFile, initrdFile, cmdlineFile} {
		if err := os.WriteFile(f, []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create fake input file: %v", err)
		}
	}

	// Make kernel file unreadable so copyFile fails
	if err := os.Chmod(kernelFile, 0000); err != nil {
		t.Fatalf("failed to chmod kernel file: %v", err)
	}

	outputFile := filepath.Join(tmpDir, "disk.img")
	err := assembleRawBIOS(t.Context(), kernelFile, initrdFile, cmdlineFile, outputFile, "", "")
	if err == nil {
		t.Error("expected error from copyFile failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to copy kernel") {
		t.Errorf("expected 'failed to copy kernel' error, got: %v", err)
	}
}

func TestAssembleRawBIOS_CopyDiskImageError(t *testing.T) {
	tmpDir := t.TempDir()
	fakeDocker := filepath.Join(tmpDir, "docker")

	// fake docker: creates disk.img then makes it unreadable so copyFile fails
	dockerScript := `#!/bin/sh
if [ "$1" = "save" ]; then
    touch "$3"
    exit 0
fi
if [ "$1" = "run" ]; then
    WORK_DIR=""
    for arg in "$@"; do
        if echo "$arg" | grep -q ":/work"; then
            WORK_DIR=$(echo "$arg" | cut -d: -f1)
        fi
    done
    if [ -n "$WORK_DIR" ]; then
        touch "$WORK_DIR/disk.img"
        chmod 0000 "$WORK_DIR/disk.img"
    fi
    exit 0
fi
exit 0
`
	if err := os.WriteFile(fakeDocker, []byte(dockerScript), 0755); err != nil {
		t.Fatalf("failed to create fake docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	kernelFile := filepath.Join(tmpDir, "kernel")
	initrdFile := filepath.Join(tmpDir, "initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "cmdline")
	outputFile := filepath.Join(tmpDir, "disk.img")

	for _, f := range []string{kernelFile, initrdFile, cmdlineFile} {
		if err := os.WriteFile(f, []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create fake input file: %v", err)
		}
	}

	err := assembleRawBIOS(t.Context(), kernelFile, initrdFile, cmdlineFile, outputFile, "", "")
	if err == nil {
		t.Error("expected error from copyFile disk image failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to copy disk image") {
		t.Errorf("expected 'failed to copy disk image' error, got: %v", err)
	}
}

func TestAssembleRawBIOS_BootScriptWriteError(t *testing.T) {
	tmpDir := t.TempDir()
	fakeDocker := filepath.Join(tmpDir, "docker")

	// Fake docker: creates image.tar then makes workDir read-only so boot.sh write fails
	dockerScript := `#!/bin/sh
if [ "$1" = "save" ]; then
    touch "$3"
    WORK_DIR=$(dirname "$3")
    chmod 0555 "$WORK_DIR"
    exit 0
fi
exit 1
`
	if err := os.WriteFile(fakeDocker, []byte(dockerScript), 0755); err != nil {
		t.Fatalf("failed to create fake docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	kernelFile := filepath.Join(tmpDir, "kernel")
	initrdFile := filepath.Join(tmpDir, "initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "cmdline")
	outputFile := filepath.Join(tmpDir, "disk.img")

	for _, f := range []string{kernelFile, initrdFile, cmdlineFile} {
		if err := os.WriteFile(f, []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create fake input file: %v", err)
		}
	}

	err := assembleRawBIOS(
		t.Context(),
		kernelFile,
		initrdFile,
		cmdlineFile,
		outputFile,
		"fake-image",
		"dummy-boot-script",
	)
	if err == nil {
		t.Error("expected error from boot script write failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to write boot script") {
		t.Errorf("expected 'failed to write boot script' error, got: %v", err)
	}
}

func TestAssembleRawBIOS_WriteAssembleScriptError(t *testing.T) {
	tmpDir := t.TempDir()
	fakeDocker := filepath.Join(tmpDir, "docker")

	// Same fake docker: creates image.tar then makes workDir read-only so assemble script write fails
	dockerScript := `#!/bin/sh
if [ "$1" = "save" ]; then
    touch "$3"
    WORK_DIR=$(dirname "$3")
    chmod 0555 "$WORK_DIR"
    exit 0
fi
exit 1
`
	if err := os.WriteFile(fakeDocker, []byte(dockerScript), 0755); err != nil {
		t.Fatalf("failed to create fake docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	kernelFile := filepath.Join(tmpDir, "kernel")
	initrdFile := filepath.Join(tmpDir, "initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "cmdline")
	outputFile := filepath.Join(tmpDir, "disk.img")

	for _, f := range []string{kernelFile, initrdFile, cmdlineFile} {
		if err := os.WriteFile(f, []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create fake input file: %v", err)
		}
	}

	// bootScript is empty so boot.sh write is skipped, then writeAssembleScript fails
	err := assembleRawBIOS(
		t.Context(),
		kernelFile,
		initrdFile,
		cmdlineFile,
		outputFile,
		"fake-image",
		"",
	)
	if err == nil {
		t.Error("expected error from writeAssembleScript failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to write assembly script") {
		t.Errorf("expected 'failed to write assembly script' error, got: %v", err)
	}
}

func TestCopyFile_Success(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")
	dst := filepath.Join(tmpDir, "dest.txt")

	content := []byte("hello, world")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}

	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestCopyFile_SourceNotExist(t *testing.T) {
	tmpDir := t.TempDir()
	dst := filepath.Join(tmpDir, "dest.txt")

	err := copyFile("/nonexistent/source.txt", dst)
	if err == nil {
		t.Error("expected error when source does not exist, got nil")
	}
}

func TestCopyFile_DestInvalidDir(t *testing.T) {
	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "source.txt")

	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	err := copyFile(src, "/nonexistent/dir/dest.txt")
	if err == nil {
		t.Error("expected error when destination directory does not exist, got nil")
	}
}

func TestAssembleRawBIOS_MkdirTempError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires filesystem permission manipulation")
	}

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create the cache directory chain with write permission, then make the
	// leaf directory read-only so os.MkdirTemp inside it fails with EACCES.
	cacheDir := filepath.Join(tmpDir, ".cache", "kdeps")
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	if err := os.Chmod(cacheDir, 0555); err != nil {
		t.Fatalf("failed to chmod cache dir to 0555: %v", err)
	}

	err := assembleRawBIOS(t.Context(), "/fake/kernel", "/fake/initrd", "/fake/cmdline", "/fake/output", "", "")
	if err == nil {
		t.Error("expected error from MkdirTemp failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create work directory") {
		t.Fatalf("expected 'failed to create work directory' error, got: %v", err)
	}
}

func TestAssembleRawBIOS_MkdirAllError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires filesystem manipulation")
	}

	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	cacheParent := filepath.Join(tmpDir, ".cache")
	if err := os.MkdirAll(cacheParent, 0750); err != nil {
		t.Fatalf("failed to create cache parent: %v", err)
	}
	cachePath := filepath.Join(cacheParent, "kdeps")
	if err := os.WriteFile(cachePath, []byte("block"), 0644); err != nil {
		t.Fatalf("failed to create blocking file: %v", err)
	}

	err := assembleRawBIOS(t.Context(), "/fake/kernel", "/fake/initrd", "/fake/cmdline", "/fake/output", "", "")
	if err == nil {
		t.Error("expected error from MkdirAll failure, got nil")
	}
	if !strings.Contains(err.Error(), "failed to create cache directory") {
		t.Fatalf("expected 'failed to create cache directory' error, got: %v", err)
	}
}

func TestCreateRawBIOSWorkDir_MkdirTempError(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	cacheDir := filepath.Join(tmpDir, ".cache", "kdeps")
	require.NoError(t, os.MkdirAll(cacheDir, 0750))
	require.NoError(t, os.Chmod(cacheDir, 0555))

	_, err := createRawBIOSWorkDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create work directory")
}

func TestCreateRawBIOSWorkDir_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	cacheParent := filepath.Join(tmpDir, ".cache")
	require.NoError(t, os.MkdirAll(cacheParent, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(cacheParent, "kdeps"), []byte("block"), 0644))

	_, err := createRawBIOSWorkDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create cache directory")
}

func TestAssembleRawBIOS_WorkDirError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cacheParent := filepath.Join(tmpDir, ".cache")
	require.NoError(t, os.MkdirAll(cacheParent, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(cacheParent, "kdeps"), []byte("block"), 0644))

	kernelFile := filepath.Join(tmpDir, "kernel")
	initrdFile := filepath.Join(tmpDir, "initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "cmdline")
	outputFile := filepath.Join(tmpDir, "disk.img")
	for _, f := range []string{kernelFile, initrdFile, cmdlineFile} {
		require.NoError(t, os.WriteFile(f, []byte("fake"), 0644))
	}

	err := assembleRawBIOS(t.Context(), kernelFile, initrdFile, cmdlineFile, outputFile, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create cache directory")
}
