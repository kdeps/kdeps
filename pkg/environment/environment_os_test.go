package environment

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// TestNewEnvironmentWithOsFs verifies that the environment loader correctly
// detects a real .kdeps.pkl that lives on the host *disk* (not in-memory) when
// ROOT_DIR, HOME and PWD all point to the same temporary directory.
func TestNewEnvironmentWithOsFs(t *testing.T) {
	tmp := t.TempDir()

	// Create a real .kdeps.pkl in the temp directory.
	fs := afero.NewOsFs()
	configPath := filepath.Join(tmp, SystemConfigFileName)
	require.NoError(t, afero.WriteFile(fs, configPath, []byte(""), 0o644))

	// Point the relevant environment variables to the temporary directory so
	// NewEnvironment will search there.
	t.Setenv("ROOT_DIR", tmp)
	t.Setenv("HOME", tmp)
	t.Setenv("PWD", tmp)

	env, err := NewEnvironment(fs, nil)
	require.NoError(t, err)
	require.Equal(t, configPath, env.KdepsConfig, "expected to locate .kdeps.pkl in temp dir")
}
