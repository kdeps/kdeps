package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestCopyFileSimple verifies that copyFile copies contents when destination
// is absent.
func TestCopyFileSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")

	if err := afero.WriteFile(fs, src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := copyFile(fs, src, dst); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	data, _ := afero.ReadFile(fs, dst)
	if string(data) != "hello" {
		t.Fatalf("content mismatch: %s", string(data))
	}

	_ = schema.SchemaVersion(context.Background())
}

// TestCopyFileOverwrite ensures that copyFile overwrites an existing file.
func TestCopyFileOverwrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	src := filepath.Join(dir, "s.txt")
	dst := filepath.Join(dir, "d.txt")

	_ = afero.WriteFile(fs, src, []byte("new"), 0o644)
	_ = afero.WriteFile(fs, dst, []byte("old"), 0o644)

	if err := copyFile(fs, src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	data, _ := afero.ReadFile(fs, dst)
	if string(data) != "new" {
		t.Fatalf("overwrite failed, got %s", string(data))
	}

	_ = schema.SchemaVersion(context.Background())
}
