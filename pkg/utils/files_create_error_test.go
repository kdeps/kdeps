package utils

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

// failCreateFs returns error on Create to hit the error branch inside CreateFiles.
type failCreateFs struct{ afero.Fs }

func (f failCreateFs) Create(name string) (afero.File, error) {
	return nil, errors.New("create error")
}

func TestCreateFiles_CreateError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := failCreateFs{afero.NewOsFs()}
	files := []string{filepath.Join(tmpDir, "cannot.txt")}
	err := CreateFiles(fs, context.Background(), files)
	if err == nil {
		t.Fatalf("expected error from CreateFiles when underlying fs.Create fails")
	}
}
