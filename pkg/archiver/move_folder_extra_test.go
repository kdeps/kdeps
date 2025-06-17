package archiver

import (
	"testing"

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
