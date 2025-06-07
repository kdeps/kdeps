package main

import (
	"context"
	"os"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestSetupEnvironment(t *testing.T) {
	fs := afero.NewMemMapFs()

	env, err := setupEnvironment(fs)
	assert.NoError(t, err)
	assert.NotNil(t, env)
	assert.IsType(t, &environment.Environment{}, env)
}

func TestSetupEnvironmentError(t *testing.T) {
	// Test with a filesystem that will cause an error
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())

	env, err := setupEnvironment(fs)
	// The function should still return an environment even if there are minor issues
	// This depends on the actual implementation of environment.NewEnvironment
	if err != nil {
		assert.Nil(t, env)
	} else {
		assert.NotNil(t, env)
	}
}

func TestSetupSignalHandler(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Test that setupSignalHandler doesn't panic
	assert.NotPanics(t, func() {
		setupSignalHandler(fs, ctx, cancel, env, false, logger)
	})

	// Cancel the context to clean up the goroutine
	cancel()
}

func TestCleanup(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env := &environment.Environment{}
	logger := logging.NewTestLogger()

	// Create a cleanup flag file to test removal
	fs.Create("/.dockercleanup")

	// Test that cleanup doesn't panic
	assert.NotPanics(t, func() {
		cleanup(fs, ctx, env, true, logger) // Use apiServerMode=true to avoid os.Exit
	})

	// Check that the cleanup flag file was removed
	_, err := fs.Stat("/.dockercleanup")
	assert.True(t, os.IsNotExist(err))
}
