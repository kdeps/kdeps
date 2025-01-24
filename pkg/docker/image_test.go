package docker

import (
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckDevBuildMode(t *testing.T) {
	t.Parallel()

	fs := afero.NewMemMapFs()
	logger := logging.GetLogger()
	kdepsDir := "/test/kdeps"
	cacheDir := filepath.Join(kdepsDir, "cache")
	binaryFile := filepath.Join(cacheDir, "kdeps")

	// Test case: Binary file exists and is valid
	require.NoError(t, fs.MkdirAll(cacheDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, binaryFile, []byte("binary content"), 0o755))

	devBuildMode, err := checkDevBuildMode(fs, kdepsDir, logger)
	require.NoError(t, err)
	assert.True(t, devBuildMode, "Expected devBuildMode to be true when binary file exists")

	// Test case: Binary file does not exist
	require.NoError(t, fs.Remove(binaryFile))

	devBuildMode, err = checkDevBuildMode(fs, kdepsDir, logger)
	require.NoError(t, err)
	assert.False(t, devBuildMode, "Expected devBuildMode to be false when binary file does not exist")

	// Test case: Path exists but is not a file
	require.NoError(t, fs.Mkdir(binaryFile, 0o755))

	devBuildMode, err = checkDevBuildMode(fs, kdepsDir, logger)
	require.NoError(t, err)
	assert.False(t, devBuildMode, "Expected devBuildMode to be false when path is not a regular file")
}
