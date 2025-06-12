package archiver

import (
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
