/*
 * Copyright 2026 Kdeps, KvK 94834768
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agent

import (
	"context"
	"encoding/json"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

func rtAssistant(id string, argChars int) map[string]any {
	args := `{"file_path":"a.txt"}`
	if argChars > 0 {
		args = `{"file_path":"a.txt","content":"` + strings.Repeat("x", argChars) + `"}`
	}
	return map[string]any{
		"role":    RoleAssistant,
		"content": "",
		"tool_calls": []any{
			map[string]any{
				"id":   id,
				"type": "function",
				"function": map[string]any{
					"name":      "write_file",
					"arguments": args,
				},
			},
		},
	}
}

func rtTool(id string, resultChars int) map[string]any {
	return map[string]any{
		"role":         roleTool,
		"tool_call_id": id,
		"name":         "write_file",
		"content":      strings.Repeat("y", resultChars),
	}
}

func TestWindowToolHistory_UnderBudgetUnchanged(t *testing.T) {
	h := []map[string]any{
		{"role": RoleUser, "content": "do a thing"},
		rtAssistant("c1", 0),
		rtTool("c1", 50),
	}
	out, dropped := windowToolHistory(h, "")
	if dropped != 0 || len(out) != len(h) {
		t.Fatalf("expected no change, got dropped=%d len=%d", dropped, len(out))
	}
}

func TestWindowToolHistory_DropsOldestKeepsHeadAndRecent(t *testing.T) {
	h := []map[string]any{
		{"role": RoleUser, "content": "the question"},
	}
	// 20 round-trips, each ~2k tokens of result -> ~40k, well over the 24k budget.
	for i := range 20 {
		id := "c" + string(rune('A'+i))
		h = append(h, rtAssistant(id, 0), rtTool(id, 8*1024))
	}
	out, dropped := windowToolHistory(h, "")
	if dropped == 0 {
		t.Fatal("expected round-trips to be dropped")
	}

	// Head (the user question) must be preserved as message 0.
	if msgRole(out[0]) != RoleUser || out[0]["content"] != "the question" {
		t.Fatalf("head not preserved: %v", out[0])
	}

	// Result must be within budget.
	if got := msgsTokens(out, ""); got > toolLoopMessageBudget {
		t.Fatalf("windowed history %d tokens exceeds budget %d", got, toolLoopMessageBudget)
	}

	// Pairing must be valid: every tool message's tool_call_id must be answered
	// by a preceding assistant tool_calls entry.
	assertValidPairing(t, out)

	// The first kept round-trip carries the dropped-count note.
	firstRT := -1
	for i, m := range out {
		if isRoundTripStart(m) {
			firstRT = i
			break
		}
	}
	if firstRT < 0 {
		t.Fatal("no round-trip survived")
	}
	if note, _ := out[firstRT]["content"].(string); !strings.Contains(note, "dropped to fit") {
		t.Fatalf("first kept round-trip missing drop note: %q", note)
	}
}

func TestWindowToolHistory_AlwaysKeepsOneRoundTrip(t *testing.T) {
	h := []map[string]any{
		{"role": RoleUser, "content": "q"},
		rtAssistant("c1", 0), rtTool("c1", 120*1024), // each ~30k tokens, alone over budget
		rtAssistant("c2", 0), rtTool("c2", 120*1024),
	}
	out, dropped := windowToolHistory(h, "")
	if dropped != 1 {
		t.Fatalf("expected exactly 1 dropped, got %d", dropped)
	}
	assertValidPairing(t, out)
	// The most recent round-trip (c2) must survive even though it alone busts
	// the budget.
	found := false
	for _, m := range out {
		if m["tool_call_id"] == "c2" {
			found = true
		}
	}
	if !found {
		t.Fatal("most recent round-trip was dropped")
	}
}

func TestWindowToolHistory_TruncatesOldArgsKeepsValidJSON(t *testing.T) {
	h := []map[string]any{{"role": RoleUser, "content": "q"}}
	for i := range 12 {
		id := "c" + string(rune('A'+i))
		h = append(h, rtAssistant(id, 20*1024), rtTool(id, 1024)) // big args
	}
	out, dropped := windowToolHistory(h, "")
	if dropped == 0 {
		t.Fatal("expected drops")
	}

	// Find kept round-trip assistant messages in order.
	var kept []map[string]any
	for _, m := range out {
		if isRoundTripStart(m) {
			kept = append(kept, m)
		}
	}
	if len(kept) < 3 {
		t.Skipf("only %d kept; truncation test needs >=3", len(kept))
	}
	// All but the last windowKeepRecentRoundTrips must have truncated args, and
	// every args string must remain valid JSON.
	for i, m := range kept {
		raw := m["tool_calls"].([]any)
		fn := raw[0].(map[string]any)["function"].(map[string]any)
		args := fn["arguments"].(string)
		var parsed map[string]any
		if err := json.Unmarshal([]byte(args), &parsed); err != nil {
			t.Fatalf("kept round-trip %d has invalid JSON args: %v", i, err)
		}
		if _, ok := parsed["file_path"]; !ok {
			t.Fatalf("kept round-trip %d lost its identifying file_path field", i)
		}
		older := i < len(kept)-windowKeepRecentRoundTrips
		content, _ := parsed["content"].(string)
		if older && !strings.Contains(content, "chars omitted") {
			t.Fatalf("older kept round-trip %d args not truncated: %q", i, content)
		}
		if !older && strings.Contains(content, "chars omitted") {
			t.Fatalf("recent kept round-trip %d args wrongly truncated", i)
		}
	}
}

func TestWindowToolHistory_NoRoundTripsUnchanged(t *testing.T) {
	h := []map[string]any{
		{"role": RoleUser, "content": strings.Repeat("q ", 20*1024)},
		{"role": RoleAssistant, "content": strings.Repeat("a ", 20*1024)},
	}
	out, dropped := windowToolHistory(h, "")
	if dropped != 0 || len(out) != 2 {
		t.Fatalf("plain conversation must not be windowed: dropped=%d len=%d", dropped, len(out))
	}
}

func TestWindowToolHistory_Empty(t *testing.T) {
	out, dropped := windowToolHistory(nil, "")
	if dropped != 0 || out != nil {
		t.Fatalf("nil in, nil out expected")
	}
}

// capturingStreamer records the messages array sent on the most recent call and
// replays a fixed sequence of tool-call rounds.
type capturingStreamer struct {
	responses []mockStreamResponse
	callCount int
	lastMsgs  string
}

func (c *capturingStreamer) StreamChat(
	_ context.Context, cfg *domain.ChatConfig, w io.Writer,
) (string, []domain.StreamedToolCall, error) {
	c.lastMsgs = cfg.Messages
	if c.callCount >= len(c.responses) {
		return "", nil, nil
	}
	r := c.responses[c.callCount]
	c.callCount++
	_, _ = io.WriteString(w, r.content)
	return r.content, r.toolCalls, nil
}

// TestRunStreaming_ToolLoopHistoryStaysBounded verifies the mid-loop windowing
// is wired: a long tool loop where every result is large must not let the
// re-sent message array grow without bound.
func TestRunStreaming_ToolLoopHistoryStaysBounded(t *testing.T) {
	const rounds = 40
	responses := make([]mockStreamResponse, 0, rounds+1)
	for i := range rounds {
		responses = append(responses, mockStreamResponse{
			toolCalls: []domain.StreamedToolCall{{
				ID:        strconv.Itoa(i),
				Name:      "bigout",
				Arguments: `{"i":` + strconv.Itoa(i) + `}`,
			}},
		})
	}
	responses = append(responses, mockStreamResponse{content: "done"})

	cs := &capturingStreamer{responses: responses}
	eng := executor.NewEngine(nil)
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name:        "bigout",
		Description: "returns a large payload",
		Execute: func(map[string]any) (string, error) {
			return strings.Repeat("payload line\n", 4096), nil // ~48 KB per call
		},
	})
	loop := New(eng, newTestWorkflowForSession(), reg, Config{
		Model: "test", Streamer: cs, MaxToolRounds: rounds + 2,
	})

	var buf strings.Builder
	if _, err := loop.RunStreaming(context.Background(), "go", &buf); err != nil {
		t.Fatalf("RunStreaming: %v", err)
	}
	if cs.callCount < rounds {
		t.Fatalf("loop stopped early at %d rounds", cs.callCount)
	}

	sentTokens := countTokensSilent("", cs.lastMsgs)
	// Without windowing this would be ~rounds * (16 KB cap / 4) ~= 160k tokens.
	// With it, the array is held near toolLoopMessageBudget plus at most one
	// fresh round-trip of slack.
	if sentTokens > toolLoopMessageBudget*2 {
		t.Fatalf("in-flight history grew to %d tokens; windowing not bounding it", sentTokens)
	}
}

// assertValidPairing checks that every role:tool message is preceded (anywhere
// earlier) by an assistant message whose tool_calls include its tool_call_id.
func assertValidPairing(t *testing.T, msgs []map[string]any) {
	t.Helper()
	known := map[string]bool{}
	for _, m := range msgs {
		if isRoundTripStart(m) {
			for _, tc := range m["tool_calls"].([]any) {
				if id, ok := tc.(map[string]any)["id"].(string); ok {
					known[id] = true
				}
			}
		}
		if msgRole(m) == roleTool {
			id, _ := m["tool_call_id"].(string)
			if !known[id] {
				t.Fatalf("orphan tool message: tool_call_id %q has no preceding assistant call", id)
			}
		}
	}
}
