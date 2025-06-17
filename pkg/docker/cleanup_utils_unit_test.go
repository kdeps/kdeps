package docker

import (
	"context"
	"path/filepath"
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

func TestCleanupFlagFilesMemFS(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create two temp files to be cleaned.
	dir := t.TempDir()
	file1 := filepath.Join(dir, "flag1")
	file2 := filepath.Join(dir, "flag2")
	if err := afero.WriteFile(fs, file1, []byte("ok"), 0o644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := afero.WriteFile(fs, file2, []byte("ok"), 0o644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	// Call cleanupFlagFiles and ensure files are removed without error.
	cleanupFlagFiles(fs, []string{file1, file2}, logger)

	for _, f := range []string{file1, file2} {
		exists, _ := afero.Exists(fs, f)
		if exists {
			t.Fatalf("expected %s to be removed", f)
		}
	}

	// Calling cleanupFlagFiles again should hit the os.IsNotExist branch and not fail.
	cleanupFlagFiles(fs, []string{file1, file2}, logger)
}
