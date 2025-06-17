package utils

import (
	"context"
	"testing"

	"github.com/spf13/afero"
)

func TestGenerateResourceIDFilenameMore(t *testing.T) {
	got := GenerateResourceIDFilename("@agent/data:1.0.0", "req-")
	if got != "req-_agent_data_1.0.0" {
		t.Fatalf("unexpected filename: %s", got)
	}
}

func TestCreateDirectoriesAndFilesMore(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	dirs := []string{"/a/b/c"}
	if err := CreateDirectories(fs, ctx, dirs); err != nil {
		t.Fatalf("CreateDirectories error: %v", err)
	}
	if ok, _ := afero.DirExists(fs, "/a/b/c"); !ok {
		t.Fatalf("directory not created")
	}

	files := []string{"/a/b/c/file.txt"}
	if err := CreateFiles(fs, ctx, files); err != nil {
		t.Fatalf("CreateFiles error: %v", err)
	}
	if ok, _ := afero.Exists(fs, files[0]); !ok {
		t.Fatalf("file not created")
	}
}

func TestSanitizeArchivePathMore(t *testing.T) {
	p, err := SanitizeArchivePath("/safe", "sub/dir.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p != "/safe/sub/dir.txt" {
		t.Fatalf("unexpected sanitized path: %s", p)
	}

	// attempt path traversal
	if _, err := SanitizeArchivePath("/safe", "../evil.txt"); err == nil {
		t.Fatalf("expected error for tainted path")
	}
}
