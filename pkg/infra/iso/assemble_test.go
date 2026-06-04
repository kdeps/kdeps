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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
	// Create a test HTTP server that serves binary content.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("binary content"))
	}))
	defer ts.Close()

	tmpDir := t.TempDir()

	// Make the destination path an existing directory so os.Rename fails.
	dest := filepath.Join(tmpDir, "dest")
	if err := os.Mkdir(dest, 0755); err != nil {
		t.Fatalf("failed to create dest dir: %v", err)
	}

	// downloadFile writes to dest.tmp, then tries to rename to dest.
	// Since dest is a directory, the rename on Unix returns EISDIR.
	err := downloadFile(t.Context(), ts.URL, dest)
	if err == nil {
		t.Fatal("expected error when dest is a directory, got nil")
	}
	if !strings.Contains(err.Error(), "rename") {
		t.Errorf("expected rename error, got: %v", err)
	}
}

// makeWorkDirReadOnly spawns a goroutine that watches for a rawbios-* workDir
// inside HOME/.cache/kdeps, then chmods it to 0555 when the docker-save tar
// file appears. Returns a channel that receives the workDir path (for cleanup).
//
//nolint:unused
func makeWorkDirReadOnly(tmpDir string) chan string {
	ch := make(chan string, 1)
	cacheDir := filepath.Join(tmpDir, ".cache", "kdeps")
	go func() {
		for range 200 {
			entries, err := os.ReadDir(cacheDir)
			if err != nil {
				time.Sleep(10 * time.Millisecond)
				continue
			}
			for _, e := range entries {
				if !strings.HasPrefix(e.Name(), "rawbios-") || !e.IsDir() {
					continue
				}
				wd := filepath.Join(cacheDir, e.Name())
				if _, err = os.Stat(filepath.Join(wd, "image.tar")); err == nil {
					_ = os.Chmod(wd, 0555)
					ch <- wd
					return
				}
			}
			time.Sleep(10 * time.Millisecond)
		}
		close(ch)
	}()
	return ch
}

// setupFakeDocker writes a fake docker script to tmpDir/docker and prepends
// tmpDir to PATH. The fake docker succeeds for "save" (with a 300ms delay)
// and fails for any other command.
//
//nolint:unused
func setupFakeDocker(t *testing.T, tmpDir string) {
	t.Helper()
	fakeDocker := filepath.Join(tmpDir, "docker")
	script := `#!/bin/sh
if [ "$1" = "save" ]; then
    touch "$3"
    sleep 0.3
    exit 0
fi
exit 1
`
	if err := os.WriteFile(fakeDocker, []byte(script), 0755); err != nil {
		t.Fatalf("failed to create fake docker: %v", err)
	}
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })
}

// createAssembleTestInputs writes kernel, initrd, and cmdline files to tmpDir.
//
//nolint:unused
func createAssembleTestInputs(
	t *testing.T,
	tmpDir string,
) (string, string, string, string) {
	t.Helper()
	kernel := filepath.Join(tmpDir, "kernel")
	initrd := filepath.Join(tmpDir, "initrd.img")
	cmdline := filepath.Join(tmpDir, "cmdline")
	output := filepath.Join(tmpDir, "disk.img")
	for _, f := range []string{kernel, initrd, cmdline} {
		if err := os.WriteFile(f, []byte("fake"), 0644); err != nil {
			t.Fatalf("failed to create input file %s: %v", f, err)
		}
	}
	return kernel, initrd, cmdline, output
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
