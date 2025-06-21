package utils_test

import (
	"testing"

	utilspkg "github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIServerResponse(t *testing.T) {
	// Reset PersistentErrors for a clean test state
	utilspkg.PersistentErrors = nil

	t.Run("SuccessfulResponseWithoutErrors", func(t *testing.T) {
		// Reset PersistentErrors before starting the test
		utilspkg.PersistentErrors = nil

		response := utilspkg.NewAPIServerResponse(true, []any{"data1", "data2"}, 0, "")

		assert.True(t, response.Success, "Expected success to be true")
		assert.NotNil(t, response.Response, "Response block should not be nil")
		assert.Empty(t, *response.Errors, "Errors should be empty for successful response")
		assert.Equal(t, []any{"data1", "data2"}, response.Response.Data, "Expected response data to match input")
	})

	t.Run("ResponseWithError", func(t *testing.T) {
		// Reset PersistentErrors before starting the test
		utilspkg.PersistentErrors = nil

		response := utilspkg.NewAPIServerResponse(false, nil, 404, "Resource not found")

		assert.False(t, response.Success, "Expected success to be false")
		assert.NotNil(t, response.Errors, "Errors block should not be nil")
		assert.Len(t, *response.Errors, 1, "Expected one error in the PersistentErrors slice")

		// Validate the error block
		errorBlock := (*response.Errors)[0]
		assert.Equal(t, 404, errorBlock.Code, "Expected error code to match")
		assert.Equal(t, "Resource not found", errorBlock.Message, "Expected error message to match")
	})

	t.Run("PersistentErrorStorage", func(t *testing.T) {
		// Reset PersistentErrors before starting the test
		utilspkg.PersistentErrors = nil

		// Add the first error
		utilspkg.NewAPIServerResponse(false, nil, 404, "Resource not found")

		// Add the second error
		utilspkg.NewAPIServerResponse(false, nil, 500, "Internal server error")

		// Validate persistent errors
		if assert.Len(t, utilspkg.PersistentErrors, 2, "Expected two errors in the PersistentErrors slice") {
			// Validate the second error block
			errorBlock := utilspkg.PersistentErrors[1]
			assert.Equal(t, 500, errorBlock.Code, "Expected error code to match")
			assert.Equal(t, "Internal server error", errorBlock.Message, "Expected error message to match")
		}
	})

	t.Run("ClearPersistentErrors", func(t *testing.T) {
		// Manually clear PersistentErrors
		utilspkg.PersistentErrors = nil

		// Ensure the errors slice is empty
		assert.Empty(t, utilspkg.PersistentErrors, "Persistent errors should be empty after reset")
	})
}

func TestNewAPIServerResponse_WithError(t *testing.T) {
	// Clear persistent errors before test
	utilspkg.PersistentErrors = nil

	response := utilspkg.NewAPIServerResponse(false, []any{"data"}, 400, "Bad Request")

	require.NotNil(t, response)
	require.False(t, response.Success)
	require.NotNil(t, response.Response)
	require.Equal(t, []any{"data"}, response.Response.Data)
	require.NotNil(t, response.Errors)
	require.Len(t, *response.Errors, 1)
	require.Equal(t, 400, (*response.Errors)[0].Code)
	require.Equal(t, "Bad Request", (*response.Errors)[0].Message)

	// Test that errors persist across multiple calls
	response2 := utilspkg.NewAPIServerResponse(true, []any{"success"}, 500, "Internal Error")
	require.True(t, response2.Success)
	require.Len(t, *response2.Errors, 2) // Should have accumulated errors
	require.Equal(t, 500, (*response2.Errors)[1].Code)
	require.Equal(t, "Internal Error", (*response2.Errors)[1].Message)
}
