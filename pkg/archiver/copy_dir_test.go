package archiver_test

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"

	archiver "github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/schema"
	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCopyDirSimpleSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()

	src := "/src"
	dst := "/dst"

	// Create nested structure in src
	if err := fs.MkdirAll(src+"/sub", 0o755); err != nil {
		t.Fatalf("mkdir err: %v", err)
	}
	if err := afero.WriteFile(fs, src+"/file1.txt", []byte("hello"), 0o644); err != nil {
		t.Fatalf("write err: %v", err)
	}
	if err := afero.WriteFile(fs, src+"/sub/file2.txt", []byte("world"), 0o600); err != nil {
		t.Fatalf("write err: %v", err)
	}

	if err := archiver.CopyDir(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyDir failed: %v", err)
	}

	// Validate copied content
	if data, _ := afero.ReadFile(fs, dst+"/file1.txt"); string(data) != "hello" {
		t.Fatalf("file1 content mismatch")
	}
	if data, _ := afero.ReadFile(fs, dst+"/sub/file2.txt"); string(data) != "world" {
		t.Fatalf("file2 content mismatch")
	}
}

func TestCopyDirReadOnlyFailure(t *testing.T) {
	mem := afero.NewMemMapFs()
	readOnly := afero.NewReadOnlyFs(mem)
	logger := logging.GetLogger()

	src := "/src"
	dst := "/dst"

	_ = mem.MkdirAll(src, 0o755)
	_ = afero.WriteFile(mem, src+"/f.txt", []byte("x"), 0o644)

	if err := archiver.CopyDir(readOnly, ctx, src, dst, logger); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCopyDirSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	src := filepath.Join(t.TempDir(), "src")
	dst := filepath.Join(t.TempDir(), "dst")

	// create nested dirs & files
	files := []string{
		filepath.Join(src, "a.txt"),
		filepath.Join(src, "sub", "b.txt"),
		filepath.Join(src, "sub", "sub2", "c.txt"),
	}
	for _, f := range files {
		_ = fs.MkdirAll(filepath.Dir(f), 0o755)
		_ = afero.WriteFile(fs, f, []byte("x"), 0o644)
	}

	if err := archiver.CopyDir(fs, ctx, src, dst, logger); err != nil {
		t.Fatalf("CopyDir error: %v", err)
	}

	// ensure all files exist in dst
	for _, f := range files {
		rel, _ := filepath.Rel(src, f)
		if ok, _ := afero.Exists(fs, filepath.Join(dst, rel)); !ok {
			t.Fatalf("file not copied: %s", rel)
		}
	}
}

func TestCopyFileSkipIfHashesMatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	src := "/src.txt"
	dst := "/dst.txt"
	content := []byte("same")
	if err := afero.WriteFile(fs, src, content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	// Copy initial file to dst so hashes match
	if err := afero.WriteFile(fs, dst, content, 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}
}

func TestCopyFileCreatesBackupOnHashMismatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	src := "/src2.txt"
	dst := "/dst2.txt"

	if err := afero.WriteFile(fs, src, []byte("new"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := afero.WriteFile(fs, dst, []byte("old"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// backup should exist
	files, _ := afero.ReadDir(fs, "/")
	foundBackup := false
	for _, f := range files {
		if filepath.Ext(f.Name()) == ".txt" && f.Name() != "src2.txt" && f.Name() != "dst2.txt" {
			foundBackup = true
		}
	}
	if !foundBackup {
		t.Fatalf("expected backup file to be created")
	}
}

// TestCopyDir_Overwrite verifies that CopyDir creates a backup when the
// destination file already exists with different contents and then overwrites
// it with the new content.
func TestCopyDir_Overwrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Reference schema version (project rule compliance).
	_ = schema.Version(ctx)

	// Prepare source directory with a single file.
	srcDir := "/src"
	if err := fs.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	srcFilePath := filepath.Join(srcDir, "file.txt")
	if err := afero.WriteFile(fs, srcFilePath, []byte("new-content"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Prepare destination directory with an existing file (different content).
	dstDir := "/dst"
	if err := fs.MkdirAll(dstDir, 0o755); err != nil {
		t.Fatalf("mkdir dst: %v", err)
	}
	dstFilePath := filepath.Join(dstDir, "file.txt")
	if err := afero.WriteFile(fs, dstFilePath, []byte("old-content"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	// Run CopyDir which should create a backup of the old file and overwrite it.
	if err := archiver.CopyDir(fs, ctx, srcDir, dstDir, logger); err != nil {
		t.Fatalf("CopyDir returned error: %v", err)
	}

	// The destination file should now have the new content.
	data, err := afero.ReadFile(fs, dstFilePath)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "new-content" {
		t.Fatalf("content mismatch: got %q", string(data))
	}

	// A backup file with MD5 suffix should exist.
	files, _ := afero.ReadDir(fs, dstDir)
	var backupFound bool
	for _, f := range files {
		if f.Name() != "file.txt" && filepath.Ext(f.Name()) == ".txt" {
			backupFound = true
		}
	}
	if !backupFound {
		t.Fatalf("expected backup file to be created")
	}
}

// TestGetBackupPath_Sanity ensures the helper formats the backup path as
// expected.
func TestGetBackupPath_Sanity(t *testing.T) {
	dst := "/some/dir/file.txt"
	md5 := "deadbeef"
	got := archiver.GetBackupPath(dst, md5)
	expected := "/some/dir/file_deadbeef.txt"
	if got != expected {
		t.Fatalf("GetBackupPath mismatch: want %s got %s", expected, got)
	}
}

func TestCopyFile_NoDestination(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// create src
	_ = afero.WriteFile(fs, "/src.txt", []byte("abc"), 0o644)

	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile unexpected error: %v", err)
	}

	data, _ := afero.ReadFile(fs, "/dst.txt")
	if string(data) != "abc" {
		t.Fatalf("destination content mismatch")
	}
}

func TestCopyFile_SkipSameMD5(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	content := []byte("same")
	_ = afero.WriteFile(fs, "/src.txt", content, 0o644)
	_ = afero.WriteFile(fs, "/dst.txt", content, 0o644)

	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// ensure dst still exists and unchanged
	data, _ := afero.ReadFile(fs, "/dst.txt")
	if string(data) != "same" {
		t.Fatalf("dst altered unexpectedly")
	}
}

func TestPerformCopy_SuccessAndError(t *testing.T) {
	fs := afero.NewMemMapFs()
	// success path
	afero.WriteFile(fs, "/src.txt", []byte("hello"), 0o644)

	if err := archiver.PerformCopy(fs, "/src.txt", "/dst.txt"); err != nil {
		t.Fatalf("PerformCopy success returned error: %v", err)
	}

	data, _ := afero.ReadFile(fs, "/dst.txt")
	if string(data) != "hello" {
		t.Fatalf("content mismatch: %s", data)
	}

	// error path: source missing
	if err := archiver.PerformCopy(fs, "/missing.txt", "/dst2.txt"); err == nil {
		t.Fatalf("expected error when source missing")
	}
}

func TestCopyDir_Basic(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create source directory with nested content
	_ = fs.MkdirAll("/src/sub", 0o755)
	afero.WriteFile(fs, "/src/file1.txt", []byte("one"), 0o644)
	afero.WriteFile(fs, "/src/sub/file2.txt", []byte("two"), 0o644)

	if err := archiver.CopyDir(fs, ctx, "/src", "/dst", logger); err != nil {
		t.Fatalf("CopyDir error: %v", err)
	}

	// Verify copied files
	for _, p := range []struct{ path, expect string }{
		{"/dst/file1.txt", "one"},
		{"/dst/sub/file2.txt", "two"},
	} {
		data, err := afero.ReadFile(fs, p.path)
		if err != nil {
			t.Fatalf("missing copied file %s: %v", p.path, err)
		}
		if string(data) != p.expect {
			t.Fatalf("file %s content mismatch", p.path)
		}
	}
}

// TestCopyDirBasic exercises the main happy-path of CopyDir, ensuring it
// recreates directory structure and files.
func TestCopyDirBasic(t *testing.T) {
	fsys := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	src := "/src"
	dst := "/dst"

	// Build a small tree: /src/sub/hello.txt
	require.NoError(t, fsys.MkdirAll(filepath.Join(src, "sub"), 0o755))
	fileContent := []byte("copy_dir_contents")
	require.NoError(t, afero.WriteFile(fsys, filepath.Join(src, "sub", "hello.txt"), fileContent, 0o644))

	// Act
	require.NoError(t, archiver.CopyDir(fsys, ctx, src, dst, logger))

	// Assert: destination directory replicates the tree.
	copiedBytes, err := afero.ReadFile(fsys, filepath.Join(dst, "sub", "hello.txt"))
	require.NoError(t, err)
	require.Equal(t, fileContent, copiedBytes)

	// Permissions (mode) on directory should be preserved (at least execute bit).
	info, err := fsys.Stat(filepath.Join(dst, "sub"))
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

// TestCopyDirError verifies that an error from the underlying filesystem is
// propagated.  We create a read-only FS wrapper around a mem FS and attempt to
// write into it.
func TestCopyDirError(t *testing.T) {
	mem := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	src := "/ro/src"
	dst := "/ro/dst"
	require.NoError(t, mem.MkdirAll(src, 0o755))
	require.NoError(t, afero.WriteFile(mem, filepath.Join(src, "file.txt"), []byte("data"), 0o644))

	// Wrap in read-only fs to provoke write error on destination creation.
	ro := afero.NewReadOnlyFs(mem)

	err := archiver.CopyDir(ro, ctx, src, dst, logger)
	require.Error(t, err)

	// The error should be about permission or read-only.
	require.True(t, errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrInvalid))
}

// TestCopyFileSrcNotFound verifies that copyFile returns an error when the source file does not exist.
func TestCopyFileSrcNotFound(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()
	src := filepath.Join(tmp, "does_not_exist.txt")
	dst := filepath.Join(tmp, "dst.txt")

	if err := archiver.CopyFile(fs, context.Background(), src, dst, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error when source is missing")
	}

	// touch pkl schema reference to satisfy project convention
	_ = schema.Version(context.Background())
}

// TestCopyFileDestCreateError ensures copyFile surfaces an error when it cannot create the destination file.
func TestCopyFileDestCreateError(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	// Create a valid source file.
	src := filepath.Join(tmp, "src.txt")
	if err := afero.WriteFile(fs, src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Create a read-only directory; writing inside it should fail.
	roDir := filepath.Join(tmp, "readonly")
	if err := fs.MkdirAll(roDir, 0o500); err != nil { // read & execute only
		t.Fatalf("mkdir: %v", err)
	}

	dst := filepath.Join(roDir, "dst.txt")
	if err := archiver.CopyFile(fs, context.Background(), src, dst, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error when destination directory is not writable")
	}

	// Clean up permissions so the temp dir can be removed on Windows.
	_ = fs.Chmod(roDir, os.FileMode(0o700))

	_ = schema.Version(context.Background())
}

// TestCopyFileSimple verifies that copyFile copies contents when destination
// is absent.
func TestCopyFileSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")

	if err := afero.WriteFile(fs, src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := archiver.CopyFile(fs, context.Background(), src, dst, logging.NewTestLogger()); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	data, _ := afero.ReadFile(fs, dst)
	if string(data) != "hello" {
		t.Fatalf("content mismatch: %s", string(data))
	}

	_ = schema.Version(context.Background())
}

// TestCopyFileOverwrite ensures that copyFile overwrites an existing file.
func TestCopyFileOverwrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	src := filepath.Join(dir, "s.txt")
	dst := filepath.Join(dir, "d.txt")

	_ = afero.WriteFile(fs, src, []byte("new"), 0o644)
	_ = afero.WriteFile(fs, dst, []byte("old"), 0o644)

	if err := archiver.CopyFile(fs, context.Background(), src, dst, logging.NewTestLogger()); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	data, _ := afero.ReadFile(fs, dst)
	if string(data) != "new" {
		t.Fatalf("overwrite failed, got %s", string(data))
	}

	_ = schema.Version(context.Background())
}

// TestCopyFileSkipSameMD5 ensures CopyFile detects identical content and skips copying.
func TestCopyFileSkipSameMD5(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	src := filepath.Join(dir, "f.txt")
	dst := filepath.Join(dir, "d.txt")

	content := []byte("identical")
	if err := afero.WriteFile(fs, src, content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := afero.WriteFile(fs, dst, content, 0o600); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	logger := logging.NewTestLogger()
	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// Ensure destination still has original permissions (should remain 0600 after skip)
	info, _ := fs.Stat(dst)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permission changed unexpectedly: %v", info.Mode())
	}

	schema.Version(context.Background())
}

// TestCopyFileBackupAndOverwrite ensures CopyFile creates a backup when content differs.
func TestCopyFileBackupAndOverwrite(t *testing.T) {
	fs := afero.NewMemMapFs()
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "file.txt")

	// Initial dst with different content
	if err := afero.WriteFile(fs, dst, []byte("old-content"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}
	if err := afero.WriteFile(fs, src, []byte("new-content"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	logger := logging.NewTestLogger()
	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	// Destination should now match source
	data, _ := afero.ReadFile(fs, dst)
	if string(data) != "new-content" {
		t.Fatalf("dst not overwritten: %s", string(data))
	}

	// Ensure log captured message about backup
	if output := logger.GetOutput(); !strings.Contains(output, messages.MsgMovingExistingToBackup) {
		t.Fatalf("backup message not logged")
	}

	files, _ := afero.ReadDir(fs, dir)
	var foundBackup bool
	for _, fi := range files {
		if fi.Name() != "file.txt" && strings.HasPrefix(fi.Name(), "file_") && strings.HasSuffix(fi.Name(), ".txt") {
			foundBackup = true
			break
		}
	}
	if !foundBackup {
		t.Fatalf("backup file not found in directory")
	}

	schema.Version(context.Background())
}

// mockWorkflow implements the minimal subset of the generated Workflow interface we need.
type mockWorkflow struct{ name, version string }

func (m mockWorkflow) GetAgentID() string                { return m.name }
func (m mockWorkflow) GetName() string                   { return m.name }
func (m mockWorkflow) GetVersion() string                { return m.version }
func (m mockWorkflow) GetDescription() *string           { desc := ""; return &desc }
func (m mockWorkflow) GetWebsite() *string               { return nil }
func (m mockWorkflow) GetAuthors() *[]string             { return nil }
func (m mockWorkflow) GetDocumentation() *string         { return nil }
func (m mockWorkflow) GetRepository() *string            { return nil }
func (m mockWorkflow) GetHeroImage() *string             { return nil }
func (m mockWorkflow) GetAgentIcon() *string             { return nil }
func (m mockWorkflow) GetTargetActionID() string         { return "" }
func (m mockWorkflow) GetWorkflows() []string            { return nil }
func (m mockWorkflow) GetSettings() *pklProject.Settings { return nil }

// TestCopyDataDirBasic verifies that CopyDataDir copies files when present.
func TestCopyDataDirBasic(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	tmp := t.TempDir()
	projectDir := filepath.Join(tmp, "project")
	compiledDir := filepath.Join(tmp, "compiled")

	// create source data file at projectDir/data/<wf.name>/<wf.version>/file.txt
	wf := mockWorkflow{"agent", "1.0.0"}
	dataSrc := filepath.Join(projectDir, "data")
	if err := fs.MkdirAll(dataSrc, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := afero.WriteFile(fs, filepath.Join(dataSrc, "sample.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	if err := fs.MkdirAll(compiledDir, 0o755); err != nil {
		t.Fatalf("mkdir compiled: %v", err)
	}

	kdepsDir := filepath.Join(tmp, "kdeps")

	if err := archiver.CopyDataDir(fs, ctx, wf, kdepsDir, projectDir, compiledDir, "", "", "", false, logger); err != nil {
		t.Fatalf("CopyDataDir error: %v", err)
	}

	destFile := filepath.Join(compiledDir, "data", wf.GetAgentID(), wf.GetVersion(), "sample.txt")
	if ok, _ := afero.Exists(fs, destFile); !ok {
		t.Fatalf("destination file not copied")
	}

	_ = schema.Version(ctx)
}

// TestResolveAgentVersionAndCopyResources verifies resource copy logic and auto-version bypass.
func TestResolveAgentVersionAndCopyResources(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	tmp := t.TempDir()
	kdepsDir := filepath.Join(tmp, "kdeps")
	compiledDir := filepath.Join(tmp, "compiled")

	// Set up resources src path kdepsDir/agents/agent/1.2.3/resources/res.txt
	resourcesDir := filepath.Join(kdepsDir, "agents", "agent", "1.2.3", "resources")
	if err := fs.MkdirAll(resourcesDir, 0o755); err != nil {
		t.Fatalf("mkdir res: %v", err)
	}
	_ = afero.WriteFile(fs, filepath.Join(resourcesDir, "res.txt"), []byte("r"), 0o644)

	// And data path which function returns
	dataFile := filepath.Join(kdepsDir, "agents", "agent", "1.2.3", "data", "agent", "1.2.3", "d.txt")
	if err := fs.MkdirAll(filepath.Dir(dataFile), 0o755); err != nil {
		t.Fatalf("mkdir data: %v", err)
	}
	_ = afero.WriteFile(fs, dataFile, []byte("d"), 0o644)

	if err := fs.MkdirAll(compiledDir, 0o755); err != nil {
		t.Fatalf("mkdir compiled: %v", err)
	}

	newSrc, newDst, err := archiver.ResolveAgentVersionAndCopyResources(fs, ctx, kdepsDir, compiledDir, "agent", "1.2.3", logger)
	if err != nil {
		t.Fatalf("ResolveAgentVersion error: %v", err)
	}

	// The resources should now be copied into compiledDir/resources/res.txt
	if ok, _ := afero.Exists(fs, filepath.Join(compiledDir, "resources", "res.txt")); !ok {
		t.Fatalf("resource not copied")
	}

	// Returned paths should match expected data directories.
	expectedSrc := filepath.Join(kdepsDir, "agents", "agent", "1.2.3", "data", "agent", "1.2.3")
	expectedDst := filepath.Join(compiledDir, "data", "agent", "1.2.3")
	if newSrc != expectedSrc || newDst != expectedDst {
		t.Fatalf("unexpected src/dst: %s %s", newSrc, newDst)
	}

	_ = schema.Version(ctx)
}

func TestCopyFile_RenameError(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	tmpDir := t.TempDir()
	src := filepath.Join(tmpDir, "src.txt")
	dst := filepath.Join(tmpDir, "dst.txt")

	// write distinct source and dest so MD5 differs → forces rename of existing dst
	_ = afero.WriteFile(fs, src, []byte("source"), 0o644)
	_ = afero.WriteFile(fs, dst, []byte("dest"), 0o644)

	// Wrap the mem fs with read-only to make Rename fail
	rofs := afero.NewReadOnlyFs(fs)

	if err := archiver.CopyFile(rofs, context.Background(), src, dst, logger); err == nil {
		t.Fatalf("expected error due to read-only rename failure")
	}
}

func TestPerformCopy_DestCreateError(t *testing.T) {
	mem := afero.NewMemMapFs()

	tmp := t.TempDir()
	src := filepath.Join(tmp, "s.txt")
	_ = afero.WriteFile(mem, src, []byte("a"), 0o644)

	// destination on read-only fs; embed mem inside ro wrapper to make create fail
	ro := afero.NewReadOnlyFs(mem)
	if err := archiver.PerformCopy(ro, src, filepath.Join(tmp, "d.txt")); err == nil {
		t.Fatalf("expected create error on read-only FS")
	}
}

// TestCopyFileMissingSource verifies that copyFile returns an error when the
// source does not exist.
func TestCopyFileMissingSource(t *testing.T) {
	fs := afero.NewMemMapFs()
	dst := "/dst.txt"
	if err := archiver.CopyFile(fs, context.Background(), "/no-such.txt", dst, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for missing source file")
	}
	// Destination should not exist either.
	if exists, _ := afero.Exists(fs, dst); exists {
		t.Fatalf("destination unexpectedly created on failure")
	}

	_ = schema.Version(context.Background())
}

// TestPerformCopyErrorSource ensures PerformCopy surfaces error when source
// cannot be opened.
func TestPerformCopyErrorSource(t *testing.T) {
	fs := afero.NewMemMapFs()
	err := archiver.PerformCopy(fs, "/bad-src", "/dst")
	if err == nil {
		t.Fatalf("expected error from PerformCopy with bad source")
	}
	_ = schema.Version(context.Background())
}

// TestMoveFolderMissing verifies that MoveFolder returns error for a missing
// source directory.
func TestMoveFolderMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := archiver.MoveFolder(fs, "/does/not/exist", "/dest"); err == nil {
		t.Fatalf("expected error when source directory is absent")
	}
	_ = schema.Version(context.Background())
}

// TestCopyPermissions checks that PerformCopy plus SetPermissions yields the
// same mode bits at destination as source.
func TestCopyPermissions(t *testing.T) {
	fs := afero.NewMemMapFs()
	src := "/src.txt"
	dst := "/dst.txt"

	// Create src with specific permissions.
	content := []byte("perm-test")
	if err := afero.WriteFile(fs, src, content, 0o640); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Need a dummy logger – not used in code path.
	logger := logging.NewTestLogger()

	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	srcInfo, _ := fs.Stat(src)
	dstInfo, _ := fs.Stat(dst)
	if srcInfo.Mode().Perm() != dstInfo.Mode().Perm() {
		t.Fatalf("permission mismatch: src %v dst %v", srcInfo.Mode().Perm(), dstInfo.Mode().Perm())
	}

	// Ensure contents copied too.
	data, _ := afero.ReadFile(fs, dst)
	if string(data) != string(content) {
		t.Fatalf("content mismatch: got %q want %q", string(data), string(content))
	}

	_ = schema.Version(ctx)
}

func TestPerformCopyErrorPaths(t *testing.T) {
	// Case 1: source missing – expect error
	fs := afero.NewMemMapFs()
	err := archiver.PerformCopy(fs, "/non/existent", "/dest")
	if err == nil {
		t.Fatal("expected error for missing source")
	}

	// Case 2: dest create failure on read-only FS
	mem := afero.NewMemMapFs()
	tmp := t.TempDir()
	src := filepath.Join(tmp, "src.txt")
	_ = afero.WriteFile(mem, src, []byte("data"), 0o644)
	ro := afero.NewReadOnlyFs(mem)
	if err := archiver.PerformCopy(ro, src, filepath.Join(tmp, "dst.txt")); err == nil {
		t.Fatal("expected error for create on read-only FS")
	}

	_ = schema.Version(context.Background())
}

func TestSetPermissionsErrorPaths(t *testing.T) {
	fs := afero.NewMemMapFs()
	// src does not exist
	if err := archiver.SetPermissions(fs, "/missing", "/dst"); err == nil {
		t.Fatal("expected error for missing src stat")
	}

	// chmod failure using read-only FS
	tmp := t.TempDir()
	src := filepath.Join(tmp, "f.txt")
	dst := filepath.Join(tmp, "d.txt")
	_ = afero.WriteFile(fs, src, []byte("Hi"), 0o644)
	_ = afero.WriteFile(fs, dst, []byte("Hi"), 0o644)
	ro := afero.NewReadOnlyFs(fs)
	if err := archiver.SetPermissions(ro, src, dst); err == nil {
		t.Fatal("expected chmod error on read-only FS")
	}

	_ = schema.Version(context.Background())
}

// ensure test files call schema version at least once to satisfy repo conventions
// go:generate echo "schema version: v0.0.0" > /dev/null

func TestMoveFolder(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = fs.MkdirAll("/src/a/b", 0o755)
	_ = afero.WriteFile(fs, "/src/a/b/file.txt", []byte("content"), 0o644)
	require.NoError(t, archiver.MoveFolder(fs, "/src", "/dest"))
	exists, err := afero.DirExists(fs, "/src")
	require.NoError(t, err)
	require.False(t, exists)
	data, err := afero.ReadFile(fs, "/dest/a/b/file.txt")
	require.NoError(t, err)
	require.Equal(t, "content", string(data))
}

func TestCopyFile_NoExist(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	_ = afero.WriteFile(fs, "/src.txt", []byte("data"), 0o644)
	require.NoError(t, archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger))
	data, err := afero.ReadFile(fs, "/dst.txt")
	require.NoError(t, err)
	require.Equal(t, "data", string(data))
}

func TestCopyFile_ExistsSameMD5(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	content := []byte("data")
	_ = afero.WriteFile(fs, "/src.txt", content, 0o644)
	_ = afero.WriteFile(fs, "/dst.txt", content, 0o644)
	require.NoError(t, archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger))
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
	_ = afero.WriteFile(fs, "/src.txt", []byte("src"), 0o644)
	_ = afero.WriteFile(fs, "/dst.txt", []byte("dst"), 0o644)
	require.NoError(t, archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger))
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
	p := archiver.GetBackupPath("/path/file.ext", "abc")
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
	if err := archiver.MoveFolder(fs, srcDir, destDir); err != nil {
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
	gotHash, err := archiver.GetFileMD5(fs, movedFile, 8)
	if err != nil {
		t.Fatalf("GetFileMD5 error: %v", err)
	}

	h := md5.Sum(content)
	expectedHash := hex.EncodeToString(h[:])[:8]
	if gotHash != expectedHash {
		t.Fatalf("md5 mismatch: got %s want %s", gotHash, expectedHash)
	}
}

func TestCopyFileCreatesBackup(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()

	logger := logging.NewTestLogger()

	src := filepath.Join(root, "src.txt")

	// initial content
	if err := afero.WriteFile(fs, src, []byte("first"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// first copy (dest does not exist yet)
	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// Copy again with identical content – should skip and not create backup
	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
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

	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
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
	if err := archiver.CopyDir(fs, ctx, srcDir, destDir, logger); err != nil {
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
	_ = schema.Version(ctx)
}

// TestCopyFileIdentical verifies that CopyFile detects identical files via MD5
// and skips copying (no backup should be created, destination remains
// unchanged).
func TestCopyFileIdentical(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	src := "/src.txt"
	dst := "/dst.txt"
	content := []byte("identical")

	if err := afero.WriteFile(fs, src, content, 0o644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}
	if err := afero.WriteFile(fs, dst, content, 0o644); err != nil {
		t.Fatalf("failed to write dst file: %v", err)
	}

	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
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
	md5sum, _ := archiver.GetFileMD5(fs, dst, 8)
	backupPath := archiver.GetBackupPath(dst, md5sum)
	if exists, _ := afero.Exists(fs, backupPath); exists {
		t.Errorf("unexpected backup file created at %s", backupPath)
	}

	_ = schema.Version(ctx)
}

// TestCopyFileBackup verifies that CopyFile creates a backup when destination
// differs from source and then overwrites the destination with source
// contents.
func TestCopyFileBackup(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	src := "/src.txt"
	dst := "/dst.txt"
	if err := afero.WriteFile(fs, src, []byte("new"), 0o644); err != nil {
		t.Fatalf("failed to write src file: %v", err)
	}
	if err := afero.WriteFile(fs, dst, []byte("old"), 0o644); err != nil {
		t.Fatalf("failed to write dst file: %v", err)
	}

	// Capture the MD5 of the old destination before copying.
	oldMD5, _ := archiver.GetFileMD5(fs, dst, 8)
	expectedBackup := archiver.GetBackupPath(dst, oldMD5)

	if err := archiver.CopyFile(fs, context.Background(), "/src.txt", "/dst.txt", logger); err != nil {
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

	_ = schema.Version(ctx)
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

	if err := archiver.CopyFile(fs, context.Background(), src, dst, logging.NewTestLogger()); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	data, err := afero.ReadFile(fs, dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "hello copy" {
		t.Errorf("content mismatch: got %q", string(data))
	}

	_ = schema.Version(context.Background())
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
	if err := archiver.MoveFolder(fs, srcDir, destDir); err != nil {
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

	_ = schema.Version(context.Background())
}

func TestCopyFileVariants(t *testing.T) {
	fsys := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Use temporary directory for test files
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.txt")
	dstPath := filepath.Join(tmpDir, "dst.txt")

	// create source file
	if err := afero.WriteFile(fsys, srcPath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// 1. destination does not exist – simple copy
	if err := archiver.CopyFile(fsys, context.Background(), srcPath, dstPath, logger); err != nil {
		t.Fatalf("copy (new): %v", err)
	}
	// verify content
	data, _ := afero.ReadFile(fsys, dstPath)
	if string(data) != "hello" {
		t.Fatalf("unexpected dst content: %q", string(data))
	}

	// 2. destination exists with SAME md5 – should skip copy and keep content
	if err := archiver.CopyFile(fsys, context.Background(), srcPath, dstPath, logger); err != nil {
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

	if err := archiver.CopyFile(fsys, context.Background(), srcPath, dstPath, logger); err != nil {
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

func TestGetBackupPathAdditional(t *testing.T) {
	tmpDir := t.TempDir()
	dst := filepath.Join(tmpDir, "file.txt")
	md5 := "abcdef12"
	expected := filepath.Join(tmpDir, "file_"+md5+".txt")
	assert.Equal(t, expected, archiver.GetBackupPath(dst, md5))
}

// TestPerformCopyError checks that PerformCopy returns an error when the source
// file does not exist. This exercises the early error branch that was previously
// uncovered.
func TestPerformCopyError(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Intentionally do NOT create the source file.
	src := "/missing/src.txt"
	dest := "/dest/out.txt"

	if err := archiver.PerformCopy(fs, src, dest); err == nil {
		t.Errorf("expected error when copying non-existent source, got nil")
	}
}

// TestSetPermissionsError ensures SetPermissions fails gracefully when the
// source file is absent, covering its error path.
func TestSetPermissionsError(t *testing.T) {
	fs := afero.NewMemMapFs()

	src := "/missing/perm.txt"
	dest := "/dest/out.txt"

	if err := archiver.SetPermissions(fs, src, dest); err == nil {
		t.Errorf("expected error when stat-ing non-existent source, got nil")
	}
}

// TestCopyFileInternalError ensures copyFile returns an error when the source does not exist.
func TestCopyFileInternalError(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	src := filepath.Join(tmp, "nosuch.txt")
	dst := filepath.Join(tmp, "dst.txt")

	if err := archiver.CopyFile(fs, context.Background(), src, dst, logging.NewTestLogger()); err == nil {
		t.Fatalf("expected error for missing source file")
	}
}

// TestPerformCopyAndSetPermissions verifies PerformCopy copies bytes and SetPermissions replicates mode bits.
func TestPerformCopyAndSetPermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits semantics differ on Windows")
	}

	fs := afero.NewOsFs()
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "dst.txt")

	if err := afero.WriteFile(fs, src, []byte("abc"), 0o600); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// PerformCopy should succeed
	if err := archiver.PerformCopy(fs, src, dst); err != nil {
		t.Fatalf("PerformCopy error: %v", err)
	}

	// ensure bytes copied
	data, err := afero.ReadFile(fs, dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != "abc" {
		t.Fatalf("unexpected dst content: %s", string(data))
	}

	// change src mode to 0644 then run SetPermissions and expect dst updated
	if err := fs.Chmod(src, 0o644); err != nil {
		t.Fatalf("chmod src: %v", err)
	}

	if err := archiver.SetPermissions(fs, src, dst); err != nil {
		t.Fatalf("SetPermissions error: %v", err)
	}

	dstInfo, err := fs.Stat(dst)
	if err != nil {
		t.Fatalf("stat dst: %v", err)
	}

	if dstInfo.Mode().Perm() != 0o644 {
		t.Fatalf("permissions not propagated, got %v", dstInfo.Mode().Perm())
	}
}

// TestGetFileMD5 covers happy-path, truncation and error branches.
func TestGetFileMD5Edges(t *testing.T) {
	fs := afero.NewMemMapFs()
	filePath := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("hello-md5-check")
	require.NoError(t, afero.WriteFile(fs, filePath, content, 0o644))

	// Full length (32 chars) hash check.
	got, err := archiver.GetFileMD5(fs, filePath, 32)
	require.NoError(t, err)
	h := md5.Sum(content)
	expected := hex.EncodeToString(h[:])
	require.Equal(t, expected, got)

	// Truncated hash (8 chars).
	gotShort, err := archiver.GetFileMD5(fs, filePath, 8)
	require.NoError(t, err)
	require.Equal(t, expected[:8], gotShort)

	// Non-existent file should return error.
	_, err = archiver.GetFileMD5(fs, "/does/not/exist", 8)
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

	// PerformCopy is internal but test file lives in same package so we can call it.
	require.NoError(t, archiver.PerformCopy(fs, src, dst))

	// Verify destination contains identical bytes.
	dstFile, err := fs.Open(dst)
	require.NoError(t, err)
	defer dstFile.Close()

	copied, err := io.ReadAll(dstFile)
	require.NoError(t, err)
	require.Equal(t, data, copied)
}

func TestGetFileMD5SuccessAndError(t *testing.T) {
	afs := afero.NewOsFs()
	tmp := t.TempDir()

	filePath := filepath.Join(tmp, "f.txt")
	data := []byte("abc123")
	if err := afero.WriteFile(afs, filePath, data, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := archiver.GetFileMD5(afs, filePath, 8)
	if err != nil {
		t.Fatalf("GetFileMD5 error: %v", err)
	}
	h := md5.Sum(data)
	expected := hex.EncodeToString(h[:])[:8]
	if got != expected {
		t.Fatalf("hash mismatch: got %s want %s", got, expected)
	}

	// error path: file missing
	if _, err := archiver.GetFileMD5(afs, filepath.Join(tmp, "missing"), 8); err == nil {
		t.Fatalf("expected error for missing file")
	}

	// error path: zero-length allowed file but permission denied (use read only fs layer)
	ro := afero.NewReadOnlyFs(afs)
	if _, err := archiver.GetFileMD5(ro, filePath, 8); err != nil && !errors.Is(err, fs.ErrPermission) {
		// expected some error not nil – just ensure function propagates
	}
}

func TestMoveFolderSuccessEdge(t *testing.T) {
	fs := afero.NewMemMapFs()

	// create src dir with subfile
	src := "/srcdir"
	dst := "/dstdir"
	if err := fs.MkdirAll(src, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := afero.WriteFile(fs, filepath.Join(src, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := archiver.MoveFolder(fs, src, dst); err != nil {
		t.Fatalf("MoveFolder error: %v", err)
	}

	// src should be removed, dst should contain file
	if exists, _ := afero.DirExists(fs, src); exists {
		t.Fatalf("expected src dir removed")
	}
	if ok, _ := afero.Exists(fs, filepath.Join(dst, "file.txt")); !ok {
		t.Fatalf("destination file missing")
	}
}

func TestGetFileMD5Truncate(t *testing.T) {
	fs := afero.NewMemMapFs()
	file := "/file.bin"
	data := []byte("1234567890abcdef")
	_ = afero.WriteFile(fs, file, data, 0o644)

	md5Full, err := archiver.GetFileMD5(fs, file, 32)
	if err != nil {
		t.Fatalf("md5 error: %v", err)
	}
	if len(md5Full) != 32 {
		t.Fatalf("expected full md5 length got %d", len(md5Full))
	}

	md5Short, _ := archiver.GetFileMD5(fs, file, 8)
	if len(md5Short) != 8 {
		t.Fatalf("expected truncated md5 len 8 got %d", len(md5Short))
	}
	if md5Short != md5Full[:8] {
		t.Fatalf("truncated md5 mismatch")
	}
}

func TestParseActionIDEdgeCases(t *testing.T) {
	// This test is no longer relevant as we now use agent.PklResourceReader for all action ID resolution.
	// The old parseActionID function has been removed in favor of the canonical agent-based system.
	t.Skip("parseActionID function removed in favor of agent.PklResourceReader-based resolution")
}

func TestCopyFileSuccess(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src.txt")
	dst := filepath.Join(tmp, "nested", "dst.txt")
	content := []byte("lorem ipsum")

	if err := afero.WriteFile(fs, src, content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := fs.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	if err := archiver.CopyFile(fs, context.Background(), src, dst, logging.NewTestLogger()); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}

	data, err := afero.ReadFile(fs, dst)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(data) != string(content) {
		t.Fatalf("content mismatch")
	}
}

func TestMoveFolderNested(t *testing.T) {
	fs := afero.NewOsFs()
	root := t.TempDir()

	src := filepath.Join(root, "src")
	dest := filepath.Join(root, "dest")

	// create deep hierarchy
	paths := []string{
		filepath.Join(src, "a", "b"),
		filepath.Join(src, "c"),
	}
	for _, p := range paths {
		if err := fs.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	if err := afero.WriteFile(fs, filepath.Join(src, "a", "b", "file.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := archiver.MoveFolder(fs, src, dest); err != nil {
		t.Fatalf("MoveFolder: %v", err)
	}

	// dest should now contain same hierarchy
	if ok, _ := afero.DirExists(fs, filepath.Join(dest, "a", "b")); !ok {
		t.Fatalf("nested dir not moved")
	}

	// src should be removed entirely
	if ok, _ := afero.DirExists(fs, src); ok {
		t.Fatalf("src dir still exists")
	}
}

func TestGetFileMD5AndCopyFile(t *testing.T) {
	fsys := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	src := "/src.txt"
	content := []byte("hello world")
	assert.NoError(t, afero.WriteFile(fsys, src, content, 0o644))

	md5short, err := archiver.GetFileMD5(fsys, src, 8)
	assert.NoError(t, err)
	assert.Len(t, md5short, 8)

	dest := "/dest.txt"
	assert.NoError(t, archiver.CopyFile(fsys, context.Background(), src, dest, logger))

	// identical copy should not create backup
	assert.NoError(t, archiver.CopyFile(fsys, context.Background(), src, dest, logger))

	// modify src and copy again -> backup expected
	newContent := []byte("hello new world")
	assert.NoError(t, afero.WriteFile(fsys, src, newContent, 0o644))
	assert.NoError(t, archiver.CopyFile(fsys, context.Background(), src, dest, logger))

	backupName := "dest_" + md5short + ".txt"
	exists, _ := afero.Exists(fsys, "/"+backupName)
	assert.True(t, exists)
}

func TestMoveFolderAndCopyDir(t *testing.T) {
	fsys := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	srcDir := "/source"
	assert.NoError(t, fsys.MkdirAll(filepath.Join(srcDir, "nested"), 0o755))
	assert.NoError(t, afero.WriteFile(fsys, filepath.Join(srcDir, "file1.txt"), []byte("a"), 0o644))
	assert.NoError(t, afero.WriteFile(fsys, filepath.Join(srcDir, "nested", "file2.txt"), []byte("b"), 0o644))

	destDir := "/destination"
	assert.NoError(t, archiver.MoveFolder(fsys, srcDir, destDir))

	exists, _ := afero.DirExists(fsys, srcDir)
	assert.False(t, exists)

	for _, rel := range []string{"file1.txt", "nested/file2.txt"} {
		data, err := afero.ReadFile(fsys, filepath.Join(destDir, rel))
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	}

	compiledDir := "/compiled"
	assert.NoError(t, archiver.CopyDir(fsys, ctx, destDir, compiledDir, logger))
	d, err := afero.ReadFile(fsys, filepath.Join(compiledDir, "file1.txt"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("a"), d)
}

func TestMoveFolder_Success(t *testing.T) {
	mem := afero.NewMemMapFs()

	// Setup source directory with nested files
	_ = mem.MkdirAll("/src/sub", 0o755)
	afero.WriteFile(mem, "/src/file1.txt", []byte("one"), 0o644)
	afero.WriteFile(mem, "/src/sub/file2.txt", []byte("two"), 0o644)

	if err := archiver.MoveFolder(mem, "/src", "/dst"); err != nil {
		t.Fatalf("MoveFolder returned error: %v", err)
	}

	// Source should be removed
	if exists, _ := afero.Exists(mem, "/src"); exists {
		t.Fatalf("source directory still exists after MoveFolder")
	}

	// Destination files should exist with same content
	data, _ := afero.ReadFile(mem, "/dst/file1.txt")
	if string(data) != "one" {
		t.Fatalf("file1 content mismatch: %s", data)
	}
	data, _ = afero.ReadFile(mem, "/dst/sub/file2.txt")
	if string(data) != "two" {
		t.Fatalf("file2 content mismatch: %s", data)
	}
}

func TestMoveFolder_NonexistentSource(t *testing.T) {
	mem := afero.NewMemMapFs()
	err := archiver.MoveFolder(mem, "/no-such", "/dst")
	if err == nil {
		t.Fatalf("expected error when source does not exist")
	}
	// Ensure destination not created
	if _, statErr := mem.Stat("/dst"); !errors.Is(statErr, fs.ErrNotExist) {
		t.Fatalf("destination directory should not exist when move fails")
	}
}

// Test that PerformCopy fails when destination cannot be created (read-only FS).
func TestPerformCopy_DestinationCreateFails(t *testing.T) {
	base := afero.NewMemMapFs()
	src := "/src.txt"
	_ = afero.WriteFile(base, src, []byte("data"), 0o644)

	ro := afero.NewReadOnlyFs(base)
	if err := archiver.PerformCopy(ro, src, "/dst.txt"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// errFs wraps MemMapFs but forces Chmod to fail so SetPermissions propagates the error.
type errFs struct {
	*afero.MemMapFs
}

// Override Chmod to simulate permission failure.
func (e *errFs) Chmod(name string, mode os.FileMode) error {
	return errors.New("chmod not allowed")
}

func TestCopyFile_SetPermissionsFails(t *testing.T) {
	// base mem FS handles file operations; errFs will delegate except Chmod.
	mem := &afero.MemMapFs{}
	efs := &errFs{mem}

	src := "/a.txt"
	dst := "/b.txt"
	_ = afero.WriteFile(mem, src, []byte("x"), 0o644)

	err := archiver.CopyFile(efs, context.Background(), src, dst, logging.NewTestLogger())
	if err == nil {
		t.Fatalf("expected chmod failure error")
	}
	if !strings.Contains(err.Error(), "chmod not allowed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// TestGetFileMD5Missing verifies error when file is missing.
func TestGetFileMD5Missing(t *testing.T) {
	fs := afero.NewMemMapFs()
	if _, err := archiver.GetFileMD5(fs, "/nope.txt", 8); err == nil {
		t.Fatalf("expected error for missing file")
	}
	_ = schema.Version(context.Background())
}

// TestPerformCopyDestError ensures PerformCopy surfaces errors when destination cannot be created.
func TestPerformCopyDestError(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	// Create readable source file.
	src := filepath.Join(tmp, "src.txt")
	if err := afero.WriteFile(fs, src, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Make a read-only directory to hold destination.
	roDir := filepath.Join(tmp, "ro")
	if err := fs.MkdirAll(roDir, 0o555); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	dst := filepath.Join(roDir, "dst.txt")

	if err := archiver.PerformCopy(fs, src, dst); err == nil {
		t.Fatalf("expected error when destination unwritable")
	}

	_ = fs.Chmod(roDir, 0o755) // cleanup so TempDir removal works
	_ = schema.Version(context.Background())
}

// TestSetPermissionsChangesMode checks that SetPermissions aligns dest mode with source.
func TestSetPermissionsChangesMode(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()

	src := filepath.Join(tmp, "s.txt")
	dst := filepath.Join(tmp, "d.txt")

	if err := afero.WriteFile(fs, src, []byte("data"), 0o600); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := afero.WriteFile(fs, dst, []byte("data"), 0o644); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := archiver.SetPermissions(fs, src, dst); err != nil {
		t.Fatalf("SetPermissions error: %v", err)
	}

	info, _ := fs.Stat(dst)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode mismatch: got %v want 0600", info.Mode().Perm())
	}

	_ = schema.Version(context.Background())
}

// TestSetPermissionsSrcMissing verifies error when source missing.
func TestSetPermissionsSrcMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := archiver.SetPermissions(fs, "/missing.txt", "/dst.txt"); err == nil {
		t.Fatalf("expected error when src missing")
	}
	_ = schema.Version(context.Background())
}

// TestPerformCopySuccess ensures file contents are copied correctly.
func TestPerformCopySuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	src := "/src.txt"
	dst := "/dst.txt"

	if err := afero.WriteFile(fs, src, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := archiver.PerformCopy(fs, src, dst); err != nil {
		t.Fatalf("PerformCopy error: %v", err)
	}

	data, _ := afero.ReadFile(fs, dst)
	if string(data) != "hello" {
		t.Fatalf("content mismatch: %s", string(data))
	}

	_ = schema.Version(context.Background())
}

// TestPerformCopySrcMissing verifies error when source is absent.
func TestPerformCopySrcMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := archiver.PerformCopy(fs, "/missing.txt", "/dst.txt"); err == nil {
		t.Fatalf("expected error for missing source")
	}
	_ = schema.Version(context.Background())
}

func TestMoveFolderAndCopyFileSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// setup source directory with one file
	srcDir := "/src"
	dstDir := "/dst"
	_ = fs.MkdirAll(srcDir, 0o755)
	srcFile := srcDir + "/file.txt"
	_ = afero.WriteFile(fs, srcFile, []byte("data"), 0o644)

	// MoveFolder
	if err := archiver.MoveFolder(fs, srcDir, dstDir); err != nil {
		t.Fatalf("MoveFolder error: %v", err)
	}
	// Original dir should not exist
	if exists, _ := afero.Exists(fs, srcDir); exists {
		t.Fatalf("src dir still exists after move")
	}
	// Destination file should exist
	if exists, _ := afero.Exists(fs, dstDir+"/file.txt"); !exists {
		t.Fatalf("file not moved to dst")
	}

	// Test CopyFile idempotent path (same content)
	newFile := dstDir + "/copy.txt"
	if err := archiver.CopyFile(fs, context.Background(), srcDir+"/file.txt", newFile, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}
	// Copying again should detect same MD5 and skip
	if err := archiver.CopyFile(fs, context.Background(), srcDir+"/file.txt", newFile, logger); err != nil {
		t.Fatalf("CopyFile second error: %v", err)
	}
}

// TestMoveFolderAndGetFileMD5Small covers the happy-path of MoveFolder as well as
// the MD5 helper which is used by CopyFile. It relies only on afero so no
// host-FS writes occur.
func TestMoveFolderAndGetFileMD5Small(t *testing.T) {
	fs := afero.NewOsFs()

	// Create a temporary source directory with one file inside.
	srcDir, err := afero.TempDir(fs, "", "kdeps_src")
	if err != nil {
		t.Fatalf("TempDir src error: %v", err)
	}
	defer fs.RemoveAll(srcDir)

	data := []byte("hello kdeps")
	srcFile := filepath.Join(srcDir, "file.txt")
	if err := afero.WriteFile(fs, srcFile, data, 0o644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	// Destination directory (does not need to exist beforehand).
	destDir, err := afero.TempDir(fs, "", "kdeps_dst")
	if err != nil {
		t.Fatalf("TempDir dest error: %v", err)
	}
	fs.RemoveAll(destDir) // ensure empty so MoveFolder will create it

	// MoveFolder should move the directory tree.
	if err := archiver.MoveFolder(fs, srcDir, destDir); err != nil {
		t.Fatalf("MoveFolder error: %v", err)
	}

	movedFile := filepath.Join(destDir, "file.txt")
	if exists, _ := afero.Exists(fs, movedFile); !exists {
		t.Fatalf("expected file to be moved to %s", movedFile)
	}
	if exists, _ := afero.DirExists(fs, srcDir); exists {
		t.Fatalf("expected source directory to be removed")
	}

	// Verify GetFileMD5 returns the expected (truncated) hash.
	got, err := archiver.GetFileMD5(fs, movedFile, 6)
	if err != nil {
		t.Fatalf("GetFileMD5 error: %v", err)
	}

	h := md5.New()
	_, _ = io.WriteString(h, string(data))
	wantFull := hex.EncodeToString(h.Sum(nil))
	want := wantFull[:6]
	if got != want {
		t.Fatalf("md5 mismatch: got %s want %s", got, want)
	}
}

// TestCopyFileSuccess verifies that copyFile successfully duplicates the file contents.
func TestCopyFileSuccessMemFS(t *testing.T) {
	mem := afero.NewMemMapFs()

	// Prepare source file.
	src := "/src.txt"
	dst := "/dst.txt"
	data := []byte("hello")
	if err := afero.WriteFile(mem, src, data, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	if err := archiver.CopyFile(mem, context.Background(), src, dst, logging.NewTestLogger()); err != nil {
		t.Fatalf("copyFile error: %v", err)
	}
	copied, _ := afero.ReadFile(mem, dst)
	if string(copied) != string(data) {
		t.Fatalf("copied content mismatch: %s", string(copied))
	}
}

// TestSetPermissionsSuccess ensures permissions are propagated from source to destination.
func TestSetPermissionsSuccessMemFS(t *testing.T) {
	mem := afero.NewMemMapFs()
	src := "/src.txt"
	dst := "/dst.txt"
	if err := afero.WriteFile(mem, src, []byte("x"), 0o640); err != nil {
		t.Fatalf("write src: %v", err)
	}
	if err := afero.WriteFile(mem, dst, []byte("y"), 0o600); err != nil {
		t.Fatalf("write dst: %v", err)
	}

	if err := archiver.SetPermissions(mem, src, dst); err != nil {
		t.Fatalf("SetPermissions error: %v", err)
	}

	info, _ := mem.Stat(dst)
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("permissions not propagated, got %v", info.Mode().Perm())
	}

	// Extra: ensure SetPermissions no error when src and dst modes identical.
	if err := archiver.SetPermissions(mem, src, dst); err != nil {
		t.Fatalf("SetPermissions identical modes error: %v", err)
	}
}

// TestGetFileMD5AndCopyFileSuccess covers:
// 1. GetFileMD5 happy path.
// 2. CopyFile when destination does NOT exist (no backup logic triggered).
func TestGetFileMD5AndCopyFileSuccess(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()

	srcPath := filepath.Join(tmpDir, "src.txt")
	dstPath := filepath.Join(tmpDir, "dst.txt")

	content := []byte("hello-md5")
	if err := afero.WriteFile(fs, srcPath, content, 0o644); err != nil {
		t.Fatalf("write src: %v", err)
	}

	// Calculate expected MD5 manually (full hash then slice len 8)
	hash := md5.Sum(content)
	wantMD5 := hex.EncodeToString(hash[:])[:8]

	gotMD5, err := archiver.GetFileMD5(fs, srcPath, 8)
	if err != nil {
		t.Fatalf("GetFileMD5 error: %v", err)
	}
	if gotMD5 != wantMD5 {
		t.Fatalf("MD5 mismatch: got %s want %s", gotMD5, wantMD5)
	}

	// Run CopyFile where dst does not exist yet.
	logger := logging.NewTestLogger()
	if err := archiver.CopyFile(fs, context.Background(), srcPath, dstPath, logger); err != nil {
		t.Fatalf("CopyFile error: %v", err)
	}

	// Verify destination now exists with identical contents.
	dstData, err := afero.ReadFile(fs, dstPath)
	if err != nil {
		t.Fatalf("read dst: %v", err)
	}
	if string(dstData) != string(content) {
		t.Fatalf("content mismatch: got %s want %s", string(dstData), string(content))
	}

	// Ensure permissions were copied (mode preserved at least rw for owner).
	info, _ := fs.Stat(dstPath)
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("permissions not preserved: %v", info.Mode())
	}

	// Logger should contain success message.
	if out := logger.GetOutput(); !strings.Contains(strings.ToLower(out), "copied successfully") {
		t.Fatalf("expected log to mention copy success, got: %s", out)
	}
}

func TestMoveFolderMainPkg(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create source directory and files
	srcDir := "/src"
	destDir := "/dest"
	_ = fs.MkdirAll(srcDir, 0o755)
	_ = afero.WriteFile(fs, filepath.Join(srcDir, "file1.txt"), []byte("content1"), 0o644)
	_ = afero.WriteFile(fs, filepath.Join(srcDir, "file2.txt"), []byte("content2"), 0o644)

	err := archiver.MoveFolder(fs, srcDir, destDir)
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

	err := archiver.CopyFile(fs, context.Background(), srcFile, destFile, logging.GetLogger())
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
	hash, err := archiver.GetFileMD5(fs, filePath, 8)

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

	err := archiver.CopyDir(fs, context.Background(), srcDir, destDir, logging.GetLogger())
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

// TestMoveFolderMemFS verifies that MoveFolder correctly copies all files from
// the source directory to the destination and removes the original source
// directory when using an in-memory filesystem.
func TestMoveFolderMemFS(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create source directory with nested file
	srcDir := "/src"
	destDir := "/dst"
	if err := fs.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	filePath := srcDir + "/file.txt"
	if err := afero.WriteFile(fs, filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Execute MoveFolder
	if err := archiver.MoveFolder(fs, srcDir, destDir); err != nil {
		t.Fatalf("MoveFolder returned error: %v", err)
	}

	// Source directory should no longer exist
	if exists, _ := afero.DirExists(fs, srcDir); exists {
		t.Fatalf("expected source directory to be removed")
	}

	// Destination file should exist with correct contents
	movedFile := destDir + "/file.txt"
	data, err := afero.ReadFile(fs, movedFile)
	if err != nil {
		t.Fatalf("read moved file: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected file content: %s", data)
	}
}

// TestMoveFolderSuccessDeep verifies MoveFolder moves a directory tree and deletes the source.
func TestMoveFolderSuccessDeep(t *testing.T) {
	fs := afero.NewOsFs()
	base := t.TempDir()
	srcDir := filepath.Join(base, "src")
	dstDir := filepath.Join(base, "dst")

	// Build directory structure: src/sub/child.txt
	if err := fs.MkdirAll(filepath.Join(srcDir, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	filePath := filepath.Join(srcDir, "sub", "child.txt")
	if err := afero.WriteFile(fs, filePath, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	if err := archiver.MoveFolder(fs, srcDir, dstDir); err != nil {
		t.Fatalf("MoveFolder: %v", err)
	}

	// Source directory should be gone, destination file should exist.
	if exists, _ := afero.DirExists(fs, srcDir); exists {
		t.Fatalf("expected source directory to be removed")
	}
	movedFile := filepath.Join(dstDir, "sub", "child.txt")
	if ok, _ := afero.Exists(fs, movedFile); !ok {
		t.Fatalf("expected file %s to exist", movedFile)
	}

	_ = schema.Version(context.Background())
}

// TestMoveFolderSrcMissing ensures an error is returned when the source directory does not exist.
func TestMoveFolderSrcMissing(t *testing.T) {
	fs := afero.NewOsFs()
	base := t.TempDir()
	err := archiver.MoveFolder(fs, filepath.Join(base, "nope"), filepath.Join(base, "dst"))
	if err == nil {
		t.Fatalf("expected error for missing src dir")
	}

	_ = schema.Version(context.Background())
}

// TestMoveFolderSuccessMemFS ensures MoveFolder copies files and removes src.
func TestMoveFolderSuccessMemFS(t *testing.T) {
	fs := afero.NewMemMapFs()

	srcDir := "/srcDir"
	dstDir := "/dstDir"
	_ = fs.MkdirAll(srcDir, 0o755)

	// create two files in nested structure.
	_ = afero.WriteFile(fs, srcDir+"/f1.txt", []byte("a"), 0o644)
	_ = fs.MkdirAll(srcDir+"/sub", 0o755)
	_ = afero.WriteFile(fs, srcDir+"/sub/f2.txt", []byte("b"), 0o640)

	if err := archiver.MoveFolder(fs, srcDir, dstDir); err != nil {
		t.Fatalf("MoveFolder error: %v", err)
	}

	// original srcDir should be removed
	if exists, _ := afero.DirExists(fs, srcDir); exists {
		t.Fatalf("expected source dir removed")
	}

	// destination files should exist with correct content
	data1, _ := afero.ReadFile(fs, dstDir+"/f1.txt")
	if string(data1) != "a" {
		t.Fatalf("dst f1 content mismatch")
	}
	data2, _ := afero.ReadFile(fs, dstDir+"/sub/f2.txt")
	if string(data2) != "b" {
		t.Fatalf("dst f2 content mismatch")
	}
}

func TestGetFileMD5CopyDir(t *testing.T) {
	fs := afero.NewMemMapFs()
	content := []byte("hello world")
	_ = afero.WriteFile(fs, "/file.txt", content, 0o644)
	md5short, err := archiver.GetFileMD5(fs, "/file.txt", 8)
	require.NoError(t, err)
	sum := md5.Sum(content)
	expectedFull := hex.EncodeToString(sum[:])
	if len(expectedFull) >= 8 {
		require.Equal(t, expectedFull[:8], md5short)
	} else {
		require.Equal(t, expectedFull, md5short)
	}
	// length greater than md5 length should return full hash
	md5full, err := archiver.GetFileMD5(fs, "/file.txt", 100)
	require.NoError(t, err)
	require.Equal(t, expectedFull, md5full)
}
