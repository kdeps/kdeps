package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAPIServerResponse(t *testing.T) {
	// Reset persistentErrors for a clean test state
	persistentErrors = nil

	t.Run("SuccessfulResponseWithoutErrors", func(t *testing.T) {
		// Reset persistentErrors before starting the test
		persistentErrors = nil

		response := NewAPIServerResponse(true, []any{"data1", "data2"}, 0, "")

		assert.True(t, response.Success, "Expected success to be true")
		assert.NotNil(t, response.Response, "Response block should not be nil")
		assert.Empty(t, *response.Errors, "Errors should be empty for successful response")
		assert.Equal(t, []any{"data1", "data2"}, response.Response.Data, "Expected response data to match input")
	})

	t.Run("ResponseWithError", func(t *testing.T) {
		// Reset persistentErrors before starting the test
		persistentErrors = nil

		response := NewAPIServerResponse(false, nil, 404, "Resource not found")

		assert.False(t, response.Success, "Expected success to be false")
		assert.NotNil(t, response.Errors, "Errors block should not be nil")
		assert.Len(t, *response.Errors, 1, "Expected one error in the persistentErrors slice")

		// Validate the error block
		errorBlock := (*response.Errors)[0]
		assert.Equal(t, 404, errorBlock.Code, "Expected error code to match")
		assert.Equal(t, "Resource not found", errorBlock.Message, "Expected error message to match")
	})

	t.Run("PersistentErrorStorage", func(t *testing.T) {
		// Reset persistentErrors before starting the test
		persistentErrors = nil

		// Add the first error
		NewAPIServerResponse(false, nil, 404, "Resource not found")

		// Add the second error
		NewAPIServerResponse(false, nil, 500, "Internal server error")

		// Validate persistent errors
		if assert.Len(t, persistentErrors, 2, "Expected two errors in the persistentErrors slice") {
			// Validate the second error block
			errorBlock := persistentErrors[1]
			assert.Equal(t, 500, errorBlock.Code, "Expected error code to match")
			assert.Equal(t, "Internal server error", errorBlock.Message, "Expected error message to match")
		}
	})

	t.Run("ClearPersistentErrors", func(t *testing.T) {
		// Manually clear persistentErrors
		persistentErrors = nil

		// Ensure the errors slice is empty
		assert.Empty(t, persistentErrors, "Persistent errors should be empty after reset")
	})
}
