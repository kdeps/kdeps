package resolver

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/agent"
	"github.com/kdeps/kdeps/pkg/evaluator"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/pklres"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	pklResource "github.com/kdeps/schema/gen/resource"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadResourceWithRequestContext(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create temporary directories
	tmpDir := t.TempDir()
	requestDir := filepath.Join(tmpDir, "request")
	resourceDir := filepath.Join(tmpDir, "resources")
	_ = fs.MkdirAll(requestDir, 0o755)
	_ = fs.MkdirAll(resourceDir, 0o755)

	// Initialize resource readers
	memoryReader, err := memory.InitializeMemory(filepath.Join(tmpDir, "memory.db"))
	require.NoError(t, err)
	defer memoryReader.DB.Close()

	sessionReader, err := session.InitializeSession(filepath.Join(tmpDir, "session.db"))
	require.NoError(t, err)
	defer sessionReader.DB.Close()

	toolReader, err := tool.InitializeTool(filepath.Join(tmpDir, "tool.db"))
	require.NoError(t, err)
	defer toolReader.DB.Close()

	itemReader, err := item.InitializeItem(filepath.Join(tmpDir, "item.db"), []string{})
	require.NoError(t, err)
	defer itemReader.DB.Close()

	agentReader, err := agent.InitializeAgent(fs, tmpDir, "default", "latest", logger)
	require.NoError(t, err)
	defer agentReader.Close()

	pklresReader, err := pklres.InitializePklResource(filepath.Join(tmpDir, "pklres.db"))
	require.NoError(t, err)
	defer pklresReader.DB.Close()

	// Initialize the evaluator
	evaluatorConfig := &evaluator.EvaluatorConfig{
		ResourceReaders: []pkl.ResourceReader{
			memoryReader,
			sessionReader,
			toolReader,
			itemReader,
			agentReader,
			pklresReader,
		},
		Logger: logger,
	}
	err = evaluator.InitializeEvaluator(ctx, evaluatorConfig)
	require.NoError(t, err)
	defer func() {
		if mgr, err := evaluator.GetEvaluatorManager(); err == nil {
			mgr.Close()
		}
	}()

	// Create a test request PKL file (just the object body, no amends statement)
	requestContent := `Path = "/api/v1/whois"
IP = "127.0.0.1"
ID = "test-request-123"
Data = ""
Method = "GET"
Headers {
    ["User-Agent"] = "test-agent"
    ["Accept"] = "application/json"
}
Params {
    ["q"] = "NeilArmstrong"
}
Files {
}`

	requestFile := filepath.Join(requestDir, "request.pkl")
	err = afero.WriteFile(fs, requestFile, []byte(requestContent), 0o644)
	require.NoError(t, err)

	// Create a test resource PKL file that doesn't require request context
	resourceContent := `amends "package://schema.kdeps.com/core@0.3.0#/Resource.pkl"

ActionID = "testResource"
Name = "Test Resource"
Description = "Test resource for context loading"
Category = "test"

Run {
    Expr {
        // Simple test that doesn't require request object
        output = "Test resource loaded successfully"
    }
}`

	resourceFile := filepath.Join(resourceDir, "test_resource.pkl")
	err = afero.WriteFile(fs, resourceFile, []byte(resourceContent), 0o644)
	require.NoError(t, err)

	// Create a dependency resolver
	dr := &DependencyResolver{
		Fs:             fs,
		Logger:         logger,
		Context:        ctx,
		RequestPklFile: requestFile,
		APIServerMode:  true,
	}

	// Test LoadResourceWithRequestContext
	t.Run("LoadResourceWithRequestContext", func(t *testing.T) {
		result, err := dr.LoadResourceWithRequestContext(ctx, resourceFile, Resource)
		assert.NoError(t, err)
		assert.NotNil(t, result)

		// Verify that the resource was loaded successfully
		// Note: The request object is not currently available in the PKL context
		// without explicit imports, so the resource will be loaded but without
		// request context access
		resource, ok := result.(*pklResource.Resource)
		assert.True(t, ok)
		assert.Equal(t, "testResource", resource.ActionID)
	})

	t.Run("LoadResourceWithRequestContext_NoRequestFile", func(t *testing.T) {
		// Test fallback when no request file is available
		drNoRequest := &DependencyResolver{
			Fs:             fs,
			Logger:         logger,
			Context:        ctx,
			RequestPklFile: "", // No request file
			APIServerMode:  true,
		}

		result, err := drNoRequest.LoadResourceWithRequestContext(ctx, resourceFile, Resource)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("LoadResourceWithRequestContext_InvalidRequestFile", func(t *testing.T) {
		// Test fallback when request file is invalid
		drInvalidRequest := &DependencyResolver{
			Fs:             fs,
			Logger:         logger,
			Context:        ctx,
			RequestPklFile: filepath.Join(requestDir, "nonexistent.pkl"),
			APIServerMode:  true,
		}

		result, err := drInvalidRequest.LoadResourceWithRequestContext(ctx, resourceFile, Resource)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}
