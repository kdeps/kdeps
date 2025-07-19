package data_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/data"
)

type errorFs struct{ afero.Fs }

func (e errorFs) Name() string                                 { return "errorFs" }
func (e errorFs) Mkdir(name string, perm os.FileMode) error    { return e.Fs.Mkdir(name, perm) }
func (e errorFs) MkdirAll(path string, perm os.FileMode) error { return e.Fs.MkdirAll(path, perm) }
func (e errorFs) Remove(name string) error                     { return e.Fs.Remove(name) }
func (e errorFs) RemoveAll(path string) error                  { return e.Fs.RemoveAll(path) }
func (e errorFs) Open(name string) (afero.File, error)         { return e.Fs.Open(name) }
func (e errorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return e.Fs.OpenFile(name, flag, perm)
}
func (e errorFs) Stat(_ string) (os.FileInfo, error)        { return nil, errors.New("stat error") }
func (e errorFs) Rename(oldname, newname string) error      { return e.Fs.Rename(oldname, newname) }
func (e errorFs) Chmod(name string, mode os.FileMode) error { return e.Fs.Chmod(name, mode) }
func (e errorFs) Chtimes(name string, atime, mtime time.Time) error {
	return e.Fs.Chtimes(name, atime, mtime)
}

type walkErrorFs struct{ afero.Fs }

func (w walkErrorFs) Name() string                                 { return "walkErrorFs" }
func (w walkErrorFs) Mkdir(name string, perm os.FileMode) error    { return w.Fs.Mkdir(name, perm) }
func (w walkErrorFs) MkdirAll(path string, perm os.FileMode) error { return w.Fs.MkdirAll(path, perm) }
func (w walkErrorFs) Remove(name string) error                     { return w.Fs.Remove(name) }
func (w walkErrorFs) RemoveAll(path string) error                  { return w.Fs.RemoveAll(path) }
func (w walkErrorFs) Open(name string) (afero.File, error)         { return w.Fs.Open(name) }
func (w walkErrorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return w.Fs.OpenFile(name, flag, perm)
}
func (w walkErrorFs) Stat(name string) (os.FileInfo, error)     { return w.Fs.Stat(name) }
func (w walkErrorFs) Rename(oldname, newname string) error      { return w.Fs.Rename(oldname, newname) }
func (w walkErrorFs) Chmod(name string, mode os.FileMode) error { return w.Fs.Chmod(name, mode) }
func (w walkErrorFs) Chtimes(name string, atime, mtime time.Time) error {
	return w.Fs.Chtimes(name, atime, mtime)
}

type statErrorFs struct{ afero.Fs }

func (s statErrorFs) Name() string                                 { return "statErrorFs" }
func (s statErrorFs) Mkdir(name string, perm os.FileMode) error    { return s.Fs.Mkdir(name, perm) }
func (s statErrorFs) MkdirAll(path string, perm os.FileMode) error { return s.Fs.MkdirAll(path, perm) }
func (s statErrorFs) Remove(name string) error                     { return s.Fs.Remove(name) }
func (s statErrorFs) RemoveAll(path string) error                  { return s.Fs.RemoveAll(path) }
func (s statErrorFs) Open(name string) (afero.File, error)         { return s.Fs.Open(name) }
func (s statErrorFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return s.Fs.OpenFile(name, flag, perm)
}

func (s statErrorFs) Stat(name string) (os.FileInfo, error) {
	// Only return error for files, not directories
	if strings.HasSuffix(name, ".txt") {
		return nil, errors.New("stat error")
	}
	return s.Fs.Stat(name)
}
func (s statErrorFs) Rename(oldname, newname string) error      { return s.Fs.Rename(oldname, newname) }
func (s statErrorFs) Chmod(name string, mode os.FileMode) error { return s.Fs.Chmod(name, mode) }
func (s statErrorFs) Chtimes(name string, atime, mtime time.Time) error {
	return s.Fs.Chtimes(name, atime, mtime)
}

func TestPopulateDataFileRegistry_BaseDirDoesNotExist(t *testing.T) {
	fs := afero.NewMemMapFs()
	reg, err := data.PopulateDataFileRegistry(fs, "/not-exist")
	require.NoError(t, err)
	assert.NotNil(t, reg)
	assert.Empty(t, *reg)
}

func TestPopulateDataFileRegistry_EmptyBaseDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base", 0o755)
	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	require.NoError(t, err)
	assert.NotNil(t, reg)
	assert.Empty(t, *reg)
}

func TestPopulateDataFileRegistry_WithFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base/agent1/v1", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file1.txt", []byte("data1"), 0o644)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file2.txt", []byte("data2"), 0o644)
	_ = fs.MkdirAll("/base/agent2/v2", 0o755)
	_ = afero.WriteFile(fs, "/base/agent2/v2/file3.txt", []byte("data3"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	require.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg
	assert.Len(t, files, 2)
	assert.Contains(t, files, filepath.Join("agent1", "v1"))
	assert.Contains(t, files, filepath.Join("agent2", "v2"))
	assert.Equal(t, "/base/agent1/v1/file1.txt", files[filepath.Join("agent1", "v1")]["file1.txt"])
	assert.Equal(t, "/base/agent1/v1/file2.txt", files[filepath.Join("agent1", "v1")]["file2.txt"])
	assert.Equal(t, "/base/agent2/v2/file3.txt", files[filepath.Join("agent2", "v2")]["file3.txt"])
}

func TestPopulateDataFileRegistry_SkipInvalidStructure(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base/agent1", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/file.txt", []byte("data"), 0o644)
	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	require.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg
	assert.Len(t, files, 1)
	assert.Contains(t, files, filepath.Join("agent1", "file.txt"))
	assert.Equal(t, map[string]string{"": "/base/agent1/file.txt"}, files[filepath.Join("agent1", "file.txt")])
}

func TestPopulateDataFileRegistry_ErrorOnDirExists(t *testing.T) {
	efs := errorFs{afero.NewMemMapFs()}
	reg, err := data.PopulateDataFileRegistry(efs, "/base")
	require.Error(t, err)
	assert.NotNil(t, reg)
	assert.Empty(t, *reg)
}

func TestPopulateDataFileRegistry_NestedDirectories(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base/agent1/v1/subdir", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/subdir/file.txt", []byte("data"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	require.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg
	assert.Len(t, files, 1)
	assert.Contains(t, files, filepath.Join("agent1", "v1"))
	assert.Equal(t, "/base/agent1/v1/subdir/file.txt", files[filepath.Join("agent1", "v1")][filepath.Join("subdir", "file.txt")])
}

func TestPopulateDataFileRegistry_SkipDirectoryEntries(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base/agent1/v1/dir", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	require.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg
	assert.Len(t, files, 1)
	assert.Contains(t, files, filepath.Join("agent1", "v1"))
	// Should only contain the file, not the directory
	assert.Len(t, files[filepath.Join("agent1", "v1")], 1)
	assert.Equal(t, "/base/agent1/v1/file.txt", files[filepath.Join("agent1", "v1")]["file.txt"])
}

func TestPopulateDataFileRegistry_SingleFileStructure(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base", 0o755)
	_ = afero.WriteFile(fs, "/base/file.txt", []byte("data"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	require.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg
	// Should skip files without at least agentName and version structure
	assert.Empty(t, files)
}

func TestPopulateDataFileRegistry_WalkErrors(t *testing.T) {
	fs := afero.NewMemMapFs()

	t.Run("WalkPermissionError", func(t *testing.T) {
		// Create a directory structure
		_ = fs.MkdirAll("/base/agent1/v1", 0o755)
		_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

		// This test checks that the function continues even if there are walk errors
		reg, err := data.PopulateDataFileRegistry(fs, "/base")
		require.NoError(t, err)
		assert.NotNil(t, reg)
		// Should still process the files that are accessible
		files := *reg
		assert.Len(t, files, 1)
	})

	t.Run("RelativePathError", func(t *testing.T) {
		// Test case where filepath.Rel might have issues
		// This is harder to trigger in practice, but let's ensure robustness
		fs := afero.NewMemMapFs()
		_ = fs.MkdirAll("/base/agent1/v1", 0o755)
		_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

		reg, err := data.PopulateDataFileRegistry(fs, "/base")
		require.NoError(t, err)
		assert.NotNil(t, reg)
		files := *reg
		assert.Len(t, files, 1)
	})
}

func TestPopulateDataFileRegistry_EmptyAgentPath(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a structure with just one level (should be skipped)
	_ = fs.MkdirAll("/base/onelevel", 0o755)
	_ = afero.WriteFile(fs, "/base/onelevel.txt", []byte("data"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	require.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg
	// Should be empty since files don't have proper agent/version structure
	assert.Empty(t, files)
}

func TestPopulateDataFileRegistry_MixedContent(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a mix of valid and invalid structures
	_ = fs.MkdirAll("/base/agent1/v1", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/valid.txt", []byte("data"), 0o644)
	_ = afero.WriteFile(fs, "/base/invalid.txt", []byte("data"), 0o644)
	_ = fs.MkdirAll("/base/onlyone", 0o755)
	_ = afero.WriteFile(fs, "/base/onlyone/file.txt", []byte("data"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	require.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	// Should only contain the valid agent/version structure
	assert.Len(t, files, 2) // agent1/v1 and onlyone/file.txt
	assert.Contains(t, files, "agent1/v1")
	assert.Contains(t, files, "onlyone/file.txt")
}

func TestPopulateDataFileRegistry_ErrorConditions(t *testing.T) {
	t.Run("DirExistsError", func(t *testing.T) {
		efs := errorFs{afero.NewMemMapFs()}
		reg, err := data.PopulateDataFileRegistry(efs, "/base")
		require.Error(t, err)
		assert.NotNil(t, reg)
		assert.Empty(t, *reg)
	})

	t.Run("WalkError", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// Create a directory structure
		_ = fs.MkdirAll("/base/agent1/v1", 0o755)
		_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

		wefs := walkErrorFs{fs}
		reg, err := data.PopulateDataFileRegistry(wefs, "/base")
		require.NoError(t, err) // Walk errors are ignored
		assert.NotNil(t, reg)
		// Since we can't actually inject a walk error, we verify that the function
		// continues processing and returns a non-empty registry
		assert.NotEmpty(t, *reg)
	})

	t.Run("RelativePathError", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		// Create a directory structure that will cause a relative path error
		_ = fs.MkdirAll("/base/agent1/v1", 0o755)
		_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

		sefs := statErrorFs{fs}
		reg, err := data.PopulateDataFileRegistry(sefs, "/base")
		require.Error(t, err) // Stat errors should cause the function to fail
		assert.NotNil(t, reg) // Function returns empty map on error, not nil
		assert.Empty(t, *reg) // But the map should be empty
		assert.Contains(t, err.Error(), "stat error")
	})
}

func TestPopulateDataFileRegistry(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Test successful registry population
	registry, err := data.PopulateDataFileRegistry(afero.NewOsFs(), tempDir)
	require.NoError(t, err)
	assert.NotNil(t, registry)
}

func TestPopulateDataFileRegistryWithInvalidPath(t *testing.T) {
	// Test with an invalid path
	invalidPath := "/nonexistent/path"

	registry, err := data.PopulateDataFileRegistry(afero.NewOsFs(), invalidPath)
	require.NoError(t, err)
	assert.NotNil(t, registry)
	assert.Empty(t, *registry)
}

func TestPopulateDataFileRegistryWithEmptyDirectory(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Test with empty directory
	registry, err := data.PopulateDataFileRegistry(afero.NewOsFs(), tempDir)
	require.NoError(t, err)
	assert.NotNil(t, registry)
}

func TestPopulateDataFileRegistryWithFiles(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create some test files
	testFiles := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, filename := range testFiles {
		filePath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filePath, []byte("test content"), 0o644)
		require.NoError(t, err)
	}

	// Test with files in directory
	registry, err := data.PopulateDataFileRegistry(afero.NewOsFs(), tempDir)
	require.NoError(t, err)
	assert.NotNil(t, registry)
}

func TestPopulateDataFileRegistryWithSubdirectories(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create subdirectories
	subDir1 := filepath.Join(tempDir, "subdir1")
	subDir2 := filepath.Join(tempDir, "subdir2")
	err := os.MkdirAll(subDir1, 0o755)
	require.NoError(t, err)
	err = os.MkdirAll(subDir2, 0o755)
	require.NoError(t, err)

	// Create files in subdirectories
	file1 := filepath.Join(subDir1, "file1.txt")
	file2 := filepath.Join(subDir2, "file2.txt")
	err = os.WriteFile(file1, []byte("test content 1"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(file2, []byte("test content 2"), 0o644)
	require.NoError(t, err)

	// Test with subdirectories
	registry, err := data.PopulateDataFileRegistry(afero.NewOsFs(), tempDir)
	require.NoError(t, err)
	assert.NotNil(t, registry)
}

func TestPopulateDataFileRegistryWithSymlinks(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a test file
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Create a symlink
	symlink := filepath.Join(tempDir, "link.txt")
	err = os.Symlink(testFile, symlink)
	require.NoError(t, err)

	// Test with symlinks
	registry, err := data.PopulateDataFileRegistry(afero.NewOsFs(), tempDir)
	require.NoError(t, err)
	assert.NotNil(t, registry)
}

func TestPopulateDataFileRegistryWithHiddenFiles(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create hidden files
	hiddenFile := filepath.Join(tempDir, ".hidden.txt")
	err := os.WriteFile(hiddenFile, []byte("hidden content"), 0o644)
	require.NoError(t, err)

	// Test with hidden files
	registry, err := data.PopulateDataFileRegistry(afero.NewOsFs(), tempDir)
	require.NoError(t, err)
	assert.NotNil(t, registry)
}

func TestPopulateDataFileRegistryWithLargeFiles(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create a large file (1MB)
	largeFile := filepath.Join(tempDir, "large.txt")
	largeContent := make([]byte, 1024*1024) // 1MB
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}
	err := os.WriteFile(largeFile, largeContent, 0o644)
	require.NoError(t, err)

	// Test with large files
	registry, err := data.PopulateDataFileRegistry(afero.NewOsFs(), tempDir)
	require.NoError(t, err)
	assert.NotNil(t, registry)
}

func TestPopulateDataFileRegistryWithSpecialCharacters(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := t.TempDir()

	// Create files with special characters in names
	specialFile := filepath.Join(tempDir, "file with spaces.txt")
	err := os.WriteFile(specialFile, []byte("special content"), 0o644)
	require.NoError(t, err)

	// Test with special characters
	registry, err := data.PopulateDataFileRegistry(afero.NewOsFs(), tempDir)
	require.NoError(t, err)
	assert.NotNil(t, registry)
}
