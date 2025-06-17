package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/spf13/afero"
)

// stubDockerClient satisfies DockerPruneClient for unit-testing.
// It records how many times ImagesPrune was called.
type stubDockerClient struct {
	containers []types.Container
	pruned     bool
}

func (s *stubDockerClient) ContainerList(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
	return s.containers, nil
}
func (s *stubDockerClient) ContainerRemove(ctx context.Context, id string, opts container.RemoveOptions) error {
	// simulate successful removal by deleting from slice
	for i, c := range s.containers {
		if c.ID == id {
			s.containers = append(s.containers[:i], s.containers[i+1:]...)
			break
		}
	}
	return nil
}
func (s *stubDockerClient) ImagesPrune(ctx context.Context, f filters.Args) (image.PruneReport, error) {
	s.pruned = true
	return image.PruneReport{}, nil
}

func TestCleanupDockerBuildImagesStub(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cName := "abc"
	client := &stubDockerClient{
		containers: []types.Container{{ID: "123", Names: []string{"/" + cName}}},
	}

	if err := CleanupDockerBuildImages(fs, ctx, cName, client); err != nil {
		t.Fatalf("CleanupDockerBuildImages returned error: %v", err)
	}

	if client.pruned == false {
		t.Fatalf("expected ImagesPrune to be called")
	}
	if len(client.containers) != 0 {
		t.Fatalf("expected container slice to be empty after removal, got %d", len(client.containers))
	}
}
