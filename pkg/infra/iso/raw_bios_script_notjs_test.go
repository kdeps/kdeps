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
