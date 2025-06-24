package resolver

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
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
	script := `#!/bin/sh
	output_path=
	prev=
	for arg in "$@"; do
	if [ "$prev" = "--output-path" ]; then
		output_path="$arg"
		break
	fi
	prev="$arg"
	done
	json='{"hello":"world"}'
	# emit JSON to stdout
	echo "$json"
	# if --output-path was supplied, also write JSON to that file
	if [ -n "$output_path" ]; then
	echo "$json" > "$output_path"
	fi
	`
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
