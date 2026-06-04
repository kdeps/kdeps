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

func TestComponentToolDefs_EmptySlice(t *testing.T) {
	res := tools.ComponentToolDefs(nil, nil, nil)
	if len(res) != 0 {
		t.Errorf("expected 0 tools for nil slice, got %d", len(res))
	}
}

func TestComponentToolDefs_EmptyInput(t *testing.T) {
	res := tools.ComponentToolDefs([]*domain.Component{}, nil, nil)
	if len(res) != 0 {
		t.Errorf("expected 0 tools for empty slice, got %d", len(res))
	}
}

func TestComponentToolDefs_SkipsNilComponent(t *testing.T) {
	comps := []*domain.Component{
		nil,
		{Metadata: domain.ComponentMetadata{Name: "valid"}},
		nil,
	}
	res := tools.ComponentToolDefs(comps, nil, nil)
	if len(res) != 1 {
		t.Fatalf("expected 1 tool after skipping nil entries, got %d", len(res))
	}
	if res[0].Name != "valid" {
		t.Errorf("expected tool name 'valid', got %q", res[0].Name)
	}
}

func TestComponentToolDefs_WithInputs(t *testing.T) {
	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{
			Name:        "compinputs",
			Description: "Has inputs",
		},
		Interface: &domain.ComponentInterface{
			Inputs: []domain.ComponentInput{
				{Name: "topic", Type: "string", Required: true, Description: "The topic"},
				{Name: "count", Type: "integer", Required: false},
			},
		},
	}
	res := tools.ComponentToolDefs([]*domain.Component{comp}, &domain.Workflow{}, executor.NewEngine(nil))
	if len(res) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(res))
	}
	if res[0].Name != "compinputs" {
		t.Errorf("expected name 'compinputs', got %q", res[0].Name)
	}
	if res[0].Description != "Has inputs" {
		t.Errorf("expected description 'Has inputs', got %q", res[0].Description)
	}
	p, ok := res[0].Parameters["topic"]
	if !ok {
		t.Fatal("expected 'topic' parameter")
	}
	if p.Type != "string" {
		t.Errorf("expected type 'string', got %q", p.Type)
	}
	if !p.Required {
		t.Error("expected 'topic' to be required")
	}
	if p.Description != "The topic" {
		t.Errorf("expected description 'The topic', got %q", p.Description)
	}
	if _, ok = res[0].Parameters["count"]; !ok {
		t.Error("expected 'count' parameter")
	}
}

func TestComponentToolDefs_NilEng(t *testing.T) {
	comp := &domain.Component{Metadata: domain.ComponentMetadata{Name: "nocallback"}}
	res := tools.ComponentToolDefs([]*domain.Component{comp}, &domain.Workflow{}, nil)
	if len(res) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(res))
	}
	if res[0].Execute != nil {
		t.Error("expected Execute to be nil when eng is nil")
	}
}

func TestComponentToolDefs_NilWorkflow(t *testing.T) {
	comp := &domain.Component{Metadata: domain.ComponentMetadata{Name: "nocallback"}}
	eng := executor.NewEngine(nil)
	res := tools.ComponentToolDefs([]*domain.Component{comp}, nil, eng)
	if len(res) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(res))
	}
	if res[0].Execute != nil {
		t.Error("expected Execute to be nil when workflow is nil")
	}
}

func TestComponentToolDefs_Execute_Success(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return "component result", nil
	})

	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "testcomp", Description: "A test"},
	}
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "testwf", Version: "1.0"},
	}

	res := tools.ComponentToolDefs([]*domain.Component{comp}, wf, eng)
	if len(res) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(res))
	}
	if res[0].Execute == nil {
		t.Fatal("expected Execute to be non-nil")
	}

	result, err := res[0].Execute(map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "component result" {
		t.Errorf("expected 'component result', got %q", result)
	}
}

func TestComponentToolDefs_Execute_Error(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, errors.New("component failure")
	})

	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "failcomp"},
	}
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "testwf", Version: "1.0"},
	}

	res := tools.ComponentToolDefs([]*domain.Component{comp}, wf, eng)
	_, err := res[0].Execute(map[string]interface{}{"x": "y"})
	if err == nil {
		t.Fatal("expected error from engine, got nil")
	}
}

func TestComponentToolDefs_Execute_NilResult(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, nil //nolint:nilnil
	})

	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "nilresult"},
	}
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "testwf", Version: "1.0"},
	}

	res := tools.ComponentToolDefs([]*domain.Component{comp}, wf, eng)
	result, err := res[0].Execute(map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty string for nil result, got %q", result)
	}
}

func TestComponentToolDefs_Execute_StructResult(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return struct{ Key string }{"value"}, nil
	})

	comp := &domain.Component{
		Metadata: domain.ComponentMetadata{Name: "structresult"},
	}
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "testwf", Version: "1.0"},
	}

	res := tools.ComponentToolDefs([]*domain.Component{comp}, wf, eng)
	result, err := res[0].Execute(map[string]interface{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != `{"Key":"value"}` {
		t.Errorf("expected JSON result, got %q", result)
	}
}
