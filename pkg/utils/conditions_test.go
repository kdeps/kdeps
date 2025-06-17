package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldSkip(t *testing.T) {

	t.Run("NoConditions", func(t *testing.T) {
		conditions := []any{}
		result := ShouldSkip(&conditions)
		assert.False(t, result)
	})

	t.Run("AllFalseConditions", func(t *testing.T) {
		conditions := []any{false, "false", false}
		assert.False(t, ShouldSkip(&conditions))
	})

	t.Run("SomeTrueConditions", func(t *testing.T) {
		conditions := []any{false, "true", false}
		assert.True(t, ShouldSkip(&conditions))
	})

	t.Run("AllTrueConditions", func(t *testing.T) {
		conditions := []any{true, "true", true}
		assert.True(t, ShouldSkip(&conditions))
	})

	t.Run("MixedInvalidConditions", func(t *testing.T) {
		conditions := []any{"maybe", 123, false}
		assert.False(t, ShouldSkip(&conditions))
	})
}

func TestAllConditionsMet(t *testing.T) {

	t.Run("NoConditions", func(t *testing.T) {
		conditions := []any{}
		assert.True(t, AllConditionsMet(&conditions))
	})

	t.Run("AllTrueConditions", func(t *testing.T) {
		conditions := []any{true, "true", true}
		assert.True(t, AllConditionsMet(&conditions))
	})

	t.Run("SomeFalseConditions", func(t *testing.T) {
		conditions := []any{true, "false", true}
		assert.False(t, AllConditionsMet(&conditions))
	})

	t.Run("AllFalseConditions", func(t *testing.T) {
		conditions := []any{"false", false, "false"}
		assert.False(t, AllConditionsMet(&conditions))
	})

	t.Run("MixedInvalidConditions", func(t *testing.T) {
		conditions := []any{true, "maybe", 123}
		assert.False(t, AllConditionsMet(&conditions))
	})
}

func TestShouldSkipAndAllConditionsMet(t *testing.T) {
	cases := []struct {
		name       string
		input      []interface{}
		wantSkip   bool
		wantAllMet bool
	}{
		{"all bool true", []interface{}{true, true}, true, true},
		{"mixed true string", []interface{}{false, "true"}, true, false},
		{"all false", []interface{}{false, false}, false, false},
		{"all string true", []interface{}{"true", "true"}, true, true},
		{"mixed false", []interface{}{true, "false"}, true, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldSkip(&tc.input); got != tc.wantSkip {
				t.Fatalf("ShouldSkip(%v) = %v, want %v", tc.input, got, tc.wantSkip)
			}
			if got := AllConditionsMet(&tc.input); got != tc.wantAllMet {
				t.Fatalf("AllConditionsMet(%v) = %v, want %v", tc.input, got, tc.wantAllMet)
			}
		})
	}
}
