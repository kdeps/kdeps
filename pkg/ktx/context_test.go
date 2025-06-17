package ktx

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"

	"github.com/stretchr/testify/require"
)

// Define test keys.
const (
	TestKey1 ContextKey = "testKey1"
	TestKey2 ContextKey = "testKey2"
)

// Test CreateContext and ReadContext.
func TestCreateAndReadContext(t *testing.T) {

	ctx := context.Background()

	// Create context with values.
	ctx = CreateContext(ctx, TestKey1, "Hello")
	ctx = CreateContext(ctx, TestKey2, 42)

	// Read and check values.
	if value, found := ReadContext(ctx, TestKey1); !found || value != "Hello" {
		t.Errorf("Expected 'Hello', got %v", value)
	}

	if value, found := ReadContext(ctx, TestKey2); !found || value != 42 {
		t.Errorf("Expected 42, got %v", value)
	}
}

// Test UpdateContext.
func TestUpdateContext(t *testing.T) {

	ctx := context.Background()
	ctx = CreateContext(ctx, TestKey1, "InitialValue")

	// Update value.
	ctx = UpdateContext(ctx, TestKey1, "UpdatedValue")

	// Verify updated value.
	if value, found := ReadContext(ctx, TestKey1); !found || value != "UpdatedValue" {
		t.Errorf("Expected 'UpdatedValue', got %v", value)
	}

	// Try updating a non-existing key (should not modify).
	ctx = UpdateContext(ctx, "nonExistingKey", "NewValue")
	if _, found := ReadContext(ctx, "nonExistingKey"); found {
		t.Errorf("Expected non-existing key to remain absent")
	}
}

// Test DeleteContext.
func TestDeleteContext(t *testing.T) {

	ctx := context.Background()
	ctx = CreateContext(ctx, TestKey1, "ToBeDeleted")

	// Delete context.
	ctx = DeleteContext(ctx)

	// Verify it's empty.
	if value, found := ReadContext(ctx, TestKey1); found {
		t.Errorf("Expected key to be deleted, but got %v", value)
	}
}

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
