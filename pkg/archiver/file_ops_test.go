package archiver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveFolderMainPkg(t *testing.T) {

	fs := afero.NewMemMapFs()
	// Create source directory and files
	srcDir := "/src"
	destDir := "/dest"
	_ = fs.MkdirAll(srcDir, 0o755)
	_ = afero.WriteFile(fs, filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(srcDir, "file2.txt"), []byte("content2"), 0o644)

	err := MoveFolder(fs, srcDir, destDir)
	require.NoError(t, err)

	// Assert source directory no longer exists
	exists, err := afero.Exists(fs, srcDir)
	require.NoError(t, err)
	assert.False(t, exists)

	// Assert destination directory and files exist
	exists, err = afero.DirExists(fs, destDir)
	require.NoError(t, err)
	assert.True(t, exists)

	content, err := afero.ReadFile(fs, filepath.Join(destDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content))

	content, err = afero.ReadFile(fs, filepath.Join(destDir, "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content2", string(content))
}

func TestCopyFileMainPkg(t *testing.T) {

	fs := afero.NewMemMapFs()
	// Create source file
	srcFile := "/src/file.txt"
	destFile := "/dest/file.txt"
	_ = fs.MkdirAll(filepath.Dir(srcFile), 0o755)
	_ = afero.WriteFile(fs, srcFile, []byte("file content"), 0o644)

	err := CopyFile(fs, context.Background(), srcFile, destFile, logging.GetLogger())
	require.NoError(t, err)

	// Assert destination file exists and content matches
	content, err := afero.ReadFile(fs, destFile)
	require.NoError(t, err)
	assert.Equal(t, "file content", string(content))
}

func TestGetFileMD5MainPkg(t *testing.T) {

	// Arrange: Use an in-memory filesystem to isolate the test environment
	fs := afero.NewMemMapFs()
	filePath := "/file.txt"
	testContent := []byte("test content")
	expectedHash := "9473fdd0" // Precomputed MD5 hash truncated to 8 characters

	// Write the file content and check for errors
	err := afero.WriteFile(fs, filePath, testContent, 0o644)
	require.NoError(t, err, "failed to write test file")

	// Act: Calculate the MD5 hash of the file
	hash, err := GetFileMD5(fs, filePath, 8)

	// Assert: Validate the hash and ensure no errors occurred
	require.NoError(t, err, "failed to calculate MD5 hash")
	assert.Equal(t, expectedHash, hash, "MD5 hash mismatch")

	// Additional safety check: Verify the file still exists and content is intact
	exists, err := afero.Exists(fs, filePath)
	require.NoError(t, err, "error checking file existence")
	assert.True(t, exists, "file does not exist")

	content, err := afero.ReadFile(fs, filePath)
	require.NoError(t, err, "error reading file content")
	assert.Equal(t, testContent, content, "file content mismatch")
}

func TestCopyDirMainPkg(t *testing.T) {

	fs := afero.NewMemMapFs()
	srcDir := "/src"
	destDir := "/dest"

	_ = fs.MkdirAll(srcDir, 0o755)
	_ = afero.WriteFile(fs, filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(srcDir, "file2.txt"), []byte("content2"), 0o644)

	err := CopyDir(fs, context.Background(), srcDir, destDir, logging.GetLogger())
	require.NoError(t, err)

	// Assert destination directory and files exist
	exists, err := afero.DirExists(fs, destDir)
	require.NoError(t, err)
	assert.True(t, exists)

	content, err := afero.ReadFile(fs, filepath.Join(destDir, "file1.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content1", string(content))

	content, err = afero.ReadFile(fs, filepath.Join(destDir, "file2.txt"))
	require.NoError(t, err)
	assert.Equal(t, "content2", string(content))
}
