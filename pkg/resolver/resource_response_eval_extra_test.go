package resolver

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// createStubPkl creates a dummy executable named `pkl` that prints JSON and exits 0.
func createStubPkl(t *testing.T) (stubDir string, cleanup func()) {
	t.Helper()
	dir := t.TempDir()
	exeName := "pkl"
	if runtime.GOOS == "windows" {
		exeName = "pkl.bat"
	}
	stubPath := filepath.Join(dir, exeName)
	script := "#!/bin/sh\necho '{\"hello\":\"world\"}'\n"
	if runtime.GOOS == "windows" {
		script = "@echo {\"hello\":\"world\"}\r\n"
	}
	if err := os.WriteFile(stubPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write stub: %v", err)
	}
	// ensure executable bit on unix
	if runtime.GOOS != "windows" {
		_ = os.Chmod(stubPath, 0o755)
	}
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)
	return dir, func() { os.Setenv("PATH", oldPath) }
}

func newEvalResolver(t *testing.T) *DependencyResolver {
	fs := afero.NewOsFs()
	tmp := t.TempDir()
	return &DependencyResolver{
		Fs:                 fs,
		ResponsePklFile:    filepath.Join(tmp, "resp.pkl"),
		ResponseTargetFile: filepath.Join(tmp, "resp.json"),
		Logger:             logging.NewTestLogger(),
		Context:            context.Background(),
	}
}

func TestExecutePklEvalCommand(t *testing.T) {
	_, restore := createStubPkl(t)
	defer restore()

	dr := newEvalResolver(t)
	// create dummy pkl file so existence check passes
	if err := afero.WriteFile(dr.Fs, dr.ResponsePklFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write pkl: %v", err)
	}
	res, err := dr.executePklEvalCommand()
	if err != nil {
		t.Fatalf("executePklEvalCommand error: %v", err)
	}
	if res.Stdout == "" {
		t.Errorf("expected stdout from stub pkl, got empty")
	}
}

func TestEvalPklFormattedResponseFile(t *testing.T) {
	_, restore := createStubPkl(t)
	defer restore()

	dr := newEvalResolver(t)
	// create dummy pkl file
	if err := afero.WriteFile(dr.Fs, dr.ResponsePklFile, []byte("{}"), 0o644); err != nil {
		t.Fatalf("write pkl: %v", err)
	}

	out, err := dr.EvalPklFormattedResponseFile()
	if err != nil {
		t.Fatalf("EvalPklFormattedResponseFile error: %v", err)
	}
	if out == "" {
		t.Errorf("expected non-empty JSON output")
	}
	// target file should exist and contain same output
	data, err := afero.ReadFile(dr.Fs, dr.ResponseTargetFile)
	if err != nil {
		t.Fatalf("reading target file: %v", err)
	}
	if string(data) == "" {
		t.Errorf("target file empty")
	}
}
