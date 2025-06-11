package resolver

import (
	"database/sql"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/kdeps/kdeps/pkg/utils"
	apiserverresponse "github.com/kdeps/schema/gen/api_server_response"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateResponsePklFile(t *testing.T) {
	// Initialize mock dependencies
	mockDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("failed to create mock database: %v", err)
	}
	defer mockDB.Close()

	resolver := &DependencyResolver{
		Logger:          logging.NewTestLogger(),
		Fs:              afero.NewMemMapFs(),
		DBs:             []*sql.DB{mockDB},
		ResponsePklFile: "response.pkl",
	}

	// Test cases
	t.Run("SuccessfulResponse", func(t *testing.T) {
		t.Skip("Skipping SuccessfulResponse due to external pkl binary dependency")
		response := utils.NewAPIServerResponse(true, []any{"data"}, 0, "")
		err := resolver.CreateResponsePklFile(response)
		assert.NoError(t, err)

		// Verify file was created
		exists, err := afero.Exists(resolver.Fs, resolver.ResponsePklFile)
		assert.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("NilResolver", func(t *testing.T) {
		var nilResolver *DependencyResolver
		err := nilResolver.CreateResponsePklFile(utils.NewAPIServerResponse(true, nil, 0, ""))
		assert.ErrorContains(t, err, "dependency resolver or database is nil")
	})

	t.Run("NilDatabase", func(t *testing.T) {
		resolver := &DependencyResolver{
			Logger: logging.NewTestLogger(),
			Fs:     afero.NewMemMapFs(),
			DBs:    nil,
		}
		err := resolver.CreateResponsePklFile(utils.NewAPIServerResponse(true, nil, 0, ""))
		assert.ErrorContains(t, err, "dependency resolver or database is nil")
	})
}

func TestEnsureResponsePklFileNotExists(t *testing.T) {
	dr := &DependencyResolver{
		Fs:     afero.NewMemMapFs(),
		Logger: logging.NewTestLogger(),
	}

	t.Run("FileDoesNotExist", func(t *testing.T) {
		err := dr.ensureResponsePklFileNotExists()
		assert.NoError(t, err)
	})

	t.Run("FileExists", func(t *testing.T) {
		// Create a test file
		err := afero.WriteFile(dr.Fs, dr.ResponsePklFile, []byte("test"), 0o644)
		require.NoError(t, err)

		err = dr.ensureResponsePklFileNotExists()
		assert.NoError(t, err)

		exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestBuildResponseSections(t *testing.T) {
	dr := &DependencyResolver{
		Fs:     afero.NewMemMapFs(),
		Logger: logging.NewTestLogger(),
	}

	t.Run("FullResponse", func(t *testing.T) {
		response := utils.NewAPIServerResponse(true, []any{"data1", "data2"}, 0, "")
		sections := dr.buildResponseSections("test-id", response)
		assert.NotEmpty(t, sections)
		assert.Contains(t, sections[0], "import")
		assert.Contains(t, sections[5], "success = true")
	})

	t.Run("ResponseWithError", func(t *testing.T) {
		response := utils.NewAPIServerResponse(false, nil, 404, "Resource not found")
		sections := dr.buildResponseSections("test-id", response)
		assert.NotEmpty(t, sections)
		assert.Contains(t, sections[0], "import")
		assert.Contains(t, sections[5], "success = false")
	})
}

func TestFormatResponseData(t *testing.T) {
	t.Run("NilResponse", func(t *testing.T) {
		result := formatResponseData(nil)
		assert.Empty(t, result)
	})

	t.Run("EmptyData", func(t *testing.T) {
		response := &apiserverresponse.APIServerResponseBlock{
			Data: []any{},
		}
		result := formatResponseData(response)
		assert.Empty(t, result)
	})

	t.Run("WithData", func(t *testing.T) {
		response := &apiserverresponse.APIServerResponseBlock{
			Data: []any{"test"},
		}
		result := formatResponseData(response)
		assert.Contains(t, result, "response")
		assert.Contains(t, result, "data")
	})
}

func TestFormatResponseMeta(t *testing.T) {
	t.Run("NilMeta", func(t *testing.T) {
		result := formatResponseMeta("test-id", nil)
		assert.Contains(t, result, "requestID = \"test-id\"")
	})

	t.Run("EmptyMeta", func(t *testing.T) {
		meta := &apiserverresponse.APIServerResponseMetaBlock{
			Headers:    &map[string]string{},
			Properties: &map[string]string{},
		}
		result := formatResponseMeta("test-id", meta)
		assert.Contains(t, result, "requestID = \"test-id\"")
	})

	t.Run("WithHeadersAndProperties", func(t *testing.T) {
		headers := map[string]string{"Content-Type": "application/json"}
		properties := map[string]string{"key": "value"}
		meta := &apiserverresponse.APIServerResponseMetaBlock{
			Headers:    &headers,
			Properties: &properties,
		}
		result := formatResponseMeta("test-id", meta)
		assert.Contains(t, result, "requestID = \"test-id\"")
		assert.Contains(t, result, "Content-Type")
		assert.Contains(t, result, "key")
	})
}

func TestFormatErrors(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("NilErrors", func(t *testing.T) {
		result := formatErrors(nil, logger)
		assert.Empty(t, result)
	})

	t.Run("EmptyErrors", func(t *testing.T) {
		errors := &[]*apiserverresponse.APIServerErrorsBlock{}
		result := formatErrors(errors, logger)
		assert.Empty(t, result)
	})

	t.Run("WithErrors", func(t *testing.T) {
		errors := &[]*apiserverresponse.APIServerErrorsBlock{
			{
				Code:    404,
				Message: "Resource not found",
			},
		}
		result := formatErrors(errors, logger)
		assert.Contains(t, result, "errors")
		assert.Contains(t, result, "code = 404")
		assert.Contains(t, result, "Resource not found")
	})
}

func TestDecodeErrorMessage(t *testing.T) {
	logger := logging.NewTestLogger()

	t.Run("EmptyMessage", func(t *testing.T) {
		result := decodeErrorMessage("", logger)
		assert.Empty(t, result)
	})

	t.Run("PlainMessage", func(t *testing.T) {
		message := "test message"
		result := decodeErrorMessage(message, logger)
		assert.Equal(t, message, result)
	})

	t.Run("Base64Message", func(t *testing.T) {
		message := "dGVzdCBtZXNzYWdl"
		result := decodeErrorMessage(message, logger)
		assert.Equal(t, "test message", result)
	})
}

func TestHandleAPIErrorResponse(t *testing.T) {
	t.Skip("Skipping HandleAPIErrorResponse tests due to external PKL dependency")
	dr := &DependencyResolver{
		Fs:            afero.NewMemMapFs(),
		Logger:        logging.NewTestLogger(),
		APIServerMode: true,
	}

	t.Run("ErrorResponse", func(t *testing.T) {
		fatal, err := dr.HandleAPIErrorResponse(404, "Resource not found", true)
		assert.NoError(t, err)
		assert.True(t, fatal)

		exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("NonAPIServerMode", func(t *testing.T) {
		dr.APIServerMode = false
		fatal, err := dr.HandleAPIErrorResponse(404, "Resource not found", true)
		assert.NoError(t, err)
		assert.True(t, fatal)

		exists, err := afero.Exists(dr.Fs, dr.ResponsePklFile)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}
