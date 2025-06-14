package archiver

import (
	"testing"

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
