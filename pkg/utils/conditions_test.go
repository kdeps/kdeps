package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldSkip(t *testing.T) {
	t.Run("NoConditions", func(t *testing.T) {
		conditions := []bool{}
		result := ShouldSkip(&conditions)
		assert.False(t, result, "Expected ShouldSkip to return false when there are no conditions")
	})

	t.Run("AllFalseConditions", func(t *testing.T) {
		conditions := []bool{false, false, false}
		result := ShouldSkip(&conditions)
		assert.False(t, result, "Expected ShouldSkip to return false when all conditions are false")
	})

	t.Run("SomeTrueConditions", func(t *testing.T) {
		conditions := []bool{false, true, false}
		result := ShouldSkip(&conditions)
		assert.True(t, result, "Expected ShouldSkip to return true when at least one condition is true")
	})

	t.Run("AllTrueConditions", func(t *testing.T) {
		conditions := []bool{true, true, true}
		result := ShouldSkip(&conditions)
		assert.True(t, result, "Expected ShouldSkip to return true when all conditions are true")
	})
}

func TestAllConditionsMet(t *testing.T) {
	t.Run("NoConditions", func(t *testing.T) {
		conditions := []bool{}
		result := AllConditionsMet(&conditions)
		assert.True(t, result, "Expected AllConditionsMet to return true when there are no conditions")
	})

	t.Run("AllTrueConditions", func(t *testing.T) {
		conditions := []bool{true, true, true}
		result := AllConditionsMet(&conditions)
		assert.True(t, result, "Expected AllConditionsMet to return true when all conditions are true")
	})

	t.Run("SomeFalseConditions", func(t *testing.T) {
		conditions := []bool{true, false, true}
		result := AllConditionsMet(&conditions)
		assert.False(t, result, "Expected AllConditionsMet to return false when at least one condition is false")
	})

	t.Run("AllFalseConditions", func(t *testing.T) {
		conditions := []bool{false, false, false}
		result := AllConditionsMet(&conditions)
		assert.False(t, result, "Expected AllConditionsMet to return false when all conditions are false")
	})
}
