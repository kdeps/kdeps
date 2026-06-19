// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agent

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

func TestExtractFinalAnswer(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		output string
		want   string
		ok     bool
	}{
		{
			name:   "standard",
			output: "Thought: done\nFinal Answer: 42",
			want:   "42",
			ok:     true,
		},
		{
			name:   "lowercase variant",
			output: "the final answer is: Paris",
			want:   "Paris",
			ok:     true,
		},
		{
			name:   "no final answer",
			output: "Thought: thinking\nAction: search",
			want:   "",
			ok:     false,
		},
		{
			name:   "answer is keyword",
			output: "the answer is: 7",
			want:   "7",
			ok:     true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			got, ok := extractFinalAnswer(c.output)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v", ok, c.ok)
			}
			if c.ok && got != c.want {
				t.Fatalf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestParseReactAction(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		output    string
		wantName  string
		wantInput string
		ok        bool
	}{
		{
			name:      "valid action",
			output:    "Thought: search\nAction: web_search\nAction Input: {\"query\": \"test\"}",
			wantName:  "web_search",
			wantInput: `{"query": "test"}`,
			ok:        true,
		},
		{
			name:      "case insensitive",
			output:    "Thought: x\nACTION: calc\nACTION INPUT: 2+2",
			wantName:  "calc",
			wantInput: "2+2",
			ok:        true,
		},
		{
			name:   "no action",
			output: "Final Answer: done",
			ok:     false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			name, input, ok := parseReactAction(c.output)
			if ok != c.ok {
				t.Fatalf("ok = %v, want %v (name=%q input=%q)", ok, c.ok, name, input)
			}
			if c.ok {
				if name != c.wantName {
					t.Fatalf("name = %q, want %q", name, c.wantName)
				}
				if input != c.wantInput {
					t.Fatalf("input = %q, want %q", input, c.wantInput)
				}
			}
		})
	}
}

func TestRunReact_FinalAnswerDirect(t *testing.T) {
	ms := &mockStreamer{
		responses: []mockStreamResponse{
			{content: "Thought: I know this.\nFinal Answer: Paris", toolCalls: nil},
		},
	}
	loop := newStreamingLoop(ms, 10)
	var buf bytes.Buffer
	result, err := loop.RunReact(context.Background(), "capital of France?", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "Paris" {
		t.Fatalf("expected 'Paris', got %q", result)
	}
}

func TestRunReact_ToolCallThenFinalAnswer(t *testing.T) {
	ms := &mockStreamer{
		responses: []mockStreamResponse{
			{content: "Thought: search\nAction: web_search\nAction Input: {\"query\":\"test\"}", toolCalls: nil},
			{content: "Thought: I know.\nFinal Answer: the result", toolCalls: nil},
		},
	}
	loop := newStreamingLoop(ms, 10)
	var buf bytes.Buffer
	result, err := loop.RunReact(context.Background(), "what is test?", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "result") {
		t.Fatalf("expected 'result' in response, got %q", result)
	}
}

func TestRunReact_MaxRoundsReturnsLastContent(t *testing.T) {
	toolOutput := "Thought: search\nAction: web_search\nAction Input: {}"
	ms := &mockStreamer{
		responses: []mockStreamResponse{
			{content: toolOutput, toolCalls: nil},
			{content: toolOutput, toolCalls: nil},
			{content: "fallback answer", toolCalls: nil},
		},
	}
	loop := newStreamingLoop(ms, 2)
	var buf bytes.Buffer
	result, err := loop.RunReact(context.Background(), "q?", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Fatal("expected non-empty result after max rounds")
	}
}

func TestBuildReactSystemPreamble_ContainsFormat(t *testing.T) {
	loop := newStreamingLoop(&mockStreamer{}, 5)
	preamble := loop.buildReactSystemPreamble()
	if !strings.Contains(preamble, "Thought:") {
		t.Errorf("preamble missing 'Thought:' format instruction, got: %q", preamble[:clampMax(200, len(preamble))])
	}
	if !strings.Contains(preamble, "Final Answer:") {
		t.Errorf("preamble missing 'Final Answer:' instruction")
	}
}

func clampMax(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestRunReact_StreamerError(t *testing.T) {
	// Covers lines 80-82: streamer error path.
	loop := newStreamingLoop(&errStreamer{err: errors.New("react stream error")}, 5)
	var buf bytes.Buffer
	_, err := loop.RunReact(context.Background(), "q?", &buf)
	if err == nil {
		t.Fatal("expected error from errStreamer")
	}
}

func TestBuildReactSystemPreamble_WithSystemPrompt(t *testing.T) {
	// Covers lines 131-133: extra != "" branch when SystemPrompt is set.
	eng := executor.NewEngine(nil)
	reg := tools.NewRegistry()
	loop := New(eng, newTestWorkflowForSession(), reg, Config{
		Model:         "test",
		Streamer:      &mockStreamer{responses: []mockStreamResponse{{content: "Final Answer: ok"}}},
		MaxToolRounds: 5,
		SystemPrompt:  "You are a helpful assistant",
	})
	preamble := loop.buildReactSystemPreamble()
	if !strings.Contains(preamble, "You are a helpful assistant") {
		t.Errorf("expected system prompt in preamble, got: %q", preamble[:clampMax(100, len(preamble))])
	}
}

func TestRunReact_AutoCompactFires(t *testing.T) {
	// Seeds many turns so shouldAutoCompact returns true before RunReact;
	// verifies that the onAutoCompact callback fires inside RunReact.
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		if len(wf.Resources) > 0 && wf.Resources[0].Chat.Prompt != "" {
			return "compaction summary", nil
		}
		return "", nil
	})
	ms := &mockStreamer{
		responses: []mockStreamResponse{{content: "Final Answer: done"}},
	}
	reg := tools.NewRegistry()
	loop := New(eng, newTestWorkflowForSession(), reg, Config{
		Model:                "test",
		Streamer:             ms,
		CompactTokenBudget:   1,
		AutoCompactThreshold: 1,
		MaxToolRounds:        5,
	})
	var callbackFired bool
	loop.SetOnAutoCompact(func(_ string) { callbackFired = true })
	for range compactMinTurns {
		loop.Session().Append(strings.Repeat("q", 300), strings.Repeat("a", 300))
	}

	var buf bytes.Buffer
	_, err := loop.RunReact(context.Background(), "hi", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !callbackFired {
		t.Error("expected onAutoCompact callback to fire inside RunReact")
	}
}

func TestBuildReactSystemPreamble_WithTools(t *testing.T) {
	// Covers the for-loop body: tool descriptions and name joining with ", ".
	eng := executor.NewEngine(nil)
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name:        "tool_alpha",
		Description: "does alpha",
		Parameters:  map[string]domain.ToolParam{},
		Execute:     func(_ map[string]interface{}) (string, error) { return "r", nil },
	})
	reg.Register(&tools.Tool{
		Name:        "tool_beta",
		Description: "does beta",
		Parameters:  map[string]domain.ToolParam{},
		Execute:     func(_ map[string]interface{}) (string, error) { return "r", nil },
	})
	loop := New(eng, newTestWorkflowForSession(), reg, Config{
		Model:         "test",
		Streamer:      &mockStreamer{responses: []mockStreamResponse{{content: "Final Answer: ok"}}},
		MaxToolRounds: 5,
	})
	preamble := loop.buildReactSystemPreamble()
	// Both tool names should appear in the preamble
	if !strings.Contains(preamble, "tool_alpha") {
		t.Errorf("expected tool_alpha in preamble, got: %q", preamble[:clampMax(200, len(preamble))])
	}
	if !strings.Contains(preamble, "tool_beta") {
		t.Errorf("expected tool_beta in preamble, got: %q", preamble[:clampMax(200, len(preamble))])
	}
	// The comma separator path should be exercised (tool_alpha, tool_beta)
	if !strings.Contains(preamble, "tool_alpha, tool_beta") && !strings.Contains(preamble, "tool_beta, tool_alpha") {
		t.Errorf("expected comma-joined tool names in preamble, got: %q", preamble[:clampMax(300, len(preamble))])
	}
}

func TestBuildReactChatConfig_WithSteps(t *testing.T) {
	// Covers lines 155-164: steps non-nil in buildReactChatConfig.
	loop := newStreamingLoop(&mockStreamer{}, 5)
	steps := []reactStep{
		{thought: "thought1", action: "act1", actionInput: "in1", observation: "obs1"},
		{thought: "thought2", action: "act2", actionInput: "in2", observation: "obs2"},
	}
	cfg := loop.buildReactChatConfig("question", "system preamble", steps)
	if cfg == nil {
		t.Fatal("expected non-nil config")
	}
	if len(cfg.Scenario) < 5 { // system + 2*(assistant+system)
		t.Errorf("expected at least 5 scenario items, got %d", len(cfg.Scenario))
	}
}

func TestBuildReactChatConfig_WithHistory(t *testing.T) {
	// Covers lines 168-170: history branch when session has turns.
	existing := NewSession(0)
	existing.Append("prior question", "prior answer")
	eng := executor.NewEngine(nil)
	reg := tools.NewRegistry()
	loop := New(eng, newTestWorkflowForSession(), reg, Config{
		Model:         "test",
		Streamer:      &mockStreamer{},
		ResumeSession: existing,
	})
	cfg := loop.buildReactChatConfig("new question", "preamble", nil)
	if cfg.Messages == "" {
		t.Error("expected non-empty messages from existing session history")
	}
}

func TestDispatchReactTool_NonJSONInput(t *testing.T) {
	// Covers lines 218-222: non-JSON input treated as {input: toolInput}.
	eng := executor.NewEngine(nil)
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name:        "test_tool",
		Description: "test",
		Parameters:  map[string]domain.ToolParam{},
		Execute:     func(_ map[string]interface{}) (string, error) { return "got it", nil },
	})
	loop := New(eng, newTestWorkflowForSession(), reg, Config{
		Model:    "test",
		Streamer: &mockStreamer{},
	})
	result := loop.dispatchReactTool("test_tool", "not json input")
	if result != "got it" {
		t.Errorf("expected 'got it', got %q", result)
	}
}

func TestDispatchReactTool_ToolError(t *testing.T) {
	// Covers lines 223-227: tool returns error.
	eng := executor.NewEngine(nil)
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name:        "fail_tool",
		Description: "test",
		Parameters:  map[string]domain.ToolParam{},
		Execute: func(_ map[string]interface{}) (string, error) {
			return "", errors.New("tool exploded")
		},
	})
	loop := New(eng, newTestWorkflowForSession(), reg, Config{
		Model:    "test",
		Streamer: &mockStreamer{},
	})
	result := loop.dispatchReactTool("fail_tool", `{"input":"x"}`)
	if !strings.Contains(result, "error") {
		t.Errorf("expected error in result, got %q", result)
	}
}

func TestRunReact_WithHistory(t *testing.T) {
	// Covers lines 168-170 via RunReact with existing session turns.
	existing := NewSession(0)
	existing.Append("previous q", "previous a")
	ms := &mockStreamer{
		responses: []mockStreamResponse{
			{content: "Final Answer: history tested", toolCalls: nil},
		},
	}
	eng := executor.NewEngine(nil)
	reg := tools.NewRegistry()
	loop := New(eng, newTestWorkflowForSession(), reg, Config{
		Model:         "test",
		Streamer:      ms,
		MaxToolRounds: 3,
		ResumeSession: existing,
	})
	var buf bytes.Buffer
	result, err := loop.RunReact(context.Background(), "new q?", &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "history tested" {
		t.Errorf("expected 'history tested', got %q", result)
	}
}
