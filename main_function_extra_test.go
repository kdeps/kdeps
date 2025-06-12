package main

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	kdepspkg "github.com/kdeps/schema/gen/kdeps"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

func TestMainEntry_NoDocker(t *testing.T) {
	// Ensure .dockerenv is not present so DockerMode=0
	// Stub all injectable funcs to lightweight versions.
	fs := afero.NewMemMapFs()

	withInjects(func() {
		// environment is created inside main; we can't intercept that easily.

		newGraphResolverFn = func(afero.Fs, context.Context, *environment.Environment, *gin.Context, *logging.Logger) (*resolver.DependencyResolver, error) {
			return &resolver.DependencyResolver{Fs: fs, Logger: logging.NewTestLogger()}, nil
		}
		bootstrapDockerSystemFn = func(context.Context, *resolver.DependencyResolver) (bool, error) { return false, nil }
		runGraphResolverActionsFn = func(context.Context, *resolver.DependencyResolver, bool) error { return nil }

		findConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "config", nil
		}
		generateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "config", nil
		}
		editConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "config", nil
		}
		validateConfigurationFn = func(afero.Fs, context.Context, *environment.Environment, *logging.Logger) (string, error) {
			return "config", nil
		}
		loadConfigurationFn = func(afero.Fs, context.Context, string, *logging.Logger) (*kdepspkg.Kdeps, error) {
			return &kdepspkg.Kdeps{KdepsDir: "."}, nil
		}
		getKdepsPathFn = func(context.Context, kdepspkg.Kdeps) (string, error) { return "/tmp", nil }
		newRootCommandFn = func(afero.Fs, context.Context, string, *kdepspkg.Kdeps, *environment.Environment, *logging.Logger) *cobra.Command {
			return &cobra.Command{Run: func(cmd *cobra.Command, args []string) {}}
		}
		cleanupFn = func(afero.Fs, context.Context, *environment.Environment, bool, *logging.Logger) {}
	}, t)

	// Run main. It should return without panic.
	main()
}
