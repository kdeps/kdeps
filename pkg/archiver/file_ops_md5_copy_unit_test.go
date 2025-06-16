package archiver

import (
	"crypto/md5" //nolint:gosec
	"encoding/hex"
	"io"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// TestGetFileMD5 covers happy-path, truncation and error branches.
func TestGetFileMD5Edges(t *testing.T) {
	fs := afero.NewMemMapFs()
	filePath := "/tmp/test.txt"
	content := []byte("hello-md5-check")
	require.NoError(t, afero.WriteFile(fs, filePath, content, 0o644))

	// Full length (32 chars) hash check.
	got, err := GetFileMD5(fs, filePath, 32)
	require.NoError(t, err)
	h := md5.Sum(content) //nolint:gosec
	expected := hex.EncodeToString(h[:])
	require.Equal(t, expected, got)

	// Truncated hash (8 chars).
	gotShort, err := GetFileMD5(fs, filePath, 8)
	require.NoError(t, err)
	require.Equal(t, expected[:8], gotShort)

	// Non-existent file should return error.
	_, err = GetFileMD5(fs, "/does/not/exist", 8)
	require.Error(t, err)
}

// TestPerformCopy ensures the helper copies bytes correctly and creates the
// destination file when it does not exist.
func TestPerformCopy(t *testing.T) {
	fs := afero.NewMemMapFs()
	src := filepath.Join(t.TempDir(), "src.txt")
	dst := filepath.Join(t.TempDir(), "dst.txt")

	// Create source file with known content.
	data := []byte("copy-this-data")
	require.NoError(t, afero.WriteFile(fs, src, data, 0o600))

	// performCopy is internal but test file lives in same package so we can call it.
	require.NoError(t, performCopy(fs, src, dst))

	// Verify destination contains identical bytes.
	dstFile, err := fs.Open(dst)
	require.NoError(t, err)
	defer dstFile.Close()

	copied, err := io.ReadAll(dstFile)
	require.NoError(t, err)
	require.Equal(t, data, copied)
}
