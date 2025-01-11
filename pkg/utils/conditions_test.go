package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldSkip(t *testing.T) {
	t.Run("NoConditions", func(t *testing.T) {
		conditions := []any{}
		result := ShouldSkip(&conditions)
		assert.False(t, result, "Expected ShouldSkip to return false when there are no conditions")
	})

	t.Run("AllFalseConditions", func(t *testing.T) {
		conditions := []any{false, "false", false}
		result := ShouldSkip(&conditions)
		assert.False(t, result, "Expected ShouldSkip to return false when all conditions are false or 'false'")
	})

	t.Run("SomeTrueConditions", func(t *testing.T) {
		conditions := []any{false, "true", false}
		result := ShouldSkip(&conditions)
		assert.True(t, result, "Expected ShouldSkip to return true when at least one condition is true or 'true'")
	})

	t.Run("AllTrueConditions", func(t *testing.T) {
		conditions := []any{true, "true", true}
		result := ShouldSkip(&conditions)
		assert.True(t, result, "Expected ShouldSkip to return true when all conditions are true or 'true'")
	})

	t.Run("MixedInvalidConditions", func(t *testing.T) {
		conditions := []any{"maybe", 123, false}
		result := ShouldSkip(&conditions)
		assert.False(t, result, "Expected ShouldSkip to return false for unsupported types and false conditions")
	})
}

func TestAllConditionsMet(t *testing.T) {
	t.Run("NoConditions", func(t *testing.T) {
		conditions := []any{}
		result := AllConditionsMet(&conditions)
		assert.True(t, result, "Expected AllConditionsMet to return true when there are no conditions")
	})

	t.Run("AllTrueConditions", func(t *testing.T) {
		conditions := []any{true, "true", true}
		result := AllConditionsMet(&conditions)
		assert.True(t, result, "Expected AllConditionsMet to return true when all conditions are true or 'true'")
	})

	t.Run("SomeFalseConditions", func(t *testing.T) {
		conditions := []any{true, "false", true}
		result := AllConditionsMet(&conditions)
		assert.False(t, result, "Expected AllConditionsMet to return false when at least one condition is false or 'false'")
	})

	t.Run("AllFalseConditions", func(t *testing.T) {
		conditions := []any{"false", false, "false"}
		result := AllConditionsMet(&conditions)
		assert.False(t, result, "Expected AllConditionsMet to return false when all conditions are false or 'false'")
	})

	t.Run("MixedInvalidConditions", func(t *testing.T) {
		conditions := []any{true, "maybe", 123}
		result := AllConditionsMet(&conditions)
		assert.False(t, result, "Expected AllConditionsMet to return false for unsupported types or non-true conditions")
	})
}
