package ktx

import (
	"context"
	"testing"
)

func TestContextHelpers(t *testing.T) {
	baseCtx := context.Background()
	key := ContextKey("user")

	// CreateContext should embed value
	ctx := CreateContext(baseCtx, key, "alice")
	if v, ok := ReadContext(ctx, key); !ok || v.(string) != "alice" {
		t.Fatalf("Create/ReadContext mismatch: got %v ok=%v", v, ok)
	}

	// Update existing key value
	ctx2 := UpdateContext(ctx, key, "bob")
	if v, _ := ReadContext(ctx2, key); v.(string) != "bob" {
		t.Errorf("UpdateContext failed; got %v", v)
	}

	// Update non-existent key returns same ctx (pointer equality) and prints message; we cannot capture stdout easily but ensure value unchanged
	otherKey := ContextKey("missing")
	ctx3 := UpdateContext(ctx2, otherKey, 123)
	if ctx3 != ctx2 {
		t.Errorf("UpdateContext on missing key should return original ctx")
	}

	// DeleteContext should return a fresh background context with no values.
	emptyCtx := DeleteContext(ctx2)
	if _, ok := ReadContext(emptyCtx, key); ok {
		t.Errorf("DeleteContext should remove values; got present")
	}
}
