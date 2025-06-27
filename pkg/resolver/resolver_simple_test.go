package resolver

import (
	"context"
	"testing"

	"github.com/kdeps/kdeps/pkg/schema"
	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"
)

func TestMapRoleToLLMMessageTypeCoverageBoost(t *testing.T) {
	// Test MapRoleToLLMMessageType function to improve coverage
	tests := []struct {
		role     string
		expected llms.ChatMessageType
	}{
		{"human", llms.ChatMessageTypeHuman},
		{"user", llms.ChatMessageTypeHuman},
		{"person", llms.ChatMessageTypeHuman},
		{"client", llms.ChatMessageTypeHuman},
		{"system", llms.ChatMessageTypeSystem},
		{"ai", llms.ChatMessageTypeAI},
		{"assistant", llms.ChatMessageTypeAI},
		{"bot", llms.ChatMessageTypeAI},
		{"chatbot", llms.ChatMessageTypeAI},
		{"llm", llms.ChatMessageTypeAI},
		{"function", llms.ChatMessageTypeFunction},
		{"action", llms.ChatMessageTypeFunction},
		{"tool", llms.ChatMessageTypeTool},
		{"unknown", llms.ChatMessageTypeGeneric},
		{"", llms.ChatMessageTypeGeneric},
		{"   ", llms.ChatMessageTypeGeneric},
	}

	for _, tt := range tests {
		result := MapRoleToLLMMessageType(tt.role)
		assert.Equal(t, tt.expected, result, "role: %s", tt.role)
	}
}

func TestSchemaVersionUsageBoost(t *testing.T) {
	// Ensure we're using schema.SchemaVersion as required
	ctx := context.Background()
	version := schema.SchemaVersion(ctx)
	assert.NotEmpty(t, version)
	assert.True(t, len(version) > 0)
}
