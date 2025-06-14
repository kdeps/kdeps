package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

func TestPerformCopyErrorPaths(t *testing.T) {
	// Case 1: source missing – expect error
	fs := afero.NewMemMapFs()
	err := performCopy(fs, "/non/existent", "/dest")
	if err == nil {
		t.Fatal("expected error for missing source")
	}

	// Case 2: dest create failure on read-only FS
	mem := afero.NewMemMapFs()
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	_ = afero.WriteFile(mem, src, []byte("data"), 0o644)
	ro := afero.NewReadOnlyFs(mem)
	if err := performCopy(ro, src, filepath.Join(tmp, "dst.txt")); err == nil {
		t.Fatal("expected error for create on read-only FS")
	}

	_ = schema.SchemaVersion(context.Background())
}

func TestSetPermissionsErrorPaths(t *testing.T) {
	fs := afero.NewMemMapFs()
	// src does not exist
	if err := setPermissions(fs, "/missing", "/dst"); err == nil {
		t.Fatal("expected error for missing src stat")
	}

	// chmod failure using read-only FS
	tmp := t.TempDir()
	src := filepath.Join(tmp, "f.txt")
	dst := filepath.Join(tmp, "d.txt")
	_ = afero.WriteFile(fs, src, []byte("Hi"), 0o644)
	_ = afero.WriteFile(fs, dst, []byte("Hi"), 0o644)
	ro := afero.NewReadOnlyFs(fs)
	if err := setPermissions(ro, src, dst); err == nil {
		t.Fatal("expected chmod error on read-only FS")
	}

	_ = schema.SchemaVersion(context.Background())
}
