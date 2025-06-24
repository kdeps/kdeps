package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/kdeps/kdeps/cmd"
	"github.com/kdeps/kdeps/pkg/bus"
	"github.com/kdeps/kdeps/pkg/cfg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
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

	// Signal handling functions
	SetupSignalHandlerFn = setupSignalHandler

	// Main function helpers for better testability
	SetupEnvironmentFn    = setupEnvironment
	HandleDockerModeFn    = handleDockerMode
	HandleNonDockerModeFn = handleNonDockerMode

	// Signal channel creation
	MakeSignalChanFn = func() chan os.Signal {
		return make(chan os.Signal, 1)
	}

	// Context creation
	ContextWithCancelFn = context.WithCancel

	// Afero filesystem
	NewOsFsFn = afero.NewOsFs

	// Function variables for dependency injection during tests
	runGraphResolverActionsFn = runGraphResolverActions // Reference to graph resolver actions function
	exitFn                    = OsExitFn                // Reference to the injectable exit function
	cleanupFn                 = cleanup                 // Reference to the cleanup function
)

// Global context and cancel for testing - these are set by main() and used by tests
var (
	ctx    context.Context
	cancel context.CancelFunc
)

// SignalNotifyWrapper wraps signal.Notify for easier mocking
func SignalNotifyWrapper(c chan<- os.Signal, sig ...os.Signal) {
	SignalNotifyFn(c, sig...)
}
