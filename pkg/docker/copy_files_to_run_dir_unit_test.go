package docker_test

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestCopyFilesToRunDirUnit(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	downloadDir := "/downloads"
	runDir := "/run"

	// setup downloadDir with files
	assert.NoError(t, fs.MkdirAll(downloadDir, 0o755))
	files := []string{"a.txt", "b.txt"}
	for _, f := range files {
		assert.NoError(t, afero.WriteFile(fs, downloadDir+"/"+f, []byte(f), 0o644))
	}

	assert.NoError(t, docker.CopyFilesToRunDir(fs, ctx, downloadDir, runDir, logger))

	// verify files copied into runDir/cache
	for _, f := range files {
		data, err := afero.ReadFile(fs, runDir+"/cache/"+f)
		assert.NoError(t, err)
		assert.Equal(t, []byte(f), data)
	}
}
