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
	"time"
)

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

type mockRunnerForCoverage struct{}

func (m *mockRunnerForCoverage) Build(_ context.Context, _, _, _, _, _ string) error {
	return nil
}

func (m *mockRunnerForCoverage) CacheImport(_ context.Context, _ string) error {
	return nil
}

type mockRawBIOSRunner struct{}

func (m *mockRawBIOSRunner) Build(
	_ context.Context,
	configPath, format, _, outputDir, _ string,
) error {
	if format != "kernel+initrd" {
		return nil
	}

	base := filepath.Base(configPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	files := map[string]string{
		base + "-kernel":     "fake-kernel",
		base + "-initrd.img": "fake-initrd",
		base + "-cmdline":    "console=ttyS0",
	}
	for name, content := range files {
		if writeErr := os.WriteFile(filepath.Join(outputDir, name), []byte(content), 0644); writeErr != nil {
			return writeErr
		}
	}
	return nil
}

func (m *mockRawBIOSRunner) CacheImport(_ context.Context, _ string) error {
	return nil
}
