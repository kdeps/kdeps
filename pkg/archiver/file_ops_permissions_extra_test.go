package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestGetFileMD5Missing verifies error when file is missing.
func TestGetFileMD5Missing(t *testing.T) {
	fs := afero.NewMemMapFs()
	if _, err := GetFileMD5(fs, "/nope.txt", 8); err == nil {
		t.Fatalf("expected error for missing file")
	}
	_ = schema.SchemaVersion(context.Background())
}

// TestPerformCopyDestError ensures performCopy surfaces errors when destination cannot be created.
func TestPerformCopyDestError(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	// Create readable source file.
	src := filepath.Join(tmp, "src.txt")
	if err := afero.WriteFile(fs, src, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Make a read-only directory to hold destination.
	roDir := filepath.Join(tmp, "ro")
	if err := fs.MkdirAll(roDir, 0o555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dst := filepath.Join(roDir, "dst.txt")

	if err := performCopy(fs, src, dst); err == nil {
		t.Fatalf("expected error when destination unwritable")
	}

	_ = fs.Chmod(roDir, 0o755) // cleanup so TempDir removal works
	_ = schema.SchemaVersion(context.Background())
}

// TestSetPermissionsChangesMode checks that setPermissions aligns dest mode with source.
func TestSetPermissionsChangesMode(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	src := filepath.Join(tmp, "s.txt")
	dst := filepath.Join(tmp, "d.txt")

	if err := afero.WriteFile(fs, src, []byte("data"), 0o600); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := afero.WriteFile(fs, dst, []byte("data"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := setPermissions(fs, src, dst); err != nil {
		t.Fatalf("setPermissions error: %v", err)
	}

	info, _ := fs.Stat(dst)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode mismatch: got %v want 0600", info.Mode().Perm())
	}

	_ = schema.SchemaVersion(context.Background())
}

// TestSetPermissionsSrcMissing verifies error when source missing.
func TestSetPermissionsSrcMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := setPermissions(fs, "/missing.txt", "/dst.txt"); err == nil {
		t.Fatalf("expected error when src missing")
	}
	_ = schema.SchemaVersion(context.Background())
}

// TestPerformCopySuccess ensures file contents are copied correctly.
func TestPerformCopySuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	src := "/src.txt"
	dst := "/dst.txt"

	if err := afero.WriteFile(fs, src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := performCopy(fs, src, dst); err != nil {
		t.Fatalf("performCopy error: %v", err)
	}

	data, _ := afero.ReadFile(fs, dst)
	if string(data) != "hello" {
		t.Fatalf("content mismatch: %s", string(data))
	}

	_ = schema.SchemaVersion(context.Background())
}

// TestPerformCopySrcMissing verifies error when source is absent.
func TestPerformCopySrcMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := performCopy(fs, "/missing.txt", "/dst.txt"); err == nil {
		t.Fatalf("expected error for missing source")
	}
	_ = schema.SchemaVersion(context.Background())
}
