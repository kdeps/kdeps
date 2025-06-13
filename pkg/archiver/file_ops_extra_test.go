package archiver

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestMoveFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/src/a/b", 0755)
	_ = afero.WriteFile(fs, "/src/a/b/file.txt", []byte("content"), 0644)
	require.NoError(t, MoveFolder(fs, "/src", "/dest"))
	exists, err := afero.DirExists(fs, "/src")
	require.NoError(t, err)
	require.False(t, exists)
	data, err := afero.ReadFile(fs, "/dest/a/b/file.txt")
	require.NoError(t, err)
	require.Equal(t, "content", string(data))
}

func TestGetFileMD5(t *testing.T) {
	fs := afero.NewMemMapFs()
	content := []byte("hello world")
	_ = afero.WriteFile(fs, "/file.txt", content, 0644)
	md5short, err := GetFileMD5(fs, "/file.txt", 8)
	require.NoError(t, err)
	sum := md5.Sum(content)
	expectedFull := hex.EncodeToString(sum[:])
	if len(expectedFull) >= 8 {
		require.Equal(t, expectedFull[:8], md5short)
	} else {
		require.Equal(t, expectedFull, md5short)
	}
	// length greater than md5 length should return full hash
	md5full, err := GetFileMD5(fs, "/file.txt", 100)
	require.NoError(t, err)
	require.Equal(t, expectedFull, md5full)
}

func TestCopyFile_NoExist(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	_ = afero.WriteFile(fs, "/src.txt", []byte("data"), 0644)
	require.NoError(t, CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger))
	data, err := afero.ReadFile(fs, "/dst.txt")
	require.NoError(t, err)
	require.Equal(t, "data", string(data))
}

func TestCopyFile_ExistsSameMD5(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	content := []byte("data")
	_ = afero.WriteFile(fs, "/src.txt", content, 0644)
	_ = afero.WriteFile(fs, "/dst.txt", content, 0644)
	require.NoError(t, CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger))
	data, err := afero.ReadFile(fs, "/dst.txt")
	require.NoError(t, err)
	require.Equal(t, "data", string(data))
	// Ensure no backup file created
	files, _ := afero.ReadDir(fs, "/")
	for _, f := range files {
		require.False(t, strings.HasPrefix(f.Name(), "dst_") && strings.HasSuffix(f.Name(), ".txt"), "unexpected backup file %s", f.Name())
	}
}

func TestCopyFile_ExistsDifferentMD5(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	_ = afero.WriteFile(fs, "/src.txt", []byte("src"), 0644)
	_ = afero.WriteFile(fs, "/dst.txt", []byte("dst"), 0644)
	require.NoError(t, CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger))
	data, err := afero.ReadFile(fs, "/dst.txt")
	require.NoError(t, err)
	require.Equal(t, "src", string(data))
	files, _ := afero.ReadDir(fs, "/")
	found := false
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "dst_") && strings.HasSuffix(f.Name(), ".txt") {
			found = true
		}
	}
	require.True(t, found, "backup file not found")
}

func TestGetBackupPath(t *testing.T) {
	p := getBackupPath("/path/file.ext", "abc")
	require.Equal(t, "/path/file_abc.ext", p)
}
