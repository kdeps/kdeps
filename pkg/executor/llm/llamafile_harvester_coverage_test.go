//go:build !js

package llm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHarvesterScript_ExecutableError(t *testing.T) {
	t.Setenv("KDEPS_LLAMAFILE_HARVESTER", "/nonexistent/harvest.py")
	assert.False(t, RunHarvesterScript())
}

func TestRunHarvesterScript_WithEnvScript(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "harvest.py")
	require.NoError(t, os.WriteFile(script, []byte("#!/usr/bin/env python3\nprint('ok')\n"), 0o755))
	t.Setenv("KDEPS_LLAMAFILE_HARVESTER", script)
	assert.True(t, RunHarvesterScript())
}
