package ktx

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContextHelpers(t *testing.T) {
	base := context.Background()

	// create
	ctx := CreateContext(base, CtxKeyGraphID, "123")

	// read existing
	v, ok := ReadContext(ctx, CtxKeyGraphID)
	assert.True(t, ok)
	assert.Equal(t, "123", v)

	// read missing
	_, ok = ReadContext(ctx, CtxKeyAgentDir)
	assert.False(t, ok)

	// update value
	updated := UpdateContext(ctx, CtxKeyGraphID, "456")
	v2, _ := ReadContext(updated, CtxKeyGraphID)
	assert.Equal(t, "456", v2)

	// update missing key should not panic and returns same ctx
	same := UpdateContext(ctx, ContextKey("missing"), "x")
	assert.NotNil(t, same)

	// delete returns new background context (no values)
	blank := DeleteContext(updated)
	_, ok = ReadContext(blank, CtxKeyGraphID)
	assert.False(t, ok)
}
