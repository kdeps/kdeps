package archiver

import (
	"testing"

	"github.com/spf13/afero"
)

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
