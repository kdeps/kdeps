//go:build !js

package file

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TestApplyHunk_DefaultCase covers the default branch in applyHunk (lines
// that don't start with space, +, or -). Such lines are skipped by advancing hunkPos.
func TestApplyHunk_DefaultCase(t *testing.T) {
	origLines := []string{"line1", "line2"}
	// A hunk line that doesn't start with ' ', '+', or '-' hits the default branch.
	hunkLines := []string{"no-prefix-line", "+added"}
	result, remaining, err := applyHunk(origLines, hunkLines, 0, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "no-prefix-line" is skipped by default; "+added" is appended.
	if len(result) == 0 {
		t.Fatal("expected non-empty result")
	}
	_ = remaining
}

// TestApplyHunk_RemovalOutOfRange covers the removal-out-of-range error path.
func TestApplyHunk_RemovalOutOfRange(t *testing.T) {
	origLines := []string{}
	hunkLines := []string{"-removeThis"}
	_, _, err := applyHunk(origLines, hunkLines, 0, 0)
	if err == nil {
		t.Fatal("expected removal out of range error")
	}
}

// TestApplyHunk_ContextMismatch covers the context-mismatch error path.
func TestApplyHunk_ContextMismatch(t *testing.T) {
	origLines := []string{"differentcontent"}
	hunkLines := []string{" expectedcontent"}
	_, _, err := applyHunk(origLines, hunkLines, 0, 0)
	if err == nil {
		t.Fatal("expected context mismatch error")
	}
}

// TestCopyFile_Stat covers the os.Stat path after successful copy (line 610-614).
func TestCopyFile_Stat_Success(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	if err := os.WriteFile(src, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hello" {
		t.Fatalf("expected 'hello', got %q", string(data))
	}
}

// TestAppend_AppendNewline_AlreadyHas covers the branch where content already
// ends with newline (AppendNewline=true but no suffix added).
func TestAppend_AppendNewline_AlreadyHas(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	e := NewExecutor()
	res, err := e.Execute(nil, &domain.FileResourceConfig{
		Operation:     domain.FileOpAppend,
		Path:          path,
		Content:       "already has newline\n",
		AppendNewline: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	data, _ := os.ReadFile(path)
	// Content should end with exactly one newline, not two.
	if string(data) != "already has newline\n" {
		t.Fatalf("expected single newline, got %q", string(data))
	}
}

// TestApplyHunk_ContextPosOutOfRange covers the context-out-of-range error path
// in applyHunk when origPos >= len(remaining) on a context (space-prefixed) line.
func TestApplyHunk_ContextPosOutOfRange(t *testing.T) {
	origLines := []string{}
	hunkLines := []string{" context"}
	_, _, err := applyHunk(origLines, hunkLines, 5, 0)
	if err == nil {
		t.Fatal("expected context out of range error")
	}
}

// TestCopyFile_MkdirAllError covers the os.MkdirAll error path in copyFile
// when the destination parent cannot be created because a file blocks the path.
func TestCopyFile_MkdirAllError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Create a regular file blocking the parent dir - MkdirAll returns ENOTDIR.
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(blocker, "sub", "dst.txt")
	if err := copyFile(src, dst); err == nil {
		t.Fatal("expected MkdirAll error")
	}
}

// TestCopyFile_CreateFileError covers the os.Create error path in copyFile
// when the destination directory is read-only.
func TestCopyFile_CreateFileError(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "subdir")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Remove write permission so os.Create fails.
	if err := os.Chmod(subdir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(subdir, 0o755)
	})
	dst := filepath.Join(subdir, "dst.txt")
	if err := copyFile(src, dst); err == nil {
		t.Fatal("expected os.Create error")
	}
}

// TestCopyFile_CopyError covers the io.Copy error path in copyFile when the
// source is a directory - os.Open succeeds but io.Copy fails with EISDIR.
func TestCopyFile_CopyError(t *testing.T) {
	dir := t.TempDir()
	srcdir := filepath.Join(dir, "srcdir")
	if err := os.MkdirAll(srcdir, 0o755); err != nil {
		t.Fatal(err)
	}
	dst := filepath.Join(dir, "dst.txt")
	if err := copyFile(srcdir, dst); err == nil {
		t.Fatal("expected io.Copy error from directory source")
	}
}

// TestCopyDir_CopyFileError covers the copyFile error path inside copyDir when
// a file cannot be created because the destination directory is read-only.
func TestCopyDir_CopyFileError(t *testing.T) {
	dir := t.TempDir()
	srcdir := filepath.Join(dir, "srcdir")
	if err := os.MkdirAll(srcdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcdir, "file.txt"), []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	dstdir := filepath.Join(dir, "dstdir")
	if err := os.MkdirAll(dstdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dstdir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(dstdir, 0o755)
	})
	if err := copyDir(srcdir, dstdir); err == nil {
		t.Fatal("expected copy error from copyDir")
	}
}

// TestAppend_CoverageWriteStringError covers the WriteString error path in the
// append operation by limiting the process file size via RLIMIT_FSIZE.
func TestAppend_CoverageWriteStringError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("initial"), 0o644); err != nil {
		t.Fatal(err)
	}

	var oldRlimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_FSIZE, &oldRlimit); err != nil {
		t.Skip("RLIMIT_FSIZE not supported:", err)
	}

	zeroLimit := &syscall.Rlimit{Cur: 0, Max: oldRlimit.Max}
	if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, zeroLimit); err != nil {
		t.Skip("cannot set RLIMIT_FSIZE:", err)
	}
	defer func() {
		_ = syscall.Setrlimit(syscall.RLIMIT_FSIZE, &oldRlimit)
	}()

	e := NewExecutor()
	_, err := e.Execute(nil, &domain.FileResourceConfig{
		Operation: domain.FileOpAppend,
		Path:      path,
		Content:   " more\n",
	})
	if err == nil {
		t.Fatal("expected write error from append")
	}
}
