package ktx_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ktx "github.com/kdeps/kdeps/pkg/ktx"
)

// Define test keys.
const (
	TestKey1 ktx.ContextKey = "testKey1"
	TestKey2 ktx.ContextKey = "testKey2"
)

// Test CreateContext and ReadContext.
func TestCreateAndReadContext(t *testing.T) {
	ctx := context.Background()

	// Create context with values.
	ctx = ktx.CreateContext(ctx, TestKey1, "Hello")
	ctx = ktx.CreateContext(ctx, TestKey2, 42)

	// Read and check values.
	if value, found := ktx.ReadContext(ctx, TestKey1); !found || value != "Hello" {
		t.Errorf("Expected 'Hello', got %v", value)
	}

	if value, found := ktx.ReadContext(ctx, TestKey2); !found || value != 42 {
		t.Errorf("Expected 42, got %v", value)
	}
}

// Test UpdateContext.
func TestUpdateContext(t *testing.T) {
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, TestKey1, "InitialValue")

	// Update value.
	ctx = ktx.UpdateContext(ctx, TestKey1, "UpdatedValue")

	// Verify updated value.
	if value, found := ktx.ReadContext(ctx, TestKey1); !found || value != "UpdatedValue" {
		t.Errorf("Expected 'UpdatedValue', got %v", value)
	}

	// Try updating a non-existing key (should not modify).
	ctx = ktx.UpdateContext(ctx, "nonExistingKey", "NewValue")
	if _, found := ktx.ReadContext(ctx, "nonExistingKey"); found {
		t.Errorf("Expected non-existing key to remain absent")
	}
}

// Test DeleteContext.
func TestDeleteContext(t *testing.T) {
	ctx := context.Background()
	ctx = ktx.CreateContext(ctx, TestKey1, "ToBeDeleted")

	// Delete context.
	ctx = ktx.DeleteContext(ctx)

	// Verify it's empty.
	if value, found := ktx.ReadContext(ctx, TestKey1); found {
		t.Errorf("Expected key to be deleted, but got %v", value)
	}
}

func TestContextHelpers(t *testing.T) {
	base := context.Background()

	// create
	ctx := ktx.CreateContext(base, ktx.CtxKeyGraphID, "123")

	// read existing
	v, ok := ktx.ReadContext(ctx, ktx.CtxKeyGraphID)
	assert.True(t, ok)
	assert.Equal(t, "123", v)

	// read missing
	_, ok = ktx.ReadContext(ctx, ktx.CtxKeyAgentDir)
	assert.False(t, ok)

	// update value
	updated := ktx.UpdateContext(ctx, ktx.CtxKeyGraphID, "456")
	v2, _ := ktx.ReadContext(updated, ktx.CtxKeyGraphID)
	assert.Equal(t, "456", v2)

	// update missing key should not panic and returns same ctx
	same := ktx.UpdateContext(ctx, ktx.ContextKey("missing"), "x")
	assert.NotNil(t, same)

	// delete returns new background context (no values)
	blank := ktx.DeleteContext(updated)
	_, ok = ktx.ReadContext(blank, ktx.CtxKeyGraphID)
	assert.False(t, ok)
}

func TestContextHelpersExtra(t *testing.T) {
	typeKey := ktx.ContextKey("foo")
	ctx := context.Background()

	// CreateContext adds value
	ctx2 := ktx.CreateContext(ctx, typeKey, 123)
	val, ok := ktx.ReadContext(ctx2, typeKey)
	require.True(t, ok)
	require.Equal(t, 123, val)

	// UpdateContext changes value
	ctx3 := ktx.UpdateContext(ctx2, typeKey, 456)
	v2, _ := ktx.ReadContext(ctx3, typeKey)
	require.Equal(t, 456, v2)

	// Update on missing key returns same ctx
	ctx4 := ktx.UpdateContext(ctx3, ktx.ContextKey("missing"), 1)
	require.Equal(t, ctx3, ctx4)

	// DeleteContext returns background (no value)
	ctx5 := ktx.DeleteContext(ctx3)
	_, ok = ktx.ReadContext(ctx5, typeKey)
	require.False(t, ok)
}
