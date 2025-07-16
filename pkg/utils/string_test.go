package utils_test

import (
	"testing"

	"github.com/kdeps/kdeps/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func TestStringPtr(t *testing.T) {
	t.Run("ValidString", func(t *testing.T) {
		input := "test string"
		result := utils.StringPtr(input)
		assert.NotNil(t, result)
		assert.Equal(t, input, *result)
	})

	t.Run("EmptyString", func(t *testing.T) {
		input := ""
		result := utils.StringPtr(input)
		assert.NotNil(t, result)
		assert.Equal(t, input, *result)
	})
}

func TestBoolPtr(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		result := utils.BoolPtr(true)
		assert.NotNil(t, result)
		assert.True(t, *result)
	})

	t.Run("False", func(t *testing.T) {
		result := utils.BoolPtr(false)
		assert.NotNil(t, result)
		assert.False(t, *result)
	})
}

func TestContainsString(t *testing.T) {
	slice := []string{"one", "Two", "three"}
	assert.True(t, utils.ContainsString(slice, "Two"))
	assert.False(t, utils.ContainsString(slice, "two"))
	assert.True(t, utils.ContainsStringInsensitive(slice, "two"))
	assert.False(t, utils.ContainsStringInsensitive(slice, "four"))
}

func TestContainsStringInsensitive(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		target   string
		expected bool
	}{
		{
			name:     "StringFoundCaseInsensitive",
			slice:    []string{"Apple", "Banana", "Cherry"},
			target:   "apple",
			expected: true,
		},
		{
			name:     "StringNotFound",
			slice:    []string{"Apple", "Banana", "Cherry"},
			target:   "orange",
			expected: false,
		},
		{
			name:     "EmptySlice",
			slice:    []string{},
			target:   "apple",
			expected: false,
		},
		{
			name:     "MixedCase",
			slice:    []string{"ApPlE", "BaNaNa", "ChErRy"},
			target:   "apple",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ContainsStringInsensitive(tt.slice, tt.target)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSafeDerefString(t *testing.T) {
	var ptr *string
	assert.Empty(t, utils.SafeDerefString(ptr))
	val := "value"
	ptr = &val
	assert.Equal(t, "value", utils.SafeDerefString(ptr))
}

func TestSafeDerefBool(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		input := true
		result := utils.SafeDerefBool(&input)
		assert.True(t, result)
	})

	t.Run("False", func(t *testing.T) {
		input := false
		result := utils.SafeDerefBool(&input)
		assert.False(t, result)
	})

	t.Run("NilPointer", func(t *testing.T) {
		var input *bool
		result := utils.SafeDerefBool(input)
		assert.False(t, result)
	})
}

func TestSafeDerefSlice(t *testing.T) {
	t.Run("ValidSlice", func(t *testing.T) {
		input := []string{"a", "b", "c"}
		result := utils.SafeDerefSlice(&input)
		assert.Equal(t, input, result)
	})

	t.Run("NilPointer", func(t *testing.T) {
		var input *[]string
		result := utils.SafeDerefSlice(input)
		assert.Empty(t, result)
	})
}

func TestSafeDerefMap(t *testing.T) {
	t.Run("ValidMap", func(t *testing.T) {
		input := map[string]int{"a": 1, "b": 2}
		result := utils.SafeDerefMap(&input)
		assert.Equal(t, input, result)
	})

	t.Run("NilPointer", func(t *testing.T) {
		var input *map[string]int
		result := utils.SafeDerefMap(input)
		assert.Empty(t, result)
	})
}

func TestTruncateString(t *testing.T) {
	s := "abcdefghijklmnopqrstuvwxyz"
	assert.Equal(t, s, utils.TruncateString(s, len(s)))
	assert.Equal(t, "abc...", utils.TruncateString(s, 6))
}

func TestContainsStringInsensitiveExtra(t *testing.T) {
	slice := []string{"Hello", "World"}
	if !utils.ContainsStringInsensitive(slice, "hello") {
		t.Fatalf("expected to find 'hello' case-insensitively")
	}
	if utils.ContainsStringInsensitive(slice, "missing") {
		t.Fatalf("did not expect to find 'missing'")
	}
}

func TestPointerHelpers(t *testing.T) {
	s := "test"
	if *utils.StringPtr(s) != "test" {
		t.Fatalf("StringPtr failed")
	}
	b := false
	if *utils.BoolPtr(b) != false {
		t.Fatalf("BoolPtr failed")
	}
}

func TestStringHelpers(t *testing.T) {
	slice := []string{"apple", "Banana", "cherry"}

	if !utils.ContainsString(slice, "Banana") {
		t.Fatalf("expected exact match present")
	}
	if utils.ContainsString(slice, "banana") {
		t.Fatalf("ContainsString should be case sensitive")
	}
	if !utils.ContainsStringInsensitive(slice, "banana") {
		t.Fatalf("expected case-insensitive match")
	}

	// Ptr helpers
	s := "foo"
	sptr := utils.StringPtr(s)
	if *sptr != s {
		t.Fatalf("StringPtr failed")
	}
	b := true
	bptr := utils.BoolPtr(b)
	if *bptr != b {
		t.Fatalf("BoolPtr failed")
	}
}

func TestTruncateStringEdgeCases(t *testing.T) {
	cases := []struct {
		in   string
		max  int
		want string
	}{
		{"hello", 10, "hello"},    // shorter than max
		{"longstring", 4, "l..."}, // truncated with ellipsis
		{"abc", 2, "..."},         // max <3, replace with dots
	}
	for _, c := range cases {
		got := utils.TruncateString(c.in, c.max)
		if got != c.want {
			t.Fatalf("TruncateString(%q,%d)=%q want %q", c.in, c.max, got, c.want)
		}
	}
}

func TestSafeDerefHelpersExtra(t *testing.T) {
	str := "hi"
	if utils.SafeDerefString(nil) != "" || utils.SafeDerefString(&str) != "hi" {
		t.Fatalf("SafeDerefString failed")
	}
	b := true
	if utils.SafeDerefBool(nil) || !utils.SafeDerefBool(&b) {
		t.Fatalf("SafeDerefBool failed")
	}
}
