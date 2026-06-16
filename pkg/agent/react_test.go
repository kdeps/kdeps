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
	"strings"
	"testing"
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
