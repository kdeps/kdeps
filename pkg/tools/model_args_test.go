package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestModelArgs(t *testing.T) {
	bg := make(chan struct{})
	got := ModelArgs(map[string]any{
		"query": "x", "_ctx": t.Context(), "_bg_ch": (<-chan struct{})(bg), "_interactive": true, "other": t.Context(),
	})
	assert.Equal(t, map[string]any{"query": "x"}, got)
	assert.Empty(t, ModelArgs(nil))
}
