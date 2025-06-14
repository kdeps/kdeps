package archiver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestCopyFile_NoDestination(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// create src
	_ = afero.WriteFile(fs, "/src.txt", []byte("abc"), 0o644)

	if err := CopyFile(fs, ctx, "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile unexpected error: %v", err)
	}

	data, _ := afero.ReadFile(fs, "/dst.txt")
	if string(data) != "abc" {
		t.Fatalf("destination content mismatch")
	}
}

func TestCopyFile_SkipSameMD5(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	content := []byte("same")
	_ = afero.WriteFile(fs, "/src.txt", content, 0o644)
	_ = afero.WriteFile(fs, "/dst.txt", content, 0o644)

	if err := CopyFile(fs, ctx, "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// ensure dst still exists and unchanged
	data, _ := afero.ReadFile(fs, "/dst.txt")
	if string(data) != "same" {
		t.Fatalf("dst altered unexpectedly")
	}
}
