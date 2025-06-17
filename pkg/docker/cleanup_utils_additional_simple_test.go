package docker

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestCleanupFlagFilesSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create temporary files
	files := []string{"/tmp/file1.flag", "/tmp/file2.flag", "/tmp/file3.flag"}
	for _, f := range files {
		if err := afero.WriteFile(fs, f, []byte("data"), 0o644); err != nil {
			t.Fatalf("unable to create temp file: %v", err)
		}
	}

	cleanupFlagFiles(fs, files, logger)

	// Verify they are removed
	for _, f := range files {
		if _, err := fs.Stat(f); err == nil {
			t.Fatalf("expected file %s to be removed", f)
		}
	}
}
