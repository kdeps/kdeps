package resolver

import (
	"context"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

// TestFormatDuration_Unit tests the FormatDuration function
func TestFormatDuration_Unit(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "seconds only",
			duration: 30 * time.Second,
			expected: "30s",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			expected: "2m 30s",
		},
		{
			name:     "hours, minutes, and seconds",
			duration: 1*time.Hour + 2*time.Minute + 30*time.Second,
			expected: "1h 2m 30s",
		},
		{
			name:     "exactly one hour",
			duration: 1 * time.Hour,
			expected: "1h 0m 0s",
		},
		{
			name:     "exactly one minute",
			duration: 1 * time.Minute,
			expected: "1m 0s",
		},
		{
			name:     "zero duration",
			duration: 0,
			expected: "0s",
		},
		{
			name:     "large duration",
			duration: 25*time.Hour + 30*time.Minute + 45*time.Second,
			expected: "25h 30m 45s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatDuration(tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDecodeErrorMessage_Unit tests the DecodeErrorMessage function
func TestDecodeErrorMessage_Unit(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected string
	}{
		{
			name:     "empty message",
			message:  "",
			expected: "",
		},
		{
			name:     "plain text message",
			message:  "This is a plain error message",
			expected: "This is a plain error message",
		},
		{
			name:     "base64 encoded message",
			message:  "VGhpcyBpcyBhIGJhc2U2NCBlbmNvZGVkIG1lc3NhZ2U=", // "This is a base64 encoded message"
			expected: "This is a base64 encoded message",
		},
		{
			name:     "invalid base64 (will return original)",
			message:  "not-valid-base64!@#",
			expected: "not-valid-base64!@#",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewTestLogger()
			result := DecodeErrorMessage(tt.message, logger)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDependencyResolver_ensureResponsePklFileNotExists_Unit tests the ensureResponsePklFileNotExists method
func TestDependencyResolver_ensureResponsePklFileNotExists_Unit(t *testing.T) {
	tests := []struct {
		name        string
		setupFS     func(fs afero.Fs, filepath string)
		expectError bool
	}{
		{
			name: "file does not exist - success",
			setupFS: func(fs afero.Fs, filepath string) {
				// Don't create the file
			},
			expectError: false,
		},
		{
			name: "file exists and gets removed - success",
			setupFS: func(fs afero.Fs, filepath string) {
				// Create the file
				afero.WriteFile(fs, filepath, []byte("existing content"), 0o644)
			},
			expectError: false,
		},
		{
			name: "directory with same name exists",
			setupFS: func(fs afero.Fs, filepath string) {
				// Create a directory with the same name
				fs.MkdirAll(filepath, 0o755)
			},
			expectError: false, // RemoveAll should handle directories too
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			logger := logging.NewTestLogger()
			responseFile := "/test/response.pkl"

			dr := &DependencyResolver{
				Fs:              fs,
				Logger:          logger,
				ResponsePklFile: responseFile,
			}

			tt.setupFS(fs, responseFile)

			err := dr.ensureResponsePklFileNotExists()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify file doesn't exist after the operation
				exists, _ := afero.Exists(fs, responseFile)
				assert.False(t, exists)
			}
		})
	}
}

// TestDependencyResolver_GetResourceFilePath_Unit tests the GetResourceFilePath method
func TestDependencyResolver_GetResourceFilePath_Unit(t *testing.T) {
	tests := []struct {
		name         string
		resourceType string
		actionDir    string
		requestID    string
		expectError  bool
		expectedPath string
	}{
		{
			name:         "llm resource type",
			resourceType: "llm",
			actionDir:    "/tmp/action",
			requestID:    "req-123",
			expectError:  false,
			expectedPath: "/tmp/action/llm/req-123__llm_output.pkl",
		},
		{
			name:         "client resource type",
			resourceType: "client",
			actionDir:    "/tmp/action",
			requestID:    "req-456",
			expectError:  false,
			expectedPath: "/tmp/action/client/req-456__client_output.pkl",
		},
		{
			name:         "exec resource type",
			resourceType: "exec",
			actionDir:    "/home/action",
			requestID:    "req-789",
			expectError:  false,
			expectedPath: "/home/action/exec/req-789__exec_output.pkl",
		},
		{
			name:         "python resource type",
			resourceType: "python",
			actionDir:    "/var/action",
			requestID:    "req-abc",
			expectError:  false,
			expectedPath: "/var/action/python/req-abc__python_output.pkl",
		},
		{
			name:         "invalid resource type",
			resourceType: "invalid",
			actionDir:    "/tmp/action",
			requestID:    "req-123",
			expectError:  true,
			expectedPath: "",
		},
		{
			name:         "empty resource type",
			resourceType: "",
			actionDir:    "/tmp/action",
			requestID:    "req-123",
			expectError:  true,
			expectedPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := &DependencyResolver{
				ActionDir: tt.actionDir,
				RequestID: tt.requestID,
			}

			result, err := dr.GetResourceFilePath(tt.resourceType)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPath, result)
			}
		})
	}
}

// TestFormatValue_Unit tests the FormatValue function
func TestFormatValue_Unit(t *testing.T) {
	tests := []struct {
		name     string
		value    interface{}
		contains []string // Check that the result contains these strings
	}{
		{
			name:     "nil value",
			value:    nil,
			contains: []string{"null"},
		},
		{
			name:     "string value",
			value:    "hello world",
			contains: []string{"hello world"},
		},
		{
			name:     "integer value",
			value:    42,
			contains: []string{"42"},
		},
		{
			name:     "boolean value",
			value:    true,
			contains: []string{"true"},
		},
		{
			name: "simple map",
			value: map[string]interface{}{
				"key1": "value1",
				"key2": 123,
			},
			contains: []string{"key1", "value1", "key2"},
		},
		{
			name: "interface map",
			value: map[interface{}]interface{}{
				"test": "data",
			},
			contains: []string{"new Mapping", "test"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValue(tt.value)
			assert.NotEmpty(t, result)

			for _, expectedContent := range tt.contains {
				assert.Contains(t, result, expectedContent)
			}
		})
	}
}

// TestFormatErrors_Unit tests the FormatErrors function
func TestFormatErrors_Unit(t *testing.T) {
	tests := []struct {
		name     string
		errors   *[]*apiserverresponse.APIServerErrorsBlock
		expected []string // Strings that should be in the result
	}{
		{
			name:     "nil errors",
			errors:   nil,
			expected: []string{},
		},
		{
			name:     "empty errors slice",
			errors:   &[]*apiserverresponse.APIServerErrorsBlock{},
			expected: []string{},
		},
		{
			name: "single error",
			errors: &[]*apiserverresponse.APIServerErrorsBlock{
				{
					Code:    400,
					Message: "Bad Request",
				},
			},
			expected: []string{"errors", "code = 400", "Bad Request"},
		},
		{
			name: "multiple errors",
			errors: &[]*apiserverresponse.APIServerErrorsBlock{
				{
					Code:    400,
					Message: "Bad Request",
				},
				{
					Code:    500,
					Message: "Internal Server Error",
				},
			},
			expected: []string{"errors", "code = 400", "Bad Request", "code = 500", "Internal Server Error"},
		},
		{
			name: "error with nil entry",
			errors: &[]*apiserverresponse.APIServerErrorsBlock{
				{
					Code:    404,
					Message: "Not Found",
				},
				nil, // This should be handled gracefully
				{
					Code:    403,
					Message: "Forbidden",
				},
			},
			expected: []string{"errors", "code = 404", "Not Found", "code = 403", "Forbidden"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewTestLogger()
			result := FormatErrors(tt.errors, logger)

			if len(tt.expected) == 0 {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				for _, expected := range tt.expected {
					assert.Contains(t, result, expected)
				}
			}
		})
	}
}

// TestFormatResponseMeta_Unit tests the FormatResponseMeta function
func TestFormatResponseMeta_Unit(t *testing.T) {
	tests := []struct {
		name      string
		requestID string
		meta      *apiserverresponse.APIServerResponseMetaBlock
		contains  []string
	}{
		{
			name:      "nil meta",
			requestID: "req-123",
			meta:      nil,
			contains:  []string{"meta", `requestID = "req-123"`},
		},
		{
			name:      "empty meta",
			requestID: "req-456",
			meta:      &apiserverresponse.APIServerResponseMetaBlock{},
			contains:  []string{"meta", `requestID = "req-456"`},
		},
		{
			name:      "meta with headers",
			requestID: "req-789",
			meta: &apiserverresponse.APIServerResponseMetaBlock{
				Headers: &map[string]string{
					"Content-Type": "application/json",
					"X-Custom":     "value",
				},
			},
			contains: []string{"meta", `requestID = "req-789"`, "Content-Type"},
		},
		{
			name:      "meta with properties",
			requestID: "req-abc",
			meta: &apiserverresponse.APIServerResponseMetaBlock{
				Properties: &map[string]string{
					"version": "1.0",
					"build":   "12345",
				},
			},
			contains: []string{"meta", `requestID = "req-abc"`, "version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatResponseMeta(tt.requestID, tt.meta)
			assert.NotEmpty(t, result)

			for _, expected := range tt.contains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

// TestFormatResponseData_Unit tests the FormatResponseData function
func TestFormatResponseData_Unit(t *testing.T) {
	tests := []struct {
		name     string
		response *apiserverresponse.APIServerResponseBlock
		expected string
	}{
		{
			name:     "nil response",
			response: nil,
			expected: "",
		},
		{
			name: "nil data",
			response: &apiserverresponse.APIServerResponseBlock{
				Data: nil,
			},
			expected: "",
		},
		{
			name: "empty data",
			response: &apiserverresponse.APIServerResponseBlock{
				Data: []interface{}{},
			},
			expected: "",
		},
		{
			name: "response with data",
			response: &apiserverresponse.APIServerResponseBlock{
				Data: []interface{}{
					"test data 1",
					"test data 2",
				},
			},
			expected: "response", // Should contain response block
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := logging.NewTestLogger()
			ctx := context.Background()
			evaluator, _ := pkl.NewEvaluator(ctx, func(options *pkl.EvaluatorOptions) {})
			defer evaluator.Close()
			result := FormatResponseData(ctx, tt.response, logger, evaluator)

			if tt.expected == "" {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, tt.expected)
			}
		})
	}
}

// TestDependencyResolver_GetCurrentTimestamp_Unit tests the GetCurrentTimestamp method error paths
func TestDependencyResolver_GetCurrentTimestamp_Unit(t *testing.T) {
	tests := []struct {
		name         string
		resourceID   string
		resourceType string
		setupDR      func() *DependencyResolver
		expectError  bool
	}{
		{
			name:         "invalid resource type",
			resourceID:   "test-resource",
			resourceType: "invalid-type",
			setupDR: func() *DependencyResolver {
				return &DependencyResolver{
					ActionDir: "/tmp/action",
					RequestID: "req-123",
				}
			},
			expectError: true,
		},
		{
			name:         "valid resource type but file doesn't exist",
			resourceID:   "test-resource",
			resourceType: "llm",
			setupDR: func() *DependencyResolver {
				return &DependencyResolver{
					Fs:        afero.NewMemMapFs(),
					ActionDir: "/tmp/action",
					RequestID: "req-123",
					Context:   context.Background(),
				}
			},
			expectError: true, // File doesn't exist, so should error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dr := tt.setupDR()

			_, err := dr.GetCurrentTimestamp(tt.resourceID, tt.resourceType)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGetResourceTimestamp_Unit tests the GetResourceTimestamp function
func TestGetResourceTimestamp_Unit(t *testing.T) {
	tests := []struct {
		name        string
		resourceID  string
		pklRes      interface{}
		expectError bool
	}{
		{
			name:        "unsupported PKL result type",
			resourceID:  "test-resource",
			pklRes:      "invalid-type",
			expectError: true,
		},
		{
			name:        "nil PKL result",
			resourceID:  "test-resource",
			pklRes:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := GetResourceTimestamp(tt.resourceID, tt.pklRes)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
