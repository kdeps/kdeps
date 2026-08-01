//go:build !js

package llm

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireFunctionalPython3 skips the test when "python3" isn't a real
// interpreter on PATH. On Windows without Python installed, "python3" and
// "python" resolve to the Microsoft Store app-execution-alias stub, which
// exits non-zero instead of running anything.
func requireFunctionalPython3(t *testing.T) {
	t.Helper()
	if err := exec.Command("python3", "--version").Run(); err != nil {
		t.Skip("python3 not functionally available in this environment")
	}
}

func TestRunHarvesterScript_ExecutableError(t *testing.T) {
	t.Setenv("KDEPS_LLAMAFILE_HARVESTER", "/nonexistent/harvest.py")
	assert.False(t, RunHarvesterScript())
}

func TestRunHarvesterScript_WithEnvScript(t *testing.T) {
	requireFunctionalPython3(t)
	dir := t.TempDir()
	script := filepath.Join(dir, "harvest.py")
	require.NoError(t, os.WriteFile(script, []byte("#!/usr/bin/env python3\nprint('ok')\n"), 0o755))
	t.Setenv("KDEPS_LLAMAFILE_HARVESTER", script)
	assert.True(t, RunHarvesterScript())
}
