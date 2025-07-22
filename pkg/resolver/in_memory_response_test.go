package resolver

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStoredAPIResponses(t *testing.T) {
	// Create a dependency resolver with in-memory filesystem
	dr := &DependencyResolver{
		Fs:                 afero.NewMemMapFs(),
		Logger:             logging.NewTestLogger(),
		storedAPIResponses: make(map[string]string),
		RequestID:          "test-request-123",
	}

	// Test initial state - no stored responses
	responses := dr.GetStoredAPIResponses()
	assert.Empty(t, responses, "Initial stored responses should be empty")

	// Test getting non-existent response
	response, exists := dr.GetStoredAPIResponse("nonexistent")
	assert.False(t, exists, "Non-existent response should not exist")
	assert.Empty(t, response, "Non-existent response should be empty")

	// Test storing a response
	testActionID := "test-action-1"
	testResponseJSON := `{"success":true,"response":{"data":"test data"},"meta":{"requestId":"test-request-123"}}`

	dr.storedAPIResponses[testActionID] = testResponseJSON

	// Test retrieving the stored response
	storedResponse, exists := dr.GetStoredAPIResponse(testActionID)
	assert.True(t, exists, "Stored response should exist")
	assert.Equal(t, testResponseJSON, storedResponse, "Stored response should match")

	// Test getting all stored responses
	allResponses := dr.GetStoredAPIResponses()
	assert.Len(t, allResponses, 1, "Should have one stored response")
	assert.Equal(t, testResponseJSON, allResponses[testActionID], "Stored response should match in all responses")

	// Test that returned map is a copy (modifications don't affect original)
	allResponses["new-key"] = "new-value"
	originalResponses := dr.GetStoredAPIResponses()
	assert.Len(t, originalResponses, 1, "Original responses should not be affected by modifications to returned copy")
	_, exists = originalResponses["new-key"]
	assert.False(t, exists, "New key should not exist in original responses")
}

func TestBuildResponseInMemory(t *testing.T) {
	t.Skip("Skipping BuildResponseInMemory test due to database dependency - requires integration testing")

	// This test would require a full database setup with memory/session/tool/item/agent readers
	// It's better tested as part of integration tests rather than unit tests

	// The function is tested implicitly when running the full resolver pipeline
	// with actual database connections and PKL resource processing
}

func TestNilStoredAPIResponsesHandling(t *testing.T) {
	// Test behavior when storedAPIResponses is nil (should not panic)
	dr := &DependencyResolver{
		Fs:                 afero.NewMemMapFs(),
		Logger:             logging.NewTestLogger(),
		storedAPIResponses: nil, // Explicitly set to nil
	}

	// Test getting all responses when map is nil
	responses := dr.GetStoredAPIResponses()
	require.NotNil(t, responses, "Should return an empty map, not nil")
	assert.Empty(t, responses, "Should return empty map when storedAPIResponses is nil")

	// Test getting specific response when map is nil
	response, exists := dr.GetStoredAPIResponse("any-id")
	assert.False(t, exists, "Should return false when storedAPIResponses is nil")
	assert.Empty(t, response, "Should return empty string when storedAPIResponses is nil")
}
