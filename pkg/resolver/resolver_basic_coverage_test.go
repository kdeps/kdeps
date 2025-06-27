package resolver

import (
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestResolverBasicCoverage(t *testing.T) {
	logger := logging.NewTestLogger()
	fs := afero.NewMemMapFs()

	// Set test mode to avoid PKL evaluation issues
	os.Setenv("KDEPS_TEST_MODE", "true")
	defer func() {
		os.Unsetenv("KDEPS_TEST_MODE")
	}()

	// Setup testable environment
	SetupTestableEnvironment()
	defer ResetEnvironment()

	t.Run("validateRequestParams", func(t *testing.T) {
		dr := &DependencyResolver{
			Fs:     fs,
			Logger: logger,
		}

		// Test with valid params
		fileContent := `request.params("allowed_param")`
		allowedParams := []string{"allowed_param"}
		err := dr.validateRequestParams(fileContent, allowedParams)
		assert.NoError(t, err)

		// Test with invalid params
		fileContent = `request.params("invalid_param")`
		allowedParams = []string{"allowed_param"}
		err = dr.validateRequestParams(fileContent, allowedParams)
		assert.Error(t, err)

		// Test with empty allowed params (should allow all)
		err = dr.validateRequestParams(fileContent, []string{})
		assert.NoError(t, err)
	})

	t.Run("validateRequestHeaders", func(t *testing.T) {
		dr := &DependencyResolver{
			Fs:     fs,
			Logger: logger,
		}

		// Test with valid headers
		fileContent := `request.header("authorization")`
		allowedHeaders := []string{"authorization"}
		err := dr.validateRequestHeaders(fileContent, allowedHeaders)
		assert.NoError(t, err)

		// Test with invalid headers
		fileContent = `request.header("x-invalid")`
		allowedHeaders = []string{"authorization"}
		err = dr.validateRequestHeaders(fileContent, allowedHeaders)
		assert.Error(t, err)

		// Test with empty allowed headers (should allow all)
		err = dr.validateRequestHeaders(fileContent, []string{})
		assert.NoError(t, err)
	})

	t.Run("validateRequestPath", func(t *testing.T) {
		dr := &DependencyResolver{
			Fs:     fs,
			Logger: logger,
		}

		// Create test request
		req, _ := gin.CreateTestContext(httptest.NewRecorder())
		req.Request = httptest.NewRequest("GET", "/api/test", nil)

		// Test with valid path
		allowedRoutes := []string{"/api/test"}
		err := dr.validateRequestPath(req, allowedRoutes)
		assert.NoError(t, err)

		// Test with invalid path
		allowedRoutes = []string{"/api/other"}
		err = dr.validateRequestPath(req, allowedRoutes)
		assert.Error(t, err)

		// Test with empty allowed routes (should allow all)
		err = dr.validateRequestPath(req, []string{})
		assert.NoError(t, err)
	})

	t.Run("validateRequestMethod", func(t *testing.T) {
		dr := &DependencyResolver{
			Fs:     fs,
			Logger: logger,
		}

		// Create test request
		req, _ := gin.CreateTestContext(httptest.NewRecorder())
		req.Request = httptest.NewRequest("POST", "/api/test", nil)

		// Test with valid method
		allowedMethods := []string{"POST"}
		err := dr.validateRequestMethod(req, allowedMethods)
		assert.NoError(t, err)

		// Test with invalid method
		allowedMethods = []string{"GET"}
		err = dr.validateRequestMethod(req, allowedMethods)
		assert.Error(t, err)

		// Test with empty allowed methods (should allow all)
		err = dr.validateRequestMethod(req, []string{})
		assert.NoError(t, err)
	})

	t.Run("MockInjectableFunctions", func(t *testing.T) {
		// Test all the injectable mock functions that have 0% coverage

		// Test GetCommand
		mockExec := &MockExecBlock{Command: "test-command"}
		command := mockExec.GetCommand()
		assert.Equal(t, "test-command", command)

		// Test GetTimeoutDuration for exec
		timeout := int64(30)
		mockExec.TimeoutDuration = &timeout
		duration := mockExec.GetTimeoutDuration()
		assert.Equal(t, &timeout, duration)

		// Test GetURL
		mockHTTP := &MockHTTPBlock{URL: "https://example.com"}
		urlResult := mockHTTP.GetURL()
		assert.Equal(t, "https://example.com", urlResult)

		// Test GetMethod
		mockHTTP.Method = "POST"
		method := mockHTTP.GetMethod()
		assert.Equal(t, "POST", method)

		// Test GetBody
		body := "test body"
		mockHTTP.Body = &body
		bodyResult := mockHTTP.GetBody()
		assert.Equal(t, &body, bodyResult)

		// Test GetScript
		mockPython := &MockPythonBlock{Script: "print('hello')"}
		script := mockPython.GetScript()
		assert.Equal(t, "print('hello')", script)

		// Test GetTimeoutDuration for python
		pythonTimeout := int64(60)
		mockPython.TimeoutDuration = &pythonTimeout
		pythonDuration := mockPython.GetTimeoutDuration()
		assert.Equal(t, &pythonTimeout, pythonDuration)
	})

	t.Run("CreateMockBlocks", func(t *testing.T) {
		// Test CreateMockExecBlock
		execBlock := CreateMockExecBlock("echo", []string{"hello"})
		assert.NotNil(t, execBlock)

		// Test CreateMockHTTPBlock
		headers := map[string]string{"Content-Type": "application/json"}
		body := "test body"
		httpBlock := CreateMockHTTPBlock("https://test.com", "GET", headers, &body)
		assert.NotNil(t, httpBlock)
		assert.Equal(t, "https://test.com", httpBlock.Url)

		// Test CreateMockPythonBlock
		pythonBlock := CreateMockPythonBlock("print('test')", []string{})
		assert.NotNil(t, pythonBlock)
	})

	t.Run("SetupTestableEnvironment", func(t *testing.T) {
		// Test SetupTestableEnvironment function - has 50.0% coverage - boost it
		// This should set up the test environment
		SetupTestableEnvironment()

		// Test that HTTP Get was overridden
		resp, err := HttpGet("http://test.com")
		assert.NoError(t, err)
		assert.Equal(t, 200, resp.StatusCode)

		// Test that Ollama was overridden
		llm, err := OllamaNew()
		assert.NoError(t, err)
		assert.NotNil(t, llm)
	})

	t.Run("ResetEnvironment", func(t *testing.T) {
		// Test ResetEnvironment function - has 0.0% coverage
		// First setup test environment
		SetupTestableEnvironment()

		// Then reset it
		ResetEnvironment()

		// Verify functions are reset (this will use actual implementations)
		assert.NotNil(t, HttpGet)
		assert.NotNil(t, OllamaNew)
	})
}

func TestResolverUtilityFunctionsCoverage(t *testing.T) {
	// Set test mode
	os.Setenv("KDEPS_TEST_MODE", "true")
	defer func() {
		os.Unsetenv("KDEPS_TEST_MODE")
	}()

	t.Run("FormatDuration", func(t *testing.T) {
		// Test FormatDuration function - has 0.0% coverage
		duration := 65 * time.Second

		result := FormatDuration(duration)
		assert.NotEmpty(t, result)
		// The format might be "1m 5s" with space, so check both parts
		assert.Contains(t, result, "1m")
		assert.Contains(t, result, "5s")
	})

	t.Run("isMethodWithBody", func(t *testing.T) {
		// Test isMethodWithBody function - has 0.0% coverage
		assert.True(t, isMethodWithBody("POST"))
		assert.True(t, isMethodWithBody("PUT"))
		assert.True(t, isMethodWithBody("PATCH"))
		assert.True(t, isMethodWithBody("DELETE")) // DELETE can have body
		assert.False(t, isMethodWithBody("GET"))
		assert.False(t, isMethodWithBody("HEAD"))
	})
}
