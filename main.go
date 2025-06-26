package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/bus"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	v "github.com/kdeps/kdeps/pkg/version"
	"github.com/spf13/afero"
)

var (
	version = "dev"
	commit  = ""
)

func main() {
	v.Version = version
	v.Commit = commit

	logger := GetLoggerFn()
	fs := NewOsFsFn()

	// Global context and cancel for dependency injection
	ctx, cancel = ContextWithCancelFn(context.Background())
	defer cancel() // Ensure context is canceled when main exits

	graphID := uuid.New().String()
	actionDir := filepath.Join(os.TempDir(), "kdeps", graphID)
	agentDir := filepath.FromSlash("/agent")
	sharedDir := filepath.FromSlash("/.kdeps")

	// Setup environment
	env, err := SetupEnvironmentFn(fs)
	if err != nil {
		logger.Fatalf("failed to set up environment: %v", err)
	}

	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)

	if env.DockerMode == "1" {
		ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
		ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
		ctx = ktx.CreateContext(ctx, ktx.CtxKeySharedDir, sharedDir)

		dr, err := NewGraphResolverFn(fs, ctx, env, nil, logger.With("requestID", graphID))
		if err != nil {
			logger.Fatalf("failed to create graph resolver: %v", err)
		}

		HandleDockerMode(ctx, dr, cancel)
	} else {
		HandleNonDockerMode(fs, ctx, env, logger)
	}
}

func HandleDockerMode(ctx context.Context, dr *resolver.DependencyResolver, cancel context.CancelFunc) {
	// Start the message bus server
	busService, err := StartBusServerBackgroundFn(dr.Logger)
	if err != nil {
		dr.Logger.Error("failed to start message bus server", "error", err)
		SendSigtermFn(dr.Logger)
		return
	}
	SetGlobalBusServiceFn(busService)
	dr.Logger.Info("Message bus server started successfully")

	// Initialize Docker system
	apiServerMode, err := BootstrapDockerSystemFn(ctx, dr)
	if err != nil {
		dr.Logger.Error("error during Docker bootstrap", "error", err)
		SendSigtermFn(dr.Logger)
		return
	}
	// Setup graceful shutdown handler
	SetupSignalHandlerFn(dr.Fs, ctx, cancel, dr.Environment, apiServerMode, dr.Logger)

	// Run workflow or wait for shutdown
	if !apiServerMode {
		if err := RunGraphResolverActionsFn(ctx, dr, apiServerMode); err != nil {
			dr.Logger.Error("error running graph resolver", "error", err)
			SendSigtermFn(dr.Logger)
			return
		}
	}

	// Wait for shutdown signal
	<-ctx.Done()
	dr.Logger.Debug("context canceled, shutting down gracefully...")
	cleanupFn(dr.Fs, ctx, dr.Environment, apiServerMode, dr.Logger)
}

func HandleNonDockerMode(fs afero.Fs, ctx context.Context, env *environment.Environment, logger *logging.Logger) {
	cfgFile, err := FindConfigurationFn(fs, ctx, env, logger)
	if err != nil {
		logger.Error("error occurred finding configuration")
	}

	if cfgFile == "" {
		cfgFile, err = GenerateConfigurationFn(fs, ctx, env, logger)
		if err != nil {
			logger.Fatal("error occurred generating configuration", "error", err)
			return
		}

		logger.Info("configuration file generated", "file", cfgFile)

		cfgFile, err = EditConfigurationFn(fs, ctx, env, logger)
		if err != nil {
			logger.Error("error occurred editing configuration")
		}
	}

	if cfgFile == "" {
		return
	}

	logger.Info("configuration file ready", "file", cfgFile)

	cfgFile, err = ValidateConfigurationFn(fs, ctx, env, logger)
	if err != nil {
		logger.Fatal("error occurred validating configuration", "error", err)
		return
	}

	systemCfg, err := LoadConfigurationFn(fs, ctx, cfgFile, logger)
	if err != nil {
		logger.Error("error occurred loading configuration")
		return
	}

	kdepsDir, err := GetKdepsPathFn(ctx, *systemCfg)
	if err != nil {
		logger.Error("error occurred while getting Kdeps system path")
		return
	}

	rootCmd := NewRootCommandFn(fs, ctx, kdepsDir, systemCfg, env, logger)
	if err := rootCmd.Execute(); err != nil {
		logger.Fatal(err)
	}
}

// SetupEnvironment initializes the environment using the filesystem.
func SetupEnvironment(fs afero.Fs) (*environment.Environment, error) {
	environ, err := NewEnvironmentFn(fs, nil)
	if err != nil {
		return nil, err
	}
	return environ, nil
}

// SetupSignalHandler sets up a goroutine to handle OS signals for graceful shutdown.
func SetupSignalHandler(fs afero.Fs, ctx context.Context, cancelFunc context.CancelFunc, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	sigs := MakeSignalChanFn()
	SignalNotifyWrapper(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigs
		logger.Debug(fmt.Sprintf("Received signal: %v, initiating shutdown...", sig))
		cancelFunc() // Cancel context to initiate shutdown
		cleanupFn(fs, ctx, env, apiServerMode, logger)

		var graphID string
		if value, found := ktx.ReadContext(ctx, ktx.CtxKeyGraphID); found {
			if strValue, ok := value.(string); ok {
				graphID = strValue
			}
		}

		// Wait for cleanup completion event via message bus instead of stamp file
		client, err := StartBusClientFn()
		if err != nil {
			logger.Error("failed to connect to message bus for cleanup coordination", "error", err)
			exitFn(1)
			return
		}
		defer client.Close()

		// Wait for dockercleanup event
		err = WaitForEventsFn(client, logger, func(event bus.Event) bool {
			if event.Type == "dockercleanup" && event.Payload == graphID {
				logger.Debug("Received dockercleanup event", "graphID", graphID)
				return true // Stop waiting
			}
			return false // Continue waiting
		})

		if err != nil {
			logger.Error("error waiting for cleanup event", "error", err)
		}

		exitFn(0)
	}()
}

// RunGraphResolverActions prepares and runs the graph resolver.
func RunGraphResolverActions(ctx context.Context, dr *resolver.DependencyResolver, apiServerMode bool) error {
	// Prepare workflow directory
	if err := PrepareWorkflowDirFn(dr); err != nil {
		return fmt.Errorf("failed to prepare workflow directory: %w", err)
	}

	if err := PrepareImportFilesFn(dr); err != nil {
		return fmt.Errorf("failed to prepare import files: %w", err)
	}

	// Handle run action
	fatal, err := HandleRunActionFn(dr)
	if err != nil {
		return fmt.Errorf("failed to handle run action: %w", err)
	}

	// In certain error cases, Ollama needs to be restarted
	if fatal {
		dr.Logger.Fatal("fatal error occurred")
		SendSigtermFn(dr.Logger)
	}

	cleanupFn(dr.Fs, ctx, dr.Environment, apiServerMode, dr.Logger)

	// Publish cleanup completion event instead of creating stamp file
	var graphID string
	if value, found := ktx.ReadContext(ctx, ktx.CtxKeyGraphID); found {
		if strValue, ok := value.(string); ok {
			graphID = strValue
		}
	}

	PublishGlobalEventFn("dockercleanup", graphID)
	dr.Logger.Debug("Published dockercleanup completion event", "graphID", graphID)

	return nil
}

// Cleanup performs any necessary cleanup tasks before shutting down.
func Cleanup(fs afero.Fs, ctx context.Context, env *environment.Environment, apiServerMode bool, logger *logging.Logger) {
	logger.Debug("performing cleanup tasks...")

	// Remove any old cleanup flags
	if _, err := fs.Stat("/.dockercleanup"); err == nil {
		if err := fs.RemoveAll("/.dockercleanup"); err != nil {
			logger.Error("unable to delete cleanup flag file", "cleanup-file", "/.dockercleanup", "error", err)
		}
	}

	// Perform Docker cleanup
	DockerCleanupFn(fs, ctx, env, logger)

	// Publish cleanup completion event instead of creating stamp file
	var graphID string
	if value, found := ktx.ReadContext(ctx, ktx.CtxKeyGraphID); found {
		if strValue, ok := value.(string); ok {
			graphID = strValue
		}
	}

	PublishGlobalEventFn("cleanup_completed", graphID)
	logger.Debug("Published cleanup completion event", "graphID", graphID)

	logger.Debug("cleanup complete.")

	if !apiServerMode {
		exitFn(0)
	}
}
