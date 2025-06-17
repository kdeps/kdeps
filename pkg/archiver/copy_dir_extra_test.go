package archiver

import (
	"context"
	"testing"

	"path/filepath"

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

func TestCopyDirSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")

	// create nested dirs & files
	files := []string{
		filepath.Join(src, "a.txt"),
		filepath.Join(src, "sub", "b.txt"),
		filepath.Join(src, "sub", "sub2", "c.txt"),
	}
	for _, f := range files {
		_ = fs.MkdirAll(filepath.Dir(f), 0o755)
		_ = afero.WriteFile(fs, f, []byte("x"), 0o644)
	}

	if err := CopyDir(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyDir error: %v", err)
	}

	// ensure all files exist in dst
	for _, f := range files {
		rel, _ := filepath.Rel(src, f)
		if ok, _ := afero.Exists(fs, filepath.Join(dst, rel)); !ok {
			t.Fatalf("file not copied: %s", rel)
		}
	}
}

func TestCopyFileSkipIfHashesMatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	src := "/src.txt"
	dst := "/dst.txt"
	content := []byte("same")
	if err := afero.WriteFile(fs, src, content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	// Copy initial file to dst so hashes match
	if err := afero.WriteFile(fs, dst, content, 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}
}

func TestCopyFileCreatesBackupOnHashMismatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	src := "/src2.txt"
	dst := "/dst2.txt"

	if err := afero.WriteFile(fs, src, []byte("new"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := afero.WriteFile(fs, dst, []byte("old"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// backup should exist
	files, _ := afero.ReadDir(fs, "/")
	foundBackup := false
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".txt" && f.Name() != "src2.txt" && f.Name() != "dst2.txt" {
			foundBackup = true
		}
	}
	if !foundBackup {
		t.Fatalf("expected backup file to be created")
	}
}
