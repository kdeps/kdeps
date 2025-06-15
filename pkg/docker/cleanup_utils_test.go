package docker

import (
	"context"
	"io"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDockerClient is a mock implementation of the DockerPruneClient interface
// Only the required methods are implemented
type MockDockerClient struct {
	mock.Mock
}

var _ DockerPruneClient = (*MockDockerClient)(nil)

func (m *MockDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	args := m.Called(ctx, options)
	return args.Get(0).([]types.Container), args.Error(1)
}

func (m *MockDockerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	args := m.Called(ctx, containerID, options)
	return args.Error(0)
}

func (m *MockDockerClient) ImagesPrune(ctx context.Context, pruneFilters filters.Args) (image.PruneReport, error) {
	args := m.Called(ctx, pruneFilters)
	return args.Get(0).(image.PruneReport), args.Error(1)
}

// Implement other required interface methods with empty implementations
func (m *MockDockerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerStop(ctx context.Context, containerID string, options *container.StopOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerWait(ctx context.Context, containerID string, condition container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	return nil, nil
}

func (m *MockDockerClient) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	return nil, nil
}

func (m *MockDockerClient) ContainerInspect(ctx context.Context, containerID string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}

func (m *MockDockerClient) ContainerInspectWithRaw(ctx context.Context, containerID string, getSize bool) (types.ContainerJSON, []byte, error) {
	return types.ContainerJSON{}, nil, nil
}

func (m *MockDockerClient) ContainerStats(ctx context.Context, containerID string, stream bool) (container.Stats, error) {
	return container.Stats{}, nil
}

func (m *MockDockerClient) ContainerStatsOneShot(ctx context.Context, containerID string) (container.Stats, error) {
	return container.Stats{}, nil
}

func (m *MockDockerClient) ContainerTop(ctx context.Context, containerID string, arguments []string) (container.ContainerTopOKBody, error) {
	return container.ContainerTopOKBody{}, nil
}

func (m *MockDockerClient) ContainerUpdate(ctx context.Context, containerID string, updateConfig container.UpdateConfig) (container.ContainerUpdateOKBody, error) {
	return container.ContainerUpdateOKBody{}, nil
}

func (m *MockDockerClient) ContainerPause(ctx context.Context, containerID string) error {
	return nil
}

func (m *MockDockerClient) ContainerUnpause(ctx context.Context, containerID string) error {
	return nil
}

func (m *MockDockerClient) ContainerRestart(ctx context.Context, containerID string, options *container.StopOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerKill(ctx context.Context, containerID, signal string) error {
	return nil
}

func (m *MockDockerClient) ContainerRename(ctx context.Context, containerID, newContainerName string) error {
	return nil
}

func (m *MockDockerClient) ContainerResize(ctx context.Context, containerID string, options container.ResizeOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerExecCreate(ctx context.Context, containerID string, config container.ExecOptions) (container.ExecCreateResponse, error) {
	return container.ExecCreateResponse{}, nil
}

func (m *MockDockerClient) ContainerExecStart(ctx context.Context, execID string, config container.ExecStartOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerExecAttach(ctx context.Context, execID string, config container.ExecStartOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}

func (m *MockDockerClient) ContainerExecInspect(ctx context.Context, execID string) (container.ExecInspect, error) {
	return container.ExecInspect{}, nil
}

func (m *MockDockerClient) ContainerExecResize(ctx context.Context, execID string, options container.ResizeOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerAttach(ctx context.Context, containerID string, options container.AttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}

func (m *MockDockerClient) ContainerCommit(ctx context.Context, containerID string, options container.CommitOptions) (container.CommitResponse, error) {
	return container.CommitResponse{}, nil
}

func (m *MockDockerClient) ContainerCopyFromContainer(ctx context.Context, containerID, srcPath string) (io.ReadCloser, container.PathStat, error) {
	return nil, container.PathStat{}, nil
}

func (m *MockDockerClient) ContainerCopyToContainer(ctx context.Context, containerID, path string, content io.Reader, options container.CopyToContainerOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerExport(ctx context.Context, containerID string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *MockDockerClient) ContainerArchive(ctx context.Context, containerID, srcPath string) (io.ReadCloser, error) {
	return nil, nil
}

func (m *MockDockerClient) ContainerArchiveInfo(ctx context.Context, containerID, srcPath string) (container.PathStat, error) {
	return container.PathStat{}, nil
}

func (m *MockDockerClient) ContainerExtractToDir(ctx context.Context, containerID, srcPath string, dstPath string) error {
	return nil
}

func TestCleanupDockerBuildImages(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()

	t.Run("NoContainers", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		// Setup mock expectations
		mockClient.On("ContainerList", ctx, container.ListOptions{All: true}).Return([]types.Container{}, nil)
		mockClient.On("ImagesPrune", ctx, filters.Args{}).Return(image.PruneReport{}, nil)

		err := CleanupDockerBuildImages(fs, ctx, "nonexistent", mockClient)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("ContainerExists", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		// Setup mock expectations for existing container
		containers := []types.Container{
			{
				ID:    "test-container-id",
				Names: []string{"/test-container"},
			},
		}
		mockClient.On("ContainerList", ctx, container.ListOptions{All: true}).Return(containers, nil)
		mockClient.On("ContainerRemove", ctx, "test-container-id", container.RemoveOptions{Force: true}).Return(nil)
		mockClient.On("ImagesPrune", ctx, filters.Args{}).Return(image.PruneReport{}, nil)

		err := CleanupDockerBuildImages(fs, ctx, "test-container", mockClient)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("ContainerListError", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		// Setup mock expectations for error case
		mockClient.On("ContainerList", ctx, container.ListOptions{All: true}).Return([]types.Container{}, assert.AnError)

		err := CleanupDockerBuildImages(fs, ctx, "test-container", mockClient)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error listing containers")
		mockClient.AssertExpectations(t)
	})

	t.Run("ImagesPruneError", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		// Setup mock expectations for error case
		mockClient.On("ContainerList", ctx, container.ListOptions{All: true}).Return([]types.Container{}, nil)
		mockClient.On("ImagesPrune", ctx, filters.Args{}).Return(image.PruneReport{}, assert.AnError)

		err := CleanupDockerBuildImages(fs, ctx, "test-container", mockClient)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "error pruning images")
		mockClient.AssertExpectations(t)
	})
}

func TestCleanup(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	environ := &environment.Environment{DockerMode: "1"}
	logger := logging.NewTestLogger() // Mock logger

	t.Run("NonDockerMode", func(t *testing.T) {
		environ.DockerMode = "0"
		Cleanup(fs, ctx, environ, logger)
		// No assertions, just ensure it doesn't panic
	})

	t.Run("DockerMode", func(t *testing.T) {
		environ.DockerMode = "1"
		Cleanup(fs, ctx, environ, logger)
		// No assertions, just ensure it doesn't panic
	})
}

func TestCleanupFlagFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger() // Mock logger

	t.Run("FilesExist", func(t *testing.T) {
		_ = afero.WriteFile(fs, "/tmp/flag1", []byte(""), 0o644)
		_ = afero.WriteFile(fs, "/tmp/flag2", []byte(""), 0o644)
		cleanupFlagFiles(fs, []string{"/tmp/flag1", "/tmp/flag2"}, logger)
		// No assertions, just ensure it doesn't panic
	})

	t.Run("FilesDoNotExist", func(t *testing.T) {
		cleanupFlagFiles(fs, []string{"/tmp/nonexistent1", "/tmp/nonexistent2"}, logger)
		// No assertions, just ensure it doesn't panic
	})
}
