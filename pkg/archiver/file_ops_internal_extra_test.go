package archiver

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/afero"
)

// TestPerformCopyError checks that performCopy returns an error when the source
// file does not exist. This exercises the early error branch that was previously
// uncovered.
func TestPerformCopyError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Intentionally do NOT create the source file.
	src := "/missing/src.txt"
	dest := "/dest/out.txt"

	if err := performCopy(fs, src, dest); err == nil {
		t.Errorf("expected error when copying non-existent source, got nil")
	}
}

// TestSetPermissionsError ensures setPermissions fails gracefully when the
// source file is absent, covering its error path.
func TestSetPermissionsError(t *testing.T) {
	fs := afero.NewMemMapFs()

	src := "/missing/perm.txt"
	dest := "/dest/out.txt"

	if err := setPermissions(fs, src, dest); err == nil {
		t.Errorf("expected error when stat-ing non-existent source, got nil")
	}
}

// TestCopyFileInternalError ensures copyFile returns an error when the source does not exist.
func TestCopyFileInternalError(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	src := filepath.Join(tmp, "nosuch.txt")
	dst := filepath.Join(tmp, "dst.txt")

	if err := copyFile(fs, src, dst); err == nil {
		t.Fatalf("expected error for missing source file")
	}
}

// TestPerformCopyAndSetPermissions verifies performCopy copies bytes and setPermissions replicates mode bits.
func TestPerformCopyAndSetPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits semantics differ on Windows")
	}

	fs := afero.NewOsFs()
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")

	if err := afero.WriteFile(fs, src, []byte("abc"), 0o600); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// performCopy should succeed
	if err := performCopy(fs, src, dst); err != nil {
		t.Fatalf("performCopy error: %v", err)
	}

	// ensure bytes copied
	data, err := afero.ReadFile(fs, dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "abc" {
		t.Fatalf("unexpected dst content: %s", string(data))
	}

	// change src mode to 0644 then run setPermissions and expect dst updated
	if err := fs.Chmod(src, 0o644); err != nil {
		t.Fatalf("chmod src: %v", err)
	}

	if err := setPermissions(fs, src, dst); err != nil {
		t.Fatalf("setPermissions error: %v", err)
	}

	dstInfo, err := fs.Stat(dst)
	if err != nil {
		t.Fatalf("stat dst: %v", err)
	}

	if dstInfo.Mode().Perm() != 0o644 {
		t.Fatalf("permissions not propagated, got %v", dstInfo.Mode().Perm())
	}
}
