package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/resolver"
	apiserver "github.com/kdeps/schema/gen/api_server"
	"github.com/kdeps/schema/gen/project"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestAPIServer(t *testing.T) (*resolver.DependencyResolver, *gin.Engine) {
	fs := afero.NewMemMapFs()
	logger := logging.NewTestLogger()
	ctx := context.Background()

	// Create necessary directories
	err := fs.MkdirAll("/tmp", 0o755)
	require.NoError(t, err)
	err = fs.MkdirAll("/files", 0o755)
	require.NoError(t, err)

	dr := &resolver.DependencyResolver{
		Fs:        fs,
		Logger:    logger,
		Context:   ctx,
		FilesDir:  "/files",
		ActionDir: "/action",
	}

	router := gin.Default()
	return dr, router
}

func TestHandleMultipartForm(t *testing.T) {
	dr, _ := setupTestAPIServer(t)
	fileMap := make(map[string]struct{ Filename, Filetype string })

	t.Run("NoFile", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest("POST", "/", nil)

		err := handleMultipartForm(c, dr, fileMap)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Unable to parse multipart form")
	})

	t.Run("SingleFile", func(t *testing.T) {
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

		err = handleMultipartForm(c, dr, fileMap)
		assert.NoError(t, err)
		assert.Len(t, fileMap, 1)
	})
}

func TestProcessFile(t *testing.T) {
	dr, _ := setupTestAPIServer(t)
	fileMap := make(map[string]struct{ Filename, Filetype string })

	t.Run("ValidFile", func(t *testing.T) {
		// Use multipart to create a real file upload and get the FileHeader
		body := &bytes.Buffer{}
		writer := multipart.NewWriter(body)
		part, err := writer.CreateFormFile("file", "test.txt")
		require.NoError(t, err)
		_, err = part.Write([]byte("test content"))
		require.NoError(t, err)
		writer.Close()

		request := httptest.NewRequest("POST", "/", body)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		_, fileHeader, err := request.FormFile("file")
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
}

func TestFormatResponseJSON(t *testing.T) {
	t.Run("FormatJSON", func(t *testing.T) {
		response := APIResponse{
			Success: true,
			Response: ResponseData{
				Data: []string{"test"},
			},
		}
		content, err := json.Marshal(response)
		require.NoError(t, err)

		formatted := formatResponseJSON(content)
		assert.NotEmpty(t, formatted)
	})
}

func TestCleanOldFiles(t *testing.T) {
	dr, _ := setupTestAPIServer(t)

	t.Run("NoFiles", func(t *testing.T) {
		err := cleanOldFiles(dr)
		assert.NoError(t, err)
	})

	t.Run("WithFiles", func(t *testing.T) {
		// Create a test file
		testFile := filepath.Join(dr.ActionDir, "test.txt")
		err := afero.WriteFile(dr.Fs, testFile, []byte("test"), 0o644)
		require.NoError(t, err)

		err = cleanOldFiles(dr)
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

func TestSetupRoutes(t *testing.T) {
	dr, router := setupTestAPIServer(t)
	dr.Logger = logging.NewTestLogger()
	semaphore := make(chan struct{}, 1)

	t.Run("InvalidRoute", func(t *testing.T) {
		setupRoutes(router, context.Background(), nil, nil, []*apiserver.APIServerRoutes{nil}, dr, semaphore)
		// No assertion needed as the function logs the error
	})

	t.Run("ValidRoute", func(t *testing.T) {
		routes := []*apiserver.APIServerRoutes{
			{
				Path:    "/test",
				Methods: []string{"GET"},
			},
		}
		setupRoutes(router, context.Background(), nil, nil, routes, dr, semaphore)
		// No assertion needed as the function sets up routes
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

func TestProcessWorkflow(t *testing.T) {
	dr, _ := setupTestAPIServer(t)

	t.Run("ProcessWorkflow", func(t *testing.T) {
		err := processWorkflow(context.Background(), dr)
		assert.Error(t, err) // Expected error due to missing workflow configuration
	})
}

// workflowWithNilSettings is a mock Workflow with GetSettings() and GetAgentIcon() returning nil
type workflowWithNilSettings struct{}

func (w workflowWithNilSettings) GetSettings() *project.Settings { return nil }

func (w workflowWithNilSettings) GetTargetActionID() string { return "" }

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
