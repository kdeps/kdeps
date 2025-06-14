package main

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/schema"
	kdepstype "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// TestHandleNonDockerMode_Happy mocks dependencies so the flow completes without fatal errors.
func TestHandleNonDockerMode_Happy(t *testing.T) {
	fs := afero.NewMemMapFs()
	tmp := t.TempDir()

	// Prepare a dummy config file path to be used by stubs.
	cfgPath := filepath.Join(tmp, "config.pkl")
	_ = afero.WriteFile(fs, cfgPath, []byte("config"), 0o644)

	env := &environment.Environment{
		Home:           tmp,
		Pwd:            tmp,
		NonInteractive: "1",
	}

	// Backup original function pointers.
	origFind := findConfigurationFn
	origGen := generateConfigurationFn
	origEdit := editConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGetPath := getKdepsPathFn
	origNewRoot := newRootCommandFn

	// Restore after test.
	t.Cleanup(func() {
		findConfigurationFn = origFind
		generateConfigurationFn = origGen
		editConfigurationFn = origEdit
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGetPath
		newRootCommandFn = origNewRoot
	})

	// Stubs.
	findConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "", nil // force generate path
	}
	generateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return cfgPath, nil
	}
	editConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return cfgPath, nil
	}
	validateConfigurationFn = func(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) (string, error) {
		return cfgPath, nil
	}
	loadConfigurationFn = func(fs afero.Fs, ctx context.Context, configFile string, logger *logging.Logger) (*kdepstype.Kdeps, error) {
		return &kdepstype.Kdeps{}, nil
	}
	getKdepsPathFn = func(ctx context.Context, _ kdepstype.Kdeps) (string, error) {
		return filepath.Join(tmp, "agents"), nil
	}
	newRootCommandFn = func(fs afero.Fs, ctx context.Context, kdepsDir string, _ *kdepstype.Kdeps, _ *environment.Environment, _ *logging.Logger) *cobra.Command {
		return &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
	}

	logger := logging.NewTestLogger()

	// Execute the function under test; expect it to run without panics or exits.
	handleNonDockerMode(fs, context.Background(), env, logger)

	// Sanity: ensure our logger captured the ready message.
	if out := logger.GetOutput(); out == "" {
		t.Fatalf("expected some log output, got none")
	}

	_ = schema.SchemaVersion(context.Background())
}
