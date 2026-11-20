package llm

import (
	"testing"

	"github.com/tmc/langchaingo/llms"
)

func TestToolCallArgsBlankIsEmptyObject(t *testing.T) {
	for in, want := range map[string]string{"": "{}", "  \n": "{}", `{"a":1}`: `{"a":1}`} {
		if got := toolCallArgs(in); got != want {
			t.Errorf("toolCallArgs(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseToolCallPartsBlankArgs(t *testing.T) {
	parts := parseToolCallParts([]map[string]any{
		{"id": "c1", "type": "function", "function": map[string]any{"name": "list_files", "arguments": ""}},
	})
	if len(parts) != 1 {
		t.Fatalf("got %d parts", len(parts))
	}
	tc, ok := parts[0].(llms.ToolCall)
	if !ok || tc.FunctionCall.Arguments != "{}" {
		t.Fatalf("history kept blank arguments: %#v", parts[0])
	}
}
