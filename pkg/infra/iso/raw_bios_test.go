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
	"testing"
)

// Test findKernelInitrd with valid files.
func TestFindKernelInitrd_Success(t *testing.T) {
	// Create temporary directory with expected files
	tmpDir := t.TempDir()

	// Create fake kernel, initrd, and cmdline files
	kernelFile := filepath.Join(tmpDir, "test-kernel")
	initrdFile := filepath.Join(tmpDir, "test-initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "test-cmdline")

	for _, file := range []string{kernelFile, initrdFile, cmdlineFile} {
		if err := os.WriteFile(file, []byte("fake content"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	// Test the function
	kernel, initrd, cmdline, err := findKernelInitrd(tmpDir)

	if err != nil {
		t.Fatalf("findKernelInitrd failed: %v", err)
	}

	if kernel != kernelFile {
		t.Errorf("Expected kernel %s, got %s", kernelFile, kernel)
	}

	if initrd != initrdFile {
		t.Errorf("Expected initrd %s, got %s", initrdFile, initrd)
	}

	if cmdline != cmdlineFile {
		t.Errorf("Expected cmdline %s, got %s", cmdlineFile, cmdline)
	}
}

// Test findKernelInitrd with missing kernel.
func TestFindKernelInitrd_MissingKernel(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only initrd and cmdline
	initrdFile := filepath.Join(tmpDir, "test-initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "test-cmdline")

	for _, file := range []string{initrdFile, cmdlineFile} {
		if err := os.WriteFile(file, []byte("fake"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Should fail due to missing kernel
	_, _, _, err := findKernelInitrd(tmpDir)

	if err == nil {
		t.Error("Expected error for missing kernel, got nil")
	}
}

// Test findKernelInitrd with missing initrd.
func TestFindKernelInitrd_MissingInitrd(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only kernel and cmdline
	kernelFile := filepath.Join(tmpDir, "test-kernel")
	cmdlineFile := filepath.Join(tmpDir, "test-cmdline")

	for _, file := range []string{kernelFile, cmdlineFile} {
		if err := os.WriteFile(file, []byte("fake"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Should fail due to missing initrd
	_, _, _, err := findKernelInitrd(tmpDir)

	if err == nil {
		t.Error("Expected error for missing initrd, got nil")
	}
}

// Test findKernelInitrd with missing cmdline.
func TestFindKernelInitrd_MissingCmdline(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only kernel and initrd
	kernelFile := filepath.Join(tmpDir, "test-kernel")
	initrdFile := filepath.Join(tmpDir, "test-initrd.img")

	for _, file := range []string{kernelFile, initrdFile} {
		if err := os.WriteFile(file, []byte("fake"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Should fail due to missing cmdline
	_, _, _, err := findKernelInitrd(tmpDir)

	if err == nil {
		t.Error("Expected error for missing cmdline, got nil")
	}
}

// Test findKernelInitrd with nonexistent directory.
func TestFindKernelInitrd_NonexistentDir(t *testing.T) {
	_, _, _, err := findKernelInitrd("/nonexistent/directory")

	if err == nil {
		t.Error("Expected error for nonexistent directory, got nil")
	}
}

// Mock RawBIOSAssembleFunc for testing.
func mockAssembleFunc(
	_ context.Context,
	_, _, _, outputPath, _, _ string,
) error {
	// Create a fake output file
	return os.WriteFile(outputPath, []byte("fake disk image"), 0644)
}

// Mock LinuxKitRunner for testing.
type mockLinuxKitRunner struct {
	buildErr error
}

func (m *mockLinuxKitRunner) Build(_ context.Context, _, _, _, buildDir, _ string) error {
	if m.buildErr != nil {
		return m.buildErr
	}

	// Create fake kernel, initrd, and cmdline files
	kernelFile := filepath.Join(buildDir, "test-kernel")
	initrdFile := filepath.Join(buildDir, "test-initrd.img")
	cmdlineFile := filepath.Join(buildDir, "test-cmdline")

	for _, file := range []string{kernelFile, initrdFile, cmdlineFile} {
		if err := os.WriteFile(file, []byte("fake"), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (m *mockLinuxKitRunner) CacheImport(_ context.Context, _ string) error {
	return nil
}

// Test buildRawBIOSWithImage with successful build.
func TestBuildRawBIOSWithImage_Success(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	// Create a fake config file
	if err := os.WriteFile(configPath, []byte("fake config"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	runner := &mockLinuxKitRunner{}

	outputPath, err := buildRawBIOSWithImage(
		context.Background(),
		runner,
		mockAssembleFunc,
		configPath,
		"amd64",
		tmpDir,
		"test-image",
		"",
	)

	if err != nil {
		t.Fatalf("buildRawBIOSWithImage failed: %v", err)
	}

	expectedOutput := filepath.Join(tmpDir, "disk.img")
	if outputPath != expectedOutput {
		t.Errorf("Expected output %s, got %s", expectedOutput, outputPath)
	}

	// Verify output file was created
	if _, statErr := os.Stat(outputPath); os.IsNotExist(statErr) {
		t.Error("Output file was not created")
	}
}

// Test buildRawBIOSWithImage with linuxkit build failure.
func TestBuildRawBIOSWithImage_LinuxKitBuildFailure(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")

	if err := os.WriteFile(configPath, []byte("fake config"), 0644); err != nil {
		t.Fatalf("Failed to create config file: %v", err)
	}

	runner := &mockLinuxKitRunner{
		buildErr: os.ErrNotExist,
	}

	_, err := buildRawBIOSWithImage(
		context.Background(),
		runner,
		mockAssembleFunc,
		configPath,
		"amd64",
		tmpDir,
		"test-image",
		"",
	)

	if err == nil {
		t.Error("Expected error from linuxkit build failure, got nil")
	}
}
