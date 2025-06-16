package archiver

import (
	"errors"
	"io/fs"
	"testing"

	"github.com/spf13/afero"
)

func TestMoveFolder_Success(t *testing.T) {
	mem := afero.NewMemMapFs()

	// Setup source directory with nested files
	_ = mem.MkdirAll("/src/sub", 0o755)
	afero.WriteFile(mem, "/src/file1.txt", []byte("one"), 0o644)
	afero.WriteFile(mem, "/src/sub/file2.txt", []byte("two"), 0o644)

	if err := MoveFolder(mem, "/src", "/dst"); err != nil {
		t.Fatalf("MoveFolder returned error: %v", err)
	}

	// Source should be removed
	if exists, _ := afero.Exists(mem, "/src"); exists {
		t.Fatalf("source directory still exists after MoveFolder")
	}

	// Destination files should exist with same content
	data, _ := afero.ReadFile(mem, "/dst/file1.txt")
	if string(data) != "one" {
		t.Fatalf("file1 content mismatch: %s", data)
	}
	data, _ = afero.ReadFile(mem, "/dst/sub/file2.txt")
	if string(data) != "two" {
		t.Fatalf("file2 content mismatch: %s", data)
	}
}

func TestMoveFolder_NonexistentSource(t *testing.T) {
	mem := afero.NewMemMapFs()
	err := MoveFolder(mem, "/no-such", "/dst")
	if err == nil {
		t.Fatalf("expected error when source does not exist")
	}
	// Ensure destination not created
	if _, statErr := mem.Stat("/dst"); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("destination directory should not exist when move fails")
	}
}
