package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"

	"github.com/kdeps/kdeps/pkg/schema"
)

func TestCopyFile_RenameError(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	// write distinct source and dest so MD5 differs → forces rename of existing dst
	_ = afero.WriteFile(fs, src, []byte("source"), 0o644)
	_ = afero.WriteFile(fs, dst, []byte("dest"), 0o644)

	// Wrap the mem fs with read-only to make Rename fail
	rofs := afero.NewReadOnlyFs(fs)

	if err := CopyFile(rofs, ctx, src, dst, logger); err == nil {
		t.Fatalf("expected error due to read-only rename failure")
	}
}

func TestPerformCopy_DestCreateError(t *testing.T) {
	mem := afero.NewMemMapFs()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "s.txt")
	_ = afero.WriteFile(mem, src, []byte("a"), 0o644)

	// destination on read-only fs; embed mem inside ro wrapper to make create fail
	ro := afero.NewReadOnlyFs(mem)
	if err := performCopy(ro, src, filepath.Join(tmp, "d.txt")); err == nil {
		t.Fatalf("expected create error on read-only FS")
	}
}

// TestCopyFileMissingSource verifies that copyFile returns an error when the
// source does not exist.
func TestCopyFileMissingSource(t *testing.T) {
	fs := afero.NewMemMapFs()
	dst := "/dst.txt"
	if err := copyFile(fs, "/no-such.txt", dst); err == nil {
		t.Fatalf("expected error for missing source file")
	}
	// Destination should not exist either.
	if exists, _ := afero.Exists(fs, dst); exists {
		t.Fatalf("destination unexpectedly created on failure")
	}

	_ = schema.SchemaVersion(context.Background())
}

// TestPerformCopyErrorSource ensures performCopy surfaces error when source
// cannot be opened.
func TestPerformCopyErrorSource(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := performCopy(fs, "/bad-src", "/dst")
	if err == nil {
		t.Fatalf("expected error from performCopy with bad source")
	}
	_ = schema.SchemaVersion(context.Background())
}

// TestMoveFolderMissing verifies that MoveFolder returns error for a missing
// source directory.
func TestMoveFolderMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := MoveFolder(fs, "/does/not/exist", "/dest"); err == nil {
		t.Fatalf("expected error when source directory is absent")
	}
	_ = schema.SchemaVersion(context.Background())
}

// TestCopyPermissions checks that performCopy plus setPermissions yields the
// same mode bits at destination as source.
func TestCopyPermissions(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	src := "/src.txt"
	dst := "/dst.txt"

	// Create src with specific permissions.
	content := []byte("perm-test")
	if err := afero.WriteFile(fs, src, content, 0o640); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Need a dummy logger – not used in code path.
	logger := logging.NewTestLogger()

	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	srcInfo, _ := fs.Stat(src)
	dstInfo, _ := fs.Stat(dst)
	if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
		t.Fatalf("permission mismatch: src %v dst %v", srcInfo.Mode().Perm(), dstInfo.Mode().Perm())
	}

	// Ensure contents copied too.
	data, _ := afero.ReadFile(fs, dst)
	if string(data) != string(content) {
		t.Fatalf("content mismatch: got %q want %q", string(data), string(content))
	}

	_ = schema.SchemaVersion(ctx)
}
