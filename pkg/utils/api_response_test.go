package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/pkg/utils"
)

func TestNewAPIServerResponse(t *testing.T) {
	// Test successful response
	response := utils.NewAPIServerResponse(true, []any{"data"}, 0, "", "test-request-id")
	require.NotNil(t, response)
	assert.True(t, *response.Success)
	assert.NotNil(t, response.Response)
	assert.NotNil(t, response.Errors)
	assert.Len(t, *response.Errors, 0)

	// Test error response
	response = utils.NewAPIServerResponse(false, nil, 400, "Bad Request", "test-request-id")
	require.NotNil(t, response)
	assert.False(t, *response.Success)
	assert.NotNil(t, response.Response)
	assert.NotNil(t, response.Errors)
	assert.Len(t, *response.Errors, 1)
	assert.Equal(t, 400, (*response.Errors)[0].Code)
	assert.Equal(t, "Bad Request", (*response.Errors)[0].Message)
}

func TestClearRequestErrors(t *testing.T) {
	// Create some errors first
	utils.NewAPIServerResponse(false, nil, 400, "Error 1", "test-request-id")
	utils.NewAPIServerResponse(false, nil, 500, "Error 2", "test-request-id")

	// Verify errors exist
	errors := utils.GetRequestErrors("test-request-id")
	assert.Len(t, errors, 2)

	// Clear errors
	utils.ClearRequestErrors("test-request-id")

	// Verify errors are cleared
	errors = utils.GetRequestErrors("test-request-id")
	assert.Len(t, errors, 0)
}

func TestGetRequestErrors(t *testing.T) {
	// Create some errors
	utils.NewAPIServerResponse(false, nil, 400, "Error 1", "test-request-id")
	utils.NewAPIServerResponse(false, nil, 500, "Error 2", "test-request-id")

	// Get errors
	errors := utils.GetRequestErrors("test-request-id")
	assert.Len(t, errors, 2)
	assert.Equal(t, 400, errors[0].Code)
	assert.Equal(t, "Error 1", errors[0].Message)
	assert.Equal(t, 500, errors[1].Code)
	assert.Equal(t, "Error 2", errors[1].Message)

	// Test with non-existent request ID
	errors = utils.GetRequestErrors("non-existent")
	assert.Len(t, errors, 0)
}
