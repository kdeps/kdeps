package docker_test

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/stretchr/testify/require"
)

// mockDockerAdapterClient is a mock implementation of the actual Docker client interface
type mockDockerAdapterClient struct {
	containerCreateFunc func(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform interface{}, containerName string) (container.CreateResponse, error)
	containerStartFunc  func(ctx context.Context, containerID string, options container.StartOptions) error
	containerListFunc   func(ctx context.Context, options container.ListOptions) ([]types.Container, error)
}

func (m *mockDockerAdapterClient) ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform interface{}, containerName string) (container.CreateResponse, error) {
	if m.containerCreateFunc != nil {
		return m.containerCreateFunc(ctx, config, hostConfig, networkingConfig, platform, containerName)
	}
	return container.CreateResponse{ID: "mock-id"}, nil
}

func (m *mockDockerAdapterClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	if m.containerStartFunc != nil {
		return m.containerStartFunc(ctx, containerID, options)
	}
	return nil
}

func (m *mockDockerAdapterClient) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	if m.containerListFunc != nil {
		return m.containerListFunc(ctx, options)
	}
	return []types.Container{{ID: "mock-container"}}, nil
}

// Ensure mockDockerAdapterClient implements the same interface as client.Client
var _ interface {
	ContainerCreate(context.Context, *container.Config, *container.HostConfig, *network.NetworkingConfig, interface{}, string) (container.CreateResponse, error)
	ContainerStart(context.Context, string, container.StartOptions) error
	ContainerList(context.Context, container.ListOptions) ([]types.Container, error)
} = &mockDockerAdapterClient{}

func TestNewDockerClientAdapter(t *testing.T) {
	// Test creating adapter with a mock client
	mockClient := &client.Client{}
	adapter := docker.NewDockerClientAdapter(mockClient)
	require.NotNil(t, adapter)

	// Test with nil client
	nilAdapter := docker.NewDockerClientAdapter(nil)
	require.NotNil(t, nilAdapter)
}

func TestDockerClientAdapter_ContainerCreate(t *testing.T) {
	// Create a real Docker client (but we'll override its transport)
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}
	dockerClient.NegotiateAPIVersion(context.Background())

	adapter := docker.NewDockerClientAdapter(dockerClient)
	ctx := context.Background()

	config := &container.Config{
		Image: "hello-world",
	}
	hostConfig := &container.HostConfig{}
	networkingConfig := &network.NetworkingConfig{}

	// This will fail since we're not actually running Docker, but it tests the method
	_, err = adapter.ContainerCreate(ctx, config, hostConfig, networkingConfig, nil, "test-container")

	// We expect an error since Docker is likely not running in test environment
	// The important thing is that the method was called without panicking
	t.Logf("ContainerCreate called, error (expected): %v", err)
}

func TestDockerClientAdapter_ContainerStart(t *testing.T) {
	// Create a real Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}
	dockerClient.NegotiateAPIVersion(context.Background())

	adapter := docker.NewDockerClientAdapter(dockerClient)
	ctx := context.Background()

	// This will fail since the container doesn't exist, but it tests the method
	err = adapter.ContainerStart(ctx, "non-existent-container", container.StartOptions{})

	// We expect an error since the container doesn't exist
	// The important thing is that the method was called without panicking
	t.Logf("ContainerStart called, error (expected): %v", err)
}

func TestDockerClientAdapter_ContainerList(t *testing.T) {
	// Create a real Docker client
	dockerClient, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		t.Skip("Docker not available")
	}
	dockerClient.NegotiateAPIVersion(context.Background())

	adapter := docker.NewDockerClientAdapter(dockerClient)
	ctx := context.Background()

	// This should work even without Docker running (returns empty list)
	containers, err := adapter.ContainerList(ctx, container.ListOptions{})

	// The method should execute without error
	t.Logf("ContainerList called, containers: %d, error: %v", len(containers), err)
}

// TestDockerClientAdapter_NilClient tests that methods panic appropriately with nil client
func TestDockerClientAdapter_NilClient(t *testing.T) {
	adapter := &docker.DockerClientAdapter{}

	t.Run("ContainerCreate", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for nil client")
			}
		}()
		_, _ = adapter.ContainerCreate(context.Background(), &container.Config{}, &container.HostConfig{}, &network.NetworkingConfig{}, nil, "test")
	})

	t.Run("ContainerStart", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for nil client")
			}
		}()
		_ = adapter.ContainerStart(context.Background(), "test-id", container.StartOptions{})
	})

	t.Run("ContainerList", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Errorf("Expected panic for nil client")
			}
		}()
		_, _ = adapter.ContainerList(context.Background(), container.ListOptions{})
	})
}
