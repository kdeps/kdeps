package archiver

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// Test that performCopy fails when destination cannot be created (read-only FS).
func TestPerformCopy_DestinationCreateFails(t *testing.T) {
	base := afero.NewMemMapFs()
	src := "/src.txt"
	_ = afero.WriteFile(base, src, []byte("data"), 0o644)

	ro := afero.NewReadOnlyFs(base)
	if err := performCopy(ro, src, "/dst.txt"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// errFs wraps MemMapFs but forces Chmod to fail so setPermissions propagates the error.
type errFs struct {
	*afero.MemMapFs
}

// Override Chmod to simulate permission failure.
func (e *errFs) Chmod(name string, mode os.FileMode) error {
	return errors.New("chmod not allowed")
}

func TestCopyFile_SetPermissionsFails(t *testing.T) {
	// base mem FS handles file operations; errFs will delegate except Chmod.
	mem := &afero.MemMapFs{}
	efs := &errFs{mem}

	src := "/a.txt"
	dst := "/b.txt"
	_ = afero.WriteFile(mem, src, []byte("x"), 0o644)

	err := CopyFile(efs, context.Background(), src, dst, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected chmod failure error")
	}
	if !strings.Contains(err.Error(), "chmod not allowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}
