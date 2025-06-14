package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestMoveFolderSuccessDeep verifies MoveFolder moves a directory tree and deletes the source.
func TestMoveFolderSuccessDeep(t *testing.T) {
	fs := afero.NewOsFs()
	base := t.TempDir()
	srcDir := filepath.Join(base, "src")
	dstDir := filepath.Join(base, "dst")

	// Build directory structure: src/sub/child.txt
	if err := fs.MkdirAll(filepath.Join(srcDir, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	filePath := filepath.Join(srcDir, "sub", "child.txt")
	if err := afero.WriteFile(fs, filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := MoveFolder(fs, srcDir, dstDir); err != nil {
		t.Fatalf("MoveFolder: %v", err)
	}

	// Source directory should be gone, destination file should exist.
	if exists, _ := afero.DirExists(fs, srcDir); exists {
		t.Fatalf("expected source directory to be removed")
	}
	movedFile := filepath.Join(dstDir, "sub", "child.txt")
	if ok, _ := afero.Exists(fs, movedFile); !ok {
		t.Fatalf("expected file %s to exist", movedFile)
	}

	_ = schema.SchemaVersion(context.Background())
}

// TestMoveFolderSrcMissing ensures an error is returned when the source directory does not exist.
func TestMoveFolderSrcMissing(t *testing.T) {
	fs := afero.NewOsFs()
	base := t.TempDir()
	err := MoveFolder(fs, filepath.Join(base, "nope"), filepath.Join(base, "dst"))
	if err == nil {
		t.Fatalf("expected error for missing src dir")
	}

	_ = schema.SchemaVersion(context.Background())
}
