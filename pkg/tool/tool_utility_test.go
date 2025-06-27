package tool

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToolUtilityFunctions(t *testing.T) {
	// Test the utility functions that are simple interface methods

	t.Run("Scheme", func(t *testing.T) {
		// Test Scheme function
		reader := &PklResourceReader{}
		result := reader.Scheme()
		assert.Equal(t, "tool", result) // Should return "tool"
	})

	t.Run("IsGlobbable", func(t *testing.T) {
		// Test IsGlobbable function
		reader := &PklResourceReader{}
		result := reader.IsGlobbable()
		assert.Equal(t, false, result) // Should return false
	})

	t.Run("HasHierarchicalUris", func(t *testing.T) {
		// Test HasHierarchicalUris function
		reader := &PklResourceReader{}
		result := reader.HasHierarchicalUris()
		assert.Equal(t, false, result) // Should return false
	})

	t.Run("ListElements", func(t *testing.T) {
		// Test ListElements function
		reader := &PklResourceReader{}
		result, err := reader.ListElements(url.URL{})
		assert.Nil(t, result)  // Should return nil
		assert.NoError(t, err) // Should return nil error
	})
}
