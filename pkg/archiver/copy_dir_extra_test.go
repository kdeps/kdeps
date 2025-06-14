package archiver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

func TestCopyDirSimpleSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()
	ctx := context.Background()

	src := "/src"
	dst := "/dst"

	// Create nested structure in src
	if err := fs.MkdirAll(src+"/sub", 0o755); err != nil {
		t.Fatalf("mkdir err: %v", err)
	}
	if err := afero.WriteFile(fs, src+"/file1.txt", []byte("hello"), 0o644); err != nil {
		t.Fatalf("write err: %v", err)
	}
	if err := afero.WriteFile(fs, src+"/sub/file2.txt", []byte("world"), 0o600); err != nil {
		t.Fatalf("write err: %v", err)
	}

	if err := CopyDir(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	// Validate copied content
	if data, _ := afero.ReadFile(fs, dst+"/file1.txt"); string(data) != "hello" {
		t.Fatalf("file1 content mismatch")
	}
	if data, _ := afero.ReadFile(fs, dst+"/sub/file2.txt"); string(data) != "world" {
		t.Fatalf("file2 content mismatch")
	}
}

func TestCopyDirReadOnlyFailure(t *testing.T) {
	mem := afero.NewMemMapFs()
	readOnly := afero.NewReadOnlyFs(mem)
	logger := logging.GetLogger()
	ctx := context.Background()

	src := "/src"
	dst := "/dst"

	_ = mem.MkdirAll(src, 0o755)
	_ = afero.WriteFile(mem, src+"/f.txt", []byte("x"), 0o644)

	if err := CopyDir(readOnly, ctx, src, dst, logger); err == nil {
		t.Fatalf("expected error, got nil")
	}
}
