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
	eng := newTestEngine(map[string]string{"key": "val"}, nil)
	reg := tools.NewRegistry()
	loop := agent.New(eng, newTestWorkflow(), reg, agent.Config{})

	resp, err := loop.Run(context.Background(), "data")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == "" {
		t.Fatal("expected non-empty string for map result")
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
	if res[0].Chat.Scenario[0].Prompt != "Be concise." {
		t.Fatalf("unexpected scenario prompt: %q", res[0].Chat.Scenario[0].Prompt)
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
