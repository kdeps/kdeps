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
	"testing"
)

func TestFindLinuxKitOutput_MatchingExtension(t *testing.T) {
	tmpDir := t.TempDir()

	isoFile := filepath.Join(tmpDir, "output.iso")
	if err := os.WriteFile(isoFile, []byte("iso content"), 0644); err != nil {
		t.Fatalf("failed to create iso file: %v", err)
	}

	result, err := findLinuxKitOutput(tmpDir, "iso-efi")
	if err != nil {
		t.Fatalf("findLinuxKitOutput failed: %v", err)
	}

	if result != isoFile {
		t.Errorf("expected %s, got %s", isoFile, result)
	}
}

func TestFindLinuxKitOutput_FallbackToFirstFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file with non-matching extension
	otherFile := filepath.Join(tmpDir, "output.bin")
	if err := os.WriteFile(otherFile, []byte("bin content"), 0644); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	// Use a format that expects .iso but we have .bin - should fall back to first file
	result, err := findLinuxKitOutput(tmpDir, "iso-efi")
	if err != nil {
		t.Fatalf("findLinuxKitOutput failed: %v", err)
	}

	if result != otherFile {
		t.Errorf("expected fallback to %s, got %s", otherFile, result)
	}
}

func TestFindLinuxKitOutput_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := findLinuxKitOutput(tmpDir, "iso-efi")
	if err == nil {
		t.Error("expected error for empty directory, got nil")
	}
}

func TestFindLinuxKitOutput_NonexistentDir(t *testing.T) {
	_, err := findLinuxKitOutput("/nonexistent/build/dir", "iso-efi")
	if err == nil {
		t.Error("expected error for nonexistent directory, got nil")
	}
}

func TestFindLinuxKitOutput_OnlySubdirs(t *testing.T) {
	tmpDir := t.TempDir()

	// Create only a subdirectory, no files
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	_, err := findLinuxKitOutput(tmpDir, "iso-efi")
	if err == nil {
		t.Error("expected error when no files in dir (only subdirs), got nil")
	}
}
