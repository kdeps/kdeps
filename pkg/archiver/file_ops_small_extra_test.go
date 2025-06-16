package archiver

import (
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"io"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

// TestMoveFolderAndGetFileMD5Small covers the happy-path of MoveFolder as well as
// the MD5 helper which is used by CopyFile. It relies only on afero so no
// host-FS writes occur.
func TestMoveFolderAndGetFileMD5Small(t *testing.T) {
	fs := afero.NewOsFs()

	// Create a temporary source directory with one file inside.
	srcDir, err := afero.TempDir(fs, "", "kdeps_src")
	if err != nil {
		t.Fatalf("TempDir src error: %v", err)
	}
	defer fs.RemoveAll(srcDir)

	data := []byte("hello kdeps")
	srcFile := filepath.Join(srcDir, "file.txt")
	if err := afero.WriteFile(fs, srcFile, data, 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Destination directory (does not need to exist beforehand).
	destDir, err := afero.TempDir(fs, "", "kdeps_dst")
	if err != nil {
		t.Fatalf("TempDir dest error: %v", err)
	}
	fs.RemoveAll(destDir) // ensure empty so MoveFolder will create it

	// MoveFolder should move the directory tree.
	if err := MoveFolder(fs, srcDir, destDir); err != nil {
		t.Fatalf("MoveFolder error: %v", err)
	}

	movedFile := filepath.Join(destDir, "file.txt")
	if exists, _ := afero.Exists(fs, movedFile); !exists {
		t.Fatalf("expected file to be moved to %s", movedFile)
	}
	if exists, _ := afero.DirExists(fs, srcDir); exists {
		t.Fatalf("expected source directory to be removed")
	}

	// Verify GetFileMD5 returns the expected (truncated) hash.
	got, err := GetFileMD5(fs, movedFile, 6)
	if err != nil {
		t.Fatalf("GetFileMD5 error: %v", err)
	}

	h := md5.New() //nolint:gosec
	_, _ = io.WriteString(h, string(data))
	wantFull := hex.EncodeToString(h.Sum(nil))
	want := wantFull[:6]
	if got != want {
		t.Fatalf("md5 mismatch: got %s want %s", got, want)
	}
}
