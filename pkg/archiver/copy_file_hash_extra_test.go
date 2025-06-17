package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestCopyFileSkipIfHashesMatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	src := "/src.txt"
	dst := "/dst.txt"
	content := []byte("same")
	if err := afero.WriteFile(fs, src, content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	// Copy initial file to dst so hashes match
	if err := afero.WriteFile(fs, dst, content, 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}
}

func TestCopyFileCreatesBackupOnHashMismatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	src := "/src2.txt"
	dst := "/dst2.txt"

	if err := afero.WriteFile(fs, src, []byte("new"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := afero.WriteFile(fs, dst, []byte("old"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// backup should exist
	files, _ := afero.ReadDir(fs, "/")
	foundBackup := false
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".txt" && f.Name() != "src2.txt" && f.Name() != "dst2.txt" {
			foundBackup = true
		}
	}
	if !foundBackup {
		t.Fatalf("expected backup file to be created")
	}
}
