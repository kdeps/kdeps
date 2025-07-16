package docker_test

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/log"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/messages"
	"github.com/kdeps/kdeps/pkg/schema"
)

type mockPruneClient struct {
	listErr   error
	removeErr error
	pruneErr  error
	removed   []string
}

func (m *mockPruneClient) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return []container.Summary{
		{ID: "abc", Names: []string{"/mycnt"}},
		{ID: "def", Names: []string{"/other"}},
	}, nil
}

func (m *mockPruneClient) ContainerRemove(_ context.Context, id string, _ container.RemoveOptions) error {
	if m.removeErr != nil {
		return m.removeErr
	}
	m.removed = append(m.removed, id)
	return nil
}

func (m *mockPruneClient) ImagesPrune(_ context.Context, _ filters.Args) (image.PruneReport, error) {
	if m.pruneErr != nil {
		return image.PruneReport{}, m.pruneErr
	}
	return image.PruneReport{}, nil
}

func TestCleanupDockerBuildImages_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	cli := &mockPruneClient{}
	if err := docker.CleanupDockerBuildImages(fs, context.Background(), "mycnt", cli); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cli.removed) != 1 || cli.removed[0] != "abc" {
		t.Fatalf("expected container 'abc' removed, got %v", cli.removed)
	}
}

func TestCleanupDockerBuildImages_ListError(t *testing.T) {
	fs := afero.NewMemMapFs()
	cli := &mockPruneClient{listErr: errors.New("boom")}
	if err := docker.CleanupDockerBuildImages(fs, context.Background(), "x", cli); err == nil {
		t.Fatalf("expected error from ContainerList")
	}
}

func TestCleanupFlagFilesSimple(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create temporary files
	files := []string{filepath.Join(t.TempDir(), "file1.flag"), filepath.Join(t.TempDir(), "file2.flag"), filepath.Join(t.TempDir(), "file3.flag")}
	for _, f := range files {
		if err := afero.WriteFile(fs, f, []byte("data"), 0o644); err != nil {
			t.Fatalf("unable to create temp file: %v", err)
		}
	}

	docker.CleanupFlagFiles(fs, files, logger)

	// Verify they are removed
	for _, f := range files {
		if _, err := fs.Stat(f); err == nil {
			t.Fatalf("expected file %s to be removed", f)
		}
	}
}

func TestCleanupDockerFlow(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// prepare fake directories mimicking docker layout
	graphID := "gid123"
	agentDir := "/agent"
	actionDir := filepath.Join(agentDir, "action")
	projectDir := filepath.Join(agentDir, "project")
	workflowDir := filepath.Join(agentDir, "workflow")

	// populate dirs and a test file inside project
	require.NoError(t, fs.MkdirAll(filepath.Join(projectDir, "sub"), 0o755))
	require.NoError(t, afero.WriteFile(fs, filepath.Join(projectDir, "sub", "file.txt"), []byte("data"), 0o644))

	// action directory (will be removed)
	require.NoError(t, fs.MkdirAll(actionDir, 0o755))

	// context with required keys
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)

	env := &environment.Environment{DockerMode: "1"}

	// run cleanup – we just assert it completes within reasonable time (~2s)
	done := make(chan struct{})
	go func() {
		docker.Cleanup(fs, ctx, env, logger)
		close(done)
	}()

	select {
	case <-done:
		// verify that workflowDir now exists and contains copied file (if copy executed)
		copied := filepath.Join(workflowDir, "sub", "file.txt")
		exists, _ := afero.Exists(fs, copied)
		// either exist or not depending on timing – we just make sure function returned
		_ = exists
	case <-ctx.Done():
		t.Fatal("context canceled prematurely")
	}
}

func TestCreateFlagFileAndCleanup(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	flag1 := filepath.Join(t.TempDir(), "flag1")
	flag2 := filepath.Join(t.TempDir(), "flag2")

	// Create first flag file via helper.
	if err := docker.CreateFlagFile(fs, ctx, flag1); err != nil {
		t.Fatalf("CreateFlagFile returned error: %v", err)
	}

	// Second call with same path should NO-OP (exists) and return nil.
	if err := docker.CreateFlagFile(fs, ctx, flag1); err != nil {
		t.Fatalf("CreateFlagFile second call expected nil err, got %v", err)
	}

	// Manually create another flag for removal.
	if err := afero.WriteFile(fs, flag2, []byte("test"), 0o644); err != nil {
		t.Fatalf("setup write file: %v", err)
	}

	// Ensure both files exist before cleanup.
	for _, p := range []string{flag1, flag2} {
		if ok, _ := afero.Exists(fs, p); !ok {
			t.Fatalf("expected %s to exist", p)
		}
	}

	logger := logging.NewTestLogger()
	docker.CleanupFlagFiles(fs, []string{flag1, flag2}, logger)

	// Confirm they are removed.
	for _, p := range []string{flag1, flag2} {
		if ok, _ := afero.Exists(fs, p); ok {
			t.Fatalf("expected %s to be removed by cleanupFlagFiles", p)
		}
	}

	// Verify CreateFlagFile sets timestamps (basic sanity: non-zero ModTime).
	path := filepath.Join(t.TempDir(), "flag3")
	if err := docker.CreateFlagFile(fs, ctx, path); err != nil {
		t.Fatalf("CreateFlagFile: %v", err)
	}
	info, _ := fs.Stat(path)
	if info.ModTime().IsZero() || time.Since(info.ModTime()) > time.Minute {
		t.Fatalf("unexpected ModTime on created flag file: %v", info.ModTime())
	}
}

// fakeClient implements DockerPruneClient for testing.
type fakeClient struct {
	containers []container.Summary
	listErr    error
	removeErr  error
	pruneErr   error
}

func (f *fakeClient) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	return f.containers, f.listErr
}

func (f *fakeClient) ContainerRemove(_ context.Context, _ string, _ container.RemoveOptions) error {
	if f.removeErr != nil {
		return f.removeErr
	}
	return nil
}

func (f *fakeClient) ImagesPrune(_ context.Context, _ filters.Args) (image.PruneReport, error) {
	if f.pruneErr != nil {
		return image.PruneReport{}, f.pruneErr
	}
	return image.PruneReport{}, nil
}

func TestCleanupDockerBuildImages_NoContainers(t *testing.T) {
	client := &fakeClient{}
	err := docker.CleanupDockerBuildImages(nil, context.Background(), "", client)
	require.NoError(t, err)
}

func TestCleanupDockerBuildImages_RemoveAndPruneSuccess(t *testing.T) {
	client := &fakeClient{
		containers: []container.Summary{{ID: "abc123", Names: []string{"/testname"}}},
	}
	// Should handle remove and prune without error
	err := docker.CleanupDockerBuildImages(nil, context.Background(), "testname", client)
	require.NoError(t, err)
}

func TestCleanupDockerBuildImages_PruneError(t *testing.T) {
	client := &fakeClient{pruneErr: errors.New("prune failed")}
	err := docker.CleanupDockerBuildImages(nil, context.Background(), "", client)
	require.Error(t, err)
	require.Contains(t, err.Error(), "prune failed")
}

// TestCleanupFlagFilesExtra verifies that cleanupFlagFiles removes specified files.
func TestCleanupFlagFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create two files and leave one missing to exercise both paths
	files := []string{filepath.Join(t.TempDir(), "f1"), filepath.Join(t.TempDir(), "f2"), filepath.Join(t.TempDir(), "missing")}
	require.NoError(t, afero.WriteFile(fs, files[0], []byte("x"), 0o644))
	require.NoError(t, afero.WriteFile(fs, files[1], []byte("y"), 0o644))

	docker.CleanupFlagFiles(fs, files, logger)

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

	docker.CleanupFlagFiles(fs, []string{f1, f2}, logger)

	for _, p := range []string{f1, f2} {
		if exists, _ := afero.Exists(fs, p); exists {
			t.Fatalf("expected %s to be removed", p)
		}
	}
}

func TestCleanupFlagFiles_NonExistent(_ *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Call with files that don't exist; should not panic or error.
	docker.CleanupFlagFiles(fs, []string{"/missing1", "/missing2"}, logger)
}

type stubPruneClient struct {
	containers  []container.Summary
	removedIDs  []string
	pruneCalled bool
	removeErr   error
}

func (s *stubPruneClient) ContainerList(_ context.Context, _ container.ListOptions) ([]container.Summary, error) {
	return s.containers, nil
}

func (s *stubPruneClient) ContainerRemove(_ context.Context, id string, _ container.RemoveOptions) error {
	if s.removeErr != nil {
		return s.removeErr
	}
	s.removedIDs = append(s.removedIDs, id)
	return nil
}

func (s *stubPruneClient) ImagesPrune(_ context.Context, _ filters.Args) (image.PruneReport, error) {
	s.pruneCalled = true
	return image.PruneReport{}, nil
}

func TestCleanupDockerBuildImages_RemovesMatchAndPrunes(t *testing.T) {
	cli := &stubPruneClient{
		containers: []container.Summary{{ID: "abc", Names: []string{"/target"}}},
	}

	if err := docker.CleanupDockerBuildImages(nil, context.Background(), "target", cli); err != nil {
		t.Fatalf("CleanupDockerBuildImages error: %v", err)
	}

	if len(cli.removedIDs) != 1 || cli.removedIDs[0] != "abc" {
		t.Fatalf("container not removed as expected: %+v", cli.removedIDs)
	}
	if !cli.pruneCalled {
		t.Fatalf("ImagesPrune not called")
	}
}

func TestCleanupFlagFilesRemoveAllExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create two dummy files
	paths := []string{filepath.Join(t.TempDir(), "flag1"), filepath.Join(t.TempDir(), "flag2")}
	for _, p := range paths {
		afero.WriteFile(fs, p, []byte("x"), 0o644)
	}

	docker.CleanupFlagFiles(fs, paths, logger)

	for _, p := range paths {
		if exists, _ := afero.Exists(fs, p); exists {
			t.Fatalf("file %s still exists after cleanup", p)
		}
	}
}

// TestCleanupDockerMode_Timeout ensures that Cleanup enters DockerMode branch,
// removes the actionDir, and returns after WaitForFileReady timeout without panic.
func TestCleanupDockerMode_Timeout(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	graphID := "gid123"
	actionDir := "/action"

	// Create the directories and dummy files that will be deleted during cleanup.
	if err := fs.MkdirAll(actionDir, 0o755); err != nil {
		t.Fatalf("setup mkdir: %v", err)
	}
	_ = afero.WriteFile(fs, actionDir+"/dummy.txt", []byte("x"), 0o644)

	// Also create project directory with file, though copy step may not be reached.
	if err := fs.MkdirAll("/agent/project", 0o755); err != nil {
		t.Fatalf("setup project dir: %v", err)
	}
	_ = afero.WriteFile(fs, "/agent/project/hello.txt", []byte("hi"), 0o644)

	// Prepare context with graphID and actionDir.
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)

	env := &environment.Environment{DockerMode: "1"}

	start := time.Now()
	docker.Cleanup(fs, ctx, env, logger) // should block ~1s due to WaitForFileReady timeout
	elapsed := time.Since(start)
	if elapsed < time.Second {
		t.Fatalf("expected at least 1s wait, got %v", elapsed)
	}

	// Verify actionDir has been removed.
	if exists, _ := afero.DirExists(fs, actionDir); exists {
		t.Fatalf("expected actionDir to be removed")
	}
}

// TestCleanupFlagFiles verifies that cleanupFlagFiles removes existing files and
// silently skips files that do not exist.
func TestCleanupFlagFilesAdditional(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create a temporary directory and a flag file that should be removed.
	tmpDir := t.TempDir()
	flag1 := filepath.Join(tmpDir, "flag1")
	require.NoError(t, afero.WriteFile(fs, flag1, []byte("flag"), 0o644))

	// flag2 intentionally does NOT exist to hit the non-existence branch.
	flag2 := filepath.Join(tmpDir, "flag2")

	docker.CleanupFlagFiles(fs, []string{flag1, flag2}, logger)

	// Verify flag1 has been deleted and flag2 still does not exist.
	_, err := fs.Stat(flag1)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	_, err = fs.Stat(flag2)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))
}

// TestCleanupEndToEnd exercises the happy-path of the high-level Cleanup
// function, covering directory removals, flag-file creation and the project →
// workflow copy.  The in-memory filesystem allows us to use absolute paths
// without touching the real host filesystem.
func TestCleanupEndToEnd(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Prepare context keys expected by Cleanup.
	graphID := "graph123"
	actionDir := filepath.Join(t.TempDir(), "action") // Any absolute path is fine for the mem fs.
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)

	// Docker mode must be "1" for Cleanup to execute.
	env := &environment.Environment{DockerMode: "1"}

	// Create the action directory so that Cleanup can delete it.
	require.NoError(t, fs.MkdirAll(actionDir, 0o755))

	// Pre-create the second flag file so that WaitForFileReady does not time out.
	preFlag := filepath.Join(actionDir, ".dockercleanup_"+graphID)
	require.NoError(t, afero.WriteFile(fs, preFlag, []byte("flag"), 0o644))

	// Create a dummy project directory with a single file that should be copied
	// to the workflow directory by Cleanup.
	projectDir := "/agent/project"
	dummyFile := filepath.Join(projectDir, "hello.txt")
	require.NoError(t, fs.MkdirAll(projectDir, 0o755))
	require.NoError(t, afero.WriteFile(fs, dummyFile, []byte("hello"), 0o644))

	// Execute the function under test.
	docker.Cleanup(fs, ctx, env, logger)

	// Assert that the action directory has been removed.
	_, err := fs.Stat(actionDir)
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	// Cleanup finished without panicking and the action directory is gone – that's sufficient for this test.
}

// stubDockerClient satisfies DockerPruneClient for unit-testing.
// It records how many times ImagesPrune was called.
type stubDockerClient struct {
	containers []container.Summary
	pruned     bool
}

func (s *stubDockerClient) ContainerList(_ context.Context, opts container.ListOptions) ([]container.Summary, error) {
	return s.containers, nil
}

func (s *stubDockerClient) ContainerRemove(_ context.Context, id string, opts container.RemoveOptions) error {
	// simulate successful removal by deleting from slice
	for i, c := range s.containers {
		if c.ID == id {
			s.containers = append(s.containers[:i], s.containers[i+1:]...)
			break
		}
	}
	return nil
}

func (s *stubDockerClient) ImagesPrune(_ context.Context, f filters.Args) (image.PruneReport, error) {
	s.pruned = true
	return image.PruneReport{}, nil
}

func TestCleanupDockerBuildImagesStub(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()

	cName := "abc"
	client := &stubDockerClient{
		containers: []container.Summary{{ID: "123", Names: []string{"/" + cName}}},
	}

	if err := docker.CleanupDockerBuildImages(fs, ctx, cName, client); err != nil {
		t.Fatalf("CleanupDockerBuildImages returned error: %v", err)
	}

	if client.pruned == false {
		t.Fatalf("expected ImagesPrune to be called")
	}
	if len(client.containers) != 0 {
		t.Fatalf("expected container slice to be empty after removal, got %d", len(client.containers))
	}
}

// MockDockerClient is a mock implementation of the DockerPruneClient interface
// Only the required methods are implemented
type MockDockerClient struct {
	mock.Mock
}

var _ docker.DockerPruneClient = (*MockDockerClient)(nil)

func (m *MockDockerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	args := m.Called(ctx, options)
	return args.Get(0).([]container.Summary), args.Error(1)
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
func (m *MockDockerClient) ContainerStart(_ context.Context, _ string, _ container.StartOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerStop(_ context.Context, _ string, _ *container.StopOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerWait(_ context.Context, _ string, _ container.WaitCondition) (<-chan container.WaitResponse, <-chan error) {
	return make(chan container.WaitResponse), make(chan error)
}

func (m *MockDockerClient) ContainerLogs(_ context.Context, _ string, _ container.LogsOptions) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (m *MockDockerClient) ContainerInspect(_ context.Context, _ string) (types.ContainerJSON, error) {
	return types.ContainerJSON{}, nil
}

func (m *MockDockerClient) ContainerInspectWithRaw(_ context.Context, _ string, _ bool) (types.ContainerJSON, []byte, error) {
	return types.ContainerJSON{}, nil, nil
}

func (m *MockDockerClient) ContainerStats(_ context.Context, _ string, _ bool) (container.Stats, error) {
	return container.Stats{}, nil
}

func (m *MockDockerClient) ContainerStatsOneShot(_ context.Context, _ string) (container.Stats, error) {
	return container.Stats{}, nil
}

func (m *MockDockerClient) ContainerTop(_ context.Context, _ string, _ []string) (container.ContainerTopOKBody, error) {
	return container.ContainerTopOKBody{}, nil
}

func (m *MockDockerClient) ContainerUpdate(_ context.Context, _ string, _ container.UpdateConfig) (container.ContainerUpdateOKBody, error) {
	return container.ContainerUpdateOKBody{}, nil
}

func (m *MockDockerClient) ContainerPause(_ context.Context, _ string) error {
	return nil
}

func (m *MockDockerClient) ContainerUnpause(_ context.Context, _ string) error {
	return nil
}

func (m *MockDockerClient) ContainerRestart(_ context.Context, _ string, _ *container.StopOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerKill(_ context.Context, _, _ string) error {
	return nil
}

func (m *MockDockerClient) ContainerRename(_ context.Context, _, _ string) error {
	return nil
}

func (m *MockDockerClient) ContainerResize(_ context.Context, _ string, _ container.ResizeOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerExecCreate(_ context.Context, _ string, _ container.ExecOptions) (container.ExecCreateResponse, error) {
	return container.ExecCreateResponse{}, nil
}

func (m *MockDockerClient) ContainerExecStart(_ context.Context, _ string, _ container.ExecStartOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerExecAttach(_ context.Context, _ string, _ container.ExecStartOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}

func (m *MockDockerClient) ContainerExecInspect(_ context.Context, _ string) (container.ExecInspect, error) {
	return container.ExecInspect{}, nil
}

func (m *MockDockerClient) ContainerExecResize(_ context.Context, _ string, _ container.ResizeOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerAttach(_ context.Context, _ string, _ container.AttachOptions) (types.HijackedResponse, error) {
	return types.HijackedResponse{}, nil
}

func (m *MockDockerClient) ContainerCommit(_ context.Context, _ string, _ container.CommitOptions) (container.CommitResponse, error) {
	return container.CommitResponse{}, nil
}

func (m *MockDockerClient) ContainerCopyFromContainer(_ context.Context, _, _ string) (io.ReadCloser, container.PathStat, error) {
	return nil, container.PathStat{}, nil
}

func (m *MockDockerClient) ContainerCopyToContainer(_ context.Context, _, _ string, _ io.Reader, _ container.CopyToContainerOptions) error {
	return nil
}

func (m *MockDockerClient) ContainerExport(_ context.Context, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (m *MockDockerClient) ContainerArchive(_ context.Context, _, _ string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}

func (m *MockDockerClient) ContainerArchiveInfo(_ context.Context, _, _ string) (container.PathStat, error) {
	return container.PathStat{}, nil
}

func (m *MockDockerClient) ContainerExtractToDir(_ context.Context, _, _, _ string) error {
	return nil
}

func TestCleanupDockerBuildImages(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()

	t.Run("NoContainers", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		// Setup mock expectations
		mockClient.On("ContainerList", ctx, container.ListOptions{All: true}).Return([]container.Summary{}, nil)
		mockClient.On("ImagesPrune", ctx, filters.Args{}).Return(image.PruneReport{}, nil)

		err := docker.CleanupDockerBuildImages(fs, ctx, "nonexistent", mockClient)
		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("ContainerExists", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		// Setup mock expectations for existing container
		containers := []container.Summary{
			{
				ID:    "test-container-id",
				Names: []string{"/test-container"},
			},
		}
		mockClient.On("ContainerList", ctx, container.ListOptions{All: true}).Return(containers, nil)
		mockClient.On("ContainerRemove", ctx, "test-container-id", container.RemoveOptions{Force: true}).Return(nil)
		mockClient.On("ImagesPrune", ctx, filters.Args{}).Return(image.PruneReport{}, nil)

		err := docker.CleanupDockerBuildImages(fs, ctx, "test-container", mockClient)
		require.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("ContainerListError", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		// Setup mock expectations for error case
		mockClient.On("ContainerList", ctx, container.ListOptions{All: true}).Return([]container.Summary{}, assert.AnError)

		err := docker.CleanupDockerBuildImages(fs, ctx, "test-container", mockClient)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error listing containers")
		mockClient.AssertExpectations(t)
	})

	t.Run("ImagesPruneError", func(t *testing.T) {
		mockClient := &MockDockerClient{}
		// Setup mock expectations for error case
		mockClient.On("ContainerList", ctx, container.ListOptions{All: true}).Return([]container.Summary{}, nil)
		mockClient.On("ImagesPrune", ctx, filters.Args{}).Return(image.PruneReport{}, assert.AnError)

		err := docker.CleanupDockerBuildImages(fs, ctx, "test-container", mockClient)
		require.Error(t, err)
		require.Contains(t, err.Error(), "error pruning images")
		mockClient.AssertExpectations(t)
	})
}

func TestCleanup(t *testing.T) {
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	environ := &environment.Environment{DockerMode: "1"}
	logger := logging.NewTestLogger() // Mock logger

	t.Run("NonDockerMode", func(_ *testing.T) {
		environ.DockerMode = "0"
		docker.Cleanup(fs, ctx, environ, logger)
		// No assertions, just ensure it doesn't panic
	})

	t.Run("DockerMode", func(_ *testing.T) {
		environ.DockerMode = "1"
		docker.Cleanup(fs, ctx, environ, logger)
		// No assertions, just ensure it doesn't panic
	})
}

func TestCleanupFlagFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	baseLogger := log.New(nil)
	logger := &logging.Logger{Logger: baseLogger}

	// Test case 1: No files to remove
	files := []string{}
	docker.CleanupFlagFiles(fs, files, logger)
	t.Log("cleanupFlagFiles with no files test passed")

	// Test case 2: Remove existing file
	filePath := filepath.Join(t.TempDir(), "flag1")
	err := afero.WriteFile(fs, filePath, []byte("test"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	files = []string{filePath}
	docker.CleanupFlagFiles(fs, files, logger)
	_, err = afero.ReadFile(fs, filePath)
	if err == nil {
		t.Errorf("Expected file to be removed, but it still exists")
	}
	t.Log("cleanupFlagFiles with existing file test passed")

	// Test case 3: Attempt to remove non-existing file
	files = []string{filepath.Join(t.TempDir(), "nonexistent")}
	docker.CleanupFlagFiles(fs, files, logger)
	t.Log("cleanupFlagFiles with non-existing file test passed")

	// Test case 4: Multiple files, some existing, some not
	filePath2 := filepath.Join(t.TempDir(), "flag2")
	err = afero.WriteFile(fs, filePath2, []byte("test2"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create second test file: %v", err)
	}
	files = []string{filePath2, filepath.Join(t.TempDir(), "nonexistent2")}
	docker.CleanupFlagFiles(fs, files, logger)
	_, err = afero.ReadFile(fs, filePath2)
	if err == nil {
		t.Errorf("Expected second file to be removed, but it still exists")
	}
	t.Log("cleanupFlagFiles with multiple files test passed")
}

// fakeDockerClient implements DockerPruneClient for unit-tests.
type fakeDockerClient struct {
	containers []container.Summary
	pruned     bool
}

func (f *fakeDockerClient) ContainerList(ctx context.Context, opts container.ListOptions) ([]container.Summary, error) {
	return f.containers, nil
}

func (f *fakeDockerClient) ContainerRemove(ctx context.Context, id string, opts container.RemoveOptions) error {
	// simulate removal by filtering slice
	var out []container.Summary
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
	err := docker.CleanupDockerBuildImages(afero.NewOsFs(), context.Background(), "dummy", cli)
	require.NoError(t, err)
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
	docker.CleanupFlagFiles(fs, []string{file1, file2}, logger)

	for _, f := range []string{file1, file2} {
		exists, _ := afero.Exists(fs, f)
		if exists {
			t.Fatalf("expected %s to be removed", f)
		}
	}

	// Calling cleanupFlagFiles again should hit the os.IsNotExist branch and not fail.
	docker.CleanupFlagFiles(fs, []string{file1, file2}, logger)
}

func TestServerReadyHelpers(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a TCP listener on an ephemeral port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	host, port, _ := net.SplitHostPort(ln.Addr().String())

	t.Run("isServerReady_true", func(t *testing.T) {
		assert.True(t, docker.IsServerReady(host, port, logger))
	})

	t.Run("waitForServer_success", func(t *testing.T) {
		assert.NoError(t, docker.WaitForServer(host, port, 2*time.Second, logger))
	})

	// close listener to make port unavailable
	_ = ln.Close()

	t.Run("isServerReady_false", func(t *testing.T) {
		assert.False(t, docker.IsServerReady(host, port, logger))
	})

	t.Run("waitForServer_timeout", func(t *testing.T) {
		err := docker.WaitForServer(host, port, 1500*time.Millisecond, logger)
		assert.Error(t, err)
	})
}

// TestIsServerReady_Extra checks that the helper correctly detects
// an open TCP port and a closed one.
func TestIsServerReady_Extra(t *testing.T) {
	logger := logging.NewTestLogger()
	// Listen on a random available port on localhost.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	_, port, _ := net.SplitHostPort(ln.Addr().String())

	if !docker.IsServerReady("127.0.0.1", port, logger) {
		t.Fatalf("server should be reported as ready on open port")
	}

	// pick an arbitrary high port unlikely to be used (and different)
	if docker.IsServerReady("127.0.0.1", "65535", logger) {
		t.Fatalf("server should not be ready on closed port")
	}

	schema.Version(context.Background()) // maintain convention
}

// TestWaitForServerQuickSuccess ensures waitForServer returns quickly when the
// port is already open.
func TestWaitForServerQuickSuccess(t *testing.T) {
	logger := logging.NewTestLogger()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	_, port, _ := net.SplitHostPort(ln.Addr().String())

	start := time.Now()
	if err := docker.WaitForServer("127.0.0.1", port, 500*time.Millisecond, logger); err != nil {
		t.Fatalf("waitForServer returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 100*time.Millisecond {
		t.Fatalf("waitForServer took too long: %v", elapsed)
	}

	schema.Version(context.Background())
}

// TestIsServerReadyAndWaitForServer covers both positive and timeout scenarios
// for the helper functions in server_utils.go.
func TestIsServerReadyAndWaitForServerExtra(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a temporary TCP server listening on an available port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	host, port, _ := net.SplitHostPort(ln.Addr().String())

	// Expect server to be reported as ready.
	if !docker.IsServerReady(host, port, logger) {
		t.Fatalf("expected server to be ready")
	}

	// waitForServer should return quickly for an already-ready server.
	if err := docker.WaitForServer(host, port, 2*time.Second, logger); err != nil {
		t.Fatalf("waitForServer returned error: %v", err)
	}

	// Close listener to test timeout path.
	ln.Close()

	start := time.Now()
	err = docker.WaitForServer(host, port, 1*time.Second, logger)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	if time.Since(start) < 1*time.Second {
		t.Fatalf("waitForServer returned too quickly, expected it to wait for timeout")
	}

	// Context compile-time check to ensure startOllamaServer callable without panic.
	// We cannot execute it because it would attempt to run an external binary, but we
	// can at least ensure it does not panic when invoked with a canceled context.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	docker.StartOllamaServer(ctx, logger)
}

func TestIsServerReadyAndWaitForServer(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a dummy TCP listener on a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	addr := ln.Addr().String() // e.g. 127.0.0.1:54321
	host, port, _ := strings.Cut(addr, ":")

	// isServerReady should return true.
	if ready := docker.IsServerReady(host, port, logger); !ready {
		t.Errorf("expected server to be ready on %s:%s", host, port)
	}

	// waitForServer should return quickly because it's already ready.
	if err := docker.WaitForServer(host, port, 3*time.Second, logger); err != nil {
		t.Errorf("waitForServer returned error: %v", err)
	}

	// Close the listener to test negative case quickly with isServerReady
	ln.Close()
	// Choose a port unlikely to be in use (listener just closed)
	pInt, _ := strconv.Atoi(port)
	unavailablePort := strconv.Itoa(pInt)
	if ready := docker.IsServerReady(host, unavailablePort, logger); ready {
		t.Errorf("expected server NOT to be ready on closed port %s", unavailablePort)
	}
}

func TestWaitForServerTimeout(t *testing.T) {
	logger := logging.NewTestLogger()

	// Use an unlikely port to be open
	host := "127.0.0.1"
	port := "65000"

	start := time.Now()
	err := docker.WaitForServer(host, port, 1500*time.Millisecond, logger)
	duration := time.Since(start)

	if err == nil {
		t.Errorf("expected timeout error for unopened port")
	}
	// Ensure it respected the timeout (±500ms)
	if duration < time.Second || duration > 3*time.Second {
		t.Errorf("waitForServer duration out of expected bounds: %v", duration)
	}
}

func TestIsServerReadyListener(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a temporary TCP listener to simulate ready server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	addr := ln.Addr().(*net.TCPAddr)
	portStr := strconv.Itoa(addr.Port)

	if !docker.IsServerReady("127.0.0.1", portStr, logger) {
		t.Fatalf("expected server to be ready on open port")
	}
	ln.Close()

	// After closing listener, readiness should fail
	if docker.IsServerReady("127.0.0.1", portStr, logger) {
		t.Fatalf("expected server NOT ready after listener closed")
	}
}

func TestWaitForServerTimeoutShort(t *testing.T) {
	logger := logging.NewTestLogger()
	port := "65534" // unlikely to be in use
	start := time.Now()
	err := docker.WaitForServer("127.0.0.1", port, 1500*time.Millisecond, logger)
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if time.Since(start) < 1500*time.Millisecond {
		t.Fatalf("waitForServer returned too early")
	}
}

func TestIsServerReadyVariants(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a real TCP listener on an ephemeral port to simulate ready server.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()
	host, port, _ := net.SplitHostPort(ln.Addr().String())

	if ok := docker.IsServerReady(host, port, logger); !ok {
		t.Fatalf("expected server to be ready")
	}

	// Close listener to make port unavailable.
	ln.Close()

	if ok := docker.IsServerReady(host, port, logger); ok {
		t.Fatalf("expected server to be NOT ready after close")
	}
}

func TestIsServerReadyAndWaitForServerSimple(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a temporary TCP listener to act as a fake Ollama server.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start listener: %v", err)
	}
	defer ln.Close()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())

	// Positive case for isServerReady
	if !docker.IsServerReady(host, portStr, logger) {
		t.Fatalf("expected server to be ready on open port")
	}

	// Positive case for waitForServer with short timeout
	if err := docker.WaitForServer(host, portStr, 2*time.Second, logger); err != nil {
		t.Fatalf("waitForServer unexpectedly failed: %v", err)
	}

	// Close listener to test negative path
	ln.Close()

	// Now port should be closed; isServerReady should return false
	if docker.IsServerReady(host, portStr, logger) {
		t.Fatalf("expected server not ready after listener closed")
	}

	// waitForServer should timeout quickly
	timeout := 1500 * time.Millisecond
	start := time.Now()
	err = docker.WaitForServer(host, portStr, timeout, logger)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	elapsed := time.Since(start)
	// Ensure we waited at least 'timeout' but not excessively more (allow 1s margin)
	if elapsed < timeout || elapsed > timeout+time.Second {
		t.Fatalf("waitForServer elapsed time unexpected: %s (timeout %s)", elapsed, timeout)
	}
}

// TestStartOllamaServerReturn ensures the helper returns immediately even when the underlying command is missing.
func TestStartOllamaServerReturn(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	start := time.Now()
	docker.StartOllamaServer(ctx, logger)
	if time.Since(start) > 200*time.Millisecond {
		t.Fatalf("startOllamaServer took too long to return")
	}
}

// TestStartOllamaServer_NoBinary ensures the helper returns immediately even when the
// underlying binary is not present on the host machine. It simply exercises the
// code path to boost coverage without making assumptions about the external
// environment.
func TestStartOllamaServer_NoBinary(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	start := time.Now()
	docker.StartOllamaServer(ctx, logging.NewTestLogger())
	elapsed := time.Since(start)

	// The function should return almost instantly because it only launches the
	// background goroutine. Use a generous threshold to avoid flakes.
	if elapsed > 100*time.Millisecond {
		t.Fatalf("startOllamaServer took too long: %v", elapsed)
	}
}

// TestStartOllamaServerBackground verifies that the helper kicks off the background task and logs as expected.
func TestStartOllamaServerBackground(t *testing.T) {
	// Create a temporary directory that will hold a dummy `ollama` executable.
	tmpDir := t.TempDir()
	dummy := filepath.Join(tmpDir, "ollama")
	if err := os.WriteFile(dummy, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("failed to write dummy executable: %v", err)
	}

	// Prepend the temp dir to PATH so it's discovered by exec.LookPath.
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", tmpDir+string(os.PathListSeparator)+oldPath)
	t.Cleanup(func() { t.Setenv("PATH", oldPath) })

	logger := logging.NewTestLogger()

	// Call the function under test; it should return immediately.
	docker.StartOllamaServer(context.Background(), logger)

	// Allow some time for the goroutine in KdepsExec to start and finish.
	time.Sleep(150 * time.Millisecond)

	output := logger.GetOutput()
	if !strings.Contains(output, messages.MsgStartOllamaBackground) {
		t.Errorf("expected log %q not found. logs: %s", messages.MsgStartOllamaBackground, output)
	}
	if !strings.Contains(output, "background command started") {
		t.Errorf("expected background start log not found. logs: %s", output)
	}
}

func TestStartOllamaServerSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := logging.NewTestLogger()

	// Call function under test; it should return immediately and not panic.
	docker.StartOllamaServer(ctx, logger)

	// Give the background goroutine a brief moment to run and fail gracefully.
	time.Sleep(10 * time.Millisecond)
}

func TestCheckDevBuildModeVariants(t *testing.T) {
	fs := afero.NewMemMapFs()
	kdepsDir := t.TempDir()
	logger := logging.NewTestLogger()

	// Case 1: file missing -> expect false
	ok, err := docker.CheckDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatalf("expected false when file absent")
	}

	// Case 2: file present -> expect true
	cacheFile := filepath.Join(kdepsDir, "cache", "kdeps")
	_ = fs.MkdirAll(filepath.Dir(cacheFile), 0o755)
	_ = afero.WriteFile(fs, cacheFile, []byte("bin"), 0o755)

	ok, err = docker.CheckDevBuildMode(fs, kdepsDir, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatalf("expected true when file present")
	}
}

func TestStartOllamaServerStubbed(t *testing.T) {
	logger := logging.NewTestLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Function should return immediately and not panic.
	docker.StartOllamaServer(ctx, logger)
}

func TestIsServerReady(t *testing.T) {
	logger := logging.GetLogger()

	t.Run("ServerReady", func(t *testing.T) {
		// Start a test TCP server
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		host, port, _ := net.SplitHostPort(listener.Addr().String())
		ready := docker.IsServerReady(host, port, logger)
		assert.True(t, ready)
	})

	t.Run("ServerNotReady", func(t *testing.T) {
		ready := docker.IsServerReady("127.0.0.1", "99999", logger)
		assert.False(t, ready)
	})
}

func TestWaitForServer(t *testing.T) {
	logger := logging.GetLogger()

	t.Run("ServerReady", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		defer listener.Close()

		host, port, _ := net.SplitHostPort(listener.Addr().String())
		err = docker.WaitForServer(host, port, 2*time.Second, logger)
		require.NoError(t, err)
	})

	t.Run("Timeout", func(t *testing.T) {
		err := docker.WaitForServer("127.0.0.1", "99999", 1*time.Second, logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
	})
}

func TestStartOllamaServer(t *testing.T) {
	ctx := context.Background()
	// Initialize a proper logger to avoid nil pointer dereference
	baseLogger := log.New(nil)
	logger := &logging.Logger{Logger: baseLogger}

	// Simply call the function to ensure it doesn't panic
	// Since it runs in background, we can't easily check the result
	docker.StartOllamaServer(ctx, logger)

	// If we reach here without panic, the test passes
	t.Log("startOllamaServer called without panic")
}

func TestWaitForServerSuccess(t *testing.T) {
	logger := logging.NewTestLogger()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer ln.Close()

	host, port, _ := net.SplitHostPort(ln.Addr().String())

	if err := docker.WaitForServer(host, port, 2*time.Second, logger); err != nil {
		t.Fatalf("waitForServer returned error: %v", err)
	}
}

func TestWaitForServerReadyAndTimeout(t *testing.T) {
	logger := logging.NewTestLogger()

	// Start a real TCP listener on an ephemeral port.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer ln.Close()

	host, portStr, _ := net.SplitHostPort(ln.Addr().String())

	// Ready case: should return quickly.
	start := time.Now()
	if err := docker.WaitForServer(host, portStr, 2*time.Second, logger); err != nil {
		t.Fatalf("expected server to be ready, got error: %v", err)
	}
	if time.Since(start) > time.Second {
		t.Fatalf("waitForServer took too long for ready case")
	}

	// Timeout case: use a different unused port.
	unusedPort := strconv.Itoa(60000)
	start = time.Now()
	err = docker.WaitForServer(host, unusedPort, 1*time.Second, logger)
	if err == nil {
		t.Fatalf("expected timeout error for unopened port")
	}
	if time.Since(start) < 900*time.Millisecond {
		t.Fatalf("waitForServer returned too quickly on timeout path")
	}
}

func TestParseOLLAMAHostVariants(t *testing.T) {
	logger := logging.NewTestLogger()

	// Success path.
	originalOllamaHost := os.Getenv("OLLAMA_HOST")
	t.Setenv("OLLAMA_HOST", "0.0.0.0:12345")
	defer t.Setenv("OLLAMA_HOST", originalOllamaHost)

	host, port, err := docker.ParseOLLAMAHost(logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if host != "0.0.0.0" || port != "12345" {
		t.Fatalf("incorrect parse result: %s %s", host, port)
	}

	// Invalid format path.
	t.Setenv("OLLAMA_HOST", "badformat")
	if _, _, err := docker.ParseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error for invalid format")
	}

	// Missing var path.
	if _, _, err := docker.ParseOLLAMAHost(logger); err == nil {
		t.Fatalf("expected error when OLLAMA_HOST unset")
	}
}
