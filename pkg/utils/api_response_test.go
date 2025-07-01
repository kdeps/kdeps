package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAPIServerResponse(t *testing.T) {
	t.Run("SuccessfulResponseWithoutErrors", func(t *testing.T) {
		// Clear request errors before starting the test
		requestID := "test-request-success"
		ClearRequestErrors(requestID)

		response := NewAPIServerResponse(true, []any{"data1", "data2"}, 0, "", requestID)

		assert.True(t, response.Success, "Expected success to be true")
		assert.NotNil(t, response.Response, "Response block should not be nil")
		assert.Empty(t, *response.Errors, "Errors should be empty for successful response")
		assert.Equal(t, []any{"data1", "data2"}, response.Response.Data, "Expected response data to match input")
	})

	t.Run("ResponseWithError", func(t *testing.T) {
		// Clear request errors before starting the test
		requestID := "test-request-error"
		ClearRequestErrors(requestID)

		response := NewAPIServerResponse(false, nil, 404, "Resource not found", requestID)

		assert.False(t, response.Success, "Expected success to be false")
		assert.NotNil(t, response.Errors, "Errors block should not be nil")
		assert.Len(t, *response.Errors, 1, "Expected one error in the request errors slice")

		// Validate the error block
		errorBlock := (*response.Errors)[0]
		assert.Equal(t, 404, errorBlock.Code, "Expected error code to match")
		assert.Equal(t, "Resource not found", errorBlock.Message, "Expected error message to match")
	})

	t.Run("AccumulatedErrorsPerRequest", func(t *testing.T) {
		// Clear request errors before starting the test
		requestID := "test-request-accumulated"
		ClearRequestErrors(requestID)

		// Add the first error
		NewAPIServerResponse(false, nil, 404, "Resource not found", requestID)

		// Add the second error
		NewAPIServerResponse(false, nil, 500, "Internal server error", requestID)

		// Get current errors for the request
		errors := GetRequestErrors(requestID)
		assert.Len(t, errors, 2, "Expected two errors for the request")

		// Validate the first error block
		assert.Equal(t, 404, errors[0].Code, "Expected first error code to match")
		assert.Equal(t, "Resource not found", errors[0].Message, "Expected first error message to match")

		// Validate the second error block
		assert.Equal(t, 500, errors[1].Code, "Expected second error code to match")
		assert.Equal(t, "Internal server error", errors[1].Message, "Expected second error message to match")
	})

	t.Run("ErrorsIsolatedPerRequest", func(t *testing.T) {
		// Test that errors for different requests are isolated
		requestID1 := "test-request-1"
		requestID2 := "test-request-2"

		ClearRequestErrors(requestID1)
		ClearRequestErrors(requestID2)

		// Add errors to different requests
		NewAPIServerResponse(false, nil, 404, "Error for request 1", requestID1)
		NewAPIServerResponse(false, nil, 500, "Error for request 2", requestID2)

		// Verify each request has only its own errors
		errors1 := GetRequestErrors(requestID1)
		errors2 := GetRequestErrors(requestID2)

		assert.Len(t, errors1, 1, "Request 1 should have only one error")
		assert.Len(t, errors2, 1, "Request 2 should have only one error")

		assert.Equal(t, "Error for request 1", errors1[0].Message, "Request 1 should have its own error")
		assert.Equal(t, "Error for request 2", errors2[0].Message, "Request 2 should have its own error")
	})

	t.Run("ClearRequestErrors", func(t *testing.T) {
		requestID := "test-request-clear"

		// Add some errors
		NewAPIServerResponse(false, nil, 404, "Error to clear", requestID)

		// Verify errors exist
		errors := GetRequestErrors(requestID)
		assert.Len(t, errors, 1, "Should have one error before clearing")

		// Clear errors
		ClearRequestErrors(requestID)

		// Verify errors are cleared
		errors = GetRequestErrors(requestID)
		assert.Empty(t, errors, "Errors should be empty after clearing")
	})
}
