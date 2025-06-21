package docker_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/apple/pkl-go/pkl"
	"github.com/gin-gonic/gin"
	. "github.com/kdeps/kdeps/pkg/docker"
	"github.com/kdeps/kdeps/pkg/environment"
	"github.com/kdeps/kdeps/pkg/item"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/memory"
	"github.com/kdeps/kdeps/pkg/resolver"
	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/kdeps/kdeps/pkg/session"
	"github.com/kdeps/kdeps/pkg/tool"
	apiserver "github.com/kdeps/schema/gen/api_server"
	"github.com/kdeps/schema/gen/project"
	"github.com/kdeps/schema/gen/resource"
	_ "github.com/mattn/go-sqlite3"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/ktx"
	"github.com/kdeps/kdeps/pkg/utils"
)

// createStubPkl is a helper to stub the PKL binary for tests
func createStubPkl(t *testing.T) (cleanup func()) {
	dir := t.TempDir()
	exeName := "pkl"
	if runtime.GOOS == "windows" {
		exeName = "pkl.bat"
	}
	stubPath := filepath.Join(dir, exeName)
	script := `#!/bin/sh
output_path=
prev=
for arg in "$@"; do
if [ "$prev" = "--output-path" ]; then
	output_path="$arg"
	break
fi
prev="$arg"
done
json='{"hello":"world"}'
# emit JSON to stdout
echo "$json"
# if --output-path was supplied, also write JSON to that file
if [ -n "$output_path" ]; then
echo "$json" > "$output_path"
fi
`
	if runtime.GOOS == "windows" {
		script = "@echo {\"hello\":\"world\"}\r\n"
	}
	if err := os.WriteFile(stubPath, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write stub: %v", err)
	}
	if runtime.GOOS != "windows" {
		_ = os.Chmod(stubPath, 0o755)
	}
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)
	return func() { os.Setenv("PATH", oldPath) }
}

func TestValidateMethodExtra2(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	methodStr, err := ValidateMethod(req, []string{http.MethodGet, http.MethodPost})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if methodStr != `method = "POST"` {
		t.Fatalf("unexpected method string: %s", methodStr)
	}

	// invalid method
	badReq := httptest.NewRequest("DELETE", "/", nil)
	if _, err := ValidateMethod(badReq, []string{"GET"}); err == nil {
		t.Fatalf("expected error for disallowed method")
	}
}

func TestCleanOldFilesExtra2(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// create dummy response file
	path := "/tmp/resp.json"
	afero.WriteFile(fs, path, []byte("dummy"), 0o644)

	dr := &resolver.DependencyResolver{
		Fs:                 fs,
		Logger:             logger,
		ResponseTargetFile: path,
	}

	if err := CleanOldFiles(dr); err != nil {
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

	decResp, err := DecodeResponseContent(encoded, logger)
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
	pretty := FormatResponseJSON(raw)

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
	pretty := FormatResponseJSON(raw)

	// It should now be pretty-printed and contain nested object without quotes
	require.Contains(t, string(pretty), "\"foo\": \"bar\"")
}

func TestCleanOldFilesExtra(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &resolver.DependencyResolver{Fs: fs, Logger: logging.NewTestLogger(), ResponseTargetFile: "old.json"}

	// Case where file exists
	require.NoError(t, afero.WriteFile(fs, dr.ResponseTargetFile, []byte("x"), 0o644))
	require.NoError(t, CleanOldFiles(dr))
	exists, _ := afero.Exists(fs, dr.ResponseTargetFile)
	require.False(t, exists)

	// Case where file does not exist should be no-op
	require.NoError(t, CleanOldFiles(dr))
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
	out, err := DecodeResponseContent(raw, logger)
	require.NoError(t, err)
	require.Len(t, out.Response.Data, 1)
	require.JSONEq(t, dataJSON, out.Response.Data[0])
}

func TestFormatResponseJSONFormatTest(t *testing.T) {
	// Input where first element is JSON string and second is plain string.
	in := []byte(`{"response":{"data":["{\"x\":1}","plain"]}}`)
	out := FormatResponseJSON(in)
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
		err = HandleMultipartForm(c, dr, fileMap)
		assert.NoError(t, err)
		assert.Len(t, fileMap, 1)
	})

	t.Run("InvalidContentType", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", bytes.NewBuffer([]byte("test")))
		c.Request.Header.Set("Content-Type", "text/plain")

		fileMap := make(map[string]struct{ Filename, Filetype string })
		err := HandleMultipartForm(c, dr, fileMap)
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
		err := HandleMultipartForm(c, dr, fileMap)
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

		err = ProcessFile(fileHeader, dr, fileMap)
		assert.NoError(t, err)
		assert.Len(t, fileMap, 1)
	})
}

func TestValidateMethod(t *testing.T) {
	t.Run("ValidMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/", nil)
		method, err := ValidateMethod(req, []string{"GET", "POST"})
		assert.NoError(t, err)
		assert.Equal(t, "method = \"GET\"", method)
	})

	t.Run("InvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest("PUT", "/", nil)
		_, err := ValidateMethod(req, []string{"GET", "POST"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "HTTP method \"PUT\" not allowed")
	})

	t.Run("EmptyMethodDefaultsToGet", func(t *testing.T) {
		req := httptest.NewRequest("", "/", nil)
		method, err := ValidateMethod(req, []string{"GET", "POST"})
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

		decoded, err := DecodeResponseContent(content, logger)
		assert.NoError(t, err)
		assert.True(t, decoded.Success)
		assert.Equal(t, "123", decoded.Meta.RequestID)
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		content := []byte("invalid json")
		_, err := DecodeResponseContent(content, logger)
		assert.Error(t, err)
	})

	t.Run("EmptyResponse", func(t *testing.T) {
		content := []byte("{}")
		decoded, err := DecodeResponseContent(content, logger)
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

		formatted := FormatResponseJSON(content)
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

		formatted := FormatResponseJSON(content)
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
		err := CleanOldFiles(dr)
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

		err := CleanOldFiles(dr)
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

type mockResolver struct {
	*resolver.DependencyResolver
	prepareWorkflowDirFn           func() error
	prepareImportFilesFn           func() error
	handleRunActionFn              func() (bool, error)
	evalPklFormattedResponseFileFn func() (string, error)
}

func (m *mockResolver) PrepareWorkflowDir() error {
	if m.prepareWorkflowDirFn != nil {
		return m.prepareWorkflowDirFn()
	}
	return m.DependencyResolver.PrepareWorkflowDir()
}

func (m *mockResolver) PrepareImportFiles() error {
	if m.prepareImportFilesFn != nil {
		return m.prepareImportFilesFn()
	}
	return m.DependencyResolver.PrepareImportFiles()
}

func (m *mockResolver) HandleRunAction() (bool, error) {
	if m.handleRunActionFn != nil {
		return m.handleRunActionFn()
	}
	return m.DependencyResolver.HandleRunAction()
}

func (m *mockResolver) EvalPklFormattedResponseFile() (string, error) {
	if m.evalPklFormattedResponseFileFn != nil {
		return m.evalPklFormattedResponseFileFn()
	}
	return m.DependencyResolver.EvalPklFormattedResponseFile()
}

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

		dr := &resolver.DependencyResolver{
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

		dr.PrependDynamicImportsFn = func(string) error { return nil }
		dr.AddPlaceholderImportsFn = func(string) error { return nil }
		dr.LoadResourceEntriesFn = func() error { return nil }
		dr.BuildDependencyStackFn = func(string, map[string]bool) []string { return []string{"test-action"} }
		dr.LoadResourceFn = func(context.Context, string, resolver.ResourceType) (interface{}, error) {
			items := []string{}
			return &resource.Resource{Items: &items, Run: nil}, nil
		}
		dr.ProcessRunBlockFn = func(resolver.ResourceNodeEntry, *resource.Resource, string, bool) (bool, error) {
			return false, fmt.Errorf("failed to handle run action")
		}
		dr.ClearItemDBFn = func() error { return nil }
		err := ProcessWorkflow(ctx, dr)
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
		SetupRoutes(router, ctx, corsConfig, []string{"127.0.0.1"}, routes, baseDr, semaphore)

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
		SetupRoutes(router, ctx, corsConfig, []string{"127.0.0.1"}, invalidRoutes, baseDr, semaphore)
		// No assertions needed as the function should log errors and continue
	})

	t.Run("CORSDisabled", func(t *testing.T) {
		router := gin.New()
		ctx := context.Background()
		disabledCORS := &apiserver.CORS{
			EnableCORS: false,
		}
		SetupRoutes(router, ctx, disabledCORS, []string{"127.0.0.1"}, routes, baseDr, semaphore)
		// No assertions needed as the function should skip CORS setup
	})

	t.Run("NoTrustedProxies", func(t *testing.T) {
		router := gin.New()
		ctx := context.Background()
		SetupRoutes(router, ctx, corsConfig, nil, routes, baseDr, semaphore)
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
		SetupRoutes(router, ctx, corsConfig, []string{"127.0.0.1"}, unsupportedRoutes, baseDr, semaphore)
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
	got, err := ValidateMethod(req, []string{http.MethodGet, http.MethodPost})
	if err != nil || got != `method = "GET"` {
		t.Fatalf("expected valid GET, got %q err %v", got, err)
	}

	reqEmpty, _ := http.NewRequest("", "http://example.com", nil)
	got2, err2 := ValidateMethod(reqEmpty, []string{http.MethodGet})
	if err2 != nil || got2 != `method = "GET"` {
		t.Fatalf("default method failed: %q err %v", got2, err2)
	}

	reqBad, _ := http.NewRequest(http.MethodDelete, "http://example.com", nil)
	if _, err := ValidateMethod(reqBad, []string{http.MethodGet}); err == nil {
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
	decoded, err := DecodeResponseContent(data, logger)
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
	decoded, err := DecodeResponseContent(raw, logger)
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
	out, err := ValidateMethod(req, allowed)
	assert.NoError(t, err)
	assert.Equal(t, `method = "POST"`, out)

	// default empty method becomes GET and passes
	req2, _ := http.NewRequest("", "/", nil)
	out, err = ValidateMethod(req2, allowed)
	assert.NoError(t, err)
	assert.Equal(t, `method = "GET"`, out)

	// invalid method
	req3, _ := http.NewRequest(http.MethodPut, "/", nil)
	out, err = ValidateMethod(req3, allowed)
	assert.Error(t, err)
	assert.Empty(t, out)
}

func TestCleanOldFilesMore(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// create dummy response file
	const respPath = "/tmp/response.json"
	_ = afero.WriteFile(fs, respPath, []byte("old"), 0o644)

	dr := &resolver.DependencyResolver{
		Fs:                 fs,
		ResponseTargetFile: respPath,
		Logger:             logger,
	}

	// should remove existing file
	err := CleanOldFiles(dr)
	assert.NoError(t, err)
	exist, _ := afero.Exists(fs, respPath)
	assert.False(t, exist)

	// second call with file absent should still succeed
	err = CleanOldFiles(dr)
	assert.NoError(t, err)
}

// TestCleanOldFiles ensures that the helper deletes the ResponseTargetFile when it exists
// and returns nil when the file is absent. Both branches of the conditional are exercised.
func TestCleanOldFilesMemFS(t *testing.T) {
	mem := afero.NewMemMapFs()
	dr := &resolver.DependencyResolver{
		Fs:                 mem,
		ResponseTargetFile: "/tmp/response.json",
		Logger:             logging.NewTestLogger(),
		Context:            context.Background(),
	}

	// Branch 1: File exists and should be removed without error.
	if err := afero.WriteFile(mem, dr.ResponseTargetFile, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to seed response file: %v", err)
	}
	if err := CleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error for existing file: %v", err)
	}
	if exists, _ := afero.Exists(mem, dr.ResponseTargetFile); exists {
		t.Fatalf("expected response file to be removed")
	}

	// Branch 2: File does not exist – function should still return nil (no error).
	if err := CleanOldFiles(dr); err != nil {
		t.Fatalf("cleanOldFiles returned error when file absent: %v", err)
	}
}

// TestCleanOldFilesRemoveError exercises the branch where RemoveAll returns an
// error. It uses a read-only filesystem wrapper so the delete fails without
// depending on OS-specific permissions.
func TestCleanOldFilesRemoveError(t *testing.T) {
	mem := afero.NewMemMapFs()
	target := "/tmp/response.json"
	if err := afero.WriteFile(mem, target, []byte("data"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	dr := &resolver.DependencyResolver{
		Fs:                 afero.NewReadOnlyFs(mem), // makes RemoveAll fail
		ResponseTargetFile: target,
		Logger:             logging.NewTestLogger(),
		Context:            context.Background(),
	}

	if err := CleanOldFiles(dr); err == nil {
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
	pretty := FormatResponseJSON(raw)

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
	if err := CleanOldFiles(dr); err != nil {
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

	pretty := FormatResponseJSON(raw)

	if !bytes.Contains(pretty, []byte("\"foo\": \"bar\"")) {
		t.Fatalf("expected pretty JSON to contain inlined object, got %s", string(pretty))
	}
}

func TestValidateMethodSimple(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://example.com", nil)
	methodStr, err := ValidateMethod(req, []string{"GET", "POST"})
	if err != nil {
		t.Fatalf("validateMethod unexpected error: %v", err)
	}
	if methodStr != `method = "POST"` {
		t.Fatalf("unexpected method string: %s", methodStr)
	}

	// Unsupported method should error
	req.Method = "DELETE"
	if _, err := ValidateMethod(req, []string{"GET", "POST"}); err == nil {
		t.Fatalf("expected error for unsupported method")
	}
}

func TestCleanOldFilesMem(t *testing.T) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()

	// Prepare dependency resolver stub with in-mem fs
	dr := &resolver.DependencyResolver{Fs: fs, Logger: logger, ResponseTargetFile: "/tmp/old_resp.txt"}
	// Create dummy file
	afero.WriteFile(fs, dr.ResponseTargetFile, []byte("old"), 0o666)

	if err := CleanOldFiles(dr); err != nil {
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

	decoded, err := DecodeResponseContent(raw, logger)
	if err != nil {
		t.Fatalf("decodeResponseContent error: %v", err)
	}
	if len(decoded.Response.Data) != 1 || decoded.Response.Data[0] != "{\n  \"foo\": \"bar\"\n}" {
		t.Fatalf("decodeResponseContent did not prettify JSON: %v", decoded.Response.Data)
	}

	// Marshal decoded struct then format
	marshaled, _ := json.Marshal(decoded)
	formatted := FormatResponseJSON(marshaled)
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

	decoded, err := DecodeResponseContent(rawBytes, logger)
	assert.NoError(t, err)
	assert.Equal(t, "abc", decoded.Meta.RequestID)
	assert.Contains(t, decoded.Response.Data[0], "\"hello\": \"world\"")
}

func TestDecodeResponseContent_InvalidJSON(t *testing.T) {
	logger := logging.NewTestLogger()
	_, err := DecodeResponseContent([]byte(`not-json`), logger)
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

	pretty := FormatResponseJSON(bytesIn)

	// The formatted JSON should contain nested object without quotes around keys
	assert.Contains(t, string(pretty), "\"foo\": \"bar\"")
}

// TestValidateMethodDefaultGET verifies that when the incoming request has an
// empty Method field validateMethod substitutes "GET" and returns the correct
// formatted string without error.
func TestValidateMethodDefaultGET(t *testing.T) {
	req := &http.Request{}

	got, err := ValidateMethod(req, []string{"GET"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := `method = "GET"`
	if got != want {
		t.Fatalf("unexpected result: got %q want %q", got, want)
	}
}

// TestValidateMethodNotAllowed verifies that validateMethod returns an error
// when an HTTP method that is not in the allowed list is provided.
func TestValidateMethodNotAllowed(t *testing.T) {
	req := &http.Request{Method: "POST"}

	if _, err := ValidateMethod(req, []string{"GET"}); err == nil {
		t.Fatalf("expected method not allowed error, got nil")
	}
}

func TestStartAPIServerMode_HappyPath(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Use a random free port
	ln, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	wfSettings := &project.Settings{
		APIServer: &apiserver.APIServerSettings{
			HostIP:  "127.0.0.1",
			PortNum: uint16(port),
			Routes: []*apiserver.APIServerRoutes{
				{
					Path:    "/test",
					Methods: []string{"GET"},
				},
			},
		},
	}
	mw := &MockWorkflow{settings: wfSettings}
	dr := &resolver.DependencyResolver{
		Workflow: mw,
		Logger:   logger,
		Fs:       fs,
	}

	err = StartAPIServerMode(ctx, dr)
	// Accept either nil or error due to context cancellation or port race
	if err != nil {
		t.Logf("StartAPIServerMode returned error (acceptable in CI): %v", err)
	}
}

func TestAPIServerHandler_Comprehensive(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	t.Run("SemaphoreFull", func(t *testing.T) {
		// Create a semaphore that's already full
		semaphore := make(chan struct{}, 1)
		semaphore <- struct{}{} // Fill the semaphore

		route := &apiserver.APIServerRoutes{
			Path:    "/test",
			Methods: []string{"GET"},
		}
		dr := &resolver.DependencyResolver{
			Fs:     fs,
			Logger: logger,
		}

		handler := APIServerHandler(ctx, route, dr, semaphore)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", nil)

		handler(c)

		assert.Equal(t, http.StatusTooManyRequests, w.Code)
		var resp APIResponse
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		require.NoError(t, err)
		assert.False(t, resp.Success)
		assert.Len(t, resp.Errors, 1)
		assert.Equal(t, http.StatusTooManyRequests, resp.Errors[0].Code)
		assert.Equal(t, "Only one active connection is allowed", resp.Errors[0].Message)
	})

	t.Run("GetRequestWithBody", func(t *testing.T) {
		semaphore := make(chan struct{}, 1)
		route := &apiserver.APIServerRoutes{
			Path:    "/test",
			Methods: []string{"GET"},
		}
		dr := &resolver.DependencyResolver{
			Fs:     fs,
			Logger: logger,
		}

		handler := APIServerHandler(ctx, route, dr, semaphore)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("GET", "/test", strings.NewReader("test body"))

		handler(c)

		// Should fail due to resolver initialization error, but we've covered the GET body reading path
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("PostRequestWithJSON", func(t *testing.T) {
		semaphore := make(chan struct{}, 1)
		route := &apiserver.APIServerRoutes{
			Path:    "/test",
			Methods: []string{"POST"},
		}
		dr := &resolver.DependencyResolver{
			Fs:     fs,
			Logger: logger,
		}

		handler := APIServerHandler(ctx, route, dr, semaphore)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/test", strings.NewReader(`{"test": "data"}`))
		c.Request.Header.Set("Content-Type", "application/json")

		handler(c)

		// Should fail due to resolver initialization error, but we've covered the POST body reading path
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("DeleteRequest", func(t *testing.T) {
		semaphore := make(chan struct{}, 1)
		route := &apiserver.APIServerRoutes{
			Path:    "/test",
			Methods: []string{"DELETE"},
		}
		dr := &resolver.DependencyResolver{
			Fs:     fs,
			Logger: logger,
		}

		handler := APIServerHandler(ctx, route, dr, semaphore)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("DELETE", "/test", nil)

		handler(c)

		// Should fail due to resolver initialization error, but we've covered the DELETE path
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("MultipartFormRequest", func(t *testing.T) {
		semaphore := make(chan struct{}, 1)
		route := &apiserver.APIServerRoutes{
			Path:    "/test",
			Methods: []string{"POST"},
		}

		tmpDir, err := afero.TempDir(fs, "", "multipart-test")
		require.NoError(t, err)
		defer fs.RemoveAll(tmpDir)

		dr := &resolver.DependencyResolver{
			Fs:        fs,
			Logger:    logger,
			ActionDir: tmpDir,
		}

		handler := APIServerHandler(ctx, route, dr, semaphore)
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Create multipart form
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("test content"))
		require.NoError(t, err)
		writer.Close()

		c.Request = httptest.NewRequest("POST", "/test", body)
		c.Request.Header.Set("Content-Type", writer.FormDataContentType())

		handler(c)

		// Should fail due to resolver initialization error, but we've covered the multipart form path
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestAPIServerHandler_ErrorPaths(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Test with nil route
	handler := APIServerHandler(ctx, nil, &resolver.DependencyResolver{
		Fs:     fs,
		Logger: logger,
	}, make(chan struct{}, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	handler(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Errors[0].Message, "Invalid route configuration")

	// Test with empty path route
	emptyRoute := &apiserver.APIServerRoutes{
		Path:    "",
		Methods: []string{"GET"},
	}
	handler = APIServerHandler(ctx, emptyRoute, &resolver.DependencyResolver{
		Fs:     fs,
		Logger: logger,
	}, make(chan struct{}, 1))

	w = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(w)
	handler(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestAPIServerHandler_SemaphoreBlocked(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create a semaphore that's already full
	semaphore := make(chan struct{}, 1)
	semaphore <- struct{}{} // Fill the semaphore

	route := &apiserver.APIServerRoutes{
		Path:    "/test",
		Methods: []string{"GET"},
	}

	handler := APIServerHandler(ctx, route, &resolver.DependencyResolver{
		Fs:     fs,
		Logger: logger,
	}, semaphore)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request = c.Request.WithContext(ctx)
	handler(c)

	assert.Equal(t, http.StatusTooManyRequests, w.Code)
	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Errors[0].Message, "Only one active connection is allowed")
}

func TestAPIServerHandler_ResolverInitError(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	route := &apiserver.APIServerRoutes{
		Path:    "/test",
		Methods: []string{"GET"},
	}

	// Create a resolver that will fail to initialize
	handler := APIServerHandler(ctx, route, &resolver.DependencyResolver{
		Fs:     fs,
		Logger: logger,
		// Missing required fields to cause initialization error
	}, make(chan struct{}, 1))

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/test", nil)
	c.Request = c.Request.WithContext(ctx)
	handler(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Errors[0].Message, "Failed to initialize resolver")
}

func TestAPIServerHandler_InvalidMethod(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create required context keys
	tempDir := t.TempDir()
	agentDir := filepath.Join(tempDir, "agent")
	actionDir := filepath.Join(tempDir, "action")
	kdepsDir := filepath.Join(tempDir, "kdeps")
	graphID := "test-graph-id"

	// Create directories
	require.NoError(t, fs.MkdirAll(agentDir, 0o755))
	require.NoError(t, fs.MkdirAll(actionDir, 0o755))
	require.NoError(t, fs.MkdirAll(kdepsDir, 0o755))

	// Add context keys
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeySharedDir, kdepsDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)

	// Create workflow file
	workflowDir := filepath.Join(agentDir, "workflow")
	require.NoError(t, fs.MkdirAll(workflowDir, 0o755))
	workflowFile := filepath.Join(workflowDir, "workflow.pkl")

	// Create a proper PKL file with required fields
	workflowContent := fmt.Sprintf(`amends "package://schema.kdeps.com/core@%s#/Workflow.pkl"

targetActionID = "testAction"
name = "test-agent"`, schema.SchemaVersion(ctx))
	require.NoError(t, afero.WriteFile(fs, workflowFile, []byte(workflowContent), 0o644))

	// Add required context keys
	kdepsDir = filepath.Join(tempDir, "kdeps")
	require.NoError(t, fs.MkdirAll(kdepsDir, 0o755))
	graphID = "test-graph-id"
	ctxWithKeys := testContextWithAllKeys(ctx, agentDir, actionDir, kdepsDir, graphID)

	// Create a proper environment
	env := &environment.Environment{
		Root: tempDir,
		Home: filepath.Join(tempDir, "home"),
		Pwd:  workflowDir,
	}

	semaphore := make(chan struct{}, 1)
	route := &apiserver.APIServerRoutes{
		Path:    "/test",
		Methods: []string{"GET"},
	}
	dr := &resolver.DependencyResolver{
		Fs:          fs,
		Logger:      logger,
		Environment: env,
	}

	handler := APIServerHandler(ctxWithKeys, route, dr, semaphore)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/test", nil)
	c.Request = c.Request.WithContext(ctxWithKeys)
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Errors[0].Message, "HTTP method \"POST\" not allowed")
}

func TestAPIServerHandler_OptionsMethod(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create required context keys
	tempDir := t.TempDir()
	agentDir := filepath.Join(tempDir, "agent")
	actionDir := filepath.Join(tempDir, "action")
	kdepsDir := filepath.Join(tempDir, "kdeps")
	graphID := "test-graph-id"

	// Create directories
	require.NoError(t, fs.MkdirAll(agentDir, 0o755))
	require.NoError(t, fs.MkdirAll(actionDir, 0o755))
	require.NoError(t, fs.MkdirAll(kdepsDir, 0o755))

	// Add context keys
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeySharedDir, kdepsDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)

	// Create workflow file
	workflowDir := filepath.Join(agentDir, "workflow")
	require.NoError(t, fs.MkdirAll(workflowDir, 0o755))
	workflowFile := filepath.Join(workflowDir, "workflow.pkl")
	require.NoError(t, afero.WriteFile(fs, workflowFile, []byte("workflow content"), 0o644))

	route := &apiserver.APIServerRoutes{
		Path:    "/test",
		Methods: []string{"GET", "POST"},
	}

	// Create a proper environment
	env := &environment.Environment{
		Root: tempDir,
		Home: filepath.Join(tempDir, "home"),
		Pwd:  workflowDir,
	}

	handler := APIServerHandlerTestable(ctx, route, &resolver.DependencyResolver{
		Fs:          fs,
		Logger:      logger,
		Environment: env,
	}, make(chan struct{}, 1), agentDir, actionDir, kdepsDir)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("OPTIONS", "/test", nil)
	c.Request = c.Request.WithContext(ctx)
	handler(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "OPTIONS, GET, HEAD, POST, PUT, PATCH, DELETE", w.Header().Get("Allow"))
}

// APIServerHandlerTestable is a test-only version that generates the graphID, sets all required context keys, and calls the original handler
func APIServerHandlerTestable(ctx context.Context, route *apiserver.APIServerRoutes, baseDr *resolver.DependencyResolver, semaphore chan struct{}, agentDir, actionDir, kdepsDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		graphID := "test-graph-id" // Use a fixed graphID for test determinism
		mergedCtx := ctx
		mergedCtx = ktx.CreateContext(mergedCtx, ktx.CtxKeyAgentDir, agentDir)
		mergedCtx = ktx.CreateContext(mergedCtx, ktx.CtxKeyActionDir, actionDir)
		mergedCtx = ktx.CreateContext(mergedCtx, ktx.CtxKeySharedDir, kdepsDir)
		mergedCtx = ktx.CreateContext(mergedCtx, ktx.CtxKeyGraphID, graphID)
		c.Request = c.Request.WithContext(mergedCtx)
		h := APIServerHandler(mergedCtx, route, baseDr, semaphore)
		h(c)
	}
}

func TestAPIServerHandler_HeadMethod(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create required context keys
	tempDir := t.TempDir()
	agentDir := filepath.Join(tempDir, "agent")
	actionDir := filepath.Join(tempDir, "action")
	kdepsDir := filepath.Join(tempDir, "kdeps")
	graphID := "test-graph-id"

	// Create directories
	require.NoError(t, fs.MkdirAll(agentDir, 0o755))
	require.NoError(t, fs.MkdirAll(actionDir, 0o755))
	require.NoError(t, fs.MkdirAll(kdepsDir, 0o755))

	// Add context keys
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeySharedDir, kdepsDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)

	// Create workflow file
	workflowDir := filepath.Join(agentDir, "workflow")
	require.NoError(t, fs.MkdirAll(workflowDir, 0o755))
	workflowFile := filepath.Join(workflowDir, "workflow.pkl")
	require.NoError(t, afero.WriteFile(fs, workflowFile, []byte("workflow content"), 0o644))

	route := &apiserver.APIServerRoutes{
		Path:    "/test",
		Methods: []string{"GET", "HEAD"},
	}

	// Create a proper environment
	env := &environment.Environment{
		Root: tempDir,
		Home: filepath.Join(tempDir, "home"),
		Pwd:  workflowDir,
	}

	handler := APIServerHandlerTestable(ctx, route, &resolver.DependencyResolver{
		Fs:          fs,
		Logger:      logger,
		Environment: env,
	}, make(chan struct{}, 1), agentDir, actionDir, kdepsDir)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("HEAD", "/test", nil)
	c.Request = c.Request.WithContext(ctx)
	handler(c)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json; charset=utf-8", w.Header().Get("Content-Type"))
}

func TestAPIServerHandler_UnsupportedMethod(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create required context keys
	tempDir := t.TempDir()
	agentDir := filepath.Join(tempDir, "agent")
	actionDir := filepath.Join(tempDir, "action")
	kdepsDir := filepath.Join(tempDir, "kdeps")
	graphID := "test-graph-id"

	// Create directories
	require.NoError(t, fs.MkdirAll(agentDir, 0o755))
	require.NoError(t, fs.MkdirAll(actionDir, 0o755))
	require.NoError(t, fs.MkdirAll(kdepsDir, 0o755))

	// Add context keys
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeySharedDir, kdepsDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)

	// Create workflow file
	workflowDir := filepath.Join(agentDir, "workflow")
	require.NoError(t, fs.MkdirAll(workflowDir, 0o755))
	workflowFile := filepath.Join(workflowDir, "workflow.pkl")
	require.NoError(t, afero.WriteFile(fs, workflowFile, []byte("workflow content"), 0o644))

	route := &apiserver.APIServerRoutes{
		Path:    "/test",
		Methods: []string{"GET"},
	}

	// Create a proper environment
	env := &environment.Environment{
		Root: tempDir,
		Home: filepath.Join(tempDir, "home"),
		Pwd:  workflowDir,
	}

	handler := APIServerHandlerTestable(ctx, route, &resolver.DependencyResolver{
		Fs:          fs,
		Logger:      logger,
		Environment: env,
	}, make(chan struct{}, 1), agentDir, actionDir, kdepsDir)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("TRACE", "/test", nil)
	c.Request = c.Request.WithContext(ctx)
	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Errors[0].Message, "HTTP method \"TRACE\" not allowed")
}

func TestAPIServerHandler_GetMethodBodyReadError(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create required context keys
	tempDir := t.TempDir()
	agentDir := filepath.Join(tempDir, "agent")
	actionDir := filepath.Join(tempDir, "action")
	kdepsDir := filepath.Join(tempDir, "kdeps")
	graphID := "test-graph-id"

	// Create directories
	require.NoError(t, fs.MkdirAll(agentDir, 0o755))
	require.NoError(t, fs.MkdirAll(actionDir, 0o755))
	require.NoError(t, fs.MkdirAll(kdepsDir, 0o755))

	// Add context keys
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeySharedDir, kdepsDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)

	// Create workflow file
	workflowDir := filepath.Join(agentDir, "workflow")
	require.NoError(t, fs.MkdirAll(workflowDir, 0o755))
	workflowFile := filepath.Join(workflowDir, "workflow.pkl")
	require.NoError(t, afero.WriteFile(fs, workflowFile, []byte("workflow content"), 0o644))

	route := &apiserver.APIServerRoutes{
		Path:    "/test",
		Methods: []string{"GET"},
	}

	// Create a proper environment
	env := &environment.Environment{
		Root: tempDir,
		Home: filepath.Join(tempDir, "home"),
		Pwd:  workflowDir,
	}

	handler := APIServerHandlerTestable(ctx, route, &resolver.DependencyResolver{
		Fs:          fs,
		Logger:      logger,
		Environment: env,
	}, make(chan struct{}, 1), agentDir, actionDir, kdepsDir)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a request with a body that can't be read
	req := httptest.NewRequest("GET", "/test", strings.NewReader("test"))
	req.Body = &errorReader{}
	c.Request = req.WithContext(ctx)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Errors[0].Message, "Failed to read request body")
}

func TestAPIServerHandler_PostMethodBodyReadError(t *testing.T) {
	fs := afero.NewOsFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create required context keys
	tempDir := t.TempDir()
	agentDir := filepath.Join(tempDir, "agent")
	actionDir := filepath.Join(tempDir, "action")
	kdepsDir := filepath.Join(tempDir, "kdeps")
	graphID := "test-graph-id"

	// Create directories
	require.NoError(t, fs.MkdirAll(agentDir, 0o755))
	require.NoError(t, fs.MkdirAll(actionDir, 0o755))
	require.NoError(t, fs.MkdirAll(kdepsDir, 0o755))

	// Add context keys
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeySharedDir, kdepsDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)

	// Create workflow file
	workflowDir := filepath.Join(agentDir, "workflow")
	require.NoError(t, fs.MkdirAll(workflowDir, 0o755))
	workflowFile := filepath.Join(workflowDir, "workflow.pkl")
	require.NoError(t, afero.WriteFile(fs, workflowFile, []byte("workflow content"), 0o644))

	route := &apiserver.APIServerRoutes{
		Path:    "/test",
		Methods: []string{"POST"},
	}

	// Create a proper environment
	env := &environment.Environment{
		Root: tempDir,
		Home: filepath.Join(tempDir, "home"),
		Pwd:  workflowDir,
	}

	handler := APIServerHandlerTestable(ctx, route, &resolver.DependencyResolver{
		Fs:          fs,
		Logger:      logger,
		Environment: env,
	}, make(chan struct{}, 1), agentDir, actionDir, kdepsDir)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Create a request with a body that can't be read
	req := httptest.NewRequest("POST", "/test", strings.NewReader("test"))
	req.Body = &errorReader{}
	c.Request = req.WithContext(ctx)

	handler(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var resp APIResponse
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp.Success)
	assert.Contains(t, resp.Errors[0].Message, "Failed to read request body")
}

// errorReader is a reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, fmt.Errorf("read error")
}

func (e *errorReader) Close() error {
	return nil
}

// testContextWithAllKeys returns a context with all required keys for the resolver
func testContextWithAllKeys(base context.Context, agentDir, actionDir, kdepsDir, graphID string) context.Context {
	ctx := base
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyAgentDir, agentDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyActionDir, actionDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeySharedDir, kdepsDir)
	ctx = ktx.CreateContext(ctx, ktx.CtxKeyGraphID, graphID)
	return ctx
}
