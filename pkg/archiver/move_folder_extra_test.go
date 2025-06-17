package archiver

import (
	"testing"

	"context"
	"path/filepath"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestMoveFolderMemFS verifies that MoveFolder correctly copies all files from
// the source directory to the destination and removes the original source
// directory when using an in-memory filesystem.
func TestMoveFolderMemFS(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create source directory with nested file
	srcDir := "/src"
	destDir := "/dst"
	if err := fs.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	filePath := srcDir + "/file.txt"
	if err := afero.WriteFile(fs, filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Execute MoveFolder
	if err := MoveFolder(fs, srcDir, destDir); err != nil {
		t.Fatalf("MoveFolder returned error: %v", err)
	}

	// Source directory should no longer exist
	if exists, _ := afero.DirExists(fs, srcDir); exists {
		t.Fatalf("expected source directory to be removed")
	}

	// Destination file should exist with correct contents
	movedFile := destDir + "/file.txt"
	data, err := afero.ReadFile(fs, movedFile)
	if err != nil {
		t.Fatalf("read moved file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %s", data)
	}
}

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

// TestMoveFolderSuccessMemFS ensures MoveFolder copies files and removes src.
func TestMoveFolderSuccessMemFS(t *testing.T) {
	fs := afero.NewMemMapFs()

	srcDir := "/srcDir"
	dstDir := "/dstDir"
	_ = fs.MkdirAll(srcDir, 0o755)

	// create two files in nested structure.
	_ = afero.WriteFile(fs, srcDir+"/f1.txt", []byte("a"), 0o644)
	_ = fs.MkdirAll(srcDir+"/sub", 0o755)
	_ = afero.WriteFile(fs, srcDir+"/sub/f2.txt", []byte("b"), 0o640)

	if err := MoveFolder(fs, srcDir, dstDir); err != nil {
		t.Fatalf("MoveFolder error: %v", err)
	}

	// original srcDir should be removed
	if exists, _ := afero.DirExists(fs, srcDir); exists {
		t.Fatalf("expected source dir removed")
	}

	// destination files should exist with correct content
	data1, _ := afero.ReadFile(fs, dstDir+"/f1.txt")
	if string(data1) != "a" {
		t.Fatalf("dst f1 content mismatch")
	}
	data2, _ := afero.ReadFile(fs, dstDir+"/sub/f2.txt")
	if string(data2) != "b" {
		t.Fatalf("dst f2 content mismatch")
	}
}
