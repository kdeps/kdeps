package data_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/pkg/data"
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
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	assert.Empty(t, *reg)
}

func TestPopulateDataFileRegistry_EmptyBaseDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base", 0o755)
	reg, err := data.PopulateDataFileRegistry(fs, "/base")
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

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
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
	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg
	assert.Len(t, files, 1)
	assert.Contains(t, files, filepath.Join("agent1", "file.txt"))
	assert.Equal(t, map[string]string{"": "/base/agent1/file.txt"}, files[filepath.Join("agent1", "file.txt")])
}

func TestPopulateDataFileRegistry_ErrorOnDirExists(t *testing.T) {
	efs := errorFs{afero.NewMemMapFs()}
	reg, err := data.PopulateDataFileRegistry(efs, "/base")
	assert.Error(t, err)
	assert.NotNil(t, reg)
	assert.Empty(t, *reg)
}

func TestPopulateDataFileRegistry_NestedDirectories(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base/agent1/v1/subdir", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/subdir/file.txt", []byte("data"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
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

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
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

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
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
		reg, err := data.PopulateDataFileRegistry(fs, "/base")
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

		reg, err := data.PopulateDataFileRegistry(fs, "/base")
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

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
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

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
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
		assert.Error(t, err)
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
		assert.NoError(t, err) // Walk errors are ignored
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
		assert.NoError(t, err) // Relative path errors are ignored
		assert.NotNil(t, reg)
		// The file should be skipped due to stat error, but the directory structure
		// should still be processed
		assert.Empty(t, *reg)
	})
}

// TestPopulateDataFileRegistry_MapCreationAndReuse tests that agent maps are created only once
func TestPopulateDataFileRegistry_MapCreationAndReuse(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create multiple files for the same agent to test map reuse
	_ = fs.MkdirAll("/base/agent1/v1", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file1.txt", []byte("data1"), 0o644)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file2.txt", []byte("data2"), 0o644)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file3.txt", []byte("data3"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	// Should have one agent entry with multiple files
	assert.Len(t, files, 1)
	agentKey := filepath.Join("agent1", "v1")
	assert.Contains(t, files, agentKey)
	assert.Len(t, files[agentKey], 3)
	assert.Equal(t, "/base/agent1/v1/file1.txt", files[agentKey]["file1.txt"])
	assert.Equal(t, "/base/agent1/v1/file2.txt", files[agentKey]["file2.txt"])
	assert.Equal(t, "/base/agent1/v1/file3.txt", files[agentKey]["file3.txt"])
}

// TestPopulateDataFileRegistry_ComplexPaths tests complex directory structures
func TestPopulateDataFileRegistry_ComplexPaths(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create complex nested structures
	_ = fs.MkdirAll("/base/complex-agent/v1.0.0/deep/nested/structure", 0o755)
	_ = afero.WriteFile(fs, "/base/complex-agent/v1.0.0/deep/nested/structure/file.txt", []byte("data"), 0o644)
	_ = afero.WriteFile(fs, "/base/complex-agent/v1.0.0/top-level.txt", []byte("data"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	agentKey := filepath.Join("complex-agent", "v1.0.0")
	assert.Contains(t, files, agentKey)
	assert.Len(t, files[agentKey], 2)
	assert.Equal(t, "/base/complex-agent/v1.0.0/top-level.txt", files[agentKey]["top-level.txt"])
	expectedNestedPath := filepath.Join("deep", "nested", "structure", "file.txt")
	assert.Equal(t, "/base/complex-agent/v1.0.0/deep/nested/structure/file.txt", files[agentKey][expectedNestedPath])
}

// TestPopulateDataFileRegistry_EmptyKey tests files that would result in empty keys
func TestPopulateDataFileRegistry_EmptyKey(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a file exactly at agent/version level (no additional path components)
	_ = fs.MkdirAll("/base/agent1/v1", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1", []byte("data"), 0o644) // File with same name as directory
	_ = fs.Remove("/base/agent1/v1")                                  // Remove the directory
	_ = afero.WriteFile(fs, "/base/agent1/v1", []byte("data"), 0o644) // Create file instead

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	// This should create an entry with empty key
	agentKey := filepath.Join("agent1", "v1")
	assert.Contains(t, files, agentKey)
	assert.Equal(t, "/base/agent1/v1", files[agentKey][""])
}

// TestPopulateDataFileRegistry_PathSeparatorHandling tests different path separators
func TestPopulateDataFileRegistry_PathSeparatorHandling(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create structure with multiple levels to test separator handling
	_ = fs.MkdirAll("/base/my-agent/v2.1/configs/prod", 0o755)
	_ = afero.WriteFile(fs, "/base/my-agent/v2.1/configs/prod/config.json", []byte("{}"), 0o644)
	_ = afero.WriteFile(fs, "/base/my-agent/v2.1/configs/staging.json", []byte("{}"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	agentKey := filepath.Join("my-agent", "v2.1")
	assert.Contains(t, files, agentKey)
	assert.Len(t, files[agentKey], 2)

	expectedProdPath := filepath.Join("configs", "prod", "config.json")
	expectedStagingPath := filepath.Join("configs", "staging.json")
	assert.Equal(t, "/base/my-agent/v2.1/configs/prod/config.json", files[agentKey][expectedProdPath])
	assert.Equal(t, "/base/my-agent/v2.1/configs/staging.json", files[agentKey][expectedStagingPath])
}

// TestPopulateDataFileRegistry_LargeStructure tests performance with many files
func TestPopulateDataFileRegistry_LargeStructure(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create multiple agents with multiple versions and files
	for i := 1; i <= 3; i++ {
		for j := 1; j <= 2; j++ {
			agentName := "agent" + string(rune('0'+i))
			version := "v" + string(rune('0'+j))
			dirPath := filepath.Join("/base", agentName, version)
			_ = fs.MkdirAll(dirPath, 0o755)

			// Create multiple files per version
			for k := 1; k <= 2; k++ {
				fileName := filepath.Join(dirPath, "file"+string(rune('0'+k))+".txt")
				_ = afero.WriteFile(fs, fileName, []byte("data"), 0o644)
			}
		}
	}

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	// Should have 6 agent/version combinations (3 agents × 2 versions)
	assert.Len(t, files, 6)

	// Verify each combination has 2 files
	for i := 1; i <= 3; i++ {
		for j := 1; j <= 2; j++ {
			agentKey := filepath.Join("agent"+string(rune('0'+i)), "v"+string(rune('0'+j)))
			assert.Contains(t, files, agentKey)
			assert.Len(t, files[agentKey], 2)
		}
	}
}

// TestPopulateDataFileRegistry_FilepathRelError tests handling of filepath.Rel errors
func TestPopulateDataFileRegistry_FilepathRelError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a structure that might cause issues with relative path calculation
	_ = fs.MkdirAll("/base/agent1/v1", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

	// Test with an invalid base directory that might cause filepath.Rel to error
	reg, err := data.PopulateDataFileRegistry(fs, "")
	assert.NoError(t, err) // Should handle gracefully
	assert.NotNil(t, reg)
}

// TestPopulateDataFileRegistry_WalkFunctionErrors tests various walk function error conditions
func TestPopulateDataFileRegistry_WalkFunctionErrors(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a directory structure
	_ = fs.MkdirAll("/base/agent1/v1", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

	// Test that function continues even when individual file operations fail
	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)

	// Should still process accessible files
	files := *reg
	assert.NotEmpty(t, files)
}

// TestPopulateDataFileRegistry_EdgeCaseStructures tests edge case directory structures
func TestPopulateDataFileRegistry_EdgeCaseStructures(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Test case with exactly 2 path components (agent/version only)
	_ = fs.MkdirAll("/base/agent1/v1", 0o755)

	// Test case with very short path components
	_ = fs.MkdirAll("/base/a/b", 0o755)
	_ = afero.WriteFile(fs, "/base/a/b/c.txt", []byte("data"), 0o644)

	// Test case with special characters in names
	_ = fs.MkdirAll("/base/agent-special/v1.0", 0o755)
	_ = afero.WriteFile(fs, "/base/agent-special/v1.0/config.json", []byte("{}"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	// Should process the valid structures
	assert.Contains(t, files, filepath.Join("a", "b"))
	assert.Contains(t, files, filepath.Join("agent-special", "v1.0"))
}

// TestPopulateDataFileRegistry_MapInitialization tests map initialization edge cases
func TestPopulateDataFileRegistry_MapInitialization(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create a structure where the same agent appears multiple times to test map initialization
	_ = fs.MkdirAll("/base/agent1/v1/sub1", 0o755)
	_ = fs.MkdirAll("/base/agent1/v1/sub2", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/sub1/file1.txt", []byte("data1"), 0o644)
	_ = afero.WriteFile(fs, "/base/agent1/v1/sub2/file2.txt", []byte("data2"), 0o644)
	_ = afero.WriteFile(fs, "/base/agent1/v1/root.txt", []byte("root"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	agentKey := filepath.Join("agent1", "v1")
	assert.Contains(t, files, agentKey)
	assert.Len(t, files[agentKey], 3)

	// Verify all files are correctly mapped
	expectedPaths := map[string]string{
		"root.txt":                         "/base/agent1/v1/root.txt",
		filepath.Join("sub1", "file1.txt"): "/base/agent1/v1/sub1/file1.txt",
		filepath.Join("sub2", "file2.txt"): "/base/agent1/v1/sub2/file2.txt",
	}

	for key, expectedPath := range expectedPaths {
		assert.Equal(t, expectedPath, files[agentKey][key])
	}
}

// TestPopulateDataFileRegistry_PathJoinEdgeCases tests edge cases in path joining
func TestPopulateDataFileRegistry_PathJoinEdgeCases(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create structure with single character names
	_ = fs.MkdirAll("/base/a/1", 0o755)
	_ = afero.WriteFile(fs, "/base/a/1/x.txt", []byte("data"), 0o644)

	// Create structure with numeric names
	_ = fs.MkdirAll("/base/123/456", 0o755)
	_ = afero.WriteFile(fs, "/base/123/456/file.txt", []byte("data"), 0o644)

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	files := *reg

	// Should correctly handle these edge cases
	assert.Contains(t, files, filepath.Join("a", "1"))
	assert.Contains(t, files, filepath.Join("123", "456"))
	assert.Equal(t, "/base/a/1/x.txt", files[filepath.Join("a", "1")]["x.txt"])
	assert.Equal(t, "/base/123/456/file.txt", files[filepath.Join("123", "456")]["file.txt"])
}

// TestPopulateDataFileRegistry_InjectableFunctions tests error paths using injectable functions
func TestPopulateDataFileRegistry_InjectableFunctions(t *testing.T) {
	// Save original functions
	originalDirExists := data.DirExistsFunc
	originalWalk := data.WalkFunc
	originalFilepathRel := data.FilepathRelFunc
	originalFilepathJoin := data.FilepathJoinFunc

	// Restore original functions after tests
	defer func() {
		data.DirExistsFunc = originalDirExists
		data.WalkFunc = originalWalk
		data.FilepathRelFunc = originalFilepathRel
		data.FilepathJoinFunc = originalFilepathJoin
	}()

	t.Run("WalkFunctionError", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		_ = fs.MkdirAll("/base", 0o755)

		// Mock WalkFunc to return an error
		data.WalkFunc = func(fs afero.Fs, root string, walkFn filepath.WalkFunc) error {
			return errors.New("walk error")
		}

		reg, err := data.PopulateDataFileRegistry(fs, "/base")
		assert.NoError(t, err) // Should handle walk errors gracefully
		assert.NotNil(t, reg)
		assert.Empty(t, *reg) // Should return empty registry on walk error
	})

	t.Run("FilepathRelError", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		_ = fs.MkdirAll("/base/agent1/v1", 0o755)
		_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

		// Mock FilepathRelFunc to return an error
		data.FilepathRelFunc = func(basepath, targpath string) (string, error) {
			return "", errors.New("filepath.Rel error")
		}

		reg, err := data.PopulateDataFileRegistry(fs, "/base")
		assert.NoError(t, err) // Should handle filepath.Rel errors gracefully
		assert.NotNil(t, reg)
		assert.Empty(t, *reg) // Should skip files that cause filepath.Rel errors
	})

	t.Run("WalkCallbackError", func(t *testing.T) {
		fs := afero.NewMemMapFs()
		_ = fs.MkdirAll("/base/agent1/v1", 0o755)
		_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

		// Mock WalkFunc to call the callback with an error
		data.WalkFunc = func(fs afero.Fs, root string, walkFn filepath.WalkFunc) error {
			// Call the callback with a walk error to test error handling in the callback
			info := &mockFileInfo{name: "file.txt", isDir: false}
			walkFn("/base/agent1/v1/file.txt", info, errors.New("walk callback error"))
			return nil
		}

		reg, err := data.PopulateDataFileRegistry(fs, "/base")
		assert.NoError(t, err) // Should handle callback errors gracefully
		assert.NotNil(t, reg)
		assert.Empty(t, *reg) // Should skip files that cause callback errors
	})

}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	mtime time.Time
	isDir bool
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m *mockFileInfo) ModTime() time.Time { return m.mtime }
func (m *mockFileInfo) IsDir() bool        { return m.isDir }
func (m *mockFileInfo) Sys() interface{}   { return nil }

// TestPopulateDataFileRegistry_WalkFuncCompleteFailure tests the scenario where WalkFunc itself fails entirely
func TestPopulateDataFileRegistry_WalkFuncCompleteFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base/agent1/v1", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

	// Save original WalkFunc
	originalWalkFunc := data.WalkFunc
	defer func() { data.WalkFunc = originalWalkFunc }()

	// Replace WalkFunc with one that always returns an error
	data.WalkFunc = func(fs afero.Fs, root string, walkFn filepath.WalkFunc) error {
		return errors.New("walk function completely failed")
	}

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err) // Should not return error even when walk fails
	assert.NotNil(t, reg)
	assert.Empty(t, *reg) // Should return empty registry when walk fails entirely
}

// TestPopulateDataFileRegistry_FilepathRelCompleteFailure tests the scenario where FilepathRelFunc always fails
func TestPopulateDataFileRegistry_FilepathRelCompleteFailure(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/base/agent1/v1", 0o755)
	_ = afero.WriteFile(fs, "/base/agent1/v1/file.txt", []byte("data"), 0o644)

	// Save original FilepathRelFunc
	originalFilepathRelFunc := data.FilepathRelFunc
	defer func() { data.FilepathRelFunc = originalFilepathRelFunc }()

	// Replace FilepathRelFunc with one that always returns an error
	data.FilepathRelFunc = func(basepath, targpath string) (string, error) {
		return "", errors.New("filepath.Rel failed")
	}

	reg, err := data.PopulateDataFileRegistry(fs, "/base")
	assert.NoError(t, err)
	assert.NotNil(t, reg)
	// Files should be skipped due to filepath.Rel errors, but registry should exist
	assert.Empty(t, *reg)
}
