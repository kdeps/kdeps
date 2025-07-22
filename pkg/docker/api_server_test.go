package docker_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	pkg "github.com/kdeps/kdeps/pkg"
	"github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	apiserver "github.com/kdeps/schema/gen/api_server"
	pklProject "github.com/kdeps/schema/gen/project"
	pklRes "github.com/kdeps/schema/gen/resource"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMethod(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		allowedMethods []string
		expectedMethod string
		expectError    bool
	}{
		{
			name:           "valid method",
			method:         "GET",
			allowedMethods: []string{"GET", "POST"},
			expectedMethod: `Method = "GET"`,
			expectError:    false,
		},
		{
			name:           "case insensitive",
			method:         "post",
			allowedMethods: []string{"GET", "POST"},
			expectedMethod: `Method = "POST"`,
			expectError:    false,
		},
		{
			name:           "invalid method",
			method:         "PUT",
			allowedMethods: []string{"GET", "POST"},
			expectedMethod: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "/test", nil)
			method, err := docker.ValidateMethod(req, tt.allowedMethods)

			if tt.expectError {
				require.Error(t, err)
				assert.Empty(t, method)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedMethod, method)
			}
		})
	}
}

func TestCleanOldFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Create test directory structure in MemMapFs
	testDir := "/test-action"
	require.NoError(t, fs.MkdirAll(testDir, 0o755))
	require.NoError(t, fs.MkdirAll(filepath.Join(testDir, "files"), 0o755))

	// Create a test file that will be the ResponseTargetFile
	targetFile := filepath.Join(testDir, "response.json")
	require.NoError(t, afero.WriteFile(fs, targetFile, []byte("test content"), 0o644))

	// Create a mock dependency resolver with ResponseTargetFile set
	dr := &resolver.DependencyResolver{
		Fs:                 fs,
		ActionDir:          testDir,
		Logger:             logger,
		ResponseTargetFile: targetFile,
	}

	// Verify the target file exists before cleaning
	exists, err := afero.Exists(fs, targetFile)
	require.NoError(t, err)
	assert.True(t, exists, "Target file should exist before cleaning")

	// Test cleanOldFiles
	err = docker.CleanOldFiles(dr)
	require.NoError(t, err)

	// Verify the target file was cleaned
	exists, err = afero.Exists(fs, targetFile)
	require.NoError(t, err)
	assert.False(t, exists, "Target file should be cleaned")

	// Test with no ResponseTargetFile (should do nothing)
	dr.ResponseTargetFile = ""
	err = docker.CleanOldFiles(dr)
	assert.NoError(t, err)
}

func TestAPIResponse(t *testing.T) {
	// Test APIResponse struct
	response := docker.APIResponse{
		Success: true,
		Response: docker.ResponseData{
			Data: []string{"test1", "test2"},
		},
		Meta: docker.ResponseMeta{
			RequestID: "test-123",
		},
	}

	assert.True(t, response.Success)
	assert.Len(t, response.Response.Data, 2)
	assert.Equal(t, "test-123", response.Meta.RequestID)
}

func TestResponseData(t *testing.T) {
	// Test ResponseData struct
	data := docker.ResponseData{
		Data: []string{"item1", "item2", "item3"},
	}

	assert.Len(t, data.Data, 3)
	assert.Equal(t, "item1", data.Data[0])
}

func TestResponseMeta(t *testing.T) {
	// Test ResponseMeta struct
	meta := docker.ResponseMeta{
		RequestID: "req-456",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Properties: map[string]string{
			"version": "1.0.0",
		},
	}

	assert.Equal(t, "req-456", meta.RequestID)
	assert.Equal(t, "application/json", meta.Headers["Content-Type"])
	assert.Equal(t, "1.0.0", meta.Properties["version"])
}

func TestDecodeResponseContent(t *testing.T) {
	logger := logging.NewTestLogger()

	// Test valid JSON response
	validJSON := `{
		"success": true,
		"response": {
			"data": ["test1", "test2"]
		},
		"meta": {
			"requestID": "test-123"
		}
	}`

	response, err := docker.DecodeResponseContent([]byte(validJSON), logger)
	require.NoError(t, err)
	assert.True(t, response.Success)
	assert.Len(t, response.Response.Data, 2)
	assert.Equal(t, "test-123", response.Meta.RequestID)

	// Test invalid JSON
	invalidJSON := `{invalid json}`
	_, err = docker.DecodeResponseContent([]byte(invalidJSON), logger)
	require.Error(t, err)
}

func TestFormatResponseJSON(t *testing.T) {
	// Test formatting response
	response := docker.APIResponse{
		Success: true,
		Response: docker.ResponseData{
			Data: []string{"formatted", "response"},
		},
		Meta: docker.ResponseMeta{
			RequestID: "format-test",
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(response)
	require.NoError(t, err)

	// Format the JSON
	formatted := docker.FormatResponseJSON(jsonData)
	assert.NotEmpty(t, formatted)

	// Verify it's valid JSON
	var parsedResponse docker.APIResponse
	err = json.Unmarshal(formatted, &parsedResponse)
	require.NoError(t, err)
	assert.True(t, parsedResponse.Success)
}

func TestFormatResponseJSONWithInvalidInput(t *testing.T) {
	// Test with invalid input
	invalidInput := []byte("invalid json")
	formatted := docker.FormatResponseJSON(invalidInput)

	// Should return the original input if it's not valid JSON
	assert.Equal(t, invalidInput, formatted)
}

func TestValidateMethodExtra2(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	methodStr, err := docker.ValidateMethod(req, []string{http.MethodGet, http.MethodPost})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if methodStr != `Method = "POST"` {
		t.Fatalf("unexpected method string: %s", methodStr)
	}

	// invalid method
	badReq := httptest.NewRequest(http.MethodDelete, "/", nil)
	if _, err := docker.ValidateMethod(badReq, []string{"GET"}); err == nil {
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

	if err := docker.CleanOldFiles(dr); err != nil {
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
	apiResp := docker.APIResponse{
		Success: true,
		Response: docker.ResponseData{
			Data: []string{`{"foo":"bar"}`},
		},
		Meta: docker.ResponseMeta{
			Headers: map[string]string{"X-Test": "yes"},
		},
	}
	encoded, _ := json.Marshal(apiResp)

	decResp, err := docker.DecodeResponseContent(encoded, logger)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(decResp.Response.Data) != 1 || decResp.Response.Data[0] != `{"foo":"bar"}` {
		t.Fatalf("unexpected decoded data: %+v", decResp.Response.Data)
	}
}

func TestFormatResponseJSONExtra2(t *testing.T) {
	// Response with data as JSON string
	raw := []byte(`{"response":{"data":["{\"a\":1}"]}}`)
	pretty := docker.FormatResponseJSON(raw)

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
	pretty := docker.FormatResponseJSON(raw)

	// It should now be pretty-printed and contain nested object without quotes
	require.Contains(t, string(pretty), "\"foo\": \"bar\"")
}

func TestCleanOldFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &resolver.DependencyResolver{Fs: fs, Logger: logging.NewTestLogger(), ResponseTargetFile: "old.json"}

	// Case where file exists
	require.NoError(t, afero.WriteFile(fs, dr.ResponseTargetFile, []byte("x"), 0o644))
	require.NoError(t, docker.CleanOldFiles(dr))
	exists, _ := afero.Exists(fs, dr.ResponseTargetFile)
	require.False(t, exists)

	// Case where file does not exist should be no-op
	require.NoError(t, docker.CleanOldFiles(dr))
}

func TestDecodeResponseContentExtra(t *testing.T) {
	// Prepare APIResponse JSON with Base64 encoded data
	dataJSON := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(dataJSON))
	respStruct := docker.APIResponse{
		Success:  true,
		Response: docker.ResponseData{Data: []string{encoded}},
	}
	raw, _ := json.Marshal(respStruct)

	logger := logging.NewTestLogger()
	out, err := docker.DecodeResponseContent(raw, logger)
	require.NoError(t, err)
	require.Len(t, out.Response.Data, 1)
	require.JSONEq(t, dataJSON, out.Response.Data[0])
}

func TestFormatResponseJSONFormatTest(t *testing.T) {
	// Input where first element is JSON string and second is plain string.
	in := []byte(`{"response":{"data":["{\"x\":1}","plain"]}}`)
	out := docker.FormatResponseJSON(in)
	// The output should still be valid JSON and contain "x": 1 without escaped quotes.
	if !json.Valid(out) {
		t.Fatalf("output not valid JSON: %s", string(out))
	}
	if !bytes.Contains(out, []byte("\"x\": 1")) {
		t.Fatalf("expected object conversion in data array, got %s", string(out))
	}
}

func setupTestAPIServer(_ *testing.T) (*resolver.DependencyResolver, *logging.Logger) {
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
		c.Request = httptest.NewRequest(http.MethodPost, "/", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		fileMap := make(map[string]struct{ Filename, Filetype string })
		err = docker.HandleMultipartForm(c, dr, fileMap)
		require.NoError(t, err)
		assert.Len(t, fileMap, 1)
	})

	t.Run("InvalidContentType", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewBuffer([]byte("test")))
		c.Request.Header.Set("Content-Type", "text/plain")

		fileMap := make(map[string]struct{ Filename, Filetype string })
		err := docker.HandleMultipartForm(c, dr, fileMap)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Unable to parse multipart form")
	})

	t.Run("NoFileField", func(t *testing.T) {
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		writer.WriteField("other", "value")
		writer.Close()

		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodPost, "/", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		fileMap := make(map[string]struct{ Filename, Filetype string })
		err := docker.HandleMultipartForm(c, dr, fileMap)
		require.Error(t, err)
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
		c.Request = httptest.NewRequest(http.MethodPost, "/", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())
		_, fileHeader, err := c.Request.FormFile("file")
		require.NoError(t, err)

		err = docker.ProcessFile(fileHeader, dr, fileMap)
		require.NoError(t, err)
		assert.Len(t, fileMap, 1)
	})
}

func TestStartAPIServerMode(t *testing.T) {
	dr, _ := setupTestAPIServer(t)

	t.Run("MissingConfig", func(t *testing.T) {
		dr.Logger = logging.NewTestLogger()
		// Provide a mock Workflow with GetSettings() returning nil
		dr.Workflow = workflowWithNilSettings{}
		err := docker.StartAPIServerMode(context.Background(), dr)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "the API server configuration is missing")
	})
}

func TestAPIServerHandler(t *testing.T) {
	dr, _ := setupTestAPIServer(t)
	semaphore := make(chan struct{}, 1)

	t.Run("InvalidRoute", func(t *testing.T) {
		handler := docker.APIServerHandler(context.Background(), nil, dr, semaphore)
		assert.NotNil(t, handler)

		// Simulate an HTTP request
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/test", nil)
		handler(c)

		// Verify the response
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		var resp docker.APIResponse
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
		handler := docker.APIServerHandler(context.Background(), route, dr, semaphore)
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
	// PrepareWorkflowDir functionality removed - using project directory directly
	return nil
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

		// PrependDynamicImportsFn and AddPlaceholderImportsFn removed - deprecated functionality
		mock.LoadResourceEntriesFn = func() error { return nil }
		mock.BuildDependencyStackFn = func(string, map[string]bool) []string { return []string{"test-action"} }
		mock.LoadResourceFn = func(context.Context, string, resolver.ResourceType) (interface{}, error) {
			items := []string{}
			return &pklRes.ResourceImpl{Items: &items, Run: nil}, nil
		}
		mock.LoadResourceWithRequestContextFn = func(context.Context, string, resolver.ResourceType) (interface{}, error) {
			items := []string{}
			return &pklRes.ResourceImpl{Items: &items, Run: nil}, nil
		}
		mock.ProcessRunBlockFn = func(resolver.ResourceNodeEntry, pklRes.Resource, string, bool) (bool, error) {
			return false, errors.New("failed to handle run action")
		}
		mock.ClearItemDBFn = func() error { return nil }
		err := docker.ProcessWorkflow(ctx, mock)
		require.Error(t, err)
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
		docker.SetupRoutes(router, ctx, corsConfig, []string{"127.0.0.1"}, routes, baseDr, semaphore)

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

	t.Run("InvalidRoute", func(_ *testing.T) {
		router := gin.New()
		ctx := context.Background()
		invalidRoutes := []*apiserver.APIServerRoutes{
			nil,
			{Path: ""},
		}
		docker.SetupRoutes(router, ctx, corsConfig, []string{"127.0.0.1"}, invalidRoutes, baseDr, semaphore)
		// No assertions needed as the function should log errors and continue
	})

	t.Run("CORSDisabled", func(_ *testing.T) {
		router := gin.New()
		ctx := context.Background()
		disabledCORS := &apiserver.CORS{
			EnableCORS: pkg.BoolPtr(false),
		}
		docker.SetupRoutes(router, ctx, disabledCORS, []string{"127.0.0.1"}, routes, baseDr, semaphore)
		// No assertions needed as the function should skip CORS setup
	})

	t.Run("NoTrustedProxies", func(_ *testing.T) {
		router := gin.New()
		ctx := context.Background()
		docker.SetupRoutes(router, ctx, corsConfig, nil, routes, baseDr, semaphore)
		// No assertions needed as the function should skip proxy setup
	})

	t.Run("UnsupportedMethod", func(_ *testing.T) {
		router := gin.New()
		ctx := context.Background()
		unsupportedRoutes := []*apiserver.APIServerRoutes{
			{
				Path:    "/test3",
				Methods: []string{"UNSUPPORTED"},
			},
		}
		docker.SetupRoutes(router, ctx, corsConfig, []string{"127.0.0.1"}, unsupportedRoutes, baseDr, semaphore)
		// No assertions needed as the function should log a warning and continue
	})
}

// Ensure schema version gets referenced at least once in this test file.
func TestSchemaVersionReference(t *testing.T) {
	if v := schema.Version(context.Background()); v == "" {
		t.Fatalf("SchemaVersion returned empty string")
	}
}

func TestValidateMethodUtilsExtra(t *testing.T) {
	_ = schema.Version(nil)

	req, _ := http.NewRequest(http.MethodGet, "http://example.com", nil)
	got, err := docker.ValidateMethod(req, []string{http.MethodGet, http.MethodPost})
	if err != nil || got != `Method = "GET"` {
		t.Fatalf("expected valid GET, got %q err %v", got, err)
	}

	reqEmpty, _ := http.NewRequest("", "http://example.com", nil)
	got2, err2 := docker.ValidateMethod(reqEmpty, []string{http.MethodGet})
	if err2 != nil || got2 != `Method = "GET"` {
		t.Fatalf("default method failed: %q err %v", got2, err2)
	}

	reqBad, _ := http.NewRequest(http.MethodDelete, "http://example.com", nil)
	if _, err := docker.ValidateMethod(reqBad, []string{http.MethodGet}); err == nil {
		t.Fatalf("expected error for disallowed method")
	}
}

func TestDecodeResponseContentUtilsExtra(t *testing.T) {
	_ = schema.Version(nil)

	helloB64 := base64.StdEncoding.EncodeToString([]byte("hello"))
	invalidB64 := "@@invalid@@"
	raw := docker.APIResponse{
		Success:  true,
		Response: docker.ResponseData{Data: []string{helloB64, invalidB64}},
		Meta:     docker.ResponseMeta{RequestID: "abc"},
	}
	data, _ := json.Marshal(raw)
	logger := logging.NewTestLogger()
	decoded, err := docker.DecodeResponseContent(data, logger)
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

	resp := docker.APIResponse{
		Success:  true,
		Response: docker.ResponseData{Data: []string{encoded}},
		Meta:     docker.ResponseMeta{Headers: map[string]string{"X-Test": "1"}},
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	logger := logging.NewTestLogger()
	decoded, err := docker.DecodeResponseContent(raw, logger)
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
	out, err := docker.ValidateMethod(req, allowed)
	require.NoError(t, err)
	assert.Equal(t, `Method = "POST"`, out)

	// default empty method becomes GET and passes
	req2, _ := http.NewRequest("", "/", nil)
	out, err = docker.ValidateMethod(req2, allowed)
	require.NoError(t, err)
	assert.Equal(t, `Method = "GET"`, out)

	// invalid method
	req3, _ := http.NewRequest(http.MethodPut, "/", nil)
	out, err = docker.ValidateMethod(req3, allowed)
	require.Error(t, err)
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
	err := docker.CleanOldFiles(dr)
	require.NoError(t, err)
	exist, _ := afero.Exists(fs, respPath)
	assert.False(t, exist)

	// second call with file absent should still succeed
	err = docker.CleanOldFiles(dr)
	require.NoError(t, err)
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
	if err := docker.CleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error for existing file: %v", err)
	}
	if exists, _ := afero.Exists(mem, dr.ResponseTargetFile); exists {
		t.Fatalf("expected response file to be removed")
	}

	// Branch 2: File does not exist – function should still return nil (no error).
	if err := docker.CleanOldFiles(dr); err != nil {
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

	if err := docker.CleanOldFiles(dr); err == nil {
		t.Fatalf("expected error from RemoveAll, got nil")
	}
}

func TestFormatResponseJSON_NestedData(t *testing.T) {
	// Build a response where data[0] is a JSON string
	payload := docker.APIResponse{
		Success:  true,
		Response: docker.ResponseData{Data: []string{`{"foo":123}`}},
		Meta:     docker.ResponseMeta{RequestID: "id"},
	}
	raw, _ := json.Marshal(payload)
	pretty := docker.FormatResponseJSON(raw)

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
	if err := docker.CleanOldFiles(dr); err != nil {
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

	pretty := docker.FormatResponseJSON(raw)

	if !bytes.Contains(pretty, []byte("\"foo\": \"bar\"")) {
		t.Fatalf("expected pretty JSON to contain inlined object, got %s", string(pretty))
	}
}

func TestValidateMethodSimple(t *testing.T) {
	req, _ := http.NewRequest(http.MethodPost, "http://example.com", nil)
	methodStr, err := docker.ValidateMethod(req, []string{"GET", "POST"})
	if err != nil {
		t.Fatalf("validateMethod unexpected error: %v", err)
	}
	if methodStr != `Method = "POST"` {
		t.Fatalf("unexpected method string: %s", methodStr)
	}

	// Unsupported method should error
	req.Method = http.MethodDelete
	if _, err := docker.ValidateMethod(req, []string{"GET", "POST"}); err == nil {
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

	if err := docker.CleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error: %v", err)
	}
	if exists, _ := afero.Exists(fs, dr.ResponseTargetFile); exists {
		t.Fatalf("file still exists after cleanOldFiles")
	}
}

func TestDecodeAndFormatResponseSimple(t *testing.T) {
	logger := logging.NewTestLogger()

	// Build sample APIResponse JSON with base64 encoded data
	sample := docker.APIResponse{
		Success:  true,
		Response: docker.ResponseData{Data: []string{`{"foo":"bar"}`}},
		Meta:     docker.ResponseMeta{RequestID: "abc123"},
	}
	raw, _ := json.Marshal(sample)

	decoded, err := docker.DecodeResponseContent(raw, logger)
	if err != nil {
		t.Fatalf("decodeResponseContent error: %v", err)
	}
	if len(decoded.Response.Data) != 1 || decoded.Response.Data[0] != `{"foo":"bar"}` {
		t.Fatalf("decodeResponseContent did not decode JSON correctly: %v", decoded.Response.Data)
	}

	// Marshal decoded struct then format
	marshaled, _ := json.Marshal(decoded)
	formatted := docker.FormatResponseJSON(marshaled)
	if !bytes.Contains(formatted, []byte("foo")) {
		t.Fatalf("formatResponseJSON missing field")
	}
}

func TestDecodeResponseContent_Success(t *testing.T) {
	logger := logging.NewTestLogger()

	// Prepare an APIResponse JSON with base64-encoded JSON payload in data.
	inner := `{"hello":"world"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(inner))

	raw := docker.APIResponse{
		Success: true,
		Response: docker.ResponseData{
			Data: []string{encoded},
		},
		Meta: docker.ResponseMeta{
			RequestID: "abc",
		},
	}

	rawBytes, err := json.Marshal(raw)
	require.NoError(t, err)

	decoded, err := docker.DecodeResponseContent(rawBytes, logger)
	require.NoError(t, err)
	assert.Equal(t, "abc", decoded.Meta.RequestID)
	assert.Equal(t, `{"hello":"world"}`, decoded.Response.Data[0])
}

func TestDecodeResponseContent_InvalidJSON(t *testing.T) {
	logger := logging.NewTestLogger()
	_, err := docker.DecodeResponseContent([]byte(`not-json`), logger)
	require.Error(t, err)
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

	pretty := docker.FormatResponseJSON(bytesIn)

	// The formatted JSON should contain nested object without quotes around keys
	assert.Contains(t, string(pretty), "\"foo\": \"bar\"")
}
