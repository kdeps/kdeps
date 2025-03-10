package ktx

import (
	"context"
	"testing"
)

// Define test keys.
const (
	TestKey1 ContextKey = "testKey1"
	TestKey2 ContextKey = "testKey2"
)

// Test CreateContext and ReadContext.
func TestCreateAndReadContext(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

	ctx := context.Background()
	ctx = CreateContext(ctx, TestKey1, "ToBeDeleted")

	// Delete context.
	ctx = DeleteContext(ctx)

	// Verify it's empty.
	if value, found := ReadContext(ctx, TestKey1); found {
		t.Errorf("Expected key to be deleted, but got %v", value)
	}
}
