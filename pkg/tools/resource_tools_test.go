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

func testWorkflow(resources ...*domain.Resource) *domain.Workflow {
	return &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
		Resources:  resources,
	}
}

func newStubEngine(result interface{}, err error) *executor.Engine {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return result, err
	})
	return eng
}

func TestResourceToolDefs_Empty(t *testing.T) {
	wf := testWorkflow()
	eng := newStubEngine(nil, nil)
	defs := tools.ResourceToolDefs(wf, eng)
	if len(defs) != 0 {
		t.Fatalf("expected 0 tools, got %d", len(defs))
	}
}

func TestResourceToolDefs_NameAndDescription(t *testing.T) {
	wf := testWorkflow(&domain.Resource{ActionID: "myAction", Name: "My Resource"})
	eng := newStubEngine(nil, nil)
	defs := tools.ResourceToolDefs(wf, eng)
	if len(defs) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(defs))
	}
	if defs[0].Name != "myAction" {
		t.Errorf("expected name 'myAction', got %q", defs[0].Name)
	}
	if defs[0].Description != "My Resource" {
		t.Errorf("expected description 'My Resource', got %q", defs[0].Description)
	}
}

func TestResourceToolDefs_DescriptionFallback(t *testing.T) {
	wf := testWorkflow(&domain.Resource{ActionID: "noName"})
	eng := newStubEngine(nil, nil)
	defs := tools.ResourceToolDefs(wf, eng)
	if defs[0].Description != "Resource noName" {
		t.Errorf("unexpected description: %q", defs[0].Description)
	}
}

func TestResourceToolDefs_Execute_Success(t *testing.T) {
	wf := testWorkflow(&domain.Resource{ActionID: "doIt"})
	eng := newStubEngine("tool result", nil)
	defs := tools.ResourceToolDefs(wf, eng)

	result, err := defs[0].Execute(map[string]interface{}{"input": "hello"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "tool result" {
		t.Errorf("expected 'tool result', got %q", result)
	}
}

func TestResourceToolDefs_Execute_Error(t *testing.T) {
	wf := testWorkflow(&domain.Resource{ActionID: "boom"})
	eng := newStubEngine(nil, errors.New("engine failure"))
	defs := tools.ResourceToolDefs(wf, eng)

	_, err := defs[0].Execute(nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResourceToolDefs_MultipleResources(t *testing.T) {
	wf := testWorkflow(
		&domain.Resource{ActionID: "r1"},
		&domain.Resource{ActionID: "r2"},
		&domain.Resource{ActionID: "r3"},
	)
	eng := newStubEngine("ok", nil)
	defs := tools.ResourceToolDefs(wf, eng)
	if len(defs) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(defs))
	}
}

func TestMarshalResult_Nil(t *testing.T) {
	// marshalResult is internal; exercise via ResourceToolDefs Execute path.
	wf := testWorkflow(&domain.Resource{ActionID: "r"})
	eng := newStubEngine(nil, nil)
	defs := tools.ResourceToolDefs(wf, eng)
	result, err := defs[0].Execute(nil)
	if err != nil {
		t.Fatal(err)
	}
	if result != "" {
		t.Errorf("expected empty string for nil result, got %q", result)
	}
}
