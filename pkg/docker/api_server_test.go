package docker

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	pkg "github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	"github.com/kdeps/kdeps/pkg/utils"
	apiserver "github.com/kdeps/schema/gen/api_server"
	pklProject "github.com/kdeps/schema/gen/project"
	"github.com/kdeps/schema/gen/resource"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMethodExtra2(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	methodStr, err := validateMethod(req, []string{http.MethodGet, http.MethodPost})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if methodStr != `Method = "POST"` {
		t.Fatalf("unexpected method string: %s", methodStr)
	}

	// invalid method
	badReq := httptest.NewRequest("DELETE", "/", nil)
	if _, err := validateMethod(badReq, []string{"GET"}); err == nil {
		t.Fatalf("expected error for disallowed method")
	}
}

func TestCleanOldFilesExtra2(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// create dummy response file
	path := filepath.Join(t.TempDir(), "resp.json")
	afero.WriteFile(fs, path, []byte("dummy"), 0o644)

	dr := &resolver.DependencyResolver{
		Fs:                 fs,
		Logger:             logger,
		ResponseTargetFile: path,
	}

	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, _ := afero.Exists(fs, path)
	if exists {
		t.Fatalf("file should have been removed")
	}
}

func TestDecodeResponseContentExtra2(t *testing.T) {
	logger := logging.NewTestLogger()

	// prepare APIResponse with base64 encoded JSON data
	apiResp := APIResponse{
		Success: true,
		Response: ResponseData{
			Data: []string{base64.StdEncoding.EncodeToString([]byte(`{"foo":"bar"}`))},
		},
		Meta: ResponseMeta{
			Headers: map[string]string{"X-Test": "yes"},
		},
	}
	encoded, _ := json.Marshal(apiResp)

	decResp, err := decodeResponseContent(encoded, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(decResp.Response.Data) != 1 || decResp.Response.Data[0] != "{\n  \"foo\": \"bar\"\n}" {
		t.Fatalf("unexpected decoded data: %+v", decResp.Response.Data)
	}
}

func TestFormatResponseJSONExtra2(t *testing.T) {
	// Response with data as JSON string
	raw := []byte(`{"response":{"data":["{\"a\":1}"]}}`)
	pretty := formatResponseJSON(raw)

	// Should be pretty printed and data element should be object not string
	if !bytes.Contains(pretty, []byte("\"a\": 1")) {
		t.Fatalf("pretty output missing expected content: %s", string(pretty))
	}
}

func TestFormatResponseJSONExtra(t *testing.T) {
	// Prepare response with data that is itself JSON string
	inner := map[string]any{"foo": "bar"}
	innerBytes, _ := json.Marshal(inner)
	resp := map[string]any{
		"response": map[string]any{
			"data": []string{string(innerBytes)},
		},
	}
	raw, _ := json.Marshal(resp)
	pretty := formatResponseJSON(raw)

	// It should now be pretty-printed and contain nested object without quotes
	require.Contains(t, string(pretty), "\"foo\": \"bar\"")
}

func TestCleanOldFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &resolver.DependencyResolver{Fs: fs, Logger: logging.NewTestLogger(), ResponseTargetFile: "old.json"}

	// Case where file exists
	require.NoError(t, afero.WriteFile(fs, dr.ResponseTargetFile, []byte("x"), 0o644))
	require.NoError(t, cleanOldFiles(dr))
	exists, _ := afero.Exists(fs, dr.ResponseTargetFile)
	require.False(t, exists)

	// Case where file does not exist should be no-op
	require.NoError(t, cleanOldFiles(dr))
}

func TestDecodeResponseContentExtra(t *testing.T) {
	// Prepare APIResponse JSON with Base64 encoded data
	dataJSON := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(dataJSON))
	respStruct := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{encoded}},
	}
	raw, _ := json.Marshal(respStruct)

	logger := logging.NewTestLogger()
	out, err := decodeResponseContent(raw, logger)
	require.NoError(t, err)
	require.Len(t, out.Response.Data, 1)
	require.JSONEq(t, dataJSON, out.Response.Data[0])
}

func TestFormatResponseJSONFormatTest(t *testing.T) {
	// Input where first element is JSON string and second is plain string.
	in := []byte(`{"response":{"data":["{\"x\":1}","plain"]}}`)
	out := formatResponseJSON(in)
	// The output should still be valid JSON and contain "x": 1 without escaped quotes.
	if !json.Valid(out) {
		t.Fatalf("output not valid JSON: %s", string(out))
	}
	if !bytes.Contains(out, []byte("\"x\": 1")) {
		t.Fatalf("expected object conversion in data array, got %s", string(out))
	}
}

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
		assert.Equal(t, "Method = \"GET\"", method)
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
		assert.Equal(t, "Method = \"GET\"", method)
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
	err = afero.WriteFile(fs, responseFile, []byte("test response"), 0o644)
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

func (workflowWithNilSettings) GetAgentID() string        { return "test-agent" }
func (workflowWithNilSettings) GetVersion() string        { return "1.0.0" }
func (workflowWithNilSettings) GetDescription() *string   { return nil }
func (workflowWithNilSettings) GetWebsite() *string       { return nil }
func (workflowWithNilSettings) GetAuthors() *[]string     { return nil }
func (workflowWithNilSettings) GetDocumentation() *string { return nil }
func (workflowWithNilSettings) GetRepository() *string    { return nil }
func (workflowWithNilSettings) GetHeroImage() *string     { return nil }
func (workflowWithNilSettings) GetAgentIcon() *string     { return nil }
func (workflowWithNilSettings) GetTargetActionID() string { return "" }
func (workflowWithNilSettings) GetWorkflows() []string    { return nil }
func (workflowWithNilSettings) GetSettings() *pklProject.Settings {
	return nil
}

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
		EnableCORS:       pkg.BoolPtr(true),
		AllowOrigins:     &[]string{"http://localhost:3000"},
		AllowMethods:     &[]string{"GET", "POST"},
		AllowHeaders:     &[]string{"Content-Type"},
		ExposeHeaders:    &[]string{"X-Custom-Header"},
		AllowCredentials: pkg.BoolPtr(true),
		MaxAge:           &pkl.Duration{Value: 3600, Unit: pkl.Second},
	}

	// Create test routes
	routes := []*apiserver.APIServerRoutes{
		{
			Path:    "/test",
			Methods: []string{http.MethodGet},
		},
		{
			Path:    "/test-post",
			Methods: []string{http.MethodPost},
		},
		{
			Path:    "/test2",
			Methods: []string{http.MethodPut},
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
		req, _ = http.NewRequest(http.MethodPost, "/test-post", nil)
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
			EnableCORS: pkg.BoolPtr(false),
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

// Ensure schema version gets referenced at least once in this test file.
func TestSchemaVersionReference(t *testing.T) {
	if v := schema.SchemaVersion(context.Background()); v == "" {
		t.Fatalf("SchemaVersion returned empty string")
	}
}

func TestValidateMethodUtilsExtra(t *testing.T) {
	_ = schema.SchemaVersion(nil)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	got, err := validateMethod(req, []string{http.MethodGet, http.MethodPost})
	if err != nil || got != `Method = "GET"` {
		t.Fatalf("expected valid GET, got %q err %v", got, err)
	}

	reqEmpty, _ := http.NewRequest("", "http://example.com", nil)
	got2, err2 := validateMethod(reqEmpty, []string{http.MethodGet})
	if err2 != nil || got2 != `Method = "GET"` {
		t.Fatalf("default method failed: %q err %v", got2, err2)
	}

	reqBad, _ := http.NewRequest(http.MethodDelete, "http://example.com", nil)
	if _, err := validateMethod(reqBad, []string{http.MethodGet}); err == nil {
		t.Fatalf("expected error for disallowed method")
	}
}

func TestDecodeResponseContentUtilsExtra(t *testing.T) {
	_ = schema.SchemaVersion(nil)

	helloB64 := base64.StdEncoding.EncodeToString([]byte("hello"))
	invalidB64 := "@@invalid@@"
	raw := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{helloB64, invalidB64}},
		Meta:     ResponseMeta{RequestID: "abc"},
	}
	data, _ := json.Marshal(raw)
	logger := logging.NewTestLogger()
	decoded, err := decodeResponseContent(data, logger)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if decoded.Response.Data[0] != "hello" {
		t.Fatalf("expected \"hello\", got %q", decoded.Response.Data[0])
	}
	if decoded.Response.Data[1] != invalidB64 {
		t.Fatalf("invalid data should remain unchanged")
	}
}

func TestDecodeResponseContentFormattingUtilsExtra(t *testing.T) {
	jsonPayload := `{"foo":"bar"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(jsonPayload))

	resp := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{encoded}},
		Meta:     ResponseMeta{Headers: map[string]string{"X-Test": "1"}},
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	logger := logging.NewTestLogger()
	decoded, err := decodeResponseContent(raw, logger)
	if err != nil {
		t.Fatalf("decodeResponseContent error: %v", err)
	}

	if len(decoded.Response.Data) != 1 {
		t.Fatalf("expected 1 data entry, got %d", len(decoded.Response.Data))
	}

	first := decoded.Response.Data[0]
	if !bytes.Contains([]byte(first), []byte("foo")) || !bytes.Contains([]byte(first), []byte("bar")) {
		t.Fatalf("decoded data does not contain expected JSON: %s", first)
	}

	if first == encoded {
		t.Fatalf("base64 string not decoded")
	}
}

func TestValidateMethodMore(t *testing.T) {
	// allowed only GET & POST
	allowed := []string{http.MethodGet, http.MethodPost}

	req, _ := http.NewRequest(http.MethodPost, "/", nil)
	out, err := validateMethod(req, allowed)
	assert.NoError(t, err)
	assert.Equal(t, `Method = "POST"`, out)

	// default empty method becomes GET and passes
	req2, _ := http.NewRequest("", "/", nil)
	out, err = validateMethod(req2, allowed)
	assert.NoError(t, err)
	assert.Equal(t, `Method = "GET"`, out)

	// invalid method
	req3, _ := http.NewRequest(http.MethodPut, "/", nil)
	out, err = validateMethod(req3, allowed)
	assert.Error(t, err)
	assert.Empty(t, out)
}

func TestCleanOldFilesMore(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// create dummy response file
	respPath := filepath.Join(t.TempDir(), "response.json")
	_ = afero.WriteFile(fs, respPath, []byte("old"), 0o644)

	dr := &resolver.DependencyResolver{
		Fs:                 fs,
		Logger:             logger,
		ResponseTargetFile: respPath,
	}

	// should remove existing file
	err := cleanOldFiles(dr)
	assert.NoError(t, err)
	exist, _ := afero.Exists(fs, respPath)
	assert.False(t, exist)

	// second call with file absent should still succeed
	err = cleanOldFiles(dr)
	assert.NoError(t, err)
}

// TestCleanOldFiles ensures that the helper deletes the ResponseTargetFile when it exists
// and returns nil when the file is absent. Both branches of the conditional are exercised.
func TestCleanOldFilesMemFS(t *testing.T) {
	mem := afero.NewMemMapFs()
	respPath := filepath.Join(t.TempDir(), "response.json")
	dr := &resolver.DependencyResolver{
		Fs:                 mem,
		ResponseTargetFile: respPath,
		Logger:             logging.NewTestLogger(),
	}

	// Branch 1: File exists and should be removed without error.
	if err := afero.WriteFile(mem, dr.ResponseTargetFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to seed response file: %v", err)
	}
	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error for existing file: %v", err)
	}
	if exists, _ := afero.Exists(mem, dr.ResponseTargetFile); exists {
		t.Fatalf("expected response file to be removed")
	}

	// Branch 2: File does not exist – function should still return nil (no error).
	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error when file absent: %v", err)
	}
}

// TestCleanOldFilesRemoveError exercises the branch where RemoveAll returns an
// error. It uses a read-only filesystem wrapper so the delete fails without
// depending on OS-specific permissions.
func TestCleanOldFilesRemoveError(t *testing.T) {
	mem := afero.NewMemMapFs()
	target := filepath.Join(t.TempDir(), "response.json")
	if err := afero.WriteFile(mem, target, []byte("data"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	dr := &resolver.DependencyResolver{
		Fs:                 afero.NewReadOnlyFs(mem), // makes RemoveAll fail
		ResponseTargetFile: target,
		Logger:             logging.NewTestLogger(),
	}

	if err := cleanOldFiles(dr); err == nil {
		t.Fatalf("expected error from RemoveAll, got nil")
	}
}

func TestFormatResponseJSON_NestedData(t *testing.T) {
	// Build a response where data[0] is a JSON string
	payload := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{`{"foo":123}`}},
		Meta:     ResponseMeta{RequestID: "id"},
	}
	raw, _ := json.Marshal(payload)
	pretty := formatResponseJSON(raw)

	// The nested JSON should have been parsed → data[0] becomes an object not string
	var out map[string]interface{}
	if err := json.Unmarshal(pretty, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	resp, ok := out["response"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing response field")
	}
	dataArr, ok := resp["data"].([]interface{})
	if !ok || len(dataArr) != 1 {
		t.Fatalf("unexpected data field: %v", resp["data"])
	}
	first, ok := dataArr[0].(map[string]interface{})
	if !ok {
		t.Fatalf("data[0] still a string after formatting")
	}
	if val, ok := first["foo"].(float64); !ok || val != 123 {
		t.Fatalf("nested JSON not preserved: %v", first)
	}
}

func TestCleanOldFilesUnique(t *testing.T) {
	fs := afero.NewOsFs()
	tmpDir, _ := afero.TempDir(fs, "", "clean")
	target := tmpDir + "/resp.json"
	_ = afero.WriteFile(fs, target, []byte("data"), 0o644)

	dr := &resolver.DependencyResolver{Fs: fs, Logger: logging.NewTestLogger(), ResponseTargetFile: target}
	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles error: %v", err)
	}
	if exists, _ := afero.Exists(fs, target); exists {
		t.Fatalf("file still exists after cleanOldFiles")
	}
}

// TestFormatResponseJSONInlineData ensures that when the "data" field contains
// string elements that are themselves valid JSON objects, formatResponseJSON
// converts those elements into embedded objects within the final JSON.
func TestFormatResponseJSONInlineData(t *testing.T) {
	raw := []byte(`{"response": {"data": ["{\"foo\": \"bar\"}", "plain text"]}}`)

	pretty := formatResponseJSON(raw)

	if !bytes.Contains(pretty, []byte("\"foo\": \"bar\"")) {
		t.Fatalf("expected pretty JSON to contain inlined object, got %s", string(pretty))
	}
}

func TestValidateMethodSimple(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	methodStr, err := validateMethod(req, []string{"GET", "POST"})
	if err != nil {
		t.Fatalf("validateMethod unexpected error: %v", err)
	}
	if methodStr != `Method = "POST"` {
		t.Fatalf("unexpected method string: %s", methodStr)
	}

	// Unsupported method should error
	req.Method = "DELETE"
	if _, err := validateMethod(req, []string{"GET", "POST"}); err == nil {
		t.Fatalf("expected error for unsupported method")
	}
}

func TestCleanOldFilesMem(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Prepare dependency resolver stub with in-mem fs
	dr := &resolver.DependencyResolver{Fs: fs, Logger: logger, ResponseTargetFile: filepath.Join(t.TempDir(), "old_resp.txt")}
	// Create dummy file
	afero.WriteFile(fs, dr.ResponseTargetFile, []byte("old"), 0o666)

	if err := cleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error: %v", err)
	}
	if exists, _ := afero.Exists(fs, dr.ResponseTargetFile); exists {
		t.Fatalf("file still exists after cleanOldFiles")
	}
}

func TestDecodeAndFormatResponseSimple(t *testing.T) {
	logger := logging.NewTestLogger()

	// Build sample APIResponse JSON with base64 encoded data
	sample := APIResponse{
		Success:  true,
		Response: ResponseData{Data: []string{utils.EncodeBase64String(`{"foo":"bar"}`)}},
		Meta:     ResponseMeta{RequestID: "abc123"},
	}
	raw, _ := json.Marshal(sample)

	decoded, err := decodeResponseContent(raw, logger)
	if err != nil {
		t.Fatalf("decodeResponseContent error: %v", err)
	}
	if len(decoded.Response.Data) != 1 || decoded.Response.Data[0] != "{\n  \"foo\": \"bar\"\n}" {
		t.Fatalf("decodeResponseContent did not prettify JSON: %v", decoded.Response.Data)
	}

	// Marshal decoded struct then format
	marshaled, _ := json.Marshal(decoded)
	formatted := formatResponseJSON(marshaled)
	if !bytes.Contains(formatted, []byte("foo")) {
		t.Fatalf("formatResponseJSON missing field")
	}
}

func TestDecodeResponseContent_Success(t *testing.T) {
	logger := logging.NewTestLogger()

	// Prepare an APIResponse JSON with base64-encoded JSON payload in data.
	inner := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(inner))

	raw := APIResponse{
		Success: true,
		Response: ResponseData{
			Data: []string{encoded},
		},
		Meta: ResponseMeta{
			RequestID: "abc",
		},
	}

	rawBytes, err := json.Marshal(raw)
	assert.NoError(t, err)

	decoded, err := decodeResponseContent(rawBytes, logger)
	assert.NoError(t, err)
	assert.Equal(t, "abc", decoded.Meta.RequestID)
	assert.Contains(t, decoded.Response.Data[0], "\"hello\": \"world\"")
}

func TestDecodeResponseContent_InvalidJSON(t *testing.T) {
	logger := logging.NewTestLogger()
	_, err := decodeResponseContent([]byte(`not-json`), logger)
	assert.Error(t, err)
}

func TestFormatResponseJSONPretty(t *testing.T) {
	// Create a response that will be decodable by formatResponseJSON
	inner := map[string]string{"foo": "bar"}
	innerBytes, _ := json.Marshal(inner)

	resp := map[string]interface{}{
		"response": map[string]interface{}{
			"data": []interface{}{string(innerBytes)},
		},
	}
	bytesIn, _ := json.Marshal(resp)

	pretty := formatResponseJSON(bytesIn)

	// The formatted JSON should contain nested object without quotes around keys
	assert.Contains(t, string(pretty), "\"foo\": \"bar\"")
}
