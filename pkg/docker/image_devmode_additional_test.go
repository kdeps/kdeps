package docker

import (
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCheckDevBuildModeVariant(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir := t.TempDir()
	logger := logging.NewTestLogger()

	cacheDir := filepath.Join(tmpDir, "cache")
	_ = fs.MkdirAll(cacheDir, 0o755)
	kdepsBinary := filepath.Join(cacheDir, "kdeps")

	// when file absent
	dev, err := checkDevBuildMode(fs, tmpDir, logger)
	assert.NoError(t, err)
	assert.False(t, dev)

	// create file
	assert.NoError(t, afero.WriteFile(fs, kdepsBinary, []byte("binary"), 0o755))
	dev, err = checkDevBuildMode(fs, tmpDir, logger)
	assert.NoError(t, err)
	assert.True(t, dev)
}
