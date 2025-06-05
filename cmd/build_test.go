package cmd

import (
	"context"
	"errors"
	"testing"

	"github.com/docker/docker/client"
	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDockerClient is a mock implementation of the Docker client
type MockDockerClient struct {
	mock.Mock
}

func (m *MockDockerClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestNewBuildCommand(t *testing.T) {
	// Setup
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	assert.Equal(t, "build [package]", cmd.Use)
	assert.Equal(t, []string{"b"}, cmd.Aliases)
	assert.Equal(t, "$ kdeps build ./myAgent.kdeps", cmd.Example)
	assert.Equal(t, "Build a dockerized AI agent", cmd.Short)
	assert.True(t, cmd.Args != nil)

	// Save original functions to restore after test
	origExtract := extractPackageFn
	origBuildDockerfile := buildDockerfileFn
	origNewDockerClient := newDockerClientFn
	origBuildDockerImage := buildDockerImageFn
	origCleanupDockerBuildImages := cleanupDockerBuildImagesFn
	defer func() {
		extractPackageFn = origExtract
		buildDockerfileFn = origBuildDockerfile
		newDockerClientFn = origNewDockerClient
		buildDockerImageFn = origBuildDockerImage
		cleanupDockerBuildImagesFn = origCleanupDockerBuildImages
	}()

	// --- Error path: ExtractPackage fails ---
	extractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return nil, errors.New("extract error")
	}
	err := cmd.RunE(cmd, []string{"fail.kdeps"})
	assert.ErrorContains(t, err, "extract error")

	// --- Error path: BuildDockerfile fails ---
	extractPackageFn = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{}, nil
	}
	buildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
		return "", false, false, "", "", "", "", "", errors.New("dockerfile error")
	}
	err = cmd.RunE(cmd, []string{"fail.kdeps"})
	assert.ErrorContains(t, err, "dockerfile error")

	// --- Error path: NewDockerClient fails ---
	buildDockerfileFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, bool, bool, string, string, string, string, string, error) {
		return "runDir", false, false, "", "", "", "", "", nil
	}
	newDockerClientFn = func(opts ...client.Opt) (*client.Client, error) {
		return nil, errors.New("client error")
	}
	err = cmd.RunE(cmd, []string{"fail.kdeps"})
	assert.ErrorContains(t, err, "client error")

	// --- Error path: BuildDockerImage fails ---
	newDockerClientFn = func(opts ...client.Opt) (*client.Client, error) {
		return &client.Client{}, nil
	}
	buildDockerImageFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, cli *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
		return "", "", errors.New("image error")
	}
	err = cmd.RunE(cmd, []string{"fail.kdeps"})
	assert.ErrorContains(t, err, "image error")

	// --- Error path: CleanupDockerBuildImages fails ---
	buildDockerImageFn = func(fs afero.Fs, ctx context.Context, kdeps *kdeps.Kdeps, cli *client.Client, runDir, kdepsDir string, pkgProject *archiver.KdepsPackage, logger *logging.Logger) (string, string, error) {
		return "containerName", "containerName:latest", nil
	}
	cleanupDockerBuildImagesFn = func(fs afero.Fs, ctx context.Context, cName string, cli *client.Client) error {
		return errors.New("cleanup error")
	}
	err = cmd.RunE(cmd, []string{"fail.kdeps"})
	assert.ErrorContains(t, err, "cleanup error")

	// --- Success path ---
	cleanupDockerBuildImagesFn = func(fs afero.Fs, ctx context.Context, cName string, cli *client.Client) error {
		return nil
	}
	err = cmd.RunE(cmd, []string{"success.kdeps"})
	assert.NoError(t, err)
}

func TestBuildCommandArgs(t *testing.T) {
	// Setup
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	systemCfg := &kdeps.Kdeps{}
	logger := logging.NewTestLogger()

	// Create test command
	cmd := NewBuildCommand(fs, ctx, kdepsDir, systemCfg, logger)

	// Test minimum args requirement
	err := cmd.Args(cmd, []string{})
	assert.Error(t, err, "should require at least one argument")

	// Test valid args
	err = cmd.Args(cmd, []string{"package.kdeps"})
	assert.NoError(t, err, "should accept valid package path")
}
