package docker

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/afero"
)

func TestCreateFlagFile_NewFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	filename := "test_flag_file"

	if err := CreateFlagFile(fs, ctx, filename); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, _ := afero.Exists(fs, filename)
	if !exists {
		t.Fatalf("expected flag file to be created")
	}

	// Check timestamps roughly current (within 2 seconds)
	info, _ := fs.Stat(filename)
	if time.Since(info.ModTime()) > 2*time.Second {
		t.Fatalf("mod time too old: %v", info.ModTime())
	}
}

func TestCreateFlagFile_FileAlreadyExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	filename := "existing_flag"

	// pre-create file
	afero.WriteFile(fs, filename, []byte{}, 0o644)

	if err := CreateFlagFile(fs, ctx, filename); err != nil {
		t.Fatalf("expected no error when file already exists, got: %v", err)
	}
}
