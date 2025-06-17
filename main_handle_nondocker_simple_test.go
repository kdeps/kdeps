package main

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	schemaKdeps "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func TestHandleNonDockerModeBasic(t *testing.T) {
	// Setup in-memory filesystem and environment
	fs := afero.NewMemMapFs()
	homeDir := "/home"
	pwdDir := "/workspace"
	_ = fs.MkdirAll(homeDir, 0o755)
	_ = fs.MkdirAll(pwdDir, 0o755)

	env := &environment.Environment{
		Root:           "/",
		Home:           homeDir,
		Pwd:            pwdDir,
		DockerMode:     "0",
		NonInteractive: "1",
	}

	ctx := context.Background()
	logger := logging.NewTestLogger()

	// Inject stubbed dependency functions
	findConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "", nil // force generation path
	}
	generateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		confPath := env.Home + "/.kdeps.pkl"
		if err := afero.WriteFile(fs, confPath, []byte("dummy"), 0o644); err != nil {
			t.Fatalf("failed to write config: %v", err)
		}
		return confPath, nil
	}
	editConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return env.Home + "/.kdeps.pkl", nil
	}
	validateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return env.Home + "/.kdeps.pkl", nil
	}
	loadConfigurationFn = func(fs afero.Fs, ctx context.Context, path string, logger *logging.Logger) (*schemaKdeps.Kdeps, error) {
		return &schemaKdeps.Kdeps{}, nil
	}
	getKdepsPathFn = func(ctx context.Context, k schemaKdeps.Kdeps) (string, error) {
		return "/tmp/kdeps", nil
	}
	newRootCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, cfg *schemaKdeps.Kdeps, env *environment.Environment, logger *logging.Logger) *cobra.Command {
		return &cobra.Command{Use: "root", Run: func(cmd *cobra.Command, args []string) {}}
	}

	// Add context keys to mimic main
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, "graph-id")
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, "/tmp/action")

	// Invoke the function under test. It should complete without panicking or fatal logging.
	handleNonDockerMode(fs, ctx, env, logger)
}
