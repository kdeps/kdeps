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

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// ---- writeAssembleScript tests ----

func TestWriteAssembleScript_WithoutDataPartition(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "assemble.sh")

	if err := writeAssembleScript(path, tmpDir, false); err != nil {
		t.Fatalf("writeAssembleScript failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read script: %v", err)
	}

	if len(content) == 0 {
		t.Error("expected non-empty script content")
	}

	// Script should contain the shebang
	if string(content[:10]) != "#!/bin/sh\n" {
		t.Errorf("expected script to start with shebang, got: %q", string(content[:10]))
	}
}

func TestWriteAssembleScript_WithDataPartition(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "assemble.sh")

	if err := writeAssembleScript(path, tmpDir, true); err != nil {
		t.Fatalf("writeAssembleScript failed: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read script: %v", err)
	}

	// Both with and without data partition produce same script (bool arg ignored in impl)
	if len(data) == 0 {
		t.Error("expected non-empty script content")
	}
}

func TestWriteAssembleScript_NonexistentDir(t *testing.T) {
	path := "/nonexistent/dir/assemble.sh"
	err := writeAssembleScript(path, "/nonexistent/dir", false)
	if err == nil {
		t.Error("expected error when writing to nonexistent directory, got nil")
	}
}

// ---- copyFile tests ----

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

// ---- findLinuxKitOutput tests ----

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

// ---- GenerateConfigYAMLExtended tests ----

func TestGenerateConfigYAMLExtended_Thin(t *testing.T) {
	b := &Builder{
		Hostname: "test-host",
		Format:   "raw-bios",
		Arch:     "amd64",
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "thin-test",
			Version: "1.0.0",
		},
	}

	yaml, err := b.GenerateConfigYAMLExtended("my-image:latest", workflow, true)
	if err != nil {
		t.Fatalf("GenerateConfigYAMLExtended failed: %v", err)
	}

	if len(yaml) == 0 {
		t.Error("expected non-empty YAML output")
	}

	// Thin builds should contain mount-data and import-image steps
	if !strings.Contains(yaml, "mount-data") {
		t.Error("expected thin build YAML to contain 'mount-data'")
	}

	if !strings.Contains(yaml, "import-image") {
		t.Error("expected thin build YAML to contain 'import-image'")
	}
}

func TestGenerateConfigYAMLExtended_Fat(t *testing.T) {
	b := &Builder{
		Hostname: "test-host",
		Format:   "iso-efi",
		Arch:     "amd64",
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "fat-test",
			Version: "1.0.0",
		},
	}

	yaml, err := b.GenerateConfigYAMLExtended("my-image:latest", workflow, false)
	if err != nil {
		t.Fatalf("GenerateConfigYAMLExtended failed: %v", err)
	}

	if len(yaml) == 0 {
		t.Error("expected non-empty YAML output")
	}
}

func TestGenerateConfigYAMLExtended_NilWorkflow(t *testing.T) {
	b := &Builder{Hostname: "test-host"}

	_, err := b.GenerateConfigYAMLExtended("my-image:latest", nil, false)
	if err == nil {
		t.Error("expected error for nil workflow, got nil")
	}
}

func TestGenerateConfigYAMLExtended_EmptyHostname(t *testing.T) {
	b := &Builder{} // empty hostname → should default

	workflow := &domain.Workflow{}

	_, err := b.GenerateConfigYAMLExtended("my-image:latest", workflow, false)
	if err != nil {
		t.Fatalf("expected no error with empty hostname, got: %v", err)
	}
}
