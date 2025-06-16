package archiver

import (
	"context"
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestCopyFileSuccess verifies that copyFile successfully duplicates the file contents.
func TestCopyFileSuccessMemFS(t *testing.T) {
	mem := afero.NewMemMapFs()

	// Prepare source file.
	src := "/src.txt"
	dst := "/dst.txt"
	data := []byte("hello")
	if err := afero.WriteFile(mem, src, data, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := copyFile(mem, src, dst); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}
	copied, _ := afero.ReadFile(mem, dst)
	if string(copied) != string(data) {
		t.Fatalf("copied content mismatch: %s", string(copied))
	}
}

// TestSetPermissionsSuccess ensures permissions are propagated from source to destination.
func TestSetPermissionsSuccessMemFS(t *testing.T) {
	mem := afero.NewMemMapFs()
	src := "/src.txt"
	dst := "/dst.txt"
	if err := afero.WriteFile(mem, src, []byte("x"), 0o640); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := afero.WriteFile(mem, dst, []byte("y"), 0o600); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := setPermissions(mem, src, dst); err != nil {
		t.Fatalf("setPermissions error: %v", err)
	}

	info, _ := mem.Stat(dst)
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("permissions not propagated, got %v", info.Mode().Perm())
	}

	// Extra: ensure setPermissions no error when src and dst modes identical.
	if err := setPermissions(mem, src, dst); err != nil {
		t.Fatalf("setPermissions identical modes error: %v", err)
	}
}

// TestGetFileMD5AndCopyFileSuccess covers:
// 1. GetFileMD5 happy path.
// 2. CopyFile when destination does NOT exist (no backup logic triggered).
func TestGetFileMD5AndCopyFileSuccess(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.txt")
	dstPath := filepath.Join(tmpDir, "dst.txt")

	content := []byte("hello-md5")
	if err := afero.WriteFile(fs, srcPath, content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Calculate expected MD5 manually (full hash then slice len 8)
	hash := md5.Sum(content) //nolint:gosec
	wantMD5 := hex.EncodeToString(hash[:])[:8]

	gotMD5, err := GetFileMD5(fs, srcPath, 8)
	if err != nil {
		t.Fatalf("GetFileMD5 error: %v", err)
	}
	if gotMD5 != wantMD5 {
		t.Fatalf("MD5 mismatch: got %s want %s", gotMD5, wantMD5)
	}

	// Run CopyFile where dst does not exist yet.
	logger := logging.NewTestLogger()
	if err := CopyFile(fs, context.Background(), srcPath, dstPath, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// Verify destination now exists with identical contents.
	dstData, err := afero.ReadFile(fs, dstPath)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(dstData) != string(content) {
		t.Fatalf("content mismatch: got %s want %s", string(dstData), string(content))
	}

	// Ensure permissions were copied (mode preserved at least rw for owner).
	info, _ := fs.Stat(dstPath)
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("permissions not preserved: %v", info.Mode())
	}

	// Logger should contain success message.
	if out := logger.GetOutput(); !strings.Contains(strings.ToLower(out), "copied successfully") {
		t.Fatalf("expected log to mention copy success, got: %s", out)
	}
}
