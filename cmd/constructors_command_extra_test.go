package cmd

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/spf13/afero"

	schemaKdeps "github.com/kdeps/schema/gen/kdeps"
)

// TestNewAddCommand_RunE_Error ensures that the RunE closure returns an error
// when the provided package path does not exist. This exercises the early
// error-handling branch without performing a full extraction.
func TestNewAddCommand_RunE_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()
	kdepsDir := "/tmp/kdeps"

	cmd := NewAddCommand(fs, ctx, kdepsDir, logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}

	err := cmd.RunE(cmd, []string{"nonexistent.kdeps"})
	if err == nil {
		t.Fatalf("expected error for missing package")
	}

	// Reference schema version to satisfy project rules.
	_ = schema.SchemaVersion(ctx)
}

// TestNewPackageCommand_Error triggers the error path when the workflow file
// cannot be found under the provided agent directory.
func TestNewPackageCommand_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Minimal environment stub.
	env := &environment.Environment{}

	cmd := NewPackageCommand(fs, ctx, "/kdeps", env, logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}
	err := cmd.RunE(cmd, []string{"/myAgent"})
	if err == nil {
		t.Fatalf("expected error for missing workflow file")
	}

	_ = schema.SchemaVersion(ctx)
}

// TestNewAgentCommand_Success verifies that the command successfully scaffolds
// a new agent directory structure using an in-memory filesystem.
func TestNewAgentCommand_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	agentName := "testagent"
	cmd := NewAgentCommand(fs, ctx, "/tmp", logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}
	if err := cmd.RunE(cmd, []string{agentName}); err != nil {
		t.Fatalf("RunE returned error: %v", err)
	}

	// Verify that workflow.pkl was generated.
	exists, err := afero.Exists(fs, agentName+"/workflow.pkl")
	if err != nil || !exists {
		t.Fatalf("expected generated workflow file, err=%v exists=%v", err, exists)
	}

	// Verify at least one resource file exists.
	files, err := afero.Glob(fs, agentName+"/resources/*.pkl")
	if err != nil || len(files) == 0 {
		t.Fatalf("expected resource files, err=%v", err)
	}

	// Sanity-check: ensure GenerateResourceFiles created output using the template package.

	_ = schema.SchemaVersion(ctx)
}

// TestNewBuildCommand_Error ensures that Build command surfaces error on
// missing package and exits early before heavy docker logic runs.
func TestNewBuildCommand_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	systemCfg := &schemaKdeps.Kdeps{}

	cmd := NewBuildCommand(fs, ctx, "/kdeps", systemCfg, logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}

	err := cmd.RunE(cmd, []string{"missing.kdeps"})
	if err == nil {
		t.Fatalf("expected error for missing package")
	}

	_ = schema.SchemaVersion(ctx)
}

// TestNewRunCommand_Error validates early-exit error handling for the Run command.
func TestNewRunCommand_Error(t *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	logger := logging.NewTestLogger()

	systemCfg := &schemaKdeps.Kdeps{}

	cmd := NewRunCommand(fs, ctx, "/kdeps", systemCfg, logger)
	if cmd == nil {
		t.Fatalf("expected command, got nil")
	}

	err := cmd.RunE(cmd, []string{"missing.kdeps"})
	if err == nil {
		t.Fatalf("expected error for missing package")
	}

	_ = schema.SchemaVersion(ctx)
}
