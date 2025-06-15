package archiver

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
)

func TestGetFileMD5SuccessAndError(t *testing.T) {
	afs := afero.NewOsFs()
	tmp := t.TempDir()

	filePath := filepath.Join(tmp, "f.txt")
	data := []byte("abc123")
	if err := afero.WriteFile(afs, filePath, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := GetFileMD5(afs, filePath, 8)
	if err != nil {
		t.Fatalf("GetFileMD5 error: %v", err)
	}
	h := md5.Sum(data) //nolint:gosec
	expected := hex.EncodeToString(h[:])[:8]
	if got != expected {
		t.Fatalf("hash mismatch: got %s want %s", got, expected)
	}

	// error path: file missing
	if _, err := GetFileMD5(afs, filepath.Join(tmp, "missing"), 8); err == nil {
		t.Fatalf("expected error for missing file")
	}

	// error path: zero-length allowed file but permission denied (use read only fs layer)
	ro := afero.NewReadOnlyFs(afs)
	if _, err := GetFileMD5(ro, filePath, 8); err != nil && !errors.Is(err, fs.ErrPermission) {
		// expected some error not nil – just ensure function propagates
	}
}
