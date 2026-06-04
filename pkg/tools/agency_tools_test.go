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

package tools_test

import (
	"errors"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

func testAgentWorkflow(name, desc, version string) *domain.Workflow {
	return &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:        name,
			Description: desc,
			Version:     version,
		},
	}
}

func TestAgentToolDef_Name(t *testing.T) {
	wf := testAgentWorkflow("myagent", "Does stuff", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) { return "ok", nil })

	tool := tools.AgentToolDef(wf, eng)
	if tool.Name != "myagent" {
		t.Errorf("expected name 'myagent', got %q", tool.Name)
	}
}

func TestAgentToolDef_DefaultName(t *testing.T) {
	wf := testAgentWorkflow("", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) { return "ok", nil })

	tool := tools.AgentToolDef(wf, eng)
	if tool.Name != "agent" {
		t.Errorf("expected default name 'agent', got %q", tool.Name)
	}
}

func TestAgentToolDef_DescriptionFromMetadata(t *testing.T) {
	wf := testAgentWorkflow("bot", "I am a bot", "2.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) { return "ok", nil })

	tool := tools.AgentToolDef(wf, eng)
	if tool.Description != "I am a bot" {
		t.Errorf("expected 'I am a bot', got %q", tool.Description)
	}
}

func TestAgentToolDef_DefaultDescription(t *testing.T) {
	wf := testAgentWorkflow("myagent", "", "1.2.3")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) { return "ok", nil })

	tool := tools.AgentToolDef(wf, eng)
	if tool.Description == "" {
		t.Error("expected non-empty default description")
	}
}

func TestAgentToolDef_InputParam(t *testing.T) {
	wf := testAgentWorkflow("bot", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) { return "ok", nil })

	tool := tools.AgentToolDef(wf, eng)
	p, ok := tool.Parameters["input"]
	if !ok {
		t.Fatal("expected 'input' parameter")
	}
	if !p.Required {
		t.Error("expected input param to be required")
	}
}

func TestAgentToolDef_Execute_Success(t *testing.T) {
	wf := testAgentWorkflow("bot", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return "agent response", nil
	})

	tool := tools.AgentToolDef(wf, eng)
	result, err := tool.Execute(map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "agent response" {
		t.Errorf("expected 'agent response', got %q", result)
	}
}

func TestAgentToolDef_Execute_Error(t *testing.T) {
	wf := testAgentWorkflow("bot", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, errors.New("agent failure")
	})

	tool := tools.AgentToolDef(wf, eng)
	_, err := tool.Execute(map[string]interface{}{"input": "fail"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// ---- marshalResult branches via AgentToolDef ----

func TestAgentToolDef_Execute_NilResult(t *testing.T) {
	wf := testAgentWorkflow("bot", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, nil //nolint:nilnil
	})

	tool := tools.AgentToolDef(wf, eng)
	result, err := tool.Execute(map[string]interface{}{"input": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for nil result, got %q", result)
	}
}

func TestAgentToolDef_Execute_StructResult(t *testing.T) {
	wf := testAgentWorkflow("bot", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return struct{ Key string }{"value"}, nil
	})

	tool := tools.AgentToolDef(wf, eng)
	result, err := tool.Execute(map[string]interface{}{"input": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := `{"Key":"value"}`
	if result != expected {
		t.Errorf("expected JSON %q, got %q", expected, result)
	}
}

func TestAgentToolDef_Execute_UnmarshalableResult(t *testing.T) {
	wf := testAgentWorkflow("bot", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return make(chan int), nil
	})

	tool := tools.AgentToolDef(wf, eng)
	result, err := tool.Execute(map[string]interface{}{"input": "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == "" {
		t.Error("expected non-empty fallback string for unmarshalable result")
	}
}

// ---- AgentToolDefWithName ----

func TestAgentToolDefWithName_CustomNameAndDesc(t *testing.T) {
	wf := testAgentWorkflow("inner", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return "ok", nil
	})

	tool := tools.AgentToolDefWithName("agency_name", "Custom description", wf, eng)
	if tool.Name != "agency_name" {
		t.Errorf("expected name 'agency_name', got %q", tool.Name)
	}
	if tool.Description != "Custom description" {
		t.Errorf("expected 'Custom description', got %q", tool.Description)
	}
}

func TestAgentToolDefWithName_InputParam(t *testing.T) {
	wf := testAgentWorkflow("inner", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return "ok", nil
	})

	tool := tools.AgentToolDefWithName("agency", "", wf, eng)
	p, ok := tool.Parameters["input"]
	if !ok {
		t.Fatal("expected 'input' parameter")
	}
	if !p.Required {
		t.Error("expected input param to be required")
	}
}

func TestAgentToolDefWithName_Execute_Success(t *testing.T) {
	wf := testAgentWorkflow("inner", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return "agency response", nil
	})

	tool := tools.AgentToolDefWithName("agency", "desc", wf, eng)
	result, err := tool.Execute(map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "agency response" {
		t.Errorf("expected 'agency response', got %q", result)
	}
}

func TestAgentToolDefWithName_Execute_Error(t *testing.T) {
	wf := testAgentWorkflow("inner", "", "1.0.0")
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, errors.New("agency failure")
	})

	tool := tools.AgentToolDefWithName("agency", "desc", wf, eng)
	_, err := tool.Execute(map[string]interface{}{"input": "fail"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
