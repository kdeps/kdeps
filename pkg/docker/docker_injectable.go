package docker

import (
	"context"
	"net/http"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/spf13/afero"
)

// Injectable functions for testability
var (
	// HTTP client operations
	HttpNewRequestFunc = func(method, url string, body interface{}) (*http.Request, error) {
		return http.NewRequest(method, url, nil)
	}

	// File system operations
	AferoNewMemMapFsFunc = func() afero.Fs {
		return afero.NewMemMapFs()
	}

	AferoNewOsFsFunc = func() afero.Fs {
		return afero.NewOsFs()
	}

	// PKL operations
	PklNewEvaluatorFunc = func(ctx context.Context, opts func(*pkl.EvaluatorOptions)) (pkl.Evaluator, error) {
		return pkl.NewEvaluator(ctx, opts)
	}

	// Gin operations
	GinNewFunc = func() *gin.Engine {
		return gin.New()
	}

	GinDefaultFunc = func() *gin.Engine {
		return gin.Default()
	}

	// Resolver operations
	NewGraphResolverFunc = func(fs afero.Fs, ctx context.Context, env interface{}, req *gin.Context, logger *logging.Logger) (*resolver.DependencyResolver, error) {
		return resolver.NewGraphResolver(fs, ctx, env.(*environment.Environment), req, logger)
	}

	// API Server operations
	StartAPIServerModeFunc    = StartAPIServerMode
	SetupRoutesFunc           = SetupRoutes
	APIServerHandlerFunc      = APIServerHandler
	HandleMultipartFormFunc   = HandleMultipartForm
	ProcessFileFunc           = ProcessFile
	ValidateMethodFunc        = ValidateMethod
	CleanOldFilesFunc         = CleanOldFiles
	ProcessWorkflowFunc       = ProcessWorkflow
	DecodeResponseContentFunc = DecodeResponseContent
	FormatResponseJSONFunc    = FormatResponseJSON

	// Bootstrap operations
	BootstrapDockerSystemFunc = BootstrapDockerSystem
)

// SetupTestableEnvironment sets up a testable environment for docker operations
func SetupTestableEnvironment() {
	// Override operations for testing
	AferoNewMemMapFsFunc = func() afero.Fs {
		return afero.NewMemMapFs()
	}

	PklNewEvaluatorFunc = func(ctx context.Context, opts func(*pkl.EvaluatorOptions)) (pkl.Evaluator, error) {
		// Return a mock evaluator for testing
		return pkl.NewEvaluator(ctx, func(options *pkl.EvaluatorOptions) {
			options.Logger = pkl.NoopLogger
		})
	}
}

// ResetEnvironment resets functions to defaults
func ResetEnvironment() {
	AferoNewMemMapFsFunc = func() afero.Fs {
		return afero.NewMemMapFs()
	}
	AferoNewOsFsFunc = func() afero.Fs {
		return afero.NewOsFs()
	}
}
