package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// fakeDockerClient implements DockerPruneClient for unit-tests.
type fakeDockerClient struct {
	containers []types.Container
	pruned     bool
}

func (f *fakeDockerClient) ContainerList(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
	return f.containers, nil
}

func (f *fakeDockerClient) ContainerRemove(ctx context.Context, id string, opts container.RemoveOptions) error {
	// simulate removal by filtering slice
	var out []types.Container
	for _, c := range f.containers {
		if c.ID != id {
			out = append(out, c)
		}
	}
	f.containers = out
	return nil
}

func (f *fakeDockerClient) ImagesPrune(ctx context.Context, _ filters.Args) (image.PruneReport, error) {
	f.pruned = true
	return image.PruneReport{}, nil
}

func TestCleanupDockerBuildImagesUnit(t *testing.T) {
	cli := &fakeDockerClient{}
	err := CleanupDockerBuildImages(afero.NewOsFs(), context.Background(), "dummy", cli)
	assert.NoError(t, err)
	assert.True(t, cli.pruned)
}

func TestCleanupFlagFilesUnit(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()

	tmpDir := t.TempDir()
	f1 := tmpDir + "/flag1"
	f2 := tmpDir + "/flag2"
	assert.NoError(t, afero.WriteFile(fs, f1, []byte("x"), 0o644))
	assert.NoError(t, afero.WriteFile(fs, f2, []byte("x"), 0o644))

	cleanupFlagFiles(fs, []string{f1, f2}, logger)

	// files should be gone
	exists1, _ := afero.Exists(fs, f1)
	exists2, _ := afero.Exists(fs, f2)
	assert.False(t, exists1 || exists2)
}
