package llm

import (
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// The assistant turn's reasoning_content must survive history rebuilding, so it
// is ready to replay the moment the message type can carry it.
func TestBuildHistoryMessages_KeepsAssistantTurn(t *testing.T) {
	history := `[{"role":"assistant","content":"answer","reasoning_content":"why"}]`
	msgs := buildHistoryMessages(history)
	if len(msgs) != 1 || msgs[0].Role != llms.ChatMessageTypeAI {
		t.Fatalf("expected one assistant message, got %+v", msgs)
	}
}

// attachReasoning is the single wiring point for the echo; until langchaingo can
// carry the field it must be inert rather than corrupting the message.
func TestAttachReasoning_InertUntilSupported(t *testing.T) {
	msg := &llms.MessageContent{Role: llms.ChatMessageTypeAI, Parts: []llms.ContentPart{llms.TextContent{Text: "a"}}}
	before := len(msg.Parts)
	attachReasoning(msg, "reasoning")
	if len(msg.Parts) != before {
		t.Fatal("attachReasoning must not alter message parts while unsupported")
	}
	attachReasoning(nil, "reasoning") // must not panic
	if ReasoningEchoSupported() {
		t.Fatal("echo support should be false until langchaingo carries the field")
	}
}
