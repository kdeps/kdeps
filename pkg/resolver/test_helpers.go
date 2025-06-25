package resolver

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/kdeps/kdeps/pkg/version"
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
	script := `#!/bin/bash
echo "Pkl ` + version.PklVersion + ` (test environment)"
echo "  Available subcommands:"
echo "    eval    Evaluate a Pkl module"
echo "    test    Run tests for a Pkl module"
echo "    repl    Start a Pkl REPL"
`
	if runtime.GOOS == "windows" {
		script = `@echo off
if "%1"=="--version" (
    echo Pkl ` + version.PklVersion + ` (test environment)
    exit /b 0
)
if "%1"=="eval" (
    echo {"hello":"world"}
    exit /b 0
)
echo {"hello":"world"}
`
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
