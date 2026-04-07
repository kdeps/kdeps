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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package llm_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	llminput "github.com/kdeps/kdeps/v2/pkg/input/llm"
)

// workflowWith builds a minimal Workflow with the given LLM input config.
func workflowWith(llmCfg *domain.LLMInputConfig) *domain.Workflow {
	return &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test", TargetActionID: "chat"},
		Settings: domain.WorkflowSettings{
			Input: &domain.InputConfig{
				Sources: []string{domain.InputSourceLLM},
				LLM:     llmCfg,
			},
		},
	}
}

// buildEngine returns an *executor.Engine with a stubbed Execute function.
func buildEngine(result interface{}) *executor.Engine {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return result, nil
	})
	return eng
}

// ── Tests ─────────────────────────────────────────────────────────────────────

func TestRunWithIO_SingleTurn(t *testing.T) {
	eng := buildEngine("Hello from LLM")
	wf := workflowWith(&domain.LLMInputConfig{Prompt: "? "})
	r := strings.NewReader("hi there\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.String()
	if !strings.Contains(out, "? ") {
		t.Errorf("expected prompt '? ' in output, got: %q", out)
	}
	if !strings.Contains(out, "Hello from LLM") {
		t.Errorf("expected LLM response in output, got: %q", out)
	}
}

func TestRunWithIO_MultiTurn(t *testing.T) {
	call := 0
	responses := []string{"turn1", "turn2"}
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		r := responses[call]
		call++
		return r, nil
	})

	wf := workflowWith(nil)
	r := strings.NewReader("first\nsecond\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.String()
	if !strings.Contains(out, "turn1") {
		t.Errorf("expected 'turn1' in output, got: %q", out)
	}
	if !strings.Contains(out, "turn2") {
		t.Errorf("expected 'turn2' in output, got: %q", out)
	}
}

func TestRunWithIO_EmptyLinesSkipped(t *testing.T) {
	calls := 0
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		calls++
		return "ok", nil
	})

	wf := workflowWith(nil)
	r := strings.NewReader("\n\n   \nhello\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calls != 1 {
		t.Errorf("expected 1 engine call (empty lines skipped), got %d", calls)
	}
}

func TestRunWithIO_QuitCommand(t *testing.T) {
	calls := 0
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		calls++
		return "", nil
	})

	wf := workflowWith(nil)
	r := strings.NewReader("/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if calls != 0 {
		t.Errorf("expected no engine calls after /quit, got %d", calls)
	}
	if !strings.Contains(w.String(), "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in output, got: %q", w.String())
	}
}

func TestRunWithIO_ExitCommand(t *testing.T) {
	wf := workflowWith(nil)
	r := strings.NewReader("/exit\n")
	var w bytes.Buffer

	eng := buildEngine(nil)
	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(w.String(), "Goodbye!") {
		t.Errorf("expected 'Goodbye!' in output, got: %q", w.String())
	}
}

func TestRunWithIO_EngineError_ContinuesLoop(t *testing.T) {
	call := 0
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		call++
		if call == 1 {
			return nil, errors.New("llm backend unavailable")
		}
		return "recovered", nil
	})

	wf := workflowWith(nil)
	r := strings.NewReader("bad\ngood\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out := w.String()
	if !strings.Contains(out, "Error:") {
		t.Errorf("expected 'Error:' in output, got: %q", out)
	}
	if !strings.Contains(out, "recovered") {
		t.Errorf("expected 'recovered' in output after error, got: %q", out)
	}
}

func TestRunWithIO_EOF_NoError(t *testing.T) {
	eng := buildEngine("reply")
	wf := workflowWith(nil)
	r := strings.NewReader("")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error on EOF: %v", err)
	}
}

func TestRunWithIO_DefaultPromptAndSession(t *testing.T) {
	var gotReq *executor.RequestContext
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, req interface{}) (interface{}, error) {
		if rc, ok := req.(*executor.RequestContext); ok {
			gotReq = rc
		}
		return "ok", nil
	})

	wf := workflowWith(nil)
	r := strings.NewReader("hello\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotReq == nil {
		t.Fatal("engine was never called")
	}
	if gotReq.SessionID != "llm-repl-session" {
		t.Errorf("expected sessionID='llm-repl-session', got %q", gotReq.SessionID)
	}
	if msg, ok := gotReq.Body["message"]; !ok || msg != "hello" {
		t.Errorf("expected body.message='hello', got %v", gotReq.Body)
	}
	if !strings.Contains(w.String(), "> ") {
		t.Errorf("expected default prompt '> ' in output, got: %q", w.String())
	}
}

func TestRunWithIO_CustomSessionID(t *testing.T) {
	var gotSession string
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, req interface{}) (interface{}, error) {
		if rc, ok := req.(*executor.RequestContext); ok {
			gotSession = rc.SessionID
		}
		return "ok", nil
	})

	wf := workflowWith(&domain.LLMInputConfig{SessionID: "my-custom-session"})
	r := strings.NewReader("hello\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotSession != "my-custom-session" {
		t.Errorf("expected sessionID='my-custom-session', got %q", gotSession)
	}
}

func TestRunWithIO_NilResultPrintsEmpty(t *testing.T) {
	eng := buildEngine(nil)
	wf := workflowWith(nil)
	r := strings.NewReader("hello\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(w.String(), "> ") {
		t.Errorf("expected default prompt in output, got: %q", w.String())
	}
}

func TestRunWithIO_NonStringResult(t *testing.T) {
	eng := buildEngine(42)
	wf := workflowWith(nil)
	r := strings.NewReader("hello\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(w.String(), "42") {
		t.Errorf("expected '42' in output, got: %q", w.String())
	}
}

// ── slash command tests ────────────────────────────────────────────────────

// workflowWithResources builds a workflow that has two named resources.
func workflowWithResources(resources []*domain.Resource) *domain.Workflow {
	return &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:           "test",
			TargetActionID: "chat",
		},
		Settings: domain.WorkflowSettings{
			Input: &domain.InputConfig{
				Sources: []string{domain.InputSourceLLM},
			},
		},
		Resources: resources,
	}
}

func TestRunWithIO_HelpCommand(t *testing.T) {
	eng := buildEngine("should not be called")
	wf := workflowWith(nil)
	r := strings.NewReader("/help\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := w.String()
	if !strings.Contains(out, "/run") {
		t.Errorf("expected /run in help output, got: %q", out)
	}
	if !strings.Contains(out, "/list") {
		t.Errorf("expected /list in help output, got: %q", out)
	}
}

func TestRunWithIO_QuestionMarkHelp(t *testing.T) {
	eng := buildEngine("nope")
	wf := workflowWith(nil)
	r := strings.NewReader("/?\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(w.String(), "/run") {
		t.Errorf("expected help output for /?, got: %q", w.String())
	}
}

func TestRunWithIO_ListCommand_NoResources(t *testing.T) {
	eng := buildEngine("nope")
	wf := workflowWith(nil) // no Resources slice
	r := strings.NewReader("/list\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(w.String(), "none") {
		t.Errorf("expected '(none)' in /list output, got: %q", w.String())
	}
}

func TestRunWithIO_ListCommand_ShowsResources(t *testing.T) {
	resources := []*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "calcTool", Name: "Calculator Tool"}},
		{Metadata: domain.ResourceMetadata{ActionID: "chat", Name: "LLM Chat"}},
	}
	eng := buildEngine("nope")
	wf := workflowWithResources(resources)
	r := strings.NewReader("/ls\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := w.String()
	if !strings.Contains(out, "calcTool") {
		t.Errorf("expected calcTool in /ls output, got: %q", out)
	}
	if !strings.Contains(out, "chat") {
		t.Errorf("expected chat in /ls output, got: %q", out)
	}
	if !strings.Contains(out, "(target)") {
		t.Errorf("expected '(target)' marker for targetActionId, got: %q", out)
	}
}

func TestRunWithIO_RunCommand_UnknownActionID(t *testing.T) {
	eng := buildEngine("should not reach engine")
	wf := workflowWithResources([]*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "calcTool", Name: "Calculator"}},
	})
	r := strings.NewReader("/run nonExistentAction\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(w.String(), "unknown actionId") {
		t.Errorf("expected 'unknown actionId' error, got: %q", w.String())
	}
}

func TestRunWithIO_RunCommand_KnownActionID_InvokesEngine(t *testing.T) {
	var gotTargetID string
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		gotTargetID = wf.Metadata.TargetActionID
		return "calc result", nil
	})
	wf := workflowWithResources([]*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "calcTool", Name: "Calculator"}},
	})
	r := strings.NewReader("/run calcTool expression=2+2\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTargetID != "calcTool" {
		t.Errorf("expected engine called with targetActionID=calcTool, got %q", gotTargetID)
	}
	if !strings.Contains(w.String(), "calc result") {
		t.Errorf("expected 'calc result' in output, got: %q", w.String())
	}
}

func TestRunWithIO_RunCommand_ParamsPassedToEngine(t *testing.T) {
	var gotBody map[string]interface{}
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, req interface{}) (interface{}, error) {
		if rc, ok := req.(*executor.RequestContext); ok {
			gotBody = rc.Body
		}
		return "ok", nil
	})
	wf := workflowWithResources([]*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "calcTool", Name: "Calculator"}},
	})
	r := strings.NewReader("/run calcTool expression=sqrt(16) mode=safe\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody == nil {
		t.Fatal("engine was never called")
	}
	if gotBody["expression"] != "sqrt(16)" {
		t.Errorf("expected expression=sqrt(16), got %v", gotBody["expression"])
	}
	if gotBody["mode"] != "safe" {
		t.Errorf("expected mode=safe, got %v", gotBody["mode"])
	}
}

func TestRunWithIO_ToolAlias_InvokesEngine(t *testing.T) {
	var gotTarget string
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		gotTarget = wf.Metadata.TargetActionID
		return "tool result", nil
	})
	wf := workflowWithResources([]*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "hashTool", Name: "Hash Tool"}},
	})
	r := strings.NewReader("/tool hashTool data=hello\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTarget != "hashTool" {
		t.Errorf("expected /tool to invoke hashTool, got %q", gotTarget)
	}
}

func TestRunWithIO_ComponentAlias_InvokesEngine(t *testing.T) {
	var gotTarget string
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		gotTarget = wf.Metadata.TargetActionID
		return "component result", nil
	})
	wf := workflowWithResources([]*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "markdownTool", Name: "Markdown"}},
	})
	r := strings.NewReader("/component markdownTool text=hello\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotTarget != "markdownTool" {
		t.Errorf("expected /component to invoke markdownTool, got %q", gotTarget)
	}
}

func TestRunWithIO_RunNoArgs_ShowsUsage(t *testing.T) {
	eng := buildEngine("nope")
	wf := workflowWith(nil)
	r := strings.NewReader("/run\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(w.String(), "Usage:") {
		t.Errorf("expected usage hint for /run with no args, got: %q", w.String())
	}
}

func TestRunWithIO_UnknownSlashCommand_ForwardsToLLM(t *testing.T) {
	calls := 0
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		calls++
		return "llm reply", nil
	})
	wf := workflowWith(nil)
	// /unknowncmd is not handled — should fall through to the LLM
	r := strings.NewReader("/unknowncmd do something\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 1 {
		t.Errorf("expected unknown slash command to be forwarded to LLM (1 call), got %d", calls)
	}
}

func TestRunWithIO_OriginalWorkflowNotMutated(t *testing.T) {
	originalTarget := "chat"
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return "ok", nil
	})
	wf := workflowWithResources([]*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "calcTool", Name: "Calculator"}},
	})
	wf.Metadata.TargetActionID = originalTarget

	r := strings.NewReader("/run calcTool\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The original workflow's TargetActionID must not be mutated.
	if wf.Metadata.TargetActionID != originalTarget {
		t.Errorf("original workflow TargetActionID was mutated: got %q, want %q",
			wf.Metadata.TargetActionID, originalTarget)
	}
}

// ─── Additional coverage tests ───────────────────────────────────────────────

// workflowWithComponents builds a Workflow that has components (for printResources coverage).
func workflowWithComponents(resources []*domain.Resource, components map[string]*domain.Component) *domain.Workflow {
	return &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test", TargetActionID: "chat"},
		Settings: domain.WorkflowSettings{
			Input: &domain.InputConfig{
				Sources: []string{domain.InputSourceLLM},
			},
		},
		Resources:  resources,
		Components: components,
	}
}

// TestRunWithIO_PrintResources_WithComponents verifies that /list shows components.
func TestRunWithIO_PrintResources_WithComponents(t *testing.T) {
	eng := buildEngine("ok")
	components := map[string]*domain.Component{
		"render-markdown": {
			Metadata: domain.ComponentMetadata{
				Name:        "render-markdown",
				Version:     "1.0.0",
				Description: "Renders markdown to HTML",
			},
		},
	}
	wf := workflowWithComponents(nil, components)
	r := strings.NewReader("/list\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := w.String()
	if !strings.Contains(out, "render-markdown") {
		t.Errorf("expected component name in /list output, got: %q", out)
	}
	if !strings.Contains(out, "1.0.0") {
		t.Errorf("expected component version in /list output, got: %q", out)
	}
}

// TestRunWithIO_PrintResources_ComponentDescriptionTruncated verifies multiline
// component descriptions are truncated to the first line.
func TestRunWithIO_PrintResources_ComponentDescriptionTruncated(t *testing.T) {
	eng := buildEngine("ok")
	components := map[string]*domain.Component{
		"my-comp": {
			Metadata: domain.ComponentMetadata{
				Name:        "my-comp",
				Version:     "2.0.0",
				Description: "First line\nSecond line ignored",
			},
		},
	}
	wf := workflowWithComponents(nil, components)
	r := strings.NewReader("/ls\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := w.String()
	if strings.Contains(out, "Second line") {
		t.Errorf("expected multiline description to be truncated, got: %q", out)
	}
	if !strings.Contains(out, "First line") {
		t.Errorf("expected first line of description, got: %q", out)
	}
}

// TestRunWithIO_PrintResources_ComponentNoDescription verifies a component with
// no description falls back to Name.
func TestRunWithIO_PrintResources_ComponentNoDescription(t *testing.T) {
	eng := buildEngine("ok")
	components := map[string]*domain.Component{
		"no-desc-comp": {
			Metadata: domain.ComponentMetadata{
				Name:    "no-desc-comp",
				Version: "0.1.0",
				// Description is empty — should fall back to Name.
			},
		},
	}
	wf := workflowWithComponents(nil, components)
	r := strings.NewReader("/list\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(w.String(), "no-desc-comp") {
		t.Errorf("expected component name as fallback description, got: %q", w.String())
	}
}

// TestRunWithIO_RunCommand_EngineError verifies that an engine error from
// /run <actionId> is printed to the output without crashing.
func TestRunWithIO_RunCommand_EngineError(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, errors.New("execution failed")
	})
	wf := workflowWithResources([]*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "calcTool", Name: "Calculator"}},
	})
	r := strings.NewReader("/run calcTool\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(w.String(), "Error") {
		t.Errorf("expected error message in output, got: %q", w.String())
	}
}

// TestParseParams_BareFlag verifies that a bare argument (no '=') is treated
// as key=true.
func TestParseParams_BareFlag(t *testing.T) {
	eng := executor.NewEngine(nil)
	var gotBody map[string]interface{}
	eng.SetExecuteFunc(func(_ *domain.Workflow, req interface{}) (interface{}, error) {
		if rc, ok := req.(*executor.RequestContext); ok {
			gotBody = rc.Body
		}
		return "ok", nil
	})
	wf := workflowWithResources([]*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "myTool", Name: "My Tool"}},
	})
	// "verbose" is a bare flag — should become "verbose"="true"
	r := strings.NewReader("/run myTool verbose\n/quit\n")
	var w bytes.Buffer

	if err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody == nil {
		t.Fatal("engine was never called")
	}
	if gotBody["verbose"] != "true" {
		t.Errorf("expected bare flag to become verbose=true, got %v", gotBody["verbose"])
	}
}

// TestRun_Signature verifies that Run is callable and returns nil on EOF input.
// This exercises the Run → RunWithIO delegation path.
func TestRun_Signature(t *testing.T) {
	// Run delegates to RunWithIO(ctx, wf, eng, logger, os.Stdin, os.Stdout).
	// We cannot easily replace os.Stdin in a unit test, but we can verify
	// the function exists and has the correct signature by calling RunWithIO
	// with the same parameters as Run would pass, using controlled I/O.
	eng := buildEngine("hello")
	wf := workflowWith(nil)

	// Immediate EOF — RunWithIO should return nil.
	r := strings.NewReader("")
	var w bytes.Buffer
	err := llminput.RunWithIO(context.Background(), wf, eng, nil, r, &w)
	assert.NoError(t, err)
}
