package ktx

import (
	"context"
)

// ContextKey is a custom type for context keys to avoid key collisions.
type ContextKey string

// CreateContext adds a key-value pair to the context.
func CreateContext(ctx context.Context, key ContextKey, value any) context.Context {
	return context.WithValue(ctx, key, value)
}

// ReadContext retrieves a value from the context.
func ReadContext(ctx context.Context, key ContextKey) (any, bool) {
	value := ctx.Value(key) // No type assertion needed
	return value, value != nil
}

// UpdateContext modifies an existing value in the context.
func UpdateContext(ctx context.Context, key ContextKey, newValue any) context.Context {
	if ctx.Value(key) == nil {
		// Key not found in context
		return ctx
	}
	return context.WithValue(ctx, key, newValue)
}

// DeleteContext removes a key-value pair by returning a new context (contexts are immutable).
func DeleteContext(_ context.Context) context.Context {
	return context.Background() // Returns a new empty context
}
