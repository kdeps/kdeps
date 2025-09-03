package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	v "github.com/kdeps/kdeps/pkg/version"
	"github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = ""

	// Function variables for dependency injection during tests.
	newGraphResolverFn        = resolver.NewGraphResolver
	bootstrapDockerSystemFn   = docker.BootstrapDockerSystem
	runGraphResolverActionsFn = runGraphResolverActions

	findConfigurationFn     func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) = cfg.FindConfiguration
	generateConfigurationFn func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) = cfg.GenerateConfiguration
	editConfigurationFn     func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) = cfg.EditConfiguration
	validateConfigurationFn func(context.Context, afero.Fs, *environment.Environment, *logging.Logger) (string, error) = cfg.ValidateConfiguration
	loadConfigurationFn     func(context.Context, afero.Fs, string, *logging.Logger) (*kdeps.Kdeps, error)             = cfg.LoadConfiguration
	getKdepsPathFn          func(context.Context, kdeps.Kdeps) (string, error)                                         = cfg.GetKdepsPath

	newRootCommandFn func(context.Context, afero.Fs, string, *kdeps.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command = cmd.NewRootCommand

	cleanupFn = cleanup
)

func main() {
	v.Version = version
	v.Commit = commit

	logger := logging.GetLogger()
	fs := afero.NewOsFs()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	env := initializeApplication(logger, fs)
	ctx = setupApplicationContext(ctx, env)

	if env.DockerMode == "1" {
		handleDockerModeApplication(ctx, fs, env, logger, cancel)
	} else {
		handleNonDockerMode(ctx, fs, env, logger)
	}
}

func initializeApplication(logger *logging.Logger, fs afero.Fs) *environment.Environment {
	env, err := setupEnvironment(fs)
	if err != nil {
		logger.Fatalf("failed to set up environment: %v", err)
	}
	return env
}

func setupApplicationContext(ctx context.Context, env *environment.Environment) context.Context {
	graphID := uuid.New().String()
	actionDir := filepath.Join(os.TempDir(), "action")
	agentDir := filepath.Join("/", "agent")

	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	return ctx
}

func handleDockerModeApplication(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger, cancel context.CancelFunc) {
	graphID := uuid.New().String()
	dr, err := newGraphResolverFn(fs, ctx, env, nil, logger.With("requestID", graphID))
	if err != nil {
		logger.Fatalf("failed to create graph resolver: %v", err)
	}
	handleDockerMode(ctx, dr, cancel)
}

func handleDockerMode(ctx context.Context, dr *resolver.DependencyResolver, cancel context.CancelFunc) {
	apiServerMode := initializeDockerSystem(ctx, dr)
	if apiServerMode == -1 {
		return // Error occurred
	}

	setupSignalHandler(ctx, dr.Fs, cancel, dr.Environment, apiServerMode == 1, dr.Logger)
	handleDockerWorkflow(ctx, dr, apiServerMode == 1)
	waitForShutdown(ctx, dr, apiServerMode == 1)
}

func initializeDockerSystem(ctx context.Context, dr *resolver.DependencyResolver) int {
	apiServerMode, err := bootstrapDockerSystemFn(ctx, dr)
	if err != nil {
		dr.Logger.Error("error during Docker bootstrap", "error", err)
		utils.SendSigterm(dr.Logger)
		return -1 // Error
	}
	return boolToInt(apiServerMode)
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func handleDockerWorkflow(ctx context.Context, dr *resolver.DependencyResolver, apiServerMode bool) {
	if !apiServerMode {
		if err := runGraphResolverActionsFn(ctx, dr, apiServerMode); err != nil {
			dr.Logger.Error("error running graph resolver", "error", err)
			utils.SendSigterm(dr.Logger)
		}
	}
}

func waitForShutdown(ctx context.Context, dr *resolver.DependencyResolver, apiServerMode bool) {
	<-ctx.Done()
	dr.Logger.Debug("context canceled, shutting down gracefully...")
	cleanupFn(ctx, dr.Fs, dr.Environment, apiServerMode, dr.Logger)
}

func handleNonDockerMode(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) {
	cfgFile := setupConfiguration(ctx, fs, env, logger)
	if cfgFile == "" {
		return
	}

	systemCfg := loadSystemConfiguration(ctx, fs, cfgFile, logger)
	if systemCfg == nil {
		return
	}

	executeRootCommand(ctx, fs, systemCfg, env, logger)
}

func setupConfiguration(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) string {
	cfgFile := findOrCreateConfiguration(ctx, fs, env, logger)
	if cfgFile == "" {
		return ""
	}

	return validateConfigurationFile(ctx, fs, env, cfgFile, logger)
}

func findOrCreateConfiguration(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) string {
	cfgFile, err := findConfigurationFn(ctx, fs, env, logger)
	if err != nil {
		logger.Error("error occurred finding configuration")
	}

	if cfgFile == "" {
		return generateAndEditConfiguration(ctx, fs, env, logger)
	}

	return cfgFile
}

func generateAndEditConfiguration(ctx context.Context, fs afero.Fs, env *environment.Environment, logger *logging.Logger) string {
	cfgFile, err := generateConfigurationFn(ctx, fs, env, logger)
	if err != nil {
		logger.Fatal("error occurred generating configuration", "error", err)
		return ""
	}

	logger.Info("configuration file generated", "file", cfgFile)

	cfgFile, err = editConfigurationFn(ctx, fs, env, logger)
	if err != nil {
		logger.Error("error occurred editing configuration")
	}

	return cfgFile
}

func validateConfigurationFile(ctx context.Context, fs afero.Fs, env *environment.Environment, cfgFile string, logger *logging.Logger) string {
	logger.Info("configuration file ready", "file", cfgFile)

	cfgFile, err := validateConfigurationFn(ctx, fs, env, logger)
	if err != nil {
		logger.Fatal("error occurred validating configuration", "error", err)
		return ""
	}

	return cfgFile
}

func loadSystemConfiguration(ctx context.Context, fs afero.Fs, cfgFile string, logger *logging.Logger) *kdeps.Kdeps {
	systemCfg, err := loadConfigurationFn(ctx, fs, cfgFile, logger)
	if err != nil {
		logger.Error("error occurred loading configuration")
		return nil
	}

	if systemCfg == nil {
		logger.Error("system configuration is nil")
		return nil
	}

	return systemCfg
}

func executeRootCommand(ctx context.Context, fs afero.Fs, systemCfg *kdeps.Kdeps, env *environment.Environment, logger *logging.Logger) {
	kdepsDir, err := getKdepsPathFn(ctx, *systemCfg)
	if err != nil {
		logger.Error("error occurred while getting Kdeps system path")
		return
	}

	rootCmd := newRootCommandFn(ctx, fs, kdepsDir, systemCfg, env, logger)
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}

// setupEnvironment initializes the environment using the filesystem.
func setupEnvironment(fs afero.Fs) (*environment.Environment, error) {
	environ, err := environment.NewEnvironment(fs, nil)
	if err != nil {
		return nil, err
	}
	return environ, nil
}

// setupSignalHandler sets up a goroutine to handle OS signals for graceful shutdown.
func setupSignalHandler(ctx context.Context, fs afero.Fs, cancelFunc context.CancelFunc, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go handleSignal(sigs, ctx, fs, cancelFunc, env, apiServerMode, logger)
}

func handleSignal(sigs chan os.Signal, ctx context.Context, fs afero.Fs, cancelFunc context.CancelFunc, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	sig := <-sigs
	logger.Debug(fmt.Sprintf("Received signal: %v, initiating shutdown...", sig))

	cancelFunc()
	cleanupFn(ctx, fs, env, apiServerMode, logger)

	graphID, actionDir := extractContextValues(ctx)
	waitForCleanupFile(fs, actionDir, graphID, logger)
	os.Exit(0)
}

func extractContextValues(ctx context.Context) (string, string) {
	var graphID, actionDir string

	contextKeys := map[*string]ktx.ContextKey{
		&graphID:   ktx.CtxKeyGraphID,
		&actionDir: ktx.CtxKeyActionDir,
	}

	for ptr, key := range contextKeys {
		if value, found := ktx.ReadContext(ctx, key); found {
			if strValue, ok := value.(string); ok {
				*ptr = strValue
			}
		}
	}

	return graphID, actionDir
}

func waitForCleanupFile(fs afero.Fs, actionDir, graphID string, logger *logging.Logger) {
	stampFile := filepath.Join(actionDir, ".dockercleanup_"+graphID)

	if err := utils.WaitForFileReady(fs, stampFile, logger); err != nil {
		logger.Error("error occurred while waiting for file to be ready", "file", stampFile)
	}
}

// runGraphResolver prepares and runs the graph resolver.
func runGraphResolverActions(ctx context.Context, dr *resolver.DependencyResolver, apiServerMode bool) error {
	if err := prepareGraphResolver(dr); err != nil {
		return err
	}

	if err := executeGraphResolver(ctx, dr, apiServerMode); err != nil {
		return err
	}

	return finalizeGraphResolver(ctx, dr, apiServerMode)
}

func prepareGraphResolver(dr *resolver.DependencyResolver) error {
	if err := dr.PrepareWorkflowDir(); err != nil {
		return fmt.Errorf("failed to prepare workflow directory: %w", err)
	}

	if err := dr.PrepareImportFiles(); err != nil {
		return fmt.Errorf("failed to prepare import files: %w", err)
	}

	return nil
}

func executeGraphResolver(ctx context.Context, dr *resolver.DependencyResolver, apiServerMode bool) error {
	fatal, err := dr.HandleRunAction()
	if err != nil {
		return fmt.Errorf("failed to handle run action: %w", err)
	}

	if fatal {
		dr.Logger.Fatal("fatal error occurred")
		utils.SendSigterm(dr.Logger)
	}

	return nil
}

func finalizeGraphResolver(ctx context.Context, dr *resolver.DependencyResolver, apiServerMode bool) error {
	cleanupFn(ctx, dr.Fs, dr.Environment, apiServerMode, dr.Logger)

	if err := utils.WaitForFileReady(dr.Fs, "/.dockercleanup", dr.Logger); err != nil {
		return fmt.Errorf("failed to wait for file to be ready: %w", err)
	}

	return nil
}

// cleanup performs any necessary cleanup tasks before shutting down.
func cleanup(ctx context.Context, fs afero.Fs, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	logger.Debug("performing cleanup tasks...")

	// Remove any old cleanup flags
	if _, err := fs.Stat("/.dockercleanup"); err == nil {
		if err := fs.RemoveAll("/.dockercleanup"); err != nil {
			logger.Error("unable to delete cleanup flag file", "cleanup-file", "/.dockercleanup", "error", err)
		}
	}

	// Perform Docker cleanup
	docker.Cleanup(fs, ctx, env, logger)

	logger.Debug("cleanup complete.")

	if !apiServerMode {
		os.Exit(0)
	}
}
