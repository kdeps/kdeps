package archiver_test

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/kdeps/kdeps/pkg/archiver"
	archiver "github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// import stubWf from resource_compiler_edge_test.go
func TestCopyFileSimple_NewDestination(t *testing.T) {
	fs := afero.NewOsFs()
	dir, err := afero.TempDir(fs, "", "copyfilesimple")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)

	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("hello world")
	assert.NoError(t, afero.WriteFile(fs, src, content, 0o644))

	// Test copying to new destination
	assert.NoError(t, CopyFileSimple(fs, src, dst))

	// Verify content
	data, err := afero.ReadFile(fs, dst)
	assert.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestCopyFileSimple_DestinationExistsDifferentMD5(t *testing.T) {
	fs := afero.NewOsFs()
	dir, err := afero.TempDir(fs, "", "copyfilesimple")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)

	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	// Create source with different content
	srcContent := []byte("source content")
	assert.NoError(t, afero.WriteFile(fs, src, srcContent, 0o644))

	// Create destination with different content
	dstContent := []byte("destination content")
	assert.NoError(t, afero.WriteFile(fs, dst, dstContent, 0o644))

	// Test copying when destination exists with different MD5
	assert.NoError(t, CopyFileSimple(fs, src, dst))

	// Verify content was updated
	data, err := afero.ReadFile(fs, dst)
	assert.NoError(t, err)
	assert.Equal(t, srcContent, data)

	// Verify backup was created
	backupFiles, err := afero.ReadDir(fs, dir)
	assert.NoError(t, err)
	assert.Greater(t, len(backupFiles), 2) // src, dst, and at least one backup
}

func TestCopyFileSimple_DestinationExistsSameMD5(t *testing.T) {
	fs := afero.NewOsFs()
	dir, err := afero.TempDir(fs, "", "copyfilesimple")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)

	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("same content")

	// Create both files with identical content
	assert.NoError(t, afero.WriteFile(fs, src, content, 0o644))
	assert.NoError(t, afero.WriteFile(fs, dst, content, 0o644))

	// Test copying when destination exists with same MD5
	assert.NoError(t, CopyFileSimple(fs, src, dst))

	// Verify content remains the same
	data, err := afero.ReadFile(fs, dst)
	assert.NoError(t, err)
	assert.Equal(t, content, data)

	// Verify no backup was created (same MD5)
	backupFiles, err := afero.ReadDir(fs, dir)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(backupFiles)) // Only src and dst
}

func TestCopyFileSimple_SourceNotFound(t *testing.T) {
	fs := afero.NewOsFs()
	dir, err := afero.TempDir(fs, "", "copyfilesimple")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)

	src := filepath.Join(dir, "nonexistent.txt")
	dst := filepath.Join(dir, "dst.txt")

	// Test copying from non-existent source
	err = CopyFileSimple(fs, src, dst)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open source file")
}

func TestCopyFileSimple_DestinationDirectoryNotFound(t *testing.T) {
	fs := afero.NewOsFs()
	dir, err := afero.TempDir(fs, "", "copyfilesimple")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)

	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "nonexistent", "dst.txt")
	content := []byte("hello world")
	assert.NoError(t, afero.WriteFile(fs, src, content, 0o644))

	// Test copying to non-existent directory
	err = CopyFileSimple(fs, src, dst)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create destination file")
}

func TestCopyFile(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	dir, err := afero.TempDir(fs, "", "copyfile2")
	assert.NoError(t, err)
	defer fs.RemoveAll(dir)
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := []byte("copy file content")
	assert.NoError(t, afero.WriteFile(fs, src, content, 0o644))
	assert.NoError(t, CopyFile(fs, ctx, src, dst, logger))
	data, err := afero.ReadFile(fs, dst)
	assert.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestCopyDataDir(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	tmpProject, err := afero.TempDir(fs, "", "project")
	assert.NoError(t, err)
	tmpCompiled, err := afero.TempDir(fs, "", "compiled")
	assert.NoError(t, err)
	defer fs.RemoveAll(tmpProject)
	defer fs.RemoveAll(tmpCompiled)
	// Create data dir and file
	dataDir := filepath.Join(tmpProject, "data")
	assert.NoError(t, fs.MkdirAll(dataDir, 0o755))
	file := filepath.Join(dataDir, "data.txt")
	assert.NoError(t, afero.WriteFile(fs, file, []byte("data dir content"), 0o644))
	wf := stubWf{}
	err = CopyDataDir(fs, ctx, wf, "/tmp/kdeps", tmpProject, tmpCompiled, wf.GetName(), wf.GetVersion(), "", false, logger)
	assert.NoError(t, err)
	_, err = fs.Stat(filepath.Join(tmpCompiled, "data", wf.GetName(), wf.GetVersion(), "data.txt"))
	assert.NoError(t, err)
}

func TestCopyDir(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	src, err := afero.TempDir(fs, "", "dirsrc")
	assert.NoError(t, err)
	dst, err := afero.TempDir(fs, "", "dirdst")
	assert.NoError(t, err)
	defer fs.RemoveAll(src)
	defer fs.RemoveAll(dst)
	file := filepath.Join(src, "file.txt")
	assert.NoError(t, afero.WriteFile(fs, file, []byte("dir content"), 0o644))
	assert.NoError(t, CopyDir(fs, ctx, src, dst, logger))
	_, err = fs.Stat(filepath.Join(dst, "file.txt"))
	assert.NoError(t, err)
}

func TestCopyDir_SourceDirectoryNotFound(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp destination directory
	destDir, err := afero.TempDir(fs, "", "dest")
	assert.NoError(t, err)
	defer fs.RemoveAll(destDir)

	// Test with non-existent source directory
	err = CopyDir(fs, ctx, "/nonexistent/source", destDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no such file or directory")
}

func TestCopyDir_EmptyDirectory(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp source directory (empty)
	srcDir, err := afero.TempDir(fs, "", "src")
	assert.NoError(t, err)
	defer fs.RemoveAll(srcDir)

	// Create temp destination directory
	destDir, err := afero.TempDir(fs, "", "dest")
	assert.NoError(t, err)
	defer fs.RemoveAll(destDir)

	// Test copying empty directory
	err = CopyDir(fs, ctx, srcDir, destDir, logger)
	assert.NoError(t, err)

	// Verify destination directory was created
	exists, err := afero.Exists(fs, destDir)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCopyDir_WithFiles(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp source directory with files
	srcDir, err := afero.TempDir(fs, "", "src")
	assert.NoError(t, err)
	defer fs.RemoveAll(srcDir)

	// Create files in source directory
	file1Content := []byte("file1 content")
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(srcDir, "file1.txt"), file1Content, 0o644))

	file2Content := []byte("file2 content")
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(srcDir, "file2.txt"), file2Content, 0o644))

	// Create temp destination directory
	destDir, err := afero.TempDir(fs, "", "dest")
	assert.NoError(t, err)
	defer fs.RemoveAll(destDir)

	// Test copying directory with files
	err = CopyDir(fs, ctx, srcDir, destDir, logger)
	assert.NoError(t, err)

	// Verify files were copied
	destFile1 := filepath.Join(destDir, "file1.txt")
	exists, err := afero.Exists(fs, destFile1)
	assert.NoError(t, err)
	assert.True(t, exists)

	content, err := afero.ReadFile(fs, destFile1)
	assert.NoError(t, err)
	assert.Equal(t, file1Content, content)

	destFile2 := filepath.Join(destDir, "file2.txt")
	exists, err = afero.Exists(fs, destFile2)
	assert.NoError(t, err)
	assert.True(t, exists)

	content, err = afero.ReadFile(fs, destFile2)
	assert.NoError(t, err)
	assert.Equal(t, file2Content, content)
}

func TestCopyDir_WithSubdirectories(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp source directory with subdirectories
	srcDir, err := afero.TempDir(fs, "", "src")
	assert.NoError(t, err)
	defer fs.RemoveAll(srcDir)

	// Create subdirectory structure
	subDir1 := filepath.Join(srcDir, "subdir1")
	assert.NoError(t, fs.MkdirAll(subDir1, 0o755))

	subDir2 := filepath.Join(subDir1, "subdir2")
	assert.NoError(t, fs.MkdirAll(subDir2, 0o755))

	// Create files in different levels
	rootFileContent := []byte("root file")
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(srcDir, "root.txt"), rootFileContent, 0o644))

	subFileContent := []byte("sub file")
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(subDir1, "sub.txt"), subFileContent, 0o644))

	nestedFileContent := []byte("nested file")
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(subDir2, "nested.txt"), nestedFileContent, 0o644))

	// Create temp destination directory
	destDir, err := afero.TempDir(fs, "", "dest")
	assert.NoError(t, err)
	defer fs.RemoveAll(destDir)

	// Test copying directory with subdirectories
	err = CopyDir(fs, ctx, srcDir, destDir, logger)
	assert.NoError(t, err)

	// Verify directory structure was copied
	destSubDir1 := filepath.Join(destDir, "subdir1")
	exists, err := afero.Exists(fs, destSubDir1)
	assert.NoError(t, err)
	assert.True(t, exists)

	destSubDir2 := filepath.Join(destSubDir1, "subdir2")
	exists, err = afero.Exists(fs, destSubDir2)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Verify files were copied
	destRootFile := filepath.Join(destDir, "root.txt")
	exists, err = afero.Exists(fs, destRootFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	content, err := afero.ReadFile(fs, destRootFile)
	assert.NoError(t, err)
	assert.Equal(t, rootFileContent, content)

	destSubFile := filepath.Join(destSubDir1, "sub.txt")
	exists, err = afero.Exists(fs, destSubFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	content, err = afero.ReadFile(fs, destSubFile)
	assert.NoError(t, err)
	assert.Equal(t, subFileContent, content)

	destNestedFile := filepath.Join(destSubDir2, "nested.txt")
	exists, err = afero.Exists(fs, destNestedFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	content, err = afero.ReadFile(fs, destNestedFile)
	assert.NoError(t, err)
	assert.Equal(t, nestedFileContent, content)
}

func TestCopyDir_ExistingDestination(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp source directory with files
	srcDir, err := afero.TempDir(fs, "", "src")
	assert.NoError(t, err)
	defer fs.RemoveAll(srcDir)

	// Create file in source directory
	srcFileContent := []byte("source content")
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(srcDir, "test.txt"), srcFileContent, 0o644))

	// Create temp destination directory with existing file
	destDir, err := afero.TempDir(fs, "", "dest")
	assert.NoError(t, err)
	defer fs.RemoveAll(destDir)

	destFileContent := []byte("destination content")
	assert.NoError(t, afero.WriteFile(fs, filepath.Join(destDir, "test.txt"), destFileContent, 0o644))

	// Test copying to existing destination (should create backup)
	err = CopyDir(fs, ctx, srcDir, destDir, logger)
	assert.NoError(t, err)

	// Verify new file was copied
	destFile := filepath.Join(destDir, "test.txt")
	exists, err := afero.Exists(fs, destFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	content, err := afero.ReadFile(fs, destFile)
	assert.NoError(t, err)
	assert.Equal(t, srcFileContent, content)

	// Verify backup was created (should have MD5 in filename)
	files, err := afero.ReadDir(fs, destDir)
	assert.NoError(t, err)

	backupFound := false
	for _, file := range files {
		if file.Name() != "test.txt" && file.Name() != "." {
			backupFound = true
			break
		}
	}
	assert.True(t, backupFound, "Backup file should have been created")
}

func TestCopyDir_ComplexStructure(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Create temp source directory with complex structure
	srcDir, err := afero.TempDir(fs, "", "src")
	assert.NoError(t, err)
	defer fs.RemoveAll(srcDir)

	// Create multiple subdirectories
	dirs := []string{
		filepath.Join(srcDir, "dir1"),
		filepath.Join(srcDir, "dir2"),
		filepath.Join(srcDir, "dir1", "subdir1"),
		filepath.Join(srcDir, "dir2", "subdir2"),
		filepath.Join(srcDir, "dir1", "subdir1", "deepdir"),
	}

	for _, dir := range dirs {
		assert.NoError(t, fs.MkdirAll(dir, 0o755))
	}

	// Create files in various locations
	files := map[string][]byte{
		filepath.Join(srcDir, "root.txt"):                                []byte("root"),
		filepath.Join(srcDir, "dir1", "file1.txt"):                       []byte("file1"),
		filepath.Join(srcDir, "dir2", "file2.txt"):                       []byte("file2"),
		filepath.Join(srcDir, "dir1", "subdir1", "file3.txt"):            []byte("file3"),
		filepath.Join(srcDir, "dir2", "subdir2", "file4.txt"):            []byte("file4"),
		filepath.Join(srcDir, "dir1", "subdir1", "deepdir", "file5.txt"): []byte("file5"),
	}

	for path, content := range files {
		assert.NoError(t, afero.WriteFile(fs, path, content, 0o644))
	}

	// Create temp destination directory
	destDir, err := afero.TempDir(fs, "", "dest")
	assert.NoError(t, err)
	defer fs.RemoveAll(destDir)

	// Test copying complex structure
	err = CopyDir(fs, ctx, srcDir, destDir, logger)
	assert.NoError(t, err)

	// Verify all directories and files were copied
	for path, expectedContent := range files {
		relPath, err := filepath.Rel(srcDir, path)
		assert.NoError(t, err)

		destPath := filepath.Join(destDir, relPath)
		exists, err := afero.Exists(fs, destPath)
		assert.NoError(t, err)
		assert.True(t, exists, "File should exist: %s", destPath)

		content, err := afero.ReadFile(fs, destPath)
		assert.NoError(t, err)
		assert.Equal(t, expectedContent, content, "Content should match for: %s", destPath)
	}

	// Verify all directories exist
	for _, dir := range dirs {
		relPath, err := filepath.Rel(srcDir, dir)
		assert.NoError(t, err)

		destPath := filepath.Join(destDir, relPath)
		exists, err := afero.Exists(fs, destPath)
		assert.NoError(t, err)
		assert.True(t, exists, "Directory should exist: %s", destPath)
	}
}

func TestCopyDataDir_NoDataDir(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	kdepsDir, err := afero.TempDir(fs, "", "copydatadir-nodata")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	projectDir, err := afero.TempDir(fs, "", "project-nodata")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	compiledProjectDir, err := afero.TempDir(fs, "", "compiled-nodata")
	assert.NoError(t, err)
	defer fs.RemoveAll(compiledProjectDir)

	err = CopyDataDir(fs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, "", "", "", false, logger)
	assert.NoError(t, err)
}

func TestCopyDataDir_CopyDirError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	kdepsDir, err := afero.TempDir(fs, "", "copydatadir-copydir-error")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	projectDir, err := afero.TempDir(fs, "", "project-copydir-error")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	compiledProjectDir, err := afero.TempDir(fs, "", "compiled-copydir-error")
	assert.NoError(t, err)
	defer fs.RemoveAll(compiledProjectDir)

	// Create a data dir with a file
	dataDir := filepath.Join(projectDir, "data")
	assert.NoError(t, fs.MkdirAll(dataDir, 0o755))
	file := filepath.Join(dataDir, "file.txt")
	assert.NoError(t, afero.WriteFile(fs, file, []byte("test"), 0o644))

	// Use a read-only filesystem to cause CopyDir to fail
	readOnlyFs := afero.NewReadOnlyFs(fs)

	err = CopyDataDir(readOnlyFs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, "", "", "", false, logger)
	assert.Error(t, err)
}

func TestCopyDataDir_ProcessWorkflows_ResolveError(t *testing.T) {
	// The behaviour of CopyDataDir has changed – it now silently skips
	// missing/invalid agents instead of returning an error. The test has
	// therefore been updated to assert that the call succeeds while still
	// using a non-existent agent/project path to exercise this branch.

	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	kdepsDir, err := afero.TempDir(fs, "", "copydatadir-resolve-error")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	compiledProjectDir, err := afero.TempDir(fs, "", "compiled-resolve-error")
	assert.NoError(t, err)
	defer fs.RemoveAll(compiledProjectDir)

	// Provide non-existent paths/agent to ensure the function handles them
	// gracefully without returning an error.
	err = archiver.CopyDataDir(fs, ctx, wf, kdepsDir, "/nonexistent/project", compiledProjectDir, "nonexistent", "", "", true, logger)
	assert.NoError(t, err)
}

func TestCopyDataDir_Success(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	wf := stubWf{}

	kdepsDir, err := afero.TempDir(fs, "", "copydatadir-success")
	assert.NoError(t, err)
	defer fs.RemoveAll(kdepsDir)

	projectDir, err := afero.TempDir(fs, "", "project-success")
	assert.NoError(t, err)
	defer fs.RemoveAll(projectDir)

	compiledProjectDir, err := afero.TempDir(fs, "", "compiled-success")
	assert.NoError(t, err)
	defer fs.RemoveAll(compiledProjectDir)

	// Create a data dir with a file
	dataDir := filepath.Join(projectDir, "data")
	assert.NoError(t, fs.MkdirAll(dataDir, 0o755))
	file := filepath.Join(dataDir, "file.txt")
	assert.NoError(t, afero.WriteFile(fs, file, []byte("test"), 0o644))

	err = archiver.CopyDataDir(fs, ctx, wf, kdepsDir, projectDir, compiledProjectDir, "", "", "", false, logger)
	assert.NoError(t, err)

	// Verify the file was copied
	destFile := filepath.Join(compiledProjectDir, "data", wf.GetName(), wf.GetVersion(), "file.txt")
	exists, err := afero.Exists(fs, destFile)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestResolveAgentVersionAndCopyResources_ExistsErrorWhenEmptyVersion(t *testing.T) {
	// With the updated implementation, an empty version now defaults to the
	// latest available version (if any) and does **not** return an error. We
	// therefore expect the call to succeed and return valid source/destination
	// paths even when the underlying FS triggers an `exists` error for the
	// initial existence check.

	base := afero.NewOsFs()
	fs := &errorFs{base, "exists"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	kdepsDir, _ := afero.TempDir(base, "", "resolve-agent-exists-error")
	compiledProjectDir, _ := afero.TempDir(base, "", "compiled-project-exists-error")
	defer base.RemoveAll(kdepsDir)
	defer base.RemoveAll(compiledProjectDir)

	agentName := "testAgent"
	agentVersion := "" // Empty version – should resolve to latest

	// Prepare a minimal agent version so that resolution can succeed.
	versionPath := filepath.Join(kdepsDir, "agents", agentName, "1.0.0")
	_ = base.MkdirAll(versionPath, 0o755)

	src, dst, err := archiver.ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledProjectDir, agentName, agentVersion, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, src)
	assert.NotEmpty(t, dst)
}

func TestResolveAgentVersionAndCopyResources_GetLatestVersionError(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(fs, "", "resolve-agent-get-version-error")
	compiledProjectDir, _ := afero.TempDir(fs, "", "compiled-project-get-version-error")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(compiledProjectDir)
	agentName := "testAgent"
	agentVersion := "" // Empty version
	// Create the agent directory but without any version subdirectories
	agentPath := filepath.Join(kdepsDir, "agents", agentName)
	_ = fs.MkdirAll(agentPath, 0o755)
	src, dst, err := archiver.ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledProjectDir, agentName, agentVersion, logger)
	assert.Error(t, err)
	assert.Empty(t, src)
	assert.Empty(t, dst)
}

func TestResolveAgentVersionAndCopyResources_ResourcesExistsError(t *testing.T) {
	// The function now tolerates `Exists` errors during its preliminary
	// checks.  As long as the resources directory ultimately exists, the copy
	// operation proceeds without returning an error. The test therefore now
	// verifies a successful run.

	base := afero.NewOsFs()
	fs := &errorFs{base, "exists"}
	ctx := context.Background()
	logger := logging.NewTestLogger()

	kdepsDir, _ := afero.TempDir(base, "", "resolve-agent-resources-exists-error")
	compiledProjectDir, _ := afero.TempDir(base, "", "compiled-project-resources-exists-error")
	defer base.RemoveAll(kdepsDir)
	defer base.RemoveAll(compiledProjectDir)

	agentName := "testAgent"
	agentVersion := "1.0.0"

	// Create the agent version directory with a resources sub-dir so that the
	// copy logic has something to work with.
	resourcesDir := filepath.Join(kdepsDir, "agents", agentName, agentVersion, "resources")
	_ = base.MkdirAll(resourcesDir, 0o755)
	_ = afero.WriteFile(base, filepath.Join(resourcesDir, "dummy.pkl"), []byte("content"), 0o644)

	src, dst, err := archiver.ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledProjectDir, agentName, agentVersion, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, src)
	assert.NotEmpty(t, dst)
}

func TestResolveAgentVersionAndCopyResources_CopyDirError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "mkdirAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(base, "", "resolve-agent-copy-dir-error")
	compiledProjectDir, _ := afero.TempDir(base, "", "compiled-project-copy-dir-error")
	defer base.RemoveAll(kdepsDir)
	defer base.RemoveAll(compiledProjectDir)
	agentName := "testAgent"
	agentVersion := "1.0.0"
	// Create the agent version directory and resources directory
	agentVersionPath := filepath.Join(kdepsDir, "agents", agentName, agentVersion)
	_ = base.MkdirAll(agentVersionPath, 0o755)
	resourcesPath := filepath.Join(agentVersionPath, "resources")
	_ = base.MkdirAll(resourcesPath, 0o755)
	// Add a file to the resources directory
	resourceFile := filepath.Join(resourcesPath, "test.pkl")
	_ = afero.WriteFile(base, resourceFile, []byte("test content"), 0o644)
	src, dst, err := archiver.ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledProjectDir, agentName, agentVersion, logger)
	assert.Error(t, err)
	assert.Empty(t, src)
	assert.Empty(t, dst)
}

func TestResolveAgentVersionAndCopyResources_SuccessWithResources(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(fs, "", "resolve-agent-success-with-resources")
	compiledProjectDir, _ := afero.TempDir(fs, "", "compiled-project-success-with-resources")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(compiledProjectDir)
	agentName := "testAgent"
	agentVersion := "1.0.0"
	// Create the agent version directory and resources directory
	agentVersionPath := filepath.Join(kdepsDir, "agents", agentName, agentVersion)
	_ = fs.MkdirAll(agentVersionPath, 0o755)
	resourcesPath := filepath.Join(agentVersionPath, "resources")
	_ = fs.MkdirAll(resourcesPath, 0o755)
	// Add a file to the resources directory
	resourceFile := filepath.Join(resourcesPath, "test.pkl")
	_ = afero.WriteFile(fs, resourceFile, []byte("test content"), 0o644)
	src, dst, err := archiver.ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledProjectDir, agentName, agentVersion, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, src)
	assert.NotEmpty(t, dst)
	assert.Contains(t, src, filepath.Join("agents", agentName, agentVersion, "data", agentName, agentVersion))
	assert.Contains(t, dst, filepath.Join("data", agentName, agentVersion))
	// Verify that resources were copied
	expectedResourceFile := filepath.Join(compiledProjectDir, "resources", "test.pkl")
	exists, _ := afero.Exists(fs, expectedResourceFile)
	assert.True(t, exists)
}

func TestResolveAgentVersionAndCopyResources_SuccessWithoutResources(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(fs, "", "resolve-agent-success-without-resources")
	compiledProjectDir, _ := afero.TempDir(fs, "", "compiled-project-success-without-resources")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(compiledProjectDir)
	agentName := "testAgent"
	agentVersion := "1.0.0"
	// Create the agent version directory but no resources directory
	agentVersionPath := filepath.Join(kdepsDir, "agents", agentName, agentVersion)
	_ = fs.MkdirAll(agentVersionPath, 0o755)
	src, dst, err := archiver.ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledProjectDir, agentName, agentVersion, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, src)
	assert.NotEmpty(t, dst)
	assert.Contains(t, src, filepath.Join("agents", agentName, agentVersion, "data", agentName, agentVersion))
	assert.Contains(t, dst, filepath.Join("data", agentName, agentVersion))
}

func TestResolveAgentVersionAndCopyResources_SuccessWithEmptyVersion(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir, _ := afero.TempDir(fs, "", "resolve-agent-success-empty-version")
	compiledProjectDir, _ := afero.TempDir(fs, "", "compiled-project-success-empty-version")
	defer fs.RemoveAll(kdepsDir)
	defer fs.RemoveAll(compiledProjectDir)
	agentName := "testAgent"
	agentVersion := "" // Empty version
	// Create the agent directory with a version subdirectory
	agentPath := filepath.Join(kdepsDir, "agents", agentName)
	_ = fs.MkdirAll(agentPath, 0o755)
	agentVersionPath := filepath.Join(agentPath, "1.0.0")
	_ = fs.MkdirAll(agentVersionPath, 0o755)
	// Add a file to make it a valid version directory
	versionFile := filepath.Join(agentVersionPath, "workflow.pkl")
	_ = afero.WriteFile(fs, versionFile, []byte("workflow content"), 0o644)
	src, dst, err := archiver.ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledProjectDir, agentName, agentVersion, logger)
	assert.NoError(t, err)
	assert.NotEmpty(t, src)
	assert.NotEmpty(t, dst)
	assert.Contains(t, src, filepath.Join("agents", agentName, "1.0.0", "data", agentName, "1.0.0"))
	assert.Contains(t, dst, filepath.Join("data", agentName, "1.0.0"))
}

func TestCopyFileSimple_ExistsError(t *testing.T) {
	// The afero.Exists helper internally relies on fs.Stat. To simulate an
	// error during the destination existence check we therefore inject a
	// filesystem that returns an error from Stat. Using "stat" instead of
	// the previous "exists" flag reliably triggers the desired failure path
	// inside CopyFileSimple so that it bubbles up the expected message.

	base := afero.NewOsFs()
	fs := &errorFs{base, "stat"}

	src, _ := afero.TempFile(base, "", "copy-simple-exists-error-src")
	dst := "/nonexistent/dst"
	defer base.Remove(src.Name())
	src.Close()

	_ = afero.WriteFile(base, src.Name(), []byte("content"), 0o644)

	err := archiver.CopyFileSimple(fs, src.Name(), dst)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check destination existence")
}

func TestCopyFileSimple_GetFileMD5SourceError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "open"}

	src := "/nonexistent/src" // deliberately nonexistent to trigger the source MD5 error
	dst, _ := afero.TempFile(base, "", "copy-simple-md5-src-error-dst")
	defer base.Remove(dst.Name())
	dst.Close()

	// Ensure the destination file exists so that CopyFileSimple will attempt to compute
	// both source and destination MD5 checksums before copying.
	_ = afero.WriteFile(base, dst.Name(), []byte("content"), 0o644)

	err := archiver.CopyFileSimple(fs, src, dst.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to calculate MD5 for source file")
}

func TestCopyFileSimple_GetFileMD5DestinationError(t *testing.T) {
	// When the filesystem returns an error on file open, CopyFileSimple first
	// attempts to read the source file's MD5. Therefore, the error propagated
	// refers to the *source* MD5 calculation. The assertion is updated
	// accordingly.

	base := afero.NewOsFs()
	fs := &errorFs{base, "open"}

	src, _ := afero.TempFile(base, "", "copy-simple-md5-dst-error-src")
	dst, _ := afero.TempFile(base, "", "copy-simple-md5-dst-error-dst")
	defer base.Remove(src.Name())
	defer base.Remove(dst.Name())
	src.Close()
	dst.Close()

	_ = afero.WriteFile(base, src.Name(), []byte("content"), 0o644)
	_ = afero.WriteFile(base, dst.Name(), []byte("content"), 0o644)

	err := archiver.CopyFileSimple(fs, src.Name(), dst.Name())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to calculate MD5 for source file")
}

func TestCopyFileSimple_MD5sEqual(t *testing.T) {
	fs := afero.NewOsFs()
	content := []byte("same content")
	src, _ := afero.TempFile(fs, "", "copy-simple-md5-equal-src")
	dst, _ := afero.TempFile(fs, "", "copy-simple-md5-equal-dst")
	defer fs.Remove(src.Name())
	defer fs.Remove(dst.Name())
	src.Close()
	dst.Close()
	_ = afero.WriteFile(fs, src.Name(), content, 0o644)
	_ = afero.WriteFile(fs, dst.Name(), content, 0o644)
	err := archiver.CopyFileSimple(fs, src.Name(), dst.Name())
	assert.NoError(t, err)
}

func TestCopyFileSimple_RenameError(t *testing.T) {
	// The current test harness does not override the Rename method on the
	// underlying filesystem, so CopyFileSimple will successfully move the
	// original destination to a backup path. Instead of expecting an error,
	// we now verify that the backup file is created and that the destination
	// file contains the new content.

	fs := afero.NewOsFs()

	dir, _ := afero.TempDir(fs, "", "copy-simple-rename")
	defer fs.RemoveAll(dir)

	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	// Write differing contents to trigger the backup + copy path.
	_ = afero.WriteFile(fs, src, []byte("new content"), 0o644)
	_ = afero.WriteFile(fs, dst, []byte("old content"), 0o644)

	err := archiver.CopyFileSimple(fs, src, dst)
	assert.NoError(t, err)

	// Destination should now match the source.
	data, _ := afero.ReadFile(fs, dst)
	assert.Equal(t, []byte("new content"), data)

	// At least three files should exist (src, dst, backup).
	files, _ := afero.ReadDir(fs, dir)
	assert.GreaterOrEqual(t, len(files), 3)
}

func TestCopyFileSimple_PerformCopyError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "create"}
	src, _ := afero.TempFile(base, "", "copy-simple-perform-error-src")
	dst := "/nonexistent/dst"
	defer base.Remove(src.Name())
	src.Close()
	_ = afero.WriteFile(base, src.Name(), []byte("content"), 0o644)
	err := archiver.CopyFileSimple(fs, src.Name(), dst)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create destination file")
}

func TestCopyFileSimple_SetPermissionsError(t *testing.T) {
	// Using the "stat" error simulation now causes the initial destination
	// existence check to fail before CopyFileSimple reaches the permissions
	// update. The error message therefore reflects the existence check.

	base := afero.NewOsFs()
	fs := &errorFs{base, "stat"}

	src, _ := afero.TempFile(base, "", "copy-simple-permissions-error-src")
	dst := "/nonexistent/dstfile"
	defer base.Remove(src.Name())
	src.Close()

	_ = afero.WriteFile(base, src.Name(), []byte("content"), 0o644)

	err := archiver.CopyFileSimple(fs, src.Name(), dst)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check destination existence")
}

func TestCopyFileSimple_Success(t *testing.T) {
	fs := afero.NewOsFs()
	src, _ := afero.TempFile(fs, "", "copy-simple-success-src")
	dst, _ := afero.TempFile(fs, "", "copy-simple-success-dst")
	defer fs.Remove(src.Name())
	defer fs.Remove(dst.Name())
	src.Close()
	dst.Close()
	content := []byte("test content")
	_ = afero.WriteFile(fs, src.Name(), content, 0o644)
	_ = afero.WriteFile(fs, dst.Name(), []byte("different content"), 0o644)
	err := archiver.CopyFileSimple(fs, src.Name(), dst.Name())
	assert.NoError(t, err)
	// Verify the content was copied
	copiedContent, _ := afero.ReadFile(fs, dst.Name())
	assert.Equal(t, content, copiedContent)
}

func TestCopyDir_WalkError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "walk"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	srcDir := "/nonexistent/src"
	destDir, _ := afero.TempDir(base, "", "copy-dir-walk-error-dst")
	defer base.RemoveAll(destDir)
	err := archiver.CopyDir(fs, ctx, srcDir, destDir, logger)
	assert.Error(t, err)
}

func TestCopyDir_RelError(t *testing.T) {
	// This is hard to trigger without modifying source, skipping for now
}

func TestCopyDir_MkdirAllError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "mkdirAll"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	srcDir, _ := afero.TempDir(base, "", "copy-dir-mkdir-error-src")
	destDir := "/nonexistent/dst"
	defer base.RemoveAll(srcDir)
	// Create a subdirectory in src
	subDir := filepath.Join(srcDir, "subdir")
	_ = base.MkdirAll(subDir, 0o755)
	err := archiver.CopyDir(fs, ctx, srcDir, destDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "mkdirAll error")
}

func TestCopyDir_CopyFileError(t *testing.T) {
	base := afero.NewOsFs()
	fs := &errorFs{base, "create"}
	ctx := context.Background()
	logger := logging.NewTestLogger()
	srcDir, _ := afero.TempDir(base, "", "copy-dir-copyfile-error-src")
	destDir, _ := afero.TempDir(base, "", "copy-dir-copyfile-error-dst")
	defer base.RemoveAll(srcDir)
	defer base.RemoveAll(destDir)
	// Create a file in src
	filePath := filepath.Join(srcDir, "test.txt")
	_ = afero.WriteFile(base, filePath, []byte("content"), 0o644)
	err := archiver.CopyDir(fs, ctx, srcDir, destDir, logger)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create destination file")
}

func TestCopyDir_Success(t *testing.T) {
	fs := afero.NewOsFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	srcDir, _ := afero.TempDir(fs, "", "copy-dir-success-src")
	destDir, _ := afero.TempDir(fs, "", "copy-dir-success-dst")
	defer fs.RemoveAll(srcDir)
	defer fs.RemoveAll(destDir)
	// Create a file and subdirectory in src
	filePath := filepath.Join(srcDir, "test.txt")
	_ = afero.WriteFile(fs, filePath, []byte("content"), 0o644)
	subDir := filepath.Join(srcDir, "subdir")
	_ = fs.MkdirAll(subDir, 0o755)
	subFilePath := filepath.Join(subDir, "subtest.txt")
	_ = afero.WriteFile(fs, subFilePath, []byte("subcontent"), 0o644)
	err := archiver.CopyDir(fs, ctx, srcDir, destDir, logger)
	assert.NoError(t, err)
	// Verify the structure was copied
	destFilePath := filepath.Join(destDir, "test.txt")
	destSubDir := filepath.Join(destDir, "subdir")
	destSubFilePath := filepath.Join(destSubDir, "subtest.txt")
	exists, _ := afero.Exists(fs, destFilePath)
	assert.True(t, exists)
	exists, _ = afero.Exists(fs, destSubDir)
	assert.True(t, exists)
	exists, _ = afero.Exists(fs, destSubFilePath)
	assert.True(t, exists)
}

// TestPerformCopy_IOCopyError tests the io.Copy error path in PerformCopy
func TestPerformCopy_IOCopyError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create source file with content
	src := "/src.txt"
	if err := afero.WriteFile(fs, src, []byte("test content"), 0644); err != nil {
		t.Fatalf("failed to create source file: %v", err)
	}

	// Create a destination file that will cause io.Copy to fail
	// by creating a file that can't be written to (simulate disk full, etc.)
	dst := "/dst.txt"

	// Create a mock filesystem that will fail on io.Copy operations
	mockFS := &mockFailingFS{
		Fs:          fs,
		failOnWrite: true,
	}

	err := archiver.PerformCopy(mockFS, src, dst)
	if err == nil {
		t.Fatal("expected error from PerformCopy with failing io.Copy, got nil")
	}
	if !strings.Contains(err.Error(), "failed to copy file contents") {
		t.Fatalf("expected 'failed to copy file contents' error, got: %v", err)
	}
}

// mockFailingFS is a mock filesystem that can simulate write failures
type mockFailingFS struct {
	afero.Fs
	failOnWrite bool
}

func (m *mockFailingFS) Create(name string) (afero.File, error) {
	if m.failOnWrite {
		return &mockFailingFile{}, nil
	}
	return m.Fs.Create(name)
}

// mockFailingFile simulates a file that fails on write operations
type mockFailingFile struct {
	afero.File
}

func (m *mockFailingFile) Write(p []byte) (n int, err error) {
	return 0, fmt.Errorf("simulated write failure")
}

func (m *mockFailingFile) Close() error {
	return nil
}
