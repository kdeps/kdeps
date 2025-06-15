package ktx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestContextHelpersExtra(t *testing.T) {
	typeKey := ContextKey("foo")
	ctx := context.Background()

	// CreateContext adds value
	ctx2 := CreateContext(ctx, typeKey, 123)
	val, ok := ReadContext(ctx2, typeKey)
	require.True(t, ok)
	require.Equal(t, 123, val)

	// UpdateContext changes value
	ctx3 := UpdateContext(ctx2, typeKey, 456)
	v2, _ := ReadContext(ctx3, typeKey)
	require.Equal(t, 456, v2)

	// Update on missing key returns same ctx
	ctx4 := UpdateContext(ctx3, ContextKey("missing"), 1)
	require.Equal(t, ctx3, ctx4)

	// DeleteContext returns background (no value)
	ctx5 := DeleteContext(ctx3)
	_, ok = ReadContext(ctx5, typeKey)
	require.False(t, ok)
}
