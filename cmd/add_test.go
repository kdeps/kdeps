package cmd

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/archiver"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunAdd_Error(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	origExtract := extractPackage
	defer func() { extractPackage = origExtract }()

	extractPackage = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return nil, errors.New("mock error")
	}

	err := runAdd(fs, ctx, kdepsDir, "some.kdeps", logger)
	require.EqualError(t, err, "mock error")
}

func TestRunAdd_Success(t *testing.T) {
	t.Parallel()
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	origExtract := extractPackage
	defer func() { extractPackage = origExtract }()

	extractPackage = func(fs afero.Fs, ctx context.Context, kdepsDir, pkgFile string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{}, nil
	}

	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w

	err := runAdd(fs, ctx, kdepsDir, "some.kdeps", logger)
	w.Close()
	os.Stdout = stdout

	var buf bytes.Buffer
	_, copyErr := io.Copy(&buf, r)
	require.NoError(t, copyErr)

	require.NoError(t, err)
	assert.Contains(t, buf.String(), "AI agent installed locally: some.kdeps")
}

func TestNewInstallCommand(t *testing.T) {
	t.Parallel()

	// Setup test dependencies
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	kdepsDir := "/tmp/kdeps"
	logger := logging.NewTestLogger()

	// Test command creation
	cmd := NewInstallCommand(fs, ctx, kdepsDir, logger)
	assert.NotNil(t, cmd)
	assert.Equal(t, "install [package]", cmd.Use)
	assert.Equal(t, []string{"i"}, cmd.Aliases)
	assert.Equal(t, "Install an AI agent locally", cmd.Short)
	assert.Equal(t, "$ kdeps install ./myAgent.kdeps", cmd.Example)

	// Test command execution with no arguments
	err := cmd.ValidateArgs([]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "requires at least 1 arg")

	// Test command execution with invalid package file
	err = cmd.ValidateArgs([]string{"nonexistent.kdeps"})
	assert.NoError(t, err)

	// Test command execution with valid package file
	testPkg := "test.kdeps"
	_, err = fs.Create(testPkg)
	require.NoError(t, err)
	err = cmd.ValidateArgs([]string{testPkg})
	assert.NoError(t, err)

	// Test command execution with multiple arguments
	err = cmd.ValidateArgs([]string{"test1.kdeps", "test2.kdeps"})
	assert.NoError(t, err)

	// Test command execution with empty package name
	err = cmd.ValidateArgs([]string{""})
	assert.NoError(t, err)

	// Test command execution with special characters in package name
	err = cmd.ValidateArgs([]string{"test@package.kdeps"})
	assert.NoError(t, err)

	// Test RunE function
	// Override extractPackage for testing
	originalExtractPackage := extractPackage
	defer func() { extractPackage = originalExtractPackage }()

	// Test RunE with error
	extractPackage = func(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return nil, errors.New("mock error")
	}
	err = cmd.RunE(cmd, []string{"test.kdeps"})
	assert.Error(t, err)
	assert.Equal(t, "mock error", err.Error())

	// Test RunE with success
	extractPackage = func(fs afero.Fs, ctx context.Context, kdepsDir string, kdepsPackage string, logger *logging.Logger) (*archiver.KdepsPackage, error) {
		return &archiver.KdepsPackage{
			PkgFilePath: "test.kdeps",
			Workflow:    "workflow.pkl",
			Resources:   []string{},
			Data:        make(map[string]map[string][]string),
		}, nil
	}
	err = cmd.RunE(cmd, []string{"test.kdeps"})
	assert.NoError(t, err)
}
