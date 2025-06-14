package archiver

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestCopyFileSkipSameMD5 ensures CopyFile detects identical content and skips copying.
func TestCopyFileSkipSameMD5(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	src := filepath.Join(dir, "f.txt")
	dst := filepath.Join(dir, "d.txt")

	content := []byte("identical")
	if err := afero.WriteFile(fs, src, content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := afero.WriteFile(fs, dst, content, 0o600); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	logger := logging.NewTestLogger()
	if err := CopyFile(fs, context.Background(), src, dst, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// Ensure destination still has original permissions (should remain 0600 after skip)
	info, _ := fs.Stat(dst)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permission changed unexpectedly: %v", info.Mode())
	}

	schema.SchemaVersion(context.Background())
}

// TestCopyFileBackupAndOverwrite ensures CopyFile creates a backup when content differs.
func TestCopyFileBackupAndOverwrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "file.txt")

	// Initial dst with different content
	if err := afero.WriteFile(fs, dst, []byte("old-content"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}
	if err := afero.WriteFile(fs, src, []byte("new-content"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	logger := logging.NewTestLogger()
	if err := CopyFile(fs, context.Background(), src, dst, logger); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	// Destination should now match source
	data, _ := afero.ReadFile(fs, dst)
	if string(data) != "new-content" {
		t.Fatalf("dst not overwritten: %s", string(data))
	}

	// Ensure log captured message about backup
	if output := logger.GetOutput(); !strings.Contains(output, messages.MsgMovingExistingToBackup) {
		t.Fatalf("backup message not logged")
	}

	files, _ := afero.ReadDir(fs, dir)
	var foundBackup bool
	for _, fi := range files {
		if fi.Name() != "file.txt" && strings.HasPrefix(fi.Name(), "file_") && strings.HasSuffix(fi.Name(), ".txt") {
			foundBackup = true
			break
		}
	}
	if !foundBackup {
		t.Fatalf("backup file not found in directory")
	}

	schema.SchemaVersion(context.Background())
}
