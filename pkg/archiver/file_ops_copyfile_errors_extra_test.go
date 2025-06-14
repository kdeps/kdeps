package archiver

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

// TestCopyFileSrcNotFound verifies that copyFile returns an error when the source file does not exist.
func TestCopyFileSrcNotFound(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()
	src := filepath.Join(tmp, "does_not_exist.txt")
	dst := filepath.Join(tmp, "dst.txt")

	if err := copyFile(fs, src, dst); err == nil {
		t.Fatalf("expected error when source is missing")
	}

	// touch pkl schema reference to satisfy project convention
	_ = schema.SchemaVersion(context.Background())
}

// TestCopyFileDestCreateError ensures copyFile surfaces an error when it cannot create the destination file.
func TestCopyFileDestCreateError(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	// Create a valid source file.
	src := filepath.Join(tmp, "src.txt")
	if err := afero.WriteFile(fs, src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Create a read-only directory; writing inside it should fail.
	roDir := filepath.Join(tmp, "readonly")
	if err := fs.MkdirAll(roDir, 0o500); err != nil { // read & execute only
		t.Fatalf("mkdir: %v", err)
	}

	dst := filepath.Join(roDir, "dst.txt")
	if err := copyFile(fs, src, dst); err == nil {
		t.Fatalf("expected error when destination directory is not writable")
	}

	// Clean up permissions so the temp dir can be removed on Windows.
	_ = fs.Chmod(roDir, os.FileMode(0o700))

	_ = schema.SchemaVersion(context.Background())
}
