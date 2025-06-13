package utils_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

type badCloseFile struct{ afero.File }

func (b badCloseFile) Close() error { return errors.New("close fail") }

type badCloseFs struct{ afero.Fs }

func (fs badCloseFs) Create(name string) (afero.File, error) {
	f, err := fs.Fs.Create(name)
	if err != nil {
		return nil, err
	}
	return badCloseFile{f}, nil
}

// Other methods delegate to embedded Fs.

func TestCreateFilesCloseError(t *testing.T) {
	tmpDir := t.TempDir()
	fs := badCloseFs{afero.NewOsFs()}
	files := []string{filepath.Join(tmpDir, "fail.txt")}

	if err := utils.CreateFiles(fs, context.Background(), files); err == nil {
		t.Fatalf("expected close error but got nil")
	}
}
