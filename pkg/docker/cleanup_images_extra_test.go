package docker

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/spf13/afero"
)

type mockPruneClient struct {
	listErr   error
	removeErr error
	pruneErr  error
	removed   []string
}

func (m *mockPruneClient) ContainerList(ctx context.Context, opts container.ListOptions) ([]types.Container, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []types.Container{
		{ID: "abc", Names: []string{"/mycnt"}},
		{ID: "def", Names: []string{"/other"}},
	}, nil
}
func (m *mockPruneClient) ContainerRemove(ctx context.Context, id string, opts container.RemoveOptions) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removed = append(m.removed, id)
	return nil
}
func (m *mockPruneClient) ImagesPrune(ctx context.Context, f filters.Args) (image.PruneReport, error) {
	if m.pruneErr != nil {
		return image.PruneReport{}, m.pruneErr
	}
	return image.PruneReport{}, nil
}

func TestCleanupDockerBuildImages_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	cli := &mockPruneClient{}
	if err := CleanupDockerBuildImages(fs, context.Background(), "mycnt", cli); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cli.removed) != 1 || cli.removed[0] != "abc" {
		t.Fatalf("expected container 'abc' removed, got %v", cli.removed)
	}
}

func TestCleanupDockerBuildImages_ListError(t *testing.T) {
	fs := afero.NewMemMapFs()
	cli := &mockPruneClient{listErr: errors.New("boom")}
	if err := CleanupDockerBuildImages(fs, context.Background(), "x", cli); err == nil {
		t.Fatalf("expected error from ContainerList")
	}
}
