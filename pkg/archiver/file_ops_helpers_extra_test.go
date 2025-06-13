package archiver

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestCopyFileHelpers(t *testing.T) {
	fs := afero.NewOsFs()
	dir := t.TempDir()

	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")

	data := []byte("dummy-data")
	if err := afero.WriteFile(fs, src, data, 0o640); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// call internal copyFile helper
	if err := copyFile(fs, src, dst); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	// verify content matches
	got, _ := afero.ReadFile(fs, dst)
	if string(got) != string(data) {
		t.Fatalf("content mismatch: %q vs %q", got, data)
	}

	// Overwrite dst with different content then test performCopy + setPermissions
	src2 := filepath.Join(dir, "src2.bin")
	data2 := []byte("another")
	if err := afero.WriteFile(fs, src2, data2, 0o600); err != nil {
		t.Fatalf("write src2: %v", err)
	}

	if err := performCopy(fs, src2, dst); err != nil {
		t.Fatalf("performCopy error: %v", err)
	}

	if err := setPermissions(fs, src2, dst); err != nil {
		t.Fatalf("setPermissions error: %v", err)
	}

	// Check permissions replicated (only use mode bits)
	srcInfo, _ := fs.Stat(src2)
	dstInfo, _ := fs.Stat(dst)
	if srcInfo.Mode() != dstInfo.Mode() {
		t.Fatalf("permissions not replicated: src %v dst %v", srcInfo.Mode(), dstInfo.Mode())
	}
}
