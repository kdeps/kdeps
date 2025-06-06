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
