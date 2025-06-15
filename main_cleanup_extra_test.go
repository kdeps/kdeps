package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"
)

func TestCleanupFlagRemovalMemFS(t *testing.T) {
	_ = schema.SchemaVersion(nil)

	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	flag := "/.dockercleanup"
	if err := afero.WriteFile(fs, flag, []byte("flag"), 0o644); err != nil {
		t.Fatalf("write flag: %v", err)
	}

	env := &environment.Environment{DockerMode: "0"}

	cleanup(fs, ctx, env, true, logger)

	if exists, _ := afero.Exists(fs, flag); exists {
		t.Fatalf("cleanup did not remove %s", flag)
	}
}
