package llm

import (
	"encoding/json"
	"testing"

	"github.com/tmc/langchaingo/llms"
)

// The assistant turn's reasoning_content must survive history rebuilding.
func TestBuildHistoryMessages_KeepsAssistantTurn(t *testing.T) {
	history := `[{"role":"assistant","content":"answer","reasoning_content":"why"}]`
	msgs := buildHistoryMessages(history)
	if len(msgs) != 1 || msgs[0].Role != llms.ChatMessageTypeAI {
		t.Fatalf("expected one assistant message, got %+v", msgs)
	}
}

// historyReasoning extracts each assistant turn's reasoning in order, so it
// lines up with the assistant messages in the request body.
func TestHistoryReasoning(t *testing.T) {
	history := `[
		{"role":"user","content":"q"},
		{"role":"assistant","content":"a1","reasoning_content":"r1"},
		{"role":"tool","content":"result"},
		{"role":"assistant","content":"a2","reasoning_content":"r2"}
	]`
	got := historyReasoning(history)
	if len(got) != 2 || got[0] != "r1" || got[1] != "r2" {
		t.Fatalf("historyReasoning = %v, want [r1 r2]", got)
	}
	// No reasoning anywhere -> nil, so the transport stays inert.
	if historyReasoning(`[{"role":"assistant","content":"a"}]`) != nil {
		t.Fatal("history without reasoning should yield nil")
	}
}

// injectReasoning writes reasoning_content onto assistant messages that lack it,
// in order, without touching other roles or existing values.
func TestInjectReasoning(t *testing.T) {
	body := `{"messages":[
		{"role":"system","content":"s"},
		{"role":"assistant","content":"a1"},
		{"role":"user","content":"u"},
		{"role":"assistant","content":"a2","reasoning_content":"keep"}
	]}`
	out, changed := injectReasoning([]byte(body), []string{"r1", "r2"})
	if !changed {
		t.Fatal("expected the body to be rewritten")
	}
	var payload map[string]any
	if err := json.Unmarshal(out, &payload); err != nil {
		t.Fatalf("rewritten body is not valid JSON: %v", err)
	}
	msgs := payload["messages"].([]any)
	a1 := msgs[1].(map[string]any)
	if a1["reasoning_content"] != "r1" {
		t.Fatalf("first assistant turn should get r1, got %v", a1["reasoning_content"])
	}
	a2 := msgs[3].(map[string]any)
	if a2["reasoning_content"] != "keep" {
		t.Fatalf("existing reasoning_content must not be overwritten, got %v", a2["reasoning_content"])
	}
	if sys := msgs[0].(map[string]any); sys["reasoning_content"] != nil {
		t.Fatal("non-assistant messages must be left alone")
	}
}

// A best-effort echo must never corrupt a body it cannot parse.
func TestInjectReasoning_LeavesBadBodyUnchanged(t *testing.T) {
	if _, changed := injectReasoning([]byte("not json"), []string{"r"}); changed {
		t.Fatal("unparsable body must be returned unchanged")
	}
}

func TestBackendRequiresReasoningEcho(t *testing.T) {
	if !backendRequiresReasoningEcho("deepseek") {
		t.Fatal("deepseek requires the reasoning echo")
	}
	for _, b := range []string{"openai", "anthropic", ""} {
		if backendRequiresReasoningEcho(b) {
			t.Errorf("%q should not require the reasoning echo", b)
		}
	}
}
