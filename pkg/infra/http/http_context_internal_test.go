package http

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithDebugMode(t *testing.T) {
	t.Parallel()
	ctx := withDebugMode(context.Background(), true)
	assert.True(t, contextBoolValue(ctx))
}

func TestWithDebugMode_False(t *testing.T) {
	t.Parallel()
	ctx := withDebugMode(context.Background(), false)
	assert.False(t, contextBoolValue(ctx))
}

func TestWithTrustedProxies(t *testing.T) {
	t.Parallel()
	ctx := withTrustedProxies(context.Background(), []string{"10.0.0.1", "192.168.1.0/24"})
	val, ok := ctx.Value(TrustedProxiesKey).([]string)
	assert.True(t, ok)
	assert.Equal(t, []string{"10.0.0.1", "192.168.1.0/24"}, val)
}

func TestWithRequestID(t *testing.T) {
	t.Parallel()
	ctx := withRequestID(context.Background(), "req-abc")
	assert.Equal(t, "req-abc", contextStringValue(ctx, RequestIDKey))
}

func TestWithSessionIDContext(t *testing.T) {
	t.Parallel()
	ctx := withSessionIDContext(context.Background(), "sess-xyz")
	assert.Equal(t, "sess-xyz", contextStringValue(ctx, SessionIDKey))
}

func TestContextStringValue_Missing(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", contextStringValue(context.Background(), RequestIDKey))
}

func TestContextBoolValue_Missing(t *testing.T) {
	t.Parallel()
	assert.False(t, contextBoolValue(context.Background()))
}

func TestNewRequestID(t *testing.T) {
	t.Parallel()
	id1 := newRequestID()
	id2 := newRequestID()
	assert.NotEmpty(t, id1)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 36)
}
