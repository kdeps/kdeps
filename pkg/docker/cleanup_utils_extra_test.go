package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

// fakeClient implements DockerPruneClient for testing.
type fakeClient struct {
	containers []types.Container
	listErr    error
	removeErr  error
	pruneErr   error
}

func (f *fakeClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return f.containers, f.listErr
}

func (f *fakeClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	return nil
}

func (f *fakeClient) ImagesPrune(ctx context.Context, pruneFilters filters.Args) (image.PruneReport, error) {
	if f.pruneErr != nil {
		return image.PruneReport{}, f.pruneErr
	}
	return image.PruneReport{}, nil
}

func TestCleanupDockerBuildImages_NoContainers(t *testing.T) {
	client := &fakeClient{}
	err := CleanupDockerBuildImages(nil, context.Background(), "", client)
	require.NoError(t, err)
}

func TestCleanupDockerBuildImages_RemoveAndPruneSuccess(t *testing.T) {
	client := &fakeClient{
		containers: []types.Container{{ID: "abc123", Names: []string{"/testname"}}},
	}
	// Should handle remove and prune without error
	err := CleanupDockerBuildImages(nil, context.Background(), "testname", client)
	require.NoError(t, err)
}

func TestCleanupDockerBuildImages_PruneError(t *testing.T) {
	client := &fakeClient{pruneErr: errors.New("prune failed")}
	err := CleanupDockerBuildImages(nil, context.Background(), "", client)
	require.Error(t, err)
	require.Contains(t, err.Error(), "prune failed")
}

// TestCleanupFlagFilesExtra verifies that cleanupFlagFiles removes specified files.
func TestCleanupFlagFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create two files and leave one missing to exercise both paths
	files := []string{"/tmp/f1", "/tmp/f2", "/tmp/missing"}
	require.NoError(t, afero.WriteFile(fs, files[0], []byte("x"), 0644))
	require.NoError(t, afero.WriteFile(fs, files[1], []byte("y"), 0644))

	cleanupFlagFiles(fs, files, logger)

	for _, f := range files {
		exists, _ := afero.Exists(fs, f)
		require.False(t, exists, "file %s should be removed (or not exist)", f)
	}
}

func TestCleanupFlagFiles_RemovesExisting(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	f1 := "/file1.flag"
	f2 := "/file2.flag"

	// create files
	_ = afero.WriteFile(fs, f1, []byte("x"), 0o644)
	_ = afero.WriteFile(fs, f2, []byte("y"), 0o644)

	cleanupFlagFiles(fs, []string{f1, f2}, logger)

	for _, p := range []string{f1, f2} {
		if exists, _ := afero.Exists(fs, p); exists {
			t.Fatalf("expected %s to be removed", p)
		}
	}
}

func TestCleanupFlagFiles_NonExistent(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Call with files that don't exist; should not panic or error.
	cleanupFlagFiles(fs, []string{"/missing1", "/missing2"}, logger)
}
