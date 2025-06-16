package docker

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	apiserver "github.com/kdeps/schema/gen/api_server"
	"github.com/kdeps/schema/gen/project"
	"github.com/kdeps/schema/gen/resource"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestAPIServer(t *testing.T) (*resolver.DependencyResolver, *logging.Logger) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	dr := &resolver.DependencyResolver{
		Fs:     fs,
		Logger: logger,
	}
	return dr, logger
}

func TestHandleMultipartForm(t *testing.T) {
	dr, _ := setupTestAPIServer(t)

	t.Run("ValidMultipartForm", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("test content"))
		require.NoError(t, err)
		writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		fileMap := make(map[string]struct{ Filename, Filetype string })
		err = handleMultipartForm(c, dr, fileMap)
		assert.NoError(t, err)
		assert.Len(t, fileMap, 1)
	})

	t.Run("InvalidContentType", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer([]byte("test")))
		c.Request.Header.Set("Content-Type", "text/plain")

		fileMap := make(map[string]struct{ Filename, Filetype string })
		err := handleMultipartForm(c, dr, fileMap)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Unable to parse multipart form")
	})

	t.Run("NoFileField", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("other", "value")
		writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		fileMap := make(map[string]struct{ Filename, Filetype string })
		err := handleMultipartForm(c, dr, fileMap)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "No file uploaded")
	})
}

func TestProcessFile(t *testing.T) {
	dr, _ := setupTestAPIServer(t)
	fileMap := make(map[string]struct{ Filename, Filetype string })

	t.Run("ValidFile", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("test content"))
		require.NoError(t, err)
		writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		_, fileHeader, err := c.Request.FormFile("file")
		require.NoError(t, err)

		err = processFile(fileHeader, dr, fileMap)
		assert.NoError(t, err)
		assert.Len(t, fileMap, 1)
	})
}

func TestValidateMethod(t *testing.T) {
	t.Run("ValidMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		method, err := validateMethod(req, []string{"GET", "POST"})
		assert.NoError(t, err)
		assert.Equal(t, "method = \"GET\"", method)
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/", nil)
		_, err := validateMethod(req, []string{"GET", "POST"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP method \"PUT\" not allowed")
	})

	t.Run("EmptyMethodDefaultsToGet", func(t *testing.T) {
		req := httptest.NewRequest("", "/", nil)
		method, err := validateMethod(req, []string{"GET", "POST"})
		assert.NoError(t, err)
		assert.Equal(t, "method = \"GET\"", method)
	})
}

func TestDecodeResponseContent(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("ValidJSON", func(t *testing.T) {
		response := APIResponse{
			Success: true,
			Response: ResponseData{
				Data: []string{"test"},
			},
			Meta: ResponseMeta{
				RequestID: "123",
			},
		}
		content, err := json.Marshal(response)
		require.NoError(t, err)

		decoded, err := decodeResponseContent(content, logger)
		assert.NoError(t, err)
		assert.True(t, decoded.Success)
		assert.Equal(t, "123", decoded.Meta.RequestID)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		content := []byte("invalid json")
		_, err := decodeResponseContent(content, logger)
		assert.Error(t, err)
	})

	t.Run("EmptyResponse", func(t *testing.T) {
		content := []byte("{}")
		decoded, err := decodeResponseContent(content, logger)
		assert.NoError(t, err)
		assert.False(t, decoded.Success)
	})
}

func TestFormatResponseJSON(t *testing.T) {
	t.Run("ValidResponse", func(t *testing.T) {
		response := APIResponse{
			Success: true,
			Response: ResponseData{
				Data: []string{"test"},
			},
			Meta: ResponseMeta{
				RequestID: "123",
			},
		}
		content, err := json.Marshal(response)
		require.NoError(t, err)

		formatted := formatResponseJSON(content)
		var decoded APIResponse
		err = json.Unmarshal(formatted, &decoded)
		require.NoError(t, err)
		assert.True(t, decoded.Success)
		assert.Equal(t, "123", decoded.Meta.RequestID)
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		response := APIResponse{
			Success: false,
			Errors: []ErrorResponse{
				{
					Code:    400,
					Message: "test error",
				},
			},
			Meta: ResponseMeta{
				RequestID: "123",
			},
		}
		content, err := json.Marshal(response)
		require.NoError(t, err)

		formatted := formatResponseJSON(content)
		var decoded APIResponse
		err = json.Unmarshal(formatted, &decoded)
		require.NoError(t, err)
		assert.False(t, decoded.Success)
		assert.Equal(t, "test error", decoded.Errors[0].Message)
	})
}

func TestCleanOldFiles(t *testing.T) {
	// Create a temporary directory using afero
	fs := afero.NewOsFs()
	tmpDir, err := afero.TempDir(fs, "", "test-cleanup")
	require.NoError(t, err)
	defer fs.RemoveAll(tmpDir)

	// Create a test response file
	responseFile := filepath.Join(tmpDir, "response.json")
	err = afero.WriteFile(fs, responseFile, []byte("test response"), 0644)
	require.NoError(t, err)

	// Create a DependencyResolver with the test filesystem
	dr := &resolver.DependencyResolver{
		Fs:                 fs,
		Logger:             logging.NewTestLogger(),
		ResponseTargetFile: responseFile,
	}

	t.Run("FileExists", func(t *testing.T) {
		err := cleanOldFiles(dr)
		require.NoError(t, err)

		// Verify file was removed
		exists, err := afero.Exists(fs, responseFile)
		require.NoError(t, err)
		assert.False(t, exists)
	})

	t.Run("FileDoesNotExist", func(t *testing.T) {
		// Create a new resolver with a non-existent file
		dr := &resolver.DependencyResolver{
			Fs:                 fs,
			Logger:             logging.NewTestLogger(),
			ResponseTargetFile: filepath.Join(tmpDir, "nonexistent.json"),
		}

		err := cleanOldFiles(dr)
		assert.NoError(t, err)
	})
}

func TestStartAPIServerMode(t *testing.T) {
	dr, _ := setupTestAPIServer(t)

	t.Run("MissingConfig", func(t *testing.T) {
		dr.Logger = logging.NewTestLogger()
		// Provide a mock Workflow with GetSettings() returning nil
		dr.Workflow = workflowWithNilSettings{}
		err := StartAPIServerMode(context.Background(), dr)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "the API server configuration is missing")
	})
}

func TestAPIServerHandler(t *testing.T) {
	dr, _ := setupTestAPIServer(t)
	semaphore := make(chan struct{}, 1)

	t.Run("InvalidRoute", func(t *testing.T) {
		handler := APIServerHandler(context.Background(), nil, dr, semaphore)
		assert.NotNil(t, handler)

		// Simulate an HTTP request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)
		handler(c)

		// Verify the response
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Len(t, resp.Errors, 1)
		assert.Equal(t, http.StatusInternalServerError, resp.Errors[0].Code)
		assert.Equal(t, "Invalid route configuration", resp.Errors[0].Message)
	})

	t.Run("ValidRoute", func(t *testing.T) {
		route := &apiserver.APIServerRoutes{
			Path:    "/test",
			Methods: []string{"GET"},
		}
		handler := APIServerHandler(context.Background(), route, dr, semaphore)
		assert.NotNil(t, handler)
	})
}

// mockResolver implements the necessary methods for testing processWorkflow
type mockResolver struct {
	*resolver.DependencyResolver
	prepareWorkflowDirFn           func() error
	prepareImportFilesFn           func() error
	handleRunActionFn              func() (bool, error)
	evalPklFormattedResponseFileFn func() (string, error)
}

func (m *mockResolver) PrepareWorkflowDir() error {
	return m.prepareWorkflowDirFn()
}

func (m *mockResolver) PrepareImportFiles() error {
	return m.prepareImportFilesFn()
}

func (m *mockResolver) HandleRunAction() (bool, error) {
	return m.handleRunActionFn()
}

func (m *mockResolver) EvalPklFormattedResponseFile() (string, error) {
	return m.evalPklFormattedResponseFileFn()
}

// workflowWithNilSettings is a mock Workflow with GetSettings() and GetAgentIcon() returning nil
type workflowWithNilSettings struct{}

func (w workflowWithNilSettings) GetSettings() *project.Settings { return nil }

func (w workflowWithNilSettings) GetTargetActionID() string { return "test-action" }

func (w workflowWithNilSettings) GetVersion() string { return "" }

func (w workflowWithNilSettings) GetAgentIcon() *string { return nil }

func (w workflowWithNilSettings) GetAuthors() *[]string { return nil }

func (w workflowWithNilSettings) GetDescription() string { return "" }

func (w workflowWithNilSettings) GetDocumentation() *string { return nil }

func (w workflowWithNilSettings) GetHeroImage() *string { return nil }

func (w workflowWithNilSettings) GetName() string { return "" }

func (w workflowWithNilSettings) GetRepository() *string { return nil }

func (w workflowWithNilSettings) GetWebsite() *string { return nil }

func (w workflowWithNilSettings) GetWorkflows() []string { return nil }

func TestProcessWorkflow(t *testing.T) {
	// Create a test filesystem
	fs := afero.NewMemMapFs()

	// Create necessary directories
	dirs := []string{
		"/workflow",
		"/project",
		"/action",
		"/files",
		"/data",
	}
	for _, dir := range dirs {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create a test environment
	env := &environment.Environment{
		Root: "/",
		Home: "/home",
		Pwd:  "/workflow",
	}

	// Create a context
	ctx := context.Background()

	t.Run("HandleRunActionError", func(t *testing.T) {
		// Initialize DBs
		memoryDB, _ := sql.Open("sqlite3", ":memory:")
		sessionDB, _ := sql.Open("sqlite3", ":memory:")
		toolDB, _ := sql.Open("sqlite3", ":memory:")
		itemDB, _ := sql.Open("sqlite3", ":memory:")

		// Create test directories
		projectDir := "/project"
		workflowDir := "/workflow"
		actionDir := "/action"
		llmDir := filepath.Join(actionDir, "llm")
		clientDir := filepath.Join(actionDir, "client")
		execDir := filepath.Join(actionDir, "exec")
		pythonDir := filepath.Join(actionDir, "python")
		dataDir := filepath.Join(actionDir, "data")

		fs.MkdirAll(projectDir, 0o755)
		fs.MkdirAll(workflowDir, 0o755)
		fs.MkdirAll(actionDir, 0o755)
		fs.MkdirAll(llmDir, 0o755)
		fs.MkdirAll(clientDir, 0o755)
		fs.MkdirAll(execDir, 0o755)
		fs.MkdirAll(pythonDir, 0o755)
		fs.MkdirAll(dataDir, 0o755)

		// Create request file
		requestPklFile := filepath.Join(actionDir, "request.pkl")
		fs.Create(requestPklFile)

		mock := &resolver.DependencyResolver{
			Logger:               logging.NewTestLogger(),
			Fs:                   fs,
			Environment:          env,
			Context:              ctx,
			RequestPklFile:       requestPklFile,
			ResponseTargetFile:   "/response.json",
			ActionDir:            actionDir,
			ProjectDir:           projectDir,
			WorkflowDir:          workflowDir,
			FilesDir:             "/files",
			DataDir:              dataDir,
			MemoryReader:         &memory.PklResourceReader{DB: memoryDB},
			SessionReader:        &session.PklResourceReader{DB: sessionDB},
			ToolReader:           &tool.PklResourceReader{DB: toolDB},
			ItemReader:           &item.PklResourceReader{DB: itemDB},
			Workflow:             &workflowWithNilSettings{},
			FileRunCounter:       make(map[string]int),
			SessionDBPath:        "/session.db",
			ItemDBPath:           "/item.db",
			MemoryDBPath:         "/memory.db",
			ToolDBPath:           "/tool.db",
			RequestID:            "test-request",
			Resources:            []resolver.ResourceNodeEntry{{ActionID: "test-action", File: "/test.pkl"}},
			APIServerMode:        true,
			ResourceDependencies: map[string][]string{"test-action": {}},
			VisitedPaths:         make(map[string]bool),
			DBs:                  []*sql.DB{memoryDB, sessionDB, toolDB, itemDB},
		}

		mock.PrependDynamicImportsFn = func(string) error { return nil }
		mock.AddPlaceholderImportsFn = func(string) error { return nil }
		mock.LoadResourceEntriesFn = func() error { return nil }
		mock.BuildDependencyStackFn = func(string, map[string]bool) []string { return []string{"test-action"} }
		mock.LoadResourceFn = func(context.Context, string, resolver.ResourceType) (interface{}, error) {
			items := []string{}
			return &resource.Resource{Items: &items, Run: nil}, nil
		}
		mock.ProcessRunBlockFn = func(resolver.ResourceNodeEntry, *resource.Resource, string, bool) (bool, error) {
			return false, fmt.Errorf("failed to handle run action")
		}
		mock.ClearItemDBFn = func() error { return nil }
		err := processWorkflow(ctx, mock)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to handle run action")
	})
}

func TestSetupRoutes(t *testing.T) {
	// Create a test filesystem
	fs := afero.NewMemMapFs()

	// Create necessary directories
	dirs := []string{
		"/workflow",
		"/project",
		"/action",
		"/files",
		"/data",
	}
	for _, dir := range dirs {
		if err := fs.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create a test environment
	env := &environment.Environment{
		Root: "/",
		Home: "/home",
		Pwd:  "/workflow",
	}

	// Create base resolver
	baseDr := &resolver.DependencyResolver{
		Logger:             logging.NewTestLogger(),
		Fs:                 fs,
		Environment:        env,
		RequestPklFile:     "/request.pkl",
		ResponseTargetFile: "/response.json",
		ActionDir:          "/action",
		ProjectDir:         "/project",
		FilesDir:           "/files",
		DataDir:            "/data",
	}

	// Create a test CORS configuration
	corsConfig := &apiserver.CORS{
		EnableCORS:       true,
		AllowOrigins:     &[]string{"http://localhost:3000"},
		AllowMethods:     &[]string{"GET", "POST"},
		AllowHeaders:     &[]string{"Content-Type"},
		ExposeHeaders:    &[]string{"X-Custom-Header"},
		AllowCredentials: true,
		MaxAge:           &pkl.Duration{Value: 3600, Unit: pkl.Second},
	}

	// Create test routes
	routes := []*apiserver.APIServerRoutes{
		{
			Path:    "/test",
			Methods: []string{http.MethodGet, http.MethodPost},
		},
		{
			Path:    "/test2",
			Methods: []string{http.MethodPut, http.MethodDelete},
		},
	}

	// Create a semaphore channel
	semaphore := make(chan struct{}, 1)

	t.Run("ValidRoutes", func(t *testing.T) {
		router := gin.New()
		ctx := context.Background()
		setupRoutes(router, ctx, corsConfig, []string{"127.0.0.1"}, routes, baseDr, semaphore)

		// Test GET request
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code) // Expected error due to missing resolver setup

		// Test POST request
		w = httptest.NewRecorder()
		req, _ = http.NewRequest(http.MethodPost, "/test", nil)
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusInternalServerError, w.Code) // Expected error due to missing resolver setup
	})

	t.Run("InvalidRoute", func(t *testing.T) {
		router := gin.New()
		ctx := context.Background()
		invalidRoutes := []*apiserver.APIServerRoutes{
			nil,
			{Path: ""},
		}
		setupRoutes(router, ctx, corsConfig, []string{"127.0.0.1"}, invalidRoutes, baseDr, semaphore)
		// No assertions needed as the function should log errors and continue
	})

	t.Run("CORSDisabled", func(t *testing.T) {
		router := gin.New()
		ctx := context.Background()
		disabledCORS := &apiserver.CORS{
			EnableCORS: false,
		}
		setupRoutes(router, ctx, disabledCORS, []string{"127.0.0.1"}, routes, baseDr, semaphore)
		// No assertions needed as the function should skip CORS setup
	})

	t.Run("NoTrustedProxies", func(t *testing.T) {
		router := gin.New()
		ctx := context.Background()
		setupRoutes(router, ctx, corsConfig, nil, routes, baseDr, semaphore)
		// No assertions needed as the function should skip proxy setup
	})

	t.Run("UnsupportedMethod", func(t *testing.T) {
		router := gin.New()
		ctx := context.Background()
		unsupportedRoutes := []*apiserver.APIServerRoutes{
			{
				Path:    "/test3",
				Methods: []string{"UNSUPPORTED"},
			},
		}
		setupRoutes(router, ctx, corsConfig, []string{"127.0.0.1"}, unsupportedRoutes, baseDr, semaphore)
		// No assertions needed as the function should log a warning and continue
	})
}
