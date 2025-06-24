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

	// Signal channel creation
	MakeSignalChanFn = func() chan os.Signal {
		return make(chan os.Signal, 1)
	}

	// Context creation
	ContextWithCancelFn = context.WithCancel

	// Afero filesystem
	NewOsFsFn = afero.NewOsFs
)

// SignalNotifyWrapper wraps signal.Notify for easier mocking
func SignalNotifyWrapper(c chan<- os.Signal, sig ...os.Signal) {
	SignalNotifyFn(c, sig...)
}
