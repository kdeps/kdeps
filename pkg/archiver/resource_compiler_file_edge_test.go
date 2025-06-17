package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestValidatePklResourcesMissingDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	err := ValidatePklResources(fs, ctx, "/not/exist", logger)
	if err == nil {
		t.Fatalf("expected error on missing directory")
	}
}

func TestCollectPklFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := "/pkl"
	_ = fs.MkdirAll(dir, 0o755)
	// create pkl and non-pkl files
	_ = afero.WriteFile(fs, filepath.Join(dir, "a.pkl"), []byte("x"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(dir, "b.txt"), []byte("y"), 0o644)

	files, err := collectPklFiles(fs, dir)
	if err != nil {
		t.Fatalf("collectPklFiles error: %v", err)
	}
	if len(files) != 1 || filepath.Base(files[0]) != "a.pkl" {
		t.Fatalf("unexpected files slice: %v", files)
	}
}
