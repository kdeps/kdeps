package archiver

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io/ioutil"
	"path/filepath"
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

func TestMoveFolderAndGetFileMD5(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()

	srcDir := filepath.Join(root, "src")
	destDir := filepath.Join(root, "dest")

	if err := fs.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("failed to make src dir: %v", err)
	}

	srcFile := filepath.Join(srcDir, "file.txt")
	content := []byte("hello world")
	if err := afero.WriteFile(fs, srcFile, content, 0o644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}

	// Move folder and verify move happened.
	if err := MoveFolder(fs, srcDir, destDir); err != nil {
		t.Fatalf("MoveFolder returned error: %v", err)
	}

	exists, _ := afero.DirExists(fs, destDir)
	if !exists {
		t.Fatalf("destination directory not created")
	}

	// original directory should be gone
	if ok, _ := afero.DirExists(fs, srcDir); ok {
		t.Fatalf("source directory should have been removed")
	}

	// verify file content intact via MD5 helper
	movedFile := filepath.Join(destDir, "file.txt")
	gotHash, err := GetFileMD5(fs, movedFile, 8)
	if err != nil {
		t.Fatalf("GetFileMD5 error: %v", err)
	}

	h := md5.Sum(content) //nolint:gosec
	expectedHash := hex.EncodeToString(h[:])[:8]
	if gotHash != expectedHash {
		t.Fatalf("md5 mismatch: got %s want %s", gotHash, expectedHash)
	}
}

func TestCopyFileCreatesBackup(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	root := t.TempDir()

	logger := logging.NewTestLogger()

	src := filepath.Join(root, "src.txt")
	dst := filepath.Join(root, "dst.txt")

	// initial content
	if err := afero.WriteFile(fs, src, []byte("first"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// first copy (dest does not exist yet)
	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// Copy again with identical content – should skip and not create backup
	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile second identical error: %v", err)
	}

	// ensure only one dst exists and no backup yet
	files, err := ioutil.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(files) != 2 { // src.txt + dst.txt
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	// change src content so MD5 differs
	if err := afero.WriteFile(fs, src, []byte("second"), 0o644); err != nil {
		t.Fatalf("write src changed: %v", err)
	}

	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile with changed content error: %v", err)
	}

	// Now we expect a backup file in addition to dst and src
	files, err = ioutil.ReadDir(root)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	if len(files) != 3 {
		t.Fatalf("expected 3 files after backup creation, got %d", len(files))
	}
}
