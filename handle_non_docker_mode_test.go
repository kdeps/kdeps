package main

import (
	"context"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	schemaK "github.com/kdeps/schema/gen/kdeps"
	schemaPath "github.com/kdeps/schema/gen/kdeps/path"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// TestHandleNonDockerMode_GenerateFlow exercises the path where no config exists and it must be generated.
func TestHandleNonDockerMode_GenerateFlow(_ *testing.T) {
	// Prepare filesystem and env
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env, _ := environment.NewEnvironment(fs, nil)
	logger := logging.GetLogger()

	// Backup original fns
	origFind := findConfigurationFn
	origGenerate := generateConfigurationFn
	origEdit := editConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGetPath := getKdepsPathFn
	origNewRoot := newRootCommandFn

	defer func() {
		findConfigurationFn = origFind
		generateConfigurationFn = origGenerate
		editConfigurationFn = origEdit
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGetPath
		newRootCommandFn = origNewRoot
	}()

	// Mock functions with correct parameter order
	findConfigurationFn = func(_ context.Context, _ afero.Fs, _ *environment.Environment, _ *logging.Logger) (string, error) {
		return "/tmp/test-config.pkl", nil
	}
	generateConfigurationFn = func(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger, eval pkl.Evaluator) (string, error) {
		return "/tmp/test-config.pkl", nil
	}
	editConfigurationFn = func(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) (string, error) {
		return "/tmp/test-config.pkl", nil
	}
	validateConfigurationFn = func(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger, eval pkl.Evaluator) (string, error) {
		return "/tmp/test-config.pkl", nil
	}
	loadConfigurationFn = func(_ context.Context, _ afero.Fs, _ string, _ *logging.Logger) (*schemaK.Kdeps, error) {
		dir := ".kdeps"
		p := schemaPath.User
		return &schemaK.Kdeps{KdepsDir: &dir, KdepsPath: &p}, nil
	}
	getKdepsPathFn = func(context.Context, schemaK.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(_ context.Context, _ afero.Fs, _ string, _ *schemaK.Kdeps, _ *environment.Environment, logger *logging.Logger) *cobra.Command {
		return &cobra.Command{Run: func(_ *cobra.Command, _ []string) {}}
	}

	// Call the function; expecting graceful completion without panic.
	handleNonDockerMode(ctx, fs, env, logger, &dummyEvaluator{})
}

// TestHandleNonDockerMode_ExistingConfig exercises the flow when a configuration already exists.
func TestHandleNonDockerMode_ExistingConfig(_ *testing.T) {
	fs := afero.NewMemMapFs()
	ctx := context.Background()
	env, _ := environment.NewEnvironment(fs, nil)
	logger := logging.GetLogger()

	// Backup originals
	origFind := findConfigurationFn
	origValidate := validateConfigurationFn
	origLoad := loadConfigurationFn
	origGetPath := getKdepsPathFn
	origNewRoot := newRootCommandFn

	defer func() {
		findConfigurationFn = origFind
		validateConfigurationFn = origValidate
		loadConfigurationFn = origLoad
		getKdepsPathFn = origGetPath
		newRootCommandFn = origNewRoot
	}()

	// Stubs
	findConfigurationFn = func(_ context.Context, _ afero.Fs, _ *environment.Environment, logger *logging.Logger) (string, error) {
		return "/test/existing.pkl", nil
	}
	generateConfigurationFn = func(_ context.Context, _ afero.Fs, env *environment.Environment, logger *logging.Logger, eval pkl.Evaluator) (string, error) {
		return "/test/existing.pkl", nil
	}
	validateConfigurationFn = func(_ context.Context, _ afero.Fs, env *environment.Environment, logger *logging.Logger, eval pkl.Evaluator) (string, error) {
		return "/existing/config.yml", nil
	}
	loadConfigurationFn = func(_ context.Context, _ afero.Fs, _ string, logger *logging.Logger) (*schemaK.Kdeps, error) {
		dir := ".kdeps"
		p := schemaPath.User
		return &schemaK.Kdeps{KdepsDir: &dir, KdepsPath: &p}, nil
	}
	getKdepsPathFn = func(context.Context, schemaK.Kdeps) (string, error) {
		return "/kdeps", nil
	}
	newRootCommandFn = func(_ context.Context, _ afero.Fs, _ string, systemCfg *schemaK.Kdeps, env *environment.Environment, logger *logging.Logger) *cobra.Command {
		return &cobra.Command{Use: "root"}
	}

	// Create expected files in the in-memory filesystem
	afero.WriteFile(fs, "/test/existing.pkl", []byte("dummy config"), 0644)
	afero.WriteFile(fs, "/existing/config.yml", []byte("dummy config"), 0644)

	// Execute with a dummy evaluator so the test passes
	handleNonDockerMode(ctx, fs, env, logger, &dummyEvaluator{})
}

func TestSetupEnvironmentSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	env, err := setupEnvironment(fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env == nil {
		t.Fatalf("expected non-nil environment")
	}
}

// Minimal working mock for pkl.Evaluator
// Satisfies the interface and returns dummy values

type dummyEvaluator struct{}

func (d *dummyEvaluator) EvaluateModule(ctx context.Context, source *pkl.ModuleSource, out any) error {
	return nil
}
func (d *dummyEvaluator) EvaluateOutputText(ctx context.Context, source *pkl.ModuleSource) (string, error) {
	return "dummy", nil
}
func (d *dummyEvaluator) EvaluateOutputValue(ctx context.Context, source *pkl.ModuleSource, out any) error {
	return nil
}
func (d *dummyEvaluator) EvaluateOutputFiles(ctx context.Context, source *pkl.ModuleSource) (map[string]string, error) {
	return nil, nil
}
func (d *dummyEvaluator) EvaluateExpression(ctx context.Context, source *pkl.ModuleSource, expr string, out any) error {
	return nil
}
func (d *dummyEvaluator) EvaluateExpressionRaw(ctx context.Context, source *pkl.ModuleSource, expr string) ([]byte, error) {
	return nil, nil
}
func (d *dummyEvaluator) Close() error { return nil }
func (d *dummyEvaluator) Closed() bool { return false }
