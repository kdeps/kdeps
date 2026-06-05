package logging

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatSlice_NonEmpty(t *testing.T) {
	h := NewPrettyHandler(&bytes.Buffer{}, nil)
	var buf strings.Builder
	h.FormatAny(&buf, []interface{}{"item1", "item2"}, "  ")
	result := buf.String()
	assert.Contains(t, result, "[")
	assert.Contains(t, result, "- ")
	assert.Contains(t, result, "item1")
}

func TestFormatSlice_Empty(t *testing.T) {
	h := NewPrettyHandler(&bytes.Buffer{}, nil)
	var buf strings.Builder
	h.FormatAny(&buf, []interface{}{}, "  ")
	result := buf.String()
	assert.Contains(t, result, "[")
	assert.Contains(t, result, "]")
}
