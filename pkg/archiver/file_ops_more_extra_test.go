package archiver

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestCopyFileSuccess(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "nested", "dst.txt")
	content := []byte("lorem ipsum")

	if err := afero.WriteFile(fs, src, content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := fs.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	if err := copyFile(fs, src, dst); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	data, err := afero.ReadFile(fs, dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != string(content) {
		t.Fatalf("content mismatch")
	}
}

func TestMoveFolderNested(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()

	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")

	// create deep hierarchy
	paths := []string{
		filepath.Join(src, "a", "b"),
		filepath.Join(src, "c"),
	}
	for _, p := range paths {
		if err := fs.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	if err := afero.WriteFile(fs, filepath.Join(src, "a", "b", "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := MoveFolder(fs, src, dest); err != nil {
		t.Fatalf("MoveFolder: %v", err)
	}

	// dest should now contain same hierarchy
	if ok, _ := afero.DirExists(fs, filepath.Join(dest, "a", "b")); !ok {
		t.Fatalf("nested dir not moved")
	}

	// src should be removed entirely
	if ok, _ := afero.DirExists(fs, src); ok {
		t.Fatalf("src dir still exists")
	}
}
