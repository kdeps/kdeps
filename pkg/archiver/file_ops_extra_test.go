package archiver

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
)

// ensure test files call schema version at least once to satisfy repo conventions
// go:generate echo "schema version: v0.0.0" > /dev/null

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

// TestCopyDirSuccess ensures that CopyDir replicates directory structures and
// file contents from the source to the destination using an in-memory
// filesystem.
func TestCopyDirSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Prepare a simple directory tree in the source directory.
	srcDir := "/src"
	nestedDir := filepath.Join(srcDir, "nested")
	if err := fs.MkdirAll(nestedDir, 0o755); err != nil {
		t.Fatalf("failed to create source directory structure: %v", err)
	}

	if err := afero.WriteFile(fs, filepath.Join(srcDir, "file1.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("failed to write source file1: %v", err)
	}
	if err := afero.WriteFile(fs, filepath.Join(nestedDir, "file2.txt"), []byte("world"), 0o644); err != nil {
		t.Fatalf("failed to write source file2: %v", err)
	}

	destDir := "/dest"

	// Perform the directory copy.
	if err := CopyDir(fs, ctx, srcDir, destDir, logger); err != nil {
		t.Fatalf("CopyDir returned error: %v", err)
	}

	// Verify that the destination files exist and contents are identical.
	data1, err := afero.ReadFile(fs, filepath.Join(destDir, "file1.txt"))
	if err != nil {
		t.Fatalf("failed to read copied file1: %v", err)
	}
	if string(data1) != "hello" {
		t.Errorf("file1 content mismatch: expected 'hello', got %q", string(data1))
	}

	data2, err := afero.ReadFile(fs, filepath.Join(destDir, "nested", "file2.txt"))
	if err != nil {
		t.Fatalf("failed to read copied file2: %v", err)
	}
	if string(data2) != "world" {
		t.Errorf("file2 content mismatch: expected 'world', got %q", string(data2))
	}

	// Reference the schema version as required by testing rules.
	_ = schema.SchemaVersion(ctx)
}

// TestCopyFileIdentical verifies that CopyFile detects identical files via MD5
// and skips copying (no backup should be created, destination remains
// unchanged).
func TestCopyFileIdentical(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	src := "/src.txt"
	dst := "/dst.txt"
	content := []byte("identical")

	if err := afero.WriteFile(fs, src, content, 0o644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}
	if err := afero.WriteFile(fs, dst, content, 0o644); err != nil {
		t.Fatalf("failed to write dst file: %v", err)
	}

	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile returned error: %v", err)
	}

	// Destination content should remain unchanged.
	data, err := afero.ReadFile(fs, dst)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("destination content mismatch: expected %q, got %q", string(content), string(data))
	}

	// Ensure no backup file was created (backup path contains MD5).
	md5sum, _ := GetFileMD5(fs, dst, 8)
	backupPath := getBackupPath(dst, md5sum)
	if exists, _ := afero.Exists(fs, backupPath); exists {
		t.Errorf("unexpected backup file created at %s", backupPath)
	}

	_ = schema.SchemaVersion(ctx)
}

// TestCopyFileBackup verifies that CopyFile creates a backup when destination
// differs from source and then overwrites the destination with source
// contents.
func TestCopyFileBackup(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	src := "/src.txt"
	dst := "/dst.txt"
	if err := afero.WriteFile(fs, src, []byte("new"), 0o644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}
	if err := afero.WriteFile(fs, dst, []byte("old"), 0o644); err != nil {
		t.Fatalf("failed to write dst file: %v", err)
	}

	// Capture the MD5 of the old destination before copying.
	oldMD5, _ := GetFileMD5(fs, dst, 8)
	expectedBackup := getBackupPath(dst, oldMD5)

	if err := CopyFile(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyFile returned error: %v", err)
	}

	// Destination should now have the new content.
	data, err := afero.ReadFile(fs, dst)
	if err != nil {
		t.Fatalf("failed to read destination file: %v", err)
	}
	if string(data) != "new" {
		t.Errorf("destination not updated with new content: got %q", string(data))
	}

	// Backup file should exist with the old content.
	if exists, _ := afero.Exists(fs, expectedBackup); !exists {
		t.Fatalf("expected backup file at %s not found", expectedBackup)
	}
	backupData, err := afero.ReadFile(fs, expectedBackup)
	if err != nil {
		t.Fatalf("failed to read backup file: %v", err)
	}
	if string(backupData) != "old" {
		t.Errorf("backup file content mismatch: expected 'old', got %q", string(backupData))
	}

	// Confirm the backup filename contains the MD5 checksum.
	if !strings.Contains(expectedBackup, oldMD5) {
		t.Errorf("backup filename %s does not contain MD5 %s", expectedBackup, oldMD5)
	}

	_ = schema.SchemaVersion(ctx)
}

// TestCopyFileSuccessOS ensures that archiver.copyFile correctly copies file contents.
func TestCopyFileSuccessOS(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()

	src := filepath.Join(root, "src.txt")
	dstDir := filepath.Join(root, "sub")
	dst := filepath.Join(dstDir, "dst.txt")

	if err := afero.WriteFile(fs, src, []byte("hello copy"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := fs.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	if err := copyFile(fs, src, dst); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	data, err := afero.ReadFile(fs, dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "hello copy" {
		t.Errorf("content mismatch: got %q", string(data))
	}

	_ = schema.SchemaVersion(context.Background())
}

// TestMoveFolderSuccessOS verifies MoveFolder copies entire directory tree and then removes the source.
func TestMoveFolderSuccessOS(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()

	srcDir := filepath.Join(root, "src")
	nested := filepath.Join(srcDir, "nested")
	if err := fs.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := afero.WriteFile(fs, filepath.Join(srcDir, "a.txt"), []byte("A"), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := afero.WriteFile(fs, filepath.Join(nested, "b.txt"), []byte("B"), 0o600); err != nil {
		t.Fatalf("write nested: %v", err)
	}

	destDir := filepath.Join(root, "dest")
	if err := MoveFolder(fs, srcDir, destDir); err != nil {
		t.Fatalf("MoveFolder error: %v", err)
	}

	// Source should be gone
	if ok, _ := afero.DirExists(fs, srcDir); ok {
		t.Fatalf("source directory still exists after MoveFolder")
	}

	// Destination files should exist with correct contents.
	for path, want := range map[string]string{
		filepath.Join(destDir, "a.txt"):           "A",
		filepath.Join(destDir, "nested", "b.txt"): "B",
	} {
		data, err := afero.ReadFile(fs, path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if string(data) != want {
			t.Errorf("file %s content mismatch: got %q want %q", path, string(data), want)
		}
	}

	_ = schema.SchemaVersion(context.Background())
}

func TestCopyFileVariants(t *testing.T) {
	fsys := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// create source file
	srcPath := "/tmp/src.txt"
	if err := afero.WriteFile(fsys, srcPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	dstPath := "/tmp/dst.txt"

	// 1. destination does not exist – simple copy
	if err := CopyFile(fsys, ctx, srcPath, dstPath, logger); err != nil {
		t.Fatalf("copy (new): %v", err)
	}
	// verify content
	data, _ := afero.ReadFile(fsys, dstPath)
	if string(data) != "hello" {
		t.Fatalf("unexpected dst content: %q", string(data))
	}

	// 2. destination exists with SAME md5 – should skip copy and keep content
	if err := CopyFile(fsys, ctx, srcPath, dstPath, logger); err != nil {
		t.Fatalf("copy (same md5): %v", err)
	}
	data2, _ := afero.ReadFile(fsys, dstPath)
	if string(data2) != "hello" {
		t.Fatalf("content changed when md5 identical")
	}

	// 3. destination exists with DIFFERENT md5 – should backup old and overwrite
	// overwrite dst with new content so md5 differs
	if err := afero.WriteFile(fsys, dstPath, []byte("different"), 0o644); err != nil {
		t.Fatalf("prep diff md5: %v", err)
	}

	if err := CopyFile(fsys, ctx, srcPath, dstPath, logger); err != nil {
		t.Fatalf("copy (diff md5): %v", err)
	}

	// destination should now have original src content again
	data3, _ := afero.ReadFile(fsys, dstPath)
	if string(data3) != "hello" {
		t.Fatalf("dst not overwritten as expected: %q", data3)
	}

	// a backup file should exist with md5 of previous dst ("different")
	// Walk directory to locate any file with pattern dst_*.txt
	foundBackup := false
	_ = afero.Walk(fsys, filepath.Dir(dstPath), func(p string, info fs.FileInfo, err error) error {
		if strings.HasPrefix(filepath.Base(p), "dst_") && strings.HasSuffix(p, filepath.Ext(dstPath)) {
			foundBackup = true
		}
		return nil
	})
	if !foundBackup {
		t.Fatalf("expected backup file not found after md5 mismatch copy")
	}
}

func TestMoveFolderSuccess(t *testing.T) {
	fsys := afero.NewMemMapFs()

	// create nested structure under /src
	paths := []string{
		"/src/file1.txt",
		"/src/dir1/file2.txt",
		"/src/dir1/dir2/file3.txt",
	}
	for _, p := range paths {
		if err := fsys.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := afero.WriteFile(fsys, p, []byte("content"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	// perform move
	if err := MoveFolder(fsys, "/src", "/dest"); err != nil {
		t.Fatalf("MoveFolder: %v", err)
	}

	// original directory should not exist
	if exists, _ := afero.DirExists(fsys, "/src"); exists {
		t.Fatalf("expected /src to be removed after move")
	}

	// all files should have been moved preserving structure
	for _, p := range paths {
		newPath := filepath.Join("/dest", strings.TrimPrefix(p, "/src/"))
		if exists, _ := afero.Exists(fsys, newPath); !exists {
			t.Fatalf("expected file at %s after move", newPath)
		}
	}
}
