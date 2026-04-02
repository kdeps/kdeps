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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
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

// ---- linuxkitCacheDir tests ----

func TestLinuxkitCacheDir(t *testing.T) {
	dir, err := linuxkitCacheDir()
	if err != nil {
		t.Fatalf("linuxkitCacheDir returned error: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty cache dir")
	}
	// Must be under the user's home directory.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("cannot determine home dir: %v", err)
	}
	if !strings.HasPrefix(dir, home) {
		t.Errorf("cache dir %q is not under home %q", dir, home)
	}
}

// ---- LinuxKitDownloadURL tests ----

func TestLinuxKitDownloadURL(t *testing.T) {
	url := LinuxKitDownloadURL()
	if url == "" {
		t.Fatal("expected non-empty download URL")
	}
	if !strings.Contains(url, linuxkitVersion) {
		t.Errorf("URL %q does not contain version %q", url, linuxkitVersion)
	}
	if !strings.Contains(url, runtime.GOOS) {
		t.Errorf("URL %q does not contain GOOS %q", url, runtime.GOOS)
	}
	if !strings.Contains(url, runtime.GOARCH) {
		t.Errorf("URL %q does not contain GOARCH %q", url, runtime.GOARCH)
	}
}

// ---- downloadFile tests ----

func TestDownloadFile_Success(t *testing.T) {
	content := []byte("binary content")
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer ts.Close()

	dest := filepath.Join(t.TempDir(), "downloaded")
	if err := downloadFile(t.Context(), ts.URL, dest); err != nil {
		t.Fatalf("downloadFile failed: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestDownloadFile_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer ts.Close()

	dest := filepath.Join(t.TempDir(), "should-not-exist")
	err := downloadFile(t.Context(), ts.URL, dest)
	if err == nil {
		t.Fatal("expected error for HTTP 404, got nil")
	}
}

func TestDownloadFile_InvalidURL(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "out")
	err := downloadFile(t.Context(), "://invalid-url", dest)
	if err == nil {
		t.Fatal("expected error for invalid URL, got nil")
	}
}
