// mock_llm.go
package resolver

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// MockLLM is a mock implementation of the LLM interface for testing.
type MockLLM struct {
	Response  string
	ToolCalls []llms.ToolCall
}

// GenerateContent simulates a successful LLM response.
func (m *MockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, opts ...llms.CallOption) (*llms.ContentResponse, error) {
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content:   m.Response,
				ToolCalls: m.ToolCalls,
			},
		},
	}, nil
}

func (m *MockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return m.Response, nil
}
