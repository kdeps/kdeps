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

package agent_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

func newTestWorkflow() *domain.Workflow {
	return &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata: domain.WorkflowMetadata{
			Name:    "test",
			Version: "1.0.0",
		},
	}
}

func newTestEngine(result interface{}, err error) *executor.Engine {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return result, err
	})
	return eng
}

func TestNew_DefaultsFromEnv(t *testing.T) {
	t.Setenv("KDEPS_AGENT_MODEL", "mistral")
	t.Setenv("KDEPS_AGENT_BACKEND", "openai")
	t.Setenv("KDEPS_AGENT_BASE_URL", "http://example.com")

	eng := newTestEngine("hello", nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Just verify it constructs without panic; execution would fail because context cancelled.
	_ = loop
	_ = ctx
}

func TestNew_ExplicitConfig(t *testing.T) {
	eng := newTestEngine("response text", nil)
	reg := tools.NewRegistry()
	cfg := agent.Config{
		Model:        "llama3.2",
		Backend:      "ollama",
		BaseURL:      "http://localhost:11434",
		SystemPrompt: "You are helpful.",
		Role:         "user",
	}
	loop := agent.New(eng, newTestWorkflow(), reg, cfg)
	if loop == nil {
		t.Fatal("expected non-nil loop")
	}
}

func TestLoop_Run_StringResult(t *testing.T) {
	eng := newTestEngine("hello agent", nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{Model: "llama3.2"})

	resp, err := loop.Run(context.Background(), "say hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "hello agent" {
		t.Fatalf("expected 'hello agent', got %q", resp)
	}
}

func TestLoop_Run_NilResult(t *testing.T) {
	eng := newTestEngine(nil, nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	resp, err := loop.Run(context.Background(), "ping")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "" {
		t.Fatalf("expected empty string, got %q", resp)
	}
}

func TestLoop_Run_NonStringResult(t *testing.T) {
	// An unrecognized map (not the {"message": {"content": ...}} shape) returns empty.
	eng := newTestEngine(map[string]string{"key": "val"}, nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	resp, err := loop.Run(context.Background(), "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "" {
		t.Fatalf("expected empty string for unrecognized map result, got %q", resp)
	}
}

func TestLoop_Run_MessageMapResult(t *testing.T) {
	result := map[string]interface{}{
		"message": map[string]interface{}{
			"content": "hello from llm",
			"role":    "assistant",
		},
	}
	eng := newTestEngine(result, nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	resp, err := loop.Run(context.Background(), "hi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp != "hello from llm" {
		t.Fatalf("expected %q, got %q", "hello from llm", resp)
	}
}

func TestLoop_Run_EngineError(t *testing.T) {
	eng := newTestEngine(nil, errors.New("engine boom"))
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	_, err := loop.Run(context.Background(), "fail")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoop_Run_SystemPrompt(t *testing.T) {
	var capturedWorkflow *domain.Workflow
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		capturedWorkflow = wf
		return "ok", nil
	})
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{
		SystemPrompt: "Be concise.",
	})

	_, err := loop.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedWorkflow == nil {
		t.Fatal("expected captured workflow")
	}
	res := capturedWorkflow.Resources
	if len(res) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(res))
	}
	if res[0].Chat == nil {
		t.Fatal("expected chat resource")
	}
	if len(res[0].Chat.Scenario) != 1 {
		t.Fatalf("expected 1 scenario item, got %d", len(res[0].Chat.Scenario))
	}
	if !strings.Contains(res[0].Chat.Scenario[0].Prompt, "Be concise.") {
		t.Fatalf("system prompt not found in scenario: %q", res[0].Chat.Scenario[0].Prompt)
	}
}

func TestLoop_CompactWithLLM_TooFewTurns(t *testing.T) {
	eng := newTestEngine("summary text", nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	// Only 2 turns - below compactMinTurns threshold, should return empty.
	loop.Run(context.Background(), "q1") //nolint:errcheck
	loop.Run(context.Background(), "q2") //nolint:errcheck

	summary, err := loop.CompactWithLLM(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary != "" {
		t.Fatalf("expected empty summary for too-few turns, got %q", summary)
	}
}

func TestLoop_CompactWithLLM_SummarizesAndReplaces(t *testing.T) {
	const fakeSummary = "## Goal\nTest compaction.\n\n## Progress\n### Done\n- [x] Ran tests"
	var callCount int
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		callCount++
		return fakeSummary, nil
	})
	reg := tools.NewRegistry()
	// Use a tiny token budget so even short messages trigger compaction.
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{
		Model:              "llama3.2",
		CompactTokenBudget: 10, // very small: forces compaction after a few turns
	})

	// Add enough turns to trigger compaction (need compactMinTurns = 4 minimum).
	for range 10 {
		_, err := loop.Run(context.Background(), "what is 2+2?")
		if err != nil {
			t.Fatalf("Run error: %v", err)
		}
	}

	prevTurns := loop.Session().TurnCount()
	summary, err := loop.CompactWithLLM(context.Background())
	if err != nil {
		t.Fatalf("CompactWithLLM error: %v", err)
	}
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
	if summary != fakeSummary {
		t.Fatalf("expected %q, got %q", fakeSummary, summary)
	}
	afterTurns := loop.Session().TurnCount()
	if afterTurns >= prevTurns {
		t.Fatalf("expected fewer turns after compaction: before=%d after=%d", prevTurns, afterTurns)
	}
}

func TestLoop_CompactWithLLM_NoToolsInCompactionCall(t *testing.T) {
	var capturedWorkflow *domain.Workflow
	eng := executor.NewEngine(nil)
	callCount := 0
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		callCount++
		// Only capture the compaction call (the one after 10 Run calls).
		if callCount > 10 {
			capturedWorkflow = wf
		}
		return "summary output", nil
	})
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name:        "sometool",
		Description: "A tool",
		Parameters:  map[string]domain.ToolParam{},
		Execute:     func(_ map[string]interface{}) (string, error) { return "r", nil },
	})
	// Tiny budget to force compaction.
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{CompactTokenBudget: 10})

	for range 10 {
		loop.Run(context.Background(), "prompt") //nolint:errcheck
	}

	_, err := loop.CompactWithLLM(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedWorkflow == nil {
		t.Fatal("compaction workflow was not captured")
	}
	if len(capturedWorkflow.Resources[0].Chat.Tools) != 0 {
		t.Fatalf("expected 0 tools in compaction call, got %d", len(capturedWorkflow.Resources[0].Chat.Tools))
	}
}

func TestLoop_CompactWithLLM_FallsBackOnEngineError(t *testing.T) {
	callCount := 0
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		callCount++
		if callCount > 10 {
			return nil, errors.New("LLM offline")
		}
		return "answer", nil
	})
	reg := tools.NewRegistry()
	// MaxTurns=3 so truncation fallback has something to compact.
	// CompactTokenBudget=10 forces compaction trigger.
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{
		MaxTurns:           3,
		CompactTokenBudget: 10,
	})

	for range 10 {
		loop.Run(context.Background(), "question") //nolint:errcheck
	}

	// Inject extra messages directly to trigger truncation fallback.
	s := loop.Session()
	for range 5 {
		s.Append("extra", "response")
	}

	// CompactWithLLM should fall back to truncation, not return an error.
	_, err := loop.CompactWithLLM(context.Background())
	if err != nil {
		t.Fatalf("expected fallback (no error), got %v", err)
	}
}

func TestLoop_Run_ToolsWired(t *testing.T) {
	var capturedWorkflow *domain.Workflow
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		capturedWorkflow = wf
		return "done", nil
	})
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name:        "mytool",
		Description: "A test tool",
		Parameters:  map[string]domain.ToolParam{},
		Execute: func(_ map[string]interface{}) (string, error) {
			return "result", nil
		},
	})
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	_, err := loop.Run(context.Background(), "use tool")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(capturedWorkflow.Resources[0].Chat.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(capturedWorkflow.Resources[0].Chat.Tools))
	}
	if capturedWorkflow.Resources[0].Chat.Tools[0].Name != "mytool" {
		t.Fatalf("unexpected tool name: %q", capturedWorkflow.Resources[0].Chat.Tools[0].Name)
	}
}

func TestLoop_Run_AutoCompact_Fires(t *testing.T) {
	var callCount int
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		callCount++
		return "response", nil
	})
	reg := tools.NewRegistry()

	var autoCompactSummary string
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{
		AutoCompactThreshold: 1,  // fire on first turn that exceeds 1 token
		CompactTokenBudget:   10, // keep minimal recent context
	})
	loop.SetOnAutoCompact(func(summary string) {
		autoCompactSummary = summary
	})

	// First run: session empty, no auto-compact yet.
	_, err := loop.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Fill enough turns that shouldAutoCompact returns true on next Run.
	for range 5 {
		_, err = loop.Run(context.Background(), "more input")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// The callback should have fired during one of the later runs.
	if autoCompactSummary == "" {
		t.Fatal("expected auto-compact callback to fire")
	}
}

func TestLoop_SummarizeBranch_TooFewTurns(t *testing.T) {
	eng := newTestEngine("summary text", nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	// Only 2 turns - below compactMinTurns threshold.
	loop.Run(context.Background(), "q1") //nolint:errcheck
	loop.Run(context.Background(), "q2") //nolint:errcheck

	summary, err := loop.SummarizeBranch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary != "" {
		t.Fatalf("expected empty summary for too-few turns, got %q", summary)
	}
}

func TestLoop_SummarizeBranch_ReturnsPreambleAndContent(t *testing.T) {
	const fakeBody = "## Goal\nFix bug\n\n## Progress\n### Done\n- [x] Fixed it"
	eng := executor.NewEngine(nil)
	callCount := 0
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		callCount++
		return fakeBody, nil
	})
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	// Add enough turns to exceed compactMinTurns threshold.
	for range 8 {
		loop.Run(context.Background(), "work item") //nolint:errcheck
	}

	summary, err := loop.SummarizeBranch(context.Background())
	if err != nil {
		t.Fatalf("SummarizeBranch error: %v", err)
	}
	if summary == "" {
		t.Fatal("expected non-empty summary")
	}
	// Must contain the pi-style preamble.
	if !strings.Contains(summary, "summary of a branch") {
		t.Fatalf("expected preamble in summary, got %q", summary)
	}
	// Must use <summary> XML tags.
	if !strings.Contains(summary, "<summary>") {
		t.Fatalf("expected <summary> XML tag in branch summary, got %q", summary)
	}
	// Must contain the LLM body.
	if !strings.Contains(summary, fakeBody) {
		t.Fatalf("expected LLM body in summary, got %q", summary)
	}
}

func TestLoop_SummarizeBranch_EngineError(t *testing.T) {
	callCount := 0
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		callCount++
		if callCount > 8 {
			return nil, errors.New("LLM offline")
		}
		return "ok", nil
	})
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	for range 8 {
		loop.Run(context.Background(), "work") //nolint:errcheck
	}

	_, err := loop.SummarizeBranch(context.Background())
	if err == nil {
		t.Fatal("expected error when engine fails")
	}
}

func TestLoop_SummarizeBranch_NoToolsInCall(t *testing.T) {
	var capturedWorkflow *domain.Workflow
	callCount := 0
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		callCount++
		if callCount > 8 {
			capturedWorkflow = wf
		}
		return "branch summary body", nil
	})
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name:        "sometool",
		Description: "A tool",
		Parameters:  map[string]domain.ToolParam{},
		Execute:     func(_ map[string]interface{}) (string, error) { return "r", nil },
	})
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	for range 8 {
		loop.Run(context.Background(), "prompt") //nolint:errcheck
	}

	_, err := loop.SummarizeBranch(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedWorkflow == nil {
		t.Fatal("branch summary workflow not captured")
	}
	if len(capturedWorkflow.Resources[0].Chat.Tools) != 0 {
		t.Fatalf("expected 0 tools in branch summary call, got %d", len(capturedWorkflow.Resources[0].Chat.Tools))
	}
}

func TestLoop_SummarizeBranch_EmptyResult(t *testing.T) {
	// Tests the raw=="" path: engine returns empty string -> error
	callCount := 0
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		callCount++
		return "", nil // empty result for summary call
	})
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	for range 8 {
		loop.Run(context.Background(), "work") //nolint:errcheck
	}

	_, err := loop.SummarizeBranch(context.Background())
	if err == nil {
		t.Fatal("expected error for empty branch summary result")
	}
}

func TestLoop_Run_AutoCompact_Disabled(t *testing.T) {
	eng := newTestEngine("response", nil)
	reg := tools.NewRegistry()

	callbackFired := false
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{
		AutoCompactThreshold: -1, // disabled
	})
	loop.SetOnAutoCompact(func(_ string) {
		callbackFired = true
	})

	for range 10 {
		loop.Run(context.Background(), "question") //nolint:errcheck
	}

	if callbackFired {
		t.Fatal("expected auto-compact callback NOT to fire when disabled")
	}
}

func TestParseToolArguments_Map(t *testing.T) {
	t.Parallel()
	m, err := agent.ParseToolArguments(map[string]any{"key": "val"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["key"] != "val" {
		t.Fatalf("expected key=val, got %v", m)
	}
}

func TestParseToolArguments_JSONString(t *testing.T) {
	t.Parallel()
	m, err := agent.ParseToolArguments(`{"x":42}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if int(m["x"].(float64)) != 42 {
		t.Fatalf("expected x=42, got %v", m["x"])
	}
}

func TestParseToolArguments_Nil(t *testing.T) {
	t.Parallel()
	m, err := agent.ParseToolArguments(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(m) != 0 {
		t.Fatalf("expected empty map, got %v", m)
	}
}

func TestParseToolArguments_InvalidJSON(t *testing.T) {
	t.Parallel()
	_, err := agent.ParseToolArguments("not json")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseToolArguments_UnsupportedType(t *testing.T) {
	t.Parallel()
	_, err := agent.ParseToolArguments(123)
	if err == nil {
		t.Fatal("expected error for unsupported type")
	}
}

func TestLoop_Store_Nil(t *testing.T) {
	eng := newTestEngine("ok", nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})
	if loop.Store() != nil {
		t.Fatal("expected nil store when not configured")
	}
}

func TestLoop_IsStreaming_False(t *testing.T) {
	eng := newTestEngine("ok", nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})
	if loop.IsStreaming() {
		t.Fatal("expected IsStreaming() = false when no streamer configured")
	}
}

func TestLoop_Config(t *testing.T) {
	t.Parallel()
	cfg := agent.Config{Model: "gpt-4", MaxTurns: 10}
	eng := newTestEngine("ok", nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, cfg)
	if loop.Config().Model != "gpt-4" {
		t.Fatalf("expected Model=gpt-4, got %s", loop.Config().Model)
	}
	if loop.Config().MaxTurns != 10 {
		t.Fatalf("expected MaxTurns=10, got %d", loop.Config().MaxTurns)
	}
}
