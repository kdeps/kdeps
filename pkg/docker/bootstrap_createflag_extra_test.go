package docker

import (
	"context"
	"testing"

	"github.com/spf13/afero"
)

func TestCreateFlagFileNoDuplicate(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	filename := "/tmp/flag.txt"

	// First creation should succeed and file should exist.
	if err := CreateFlagFile(fs, ctx, filename); err != nil {
		t.Fatalf("CreateFlagFile error: %v", err)
	}
	if ok, _ := afero.Exists(fs, filename); !ok {
		t.Fatalf("expected file to exist after creation")
	}

	// Second creation should be no-op with no error (file already exists).
	if err := CreateFlagFile(fs, ctx, filename); err != nil {
		t.Fatalf("expected no error on second create, got %v", err)
	}
}
