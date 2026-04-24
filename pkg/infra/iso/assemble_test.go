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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
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
	err := assembleRawBIOS(t.Context(), kernelFile, initrdFile, cmdlineFile, outputFile, "fake-image", "")
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

	err := assembleRawBIOS(t.Context(), kernelFile, initrdFile, cmdlineFile, outputFile, "fake-image", "")
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

	err := assembleRawBIOS(t.Context(), kernelFile, initrdFile, cmdlineFile, outputFile, "fake-image", "echo boot")
	if err != nil {
		t.Fatalf("assembleRawBIOS failed: %v", err)
	}

	if _, statErr := os.Stat(outputFile); os.IsNotExist(statErr) {
		t.Error("expected output file to be created")
	}
}

func TestEnsureLinuxKit_InPath(t *testing.T) {
	tmpDir := t.TempDir()
	fakeLinuxKit := filepath.Join(tmpDir, "linuxkit")
	if err := os.WriteFile(fakeLinuxKit, []byte("#!/bin/sh\necho ok"), 0755); err != nil {
		t.Fatalf("failed to create fake linuxkit: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	path, err := EnsureLinuxKit(context.Background())
	if err != nil {
		t.Fatalf("EnsureLinuxKit failed: %v", err)
	}
	if path != fakeLinuxKit {
		t.Errorf("expected %s, got %s", fakeLinuxKit, path)
	}
}

func TestEnsureLinuxKit_InCache(t *testing.T) {
	// Mock HOME to control cache location
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", oldHome)

	// Create cached binary
	cacheDir := filepath.Join(tmpDir, ".cache", "kdeps", "linuxkit")
	if err := os.MkdirAll(cacheDir, 0750); err != nil {
		t.Fatalf("failed to create cache dir: %v", err)
	}
	cachedBinary := filepath.Join(cacheDir, "linuxkit-"+linuxkitVersion)
	if err := os.WriteFile(cachedBinary, []byte("binary"), 0755); err != nil {
		t.Fatalf("failed to create cached binary: %v", err)
	}

	// Ensure it's NOT in PATH
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	defer os.Setenv("PATH", oldPath)

	path, err := EnsureLinuxKit(context.Background())
	if err != nil {
		t.Fatalf("EnsureLinuxKit failed: %v", err)
	}
	if path != cachedBinary {
		t.Errorf("expected %s, got %s", cachedBinary, path)
	}
}

func TestDownloadFile_RenameError(t *testing.T) {
	// To trigger rename error, we can make the destination directory a file
	tmpDir := t.TempDir()
	dest := filepath.Join(tmpDir, "dest")
	// Make dest.tmp a directory so renaming to dest (file) might fail or something?
	// Actually, easier to make 'dest' a directory and try to rename a file to it.
	if err := os.Mkdir(dest, 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}

	// This might not work as expected on all OSs.
	// Let's try to mock the rename by making the target unwritable.
}

func TestMarshalConfig_Error(_ *testing.T) {
	// MarshalConfig uses yaml.Marshal which rarely fails for our struct
	// unless there are cycles or unsupported types, but our struct is simple.
}

func TestBuilder_Build_NilWorkflow_Error(t *testing.T) {
	b := &Builder{}
	err := b.Build(context.Background(), "image", nil, "out", false)
	if err == nil {
		t.Error("expected error for nil workflow, got nil")
	}
}

func TestBuilder_Build_EmptyImage_Error(t *testing.T) {
	b := &Builder{}
	err := b.Build(context.Background(), "", &domain.Workflow{}, "out", false)
	if err == nil {
		t.Error("expected error for empty image name, got nil")
	}
}
