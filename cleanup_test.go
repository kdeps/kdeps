package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
)

// TestCleanup_RemovesFlagFile ensures that cleanup deletes a pre-existing /.dockercleanup file
// and does NOT call os.Exit when apiServerMode=true (which would kill the test process).
func TestCleanup_RemovesFlagFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	// Create the flag file that cleanup should remove.
	if err := afero.WriteFile(fs, "/.dockercleanup", []byte("flag"), 0o644); err != nil {
		t.Fatalf("setup write: %v", err)
	}

	env, _ := environment.NewEnvironment(fs, nil) // DockerMode defaults to "0" – docker.Cleanup becomes no-op.

	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Call the helper under test. apiServerMode=true avoids the os.Exit path.
	cleanup(ctx, fs, env, true, logger)

	if exists, _ := afero.Exists(fs, "/.dockercleanup"); exists {
		t.Fatalf("expected flag file to be removed by cleanup")
	}
}
