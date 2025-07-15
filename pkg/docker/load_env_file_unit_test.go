package docker_test

import (
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestLoadEnvFileUnit(t *testing.T) {
	fs := afero.NewOsFs()
	tmp := t.TempDir()
	envPath := filepath.Join(tmp, ".env")
	content := []byte("FOO=bar\nBAZ=qux")
	assert.NoError(t, afero.WriteFile(fs, envPath, content, 0o644))

	vals, err := docker.LoadEnvFile(fs, envPath)
	assert.NoError(t, err)
	assert.ElementsMatch(t, []string{"FOO=bar", "BAZ=qux"}, vals)

	// missing file
	none, err := docker.LoadEnvFile(fs, filepath.Join(tmp, "missing.env"))
	assert.NoError(t, err)
	assert.Nil(t, none)

	// malformed path produces error by permissions (dir)
	_, err = docker.LoadEnvFile(fs, tmp)
	assert.Error(t, err)
}
