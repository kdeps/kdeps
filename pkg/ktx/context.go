package ktx

import (
	"context"
	"fmt"
)

// ContextKey is a custom type for context keys to avoid key collisions
type ContextKey string

// Create adds a key-value pair to the context
func CreateContext(ctx context.Context, key ContextKey, value any) context.Context {
	return context.WithValue(ctx, key, value)
}

// Read retrieves a value from the context
func ReadContext(ctx context.Context, key ContextKey) (any, bool) {
	value, ok := ctx.Value(key).(any)
	return value, ok
}

// Update modifies an existing value in the context
func UpdateContext(ctx context.Context, key ContextKey, newValue any) context.Context {
	_, ok := ctx.Value(key).(any)
	if !ok {
		fmt.Println("Key not found in context")
		return ctx
	}
	return context.WithValue(ctx, key, newValue)
}

// Delete removes a key-value pair by returning a new context (contexts are immutable)
func DeleteContext(ctx context.Context) context.Context {
	return context.Background() // Returns a new empty context
}
