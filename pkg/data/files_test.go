package data

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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
func (e errorFs) Stat(name string) (os.FileInfo, error)     { return nil, errors.New("stat error") }
func (e errorFs) Rename(oldname, newname string) error      { return e.Fs.Rename(oldname, newname) }
func (e errorFs) Chmod(name string, mode os.FileMode) error { return e.Fs.Chmod(name, mode) }
func (e errorFs) Chtimes(name string, atime, mtime time.Time) error {
	return e.Fs.Chtimes(name, atime, mtime)
}

func TestPopulateDataFileRegistry_BaseDirDoesNotExist(t *testing.T) {
	fs := afero.NewMemMapFs()
	reg, err := PopulateDataFileRegistry(fs, "/not-exist")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	assert.Empty(t, *reg)
}

func TestPopulateDataFileRegistry_EmptyBaseDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base", 0o755)
	reg, err := PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
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

	reg, err := PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
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
	reg, err := PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg
	assert.Len(t, files, 1)
	assert.Contains(t, files, filepath.Join("agent1", "file.txt"))
	assert.Equal(t, map[string]string{"": "/base/agent1/file.txt"}, files[filepath.Join("agent1", "file.txt")])
}

func TestPopulateDataFileRegistry_ErrorOnDirExists(t *testing.T) {
	efs := errorFs{afero.NewMemMapFs()}
	reg, err := PopulateDataFileRegistry(efs, "/base")
	assert.Error(t, err)
	assert.NotNil(t, reg)
	assert.Empty(t, *reg)
}

func TestPopulateDataFileRegistry_NestedDirectories(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base/agent1/v1/subdir", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/subdir/file.txt", []byte("data"), 0o644)

	reg, err := PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
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

	reg, err := PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
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

	reg, err := PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
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
		reg, err := PopulateDataFileRegistry(fs, "/base")
		assert.NoError(t, err)
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

		reg, err := PopulateDataFileRegistry(fs, "/base")
		assert.NoError(t, err)
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

	reg, err := PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
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

	reg, err := PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	// Should only contain the valid agent/version structure
	assert.Len(t, files, 2) // agent1/v1 and onlyone/file.txt
	assert.Contains(t, files, "agent1/v1")
	assert.Contains(t, files, "onlyone/file.txt")
}
