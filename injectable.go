package main

import (
	"context"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/bus"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/spf13/afero"
)

// Injectable functions for testability
var (
	// OS operations
	OsExitFn       = os.Exit
	SignalNotifyFn = signal.Notify

	// Environment functions
	NewEnvironmentFn = environment.NewEnvironment

	// Configuration functions
	FindConfigurationFn     = cfg.FindConfiguration
	GenerateConfigurationFn = cfg.GenerateConfiguration
	EditConfigurationFn     = cfg.EditConfiguration
	ValidateConfigurationFn = cfg.ValidateConfiguration
	LoadConfigurationFn     = cfg.LoadConfiguration
	GetKdepsPathFn          = cfg.GetKdepsPath

	// Command functions
	NewRootCommandFn = cmd.NewRootCommand

	// Resolver functions
	NewGraphResolverFn = resolver.NewGraphResolver

	// Resolver method functions for better testability
	PrepareWorkflowDirFn = func(dr *resolver.DependencyResolver) error {
		return dr.PrepareWorkflowDir()
	}
	PrepareImportFilesFn = func(dr *resolver.DependencyResolver) error {
		return dr.PrepareImportFiles()
	}
	HandleRunActionFn = func(dr *resolver.DependencyResolver) (bool, error) {
		return dr.HandleRunAction()
	}

	// Docker functions
	BootstrapDockerSystemFn = docker.BootstrapDockerSystem
	DockerCleanupFn         = docker.Cleanup

	// Bus functions
	StartBusServerBackgroundFn = bus.StartBusServerBackground
	SetGlobalBusServiceFn      = bus.SetGlobalBusService
	StartBusClientFn           = bus.StartBusClient
	WaitForEventsFn            = bus.WaitForEvents
	PublishGlobalEventFn       = bus.PublishGlobalEvent

	// Utils functions
	SendSigtermFn      = utils.SendSigterm
	WaitForFileReadyFn = utils.WaitForFileReady

	// Logging functions
	GetLoggerFn = logging.GetLogger

	// Main function helpers for better testability
	SetupEnvironmentFn    = SetupEnvironment
	HandleDockerModeFn    = HandleDockerMode
	HandleNonDockerModeFn = HandleNonDockerMode

	// Signal channel creation
	MakeSignalChanFn = func() chan os.Signal {
		return make(chan os.Signal, 1)
	}

	// Context creation
	ContextWithCancelFn = context.WithCancel

	// Afero filesystem
	NewOsFsFn = afero.NewOsFs

	// Function variables for dependency injection during tests
	runGraphResolverActionsFn = RunGraphResolverActions // Reference to graph resolver actions function
	RunGraphResolverActionsFn = RunGraphResolverActions // Export for tests
	SetupSignalHandlerFn      = SetupSignalHandler      // Export for tests
	exitFn                    = OsExitFn                // Reference to the injectable exit function
	cleanupFn                 = Cleanup                 // Reference to the cleanup function
)

// Global context and cancel for testing - these are set by main() and used by tests
var (
	ctx    context.Context
	cancel context.CancelFunc
)

// PKL Evaluator functions
var (
	// EnsurePklBinaryExistsFunc allows injection of PKL binary existence check
	EnsurePklBinaryExistsFunc = func(ctx context.Context, logger *logging.Logger) error {
		binaryNames := []string{"pkl", "pkl.exe"}
		for _, binaryName := range binaryNames {
			if _, err := exec.LookPath(binaryName); err == nil {
				return nil
			}
		}
		logger.Fatal("apple PKL not found in PATH. Please install Apple PKL (see https://pkl-lang.org/main/current/pkl-cli/index.html#installation) for more details")
		os.Exit(1)
		return nil
	}

	// EvalPklFunc allows injection of PKL evaluation
	EvalPklFunc = func(fs afero.Fs, ctx context.Context, resourcePath string, headerSection string, opts func(options *pkl.EvaluatorOptions), logger *logging.Logger) (string, error) {
		// Default implementation would call the actual EvalPkl function
		// This will be set to the real implementation in init()
		return "", nil
	}

	// NewPklEvaluatorFunc allows injection of PKL evaluator creation
	NewPklEvaluatorFunc = func(ctx context.Context, opts func(options *pkl.EvaluatorOptions)) (pkl.Evaluator, error) {
		return pkl.NewEvaluator(ctx, opts)
	}
)

// File system operations
var (
	// AferoTempFileFunc allows injection of temporary file creation
	AferoTempFileFunc = func(fs afero.Fs, dir, pattern string) (afero.File, error) {
		return afero.TempFile(fs, dir, pattern)
	}

	// AferoTempDirFunc allows injection of temporary directory creation
	AferoTempDirFunc = func(fs afero.Fs, dir, prefix string) (string, error) {
		return afero.TempDir(fs, dir, prefix)
	}

	// AferoReadFileFunc allows injection of file reading
	AferoReadFileFunc = func(fs afero.Fs, filename string) ([]byte, error) {
		return afero.ReadFile(fs, filename)
	}

	// AferoWriteFileFunc allows injection of file writing
	AferoWriteFileFunc = func(fs afero.Fs, filename string, data []byte, perm os.FileMode) error {
		return afero.WriteFile(fs, filename, data, perm)
	}

	// CreateKdepsTempDirFunc allows injection of organized kdeps temporary directory creation
	CreateKdepsTempDirFunc = func(fs afero.Fs, requestID string, suffix string) (string, error) {
		return utils.CreateKdepsTempDir(fs, requestID, suffix)
	}

	// CreateKdepsTempFileFunc allows injection of organized kdeps temporary file creation
	CreateKdepsTempFileFunc = func(fs afero.Fs, requestID string, pattern string) (afero.File, error) {
		return utils.CreateKdepsTempFile(fs, requestID, pattern)
	}
)

// OS operations
var (
	// ExecLookPathFunc allows injection of executable lookup
	ExecLookPathFunc = func(file string) (string, error) {
		return exec.LookPath(file)
	}

	// OsExitFunc allows injection of os.Exit for testing
	OsExitFunc = func(code int) {
		os.Exit(code)
	}

	// OsGetenvFunc allows injection of environment variable reading
	OsGetenvFunc = func(key string) string {
		return os.Getenv(key)
	}
)

// Signal handling
var (
	// SignalNotifyFunc allows injection of signal notification
	SignalNotifyFunc = func(c chan<- os.Signal, sig ...os.Signal) {
		signal.Notify(c, sig...)
	}
)

// SignalNotifyWrapper wraps signal.Notify for dependency injection
func SignalNotifyWrapper(c chan<- os.Signal, sig ...os.Signal) {
	SignalNotifyFunc(c, sig...)
}

// Default signal setup
func init() {
	SignalNotifyFunc = func(c chan<- os.Signal, sig ...os.Signal) {
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	}
}

// Additional injectable functions for resolver testing
var (
	// HTTP client operations
	HttpClientDoFunc = func(client interface{}, req interface{}) (interface{}, error) {
		// Default implementation - will be set properly in production
		return nil, nil
	}

	// External command execution for resolver
	ExecuteGoExecuteFunc = func(ctx context.Context, command string, args []string, env []string) (struct{ Stdout, Stderr string }, error) {
		// Default implementation - will be set properly in production
		return struct{ Stdout, Stderr string }{}, nil
	}

	// Time operations for resolver
	TimeNowFunc = func() interface{} {
		// Default implementation - will be set properly in production
		return nil
	}

	// Filesystem walk operations
	AferoWalkFunc = func(fs interface{}, root string, walkFn interface{}) error {
		// Default implementation - will be set properly in production
		return nil
	}

	// PKL loading operations
	PklLoadFromPathFunc = func(ctx context.Context, path string) (interface{}, error) {
		// Default implementation - will be set properly in production
		return nil, nil
	}
)
