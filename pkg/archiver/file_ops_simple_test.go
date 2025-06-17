package archiver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestMoveFolderAndCopyFileSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// setup source directory with one file
	srcDir := "/src"
	dstDir := "/dst"
	_ = fs.MkdirAll(srcDir, 0o755)
	srcFile := srcDir + "/file.txt"
	_ = afero.WriteFile(fs, srcFile, []byte("data"), 0o644)

	// MoveFolder
	if err := MoveFolder(fs, srcDir, dstDir); err != nil {
		t.Fatalf("MoveFolder error: %v", err)
	}
	// Original dir should not exist
	if exists, _ := afero.Exists(fs, srcDir); exists {
		t.Fatalf("src dir still exists after move")
	}
	// Destination file should exist
	if exists, _ := afero.Exists(fs, dstDir+"/file.txt"); !exists {
		t.Fatalf("file not moved to dst")
	}

	// Test CopyFile idempotent path (same content)
	newFile := dstDir + "/copy.txt"
	if err := CopyFile(fs, ctx, dstDir+"/file.txt", newFile, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}
	// Copying again should detect same MD5 and skip
	if err := CopyFile(fs, ctx, dstDir+"/file.txt", newFile, logger); err != nil {
		t.Fatalf("CopyFile second error: %v", err)
	}
}
