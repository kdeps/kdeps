package docker

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

func TestCreateFlagFile_ReadOnlyFs(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "roflag")
	if err != nil {
		t.Fatalf("TempDir: %v", err)
	}

	ro := afero.NewReadOnlyFs(fs)
	flagPath := filepath.Join(tmpDir, "flag.txt")

	// Attempting to create a new file on read-only FS should error.
	if err := CreateFlagFile(ro, context.Background(), flagPath); err == nil {
		t.Fatalf("expected error when creating flag file on read-only fs")
	}

	// Reference schema version (requirement in tests)
	_ = schema.SchemaVersion(context.Background())
}
