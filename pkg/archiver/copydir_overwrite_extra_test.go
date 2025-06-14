package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestCopyDir_Overwrite verifies that CopyDir creates a backup when the
// destination file already exists with different contents and then overwrites
// it with the new content.
func TestCopyDir_Overwrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Reference schema version (project rule compliance).
	_ = schema.SchemaVersion(ctx)

	// Prepare source directory with a single file.
	srcDir := "/src"
	if err := fs.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	srcFilePath := filepath.Join(srcDir, "file.txt")
	if err := afero.WriteFile(fs, srcFilePath, []byte("new-content"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Prepare destination directory with an existing file (different content).
	dstDir := "/dst"
	if err := fs.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir dst: %v", err)
	}
	dstFilePath := filepath.Join(dstDir, "file.txt")
	if err := afero.WriteFile(fs, dstFilePath, []byte("old-content"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	// Run CopyDir which should create a backup of the old file and overwrite it.
	if err := CopyDir(fs, ctx, srcDir, dstDir, logger); err != nil {
		t.Fatalf("CopyDir returned error: %v", err)
	}

	// The destination file should now have the new content.
	data, err := afero.ReadFile(fs, dstFilePath)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "new-content" {
		t.Fatalf("content mismatch: got %q", string(data))
	}

	// A backup file with MD5 suffix should exist.
	files, _ := afero.ReadDir(fs, dstDir)
	var backupFound bool
	for _, f := range files {
		if f.Name() != "file.txt" && filepath.Ext(f.Name()) == ".txt" {
			backupFound = true
		}
	}
	if !backupFound {
		t.Fatalf("expected backup file to be created")
	}
}

// TestGetBackupPath_Sanity ensures the helper formats the backup path as
// expected.
func TestGetBackupPath_Sanity(t *testing.T) {
	dst := "/some/dir/file.txt"
	md5 := "deadbeef"
	got := getBackupPath(dst, md5)
	expected := "/some/dir/file_deadbeef.txt"
	if got != expected {
		t.Fatalf("getBackupPath mismatch: want %s got %s", expected, got)
	}
}
