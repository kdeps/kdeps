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

package validator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

// helpers

func mkWorkflow(targetActionID string, resources ...*domain.Resource) *domain.Workflow { //nolint:unparam
	return &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:           "test",
			Version:        "1.0.0",
			TargetActionID: targetActionID,
		},
		Resources: resources,
	}
}

func mkResource(id string, requires ...string) *domain.Resource {
	return &domain.Resource{
		Metadata: domain.ResourceMetadata{
			ActionID: id,
			Requires: requires,
		},
	}
}

// AnalysisIssue.Error

func TestAnalysisIssue_String_WithActionID(t *testing.T) {
	i := validator.AnalysisIssue{ActionID: "foo", Severity: "error", Message: "bad"}
	assert.Equal(t, "[error] foo: bad", i.String())
}

func TestAnalysisIssue_String_NoActionID(t *testing.T) {
	i := validator.AnalysisIssue{Severity: "warning", Message: "stale"}
	assert.Equal(t, "[warning] stale", i.String())
}

// WorkflowAnalysis helpers

func TestWorkflowAnalysis_HasErrors_False(t *testing.T) {
	wa := &validator.WorkflowAnalysis{}
	assert.False(t, wa.HasErrors())
}

func TestWorkflowAnalysis_HasErrors_True(t *testing.T) {
	wa := &validator.WorkflowAnalysis{
		Issues: []validator.AnalysisIssue{{Severity: "error", Message: "x"}},
	}
	assert.True(t, wa.HasErrors())
}

func TestWorkflowAnalysis_Errors_Warnings(t *testing.T) {
	wa := &validator.WorkflowAnalysis{
		Issues: []validator.AnalysisIssue{
			{Severity: "error", Message: "e1"},
			{Severity: "warning", Message: "w1"},
			{Severity: "error", Message: "e2"},
		},
	}
	require.Len(t, wa.Errors(), 2)
	require.Len(t, wa.Warnings(), 1)
	assert.Equal(t, "w1", wa.Warnings()[0].Message)
}

// AnalyzeWorkflow - empty workflow

func TestAnalyzeWorkflow_EmptyResources(t *testing.T) {
	w := &domain.Workflow{}
	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Issues)
}

// Unreachable detection

func TestAnalyzeWorkflow_AllReachable(t *testing.T) {
	// target -> b -> c (all reachable)
	c := mkResource("c")
	b := mkResource("b", "c")
	a := mkResource("target", "b")
	w := mkWorkflow("target", a, b, c)

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Issues)
}

func TestAnalyzeWorkflow_UnreachableResource(t *testing.T) {
	target := mkResource("target")
	orphan := mkResource("orphan")
	w := mkWorkflow("target", target, orphan)

	wa := validator.AnalyzeWorkflow(w)
	warns := wa.Warnings()
	require.Len(t, warns, 1)
	assert.Equal(t, "orphan", warns[0].ActionID)
	assert.Contains(t, warns[0].Message, "unreachable")
}

func TestAnalyzeWorkflow_MultipleUnreachable(t *testing.T) {
	target := mkResource("target")
	orphan1 := mkResource("orphan1")
	orphan2 := mkResource("orphan2", "orphan1") // orphan2 requires orphan1, still unreachable from target
	w := mkWorkflow("target", target, orphan1, orphan2)

	wa := validator.AnalyzeWorkflow(w)
	warns := wa.Warnings()
	assert.Len(t, warns, 2)
}

func TestAnalyzeWorkflow_NoTargetActionID(t *testing.T) {
	// No target - unreachable detection skipped
	r := mkResource("a")
	w := &domain.Workflow{Resources: []*domain.Resource{r}}

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Warnings())
}

// Bad expression reference detection

func TestAnalyzeWorkflow_GoodExpressionRef(t *testing.T) {
	dep := mkResource("dep")
	r := mkResource("target", "dep")
	r.Run.Chat = &domain.ChatConfig{Prompt: "answer: {{ dep.result }}"}
	w := mkWorkflow("target", r, dep)

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestAnalyzeWorkflow_BadExpressionRef_GetFunc(t *testing.T) {
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: "{{ get('missing.field') }}"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	errs := wa.Errors()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, `"missing"`)
}

func TestAnalyzeWorkflow_BadExpressionRef_OutputFunc(t *testing.T) {
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: "output('ghost')"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	errs := wa.Errors()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, `"ghost"`)
}

func TestAnalyzeWorkflow_BadExpressionRef_Template(t *testing.T) {
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: "value: {{ nowhere.x }}"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	errs := wa.Errors()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, `"nowhere"`)
}

func TestAnalyzeWorkflow_ExpressionRef_NoFalsePositive_SelfRef(t *testing.T) {
	// Self-reference via get('target.field') should not flag if 'target' exists.
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: "get('target.field')"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionRef_DedupedPerResource(t *testing.T) {
	// Same bad ref used twice in the same resource should produce one error.
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{
		Prompt: "get('ghost.a') and get('ghost.b')",
	}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.Len(t, wa.Errors(), 1)
}

func TestAnalyzeWorkflow_ExpressionInExpr(t *testing.T) {
	r := mkResource("target")
	r.Run.Expr = []domain.Expression{{Raw: "output('missing')"}}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInExprBefore(t *testing.T) {
	r := mkResource("target")
	r.Run.ExprBefore = []domain.Expression{{Raw: "set('k', get('gone.field'))"}}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInValidations(t *testing.T) {
	r := mkResource("target")
	r.Run.Validations = &domain.ValidationsConfig{
		Skip:  []domain.Expression{{Raw: "get('absent.field')"}},
		Check: []domain.Expression{{Raw: "get('target.ok')"}}, // valid
	}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	errs := wa.Errors()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, `"absent"`)
}

func TestAnalyzeWorkflow_ExpressionInOnError(t *testing.T) {
	r := mkResource("target")
	r.Run.OnError = &domain.OnErrorConfig{
		Expr: []domain.Expression{{Raw: "get('nowhere.x')"}},
	}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInPythonScript(t *testing.T) {
	r := mkResource("target")
	r.Run.Python = &domain.PythonConfig{Script: "output('ghost')"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInExecCommand(t *testing.T) {
	r := mkResource("target")
	r.Run.Exec = &domain.ExecConfig{Command: "echo get('missing.field')"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInHTTPURL(t *testing.T) {
	r := mkResource("target")
	r.Run.HTTPClient = &domain.HTTPClientConfig{URL: "http://{{ gone.host }}/path"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInHTTPData(t *testing.T) {
	r := mkResource("target")
	r.Run.HTTPClient = &domain.HTTPClientConfig{
		URL:  "http://example.com",
		Data: "get('absent.field')",
	}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInSQLQuery(t *testing.T) {
	r := mkResource("target")
	r.Run.SQL = &domain.SQLConfig{
		Queries: []domain.QueryItem{
			{Query: "SELECT * WHERE id = output('nope')"},
		},
	}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInScraperURL(t *testing.T) {
	r := mkResource("target")
	r.Run.Scraper = &domain.ScraperConfig{URL: "{{ gone.url }}"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInEmbeddingText(t *testing.T) {
	r := mkResource("target")
	r.Run.Embedding = &domain.EmbeddingConfig{Text: "{{ gone.text }}"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInSearchLocalQuery(t *testing.T) {
	r := mkResource("target")
	r.Run.SearchLocal = &domain.SearchLocalConfig{Query: "get('gone.field')"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInSearchWebQuery(t *testing.T) {
	r := mkResource("target")
	r.Run.SearchWeb = &domain.SearchWebConfig{Query: "output('gone')"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInBeforeInline(t *testing.T) {
	r := mkResource("target")
	r.Run.Before = []domain.ActionConfig{
		{Exec: &domain.ExecConfig{Command: "output('ghost')"}},
	}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ExpressionInAfterInline(t *testing.T) {
	r := mkResource("target")
	r.Run.After = []domain.ActionConfig{
		{Python: &domain.PythonConfig{Script: "{{ gone.x }}"}},
	}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.NotEmpty(t, wa.Errors())
}

// TestAnalyzeWorkflow_BareGetNoDotNoError verifies bare get('q') without dot
// notation is NOT flagged (request-param lookups, not actionId refs).
func TestAnalyzeWorkflow_BareGetNoDotNoError(t *testing.T) {
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: "{{ get('q') }} and get('page')"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

// Component input validation

func TestAnalyzeWorkflow_ComponentMissingRequiredInput(t *testing.T) {
	comp := &domain.Component{
		Interface: &domain.ComponentInterface{
			Inputs: []domain.ComponentInput{
				{Name: "url", Required: true},
			},
		},
	}
	r := mkResource("target")
	r.Run.Component = &domain.ComponentCallConfig{
		Name: "scraper",
		With: map[string]interface{}{}, // url not provided
	}
	w := mkWorkflow("target", r)
	w.Components = map[string]*domain.Component{"scraper": comp}

	wa := validator.AnalyzeWorkflow(w)
	errs := wa.Errors()
	require.Len(t, errs, 1)
	assert.Contains(t, errs[0].Message, `"url"`)
	assert.Contains(t, errs[0].Message, `"scraper"`)
}

func TestAnalyzeWorkflow_ComponentOptionalInputNotRequired(t *testing.T) {
	comp := &domain.Component{
		Interface: &domain.ComponentInterface{
			Inputs: []domain.ComponentInput{
				{Name: "selector", Required: false},
			},
		},
	}
	r := mkResource("target")
	r.Run.Component = &domain.ComponentCallConfig{
		Name: "scraper",
		With: map[string]interface{}{},
	}
	w := mkWorkflow("target", r)
	w.Components = map[string]*domain.Component{"scraper": comp}

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ComponentAllRequiredInputsProvided(t *testing.T) {
	comp := &domain.Component{
		Interface: &domain.ComponentInterface{
			Inputs: []domain.ComponentInput{
				{Name: "url", Required: true},
				{Name: "selector", Required: false},
			},
		},
	}
	r := mkResource("target")
	r.Run.Component = &domain.ComponentCallConfig{
		Name: "scraper",
		With: map[string]interface{}{"url": "https://example.com"},
	}
	w := mkWorkflow("target", r)
	w.Components = map[string]*domain.Component{"scraper": comp}

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ComponentNotFound(t *testing.T) {
	// Component referenced but not in workflow.Components - should not error.
	r := mkResource("target")
	r.Run.Component = &domain.ComponentCallConfig{Name: "unknown"}
	w := mkWorkflow("target", r)
	w.Components = map[string]*domain.Component{}

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestAnalyzeWorkflow_ComponentNoInterface(t *testing.T) {
	// Component exists but has no Interface defined - skip input validation.
	comp := &domain.Component{}
	r := mkResource("target")
	r.Run.Component = &domain.ComponentCallConfig{Name: "simple"}
	w := mkWorkflow("target", r)
	w.Components = map[string]*domain.Component{"simple": comp}

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestAnalyzeWorkflow_NoComponents(t *testing.T) {
	r := mkResource("target")
	w := mkWorkflow("target", r)
	// workflow.Components is nil

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Issues)
}

// extractActionIDRefs (via public surface)

func TestExtractActionIDRefs_BareGetNoDetection(t *testing.T) {
	// bare get('dep') without dot is not detected (ambiguous with request params)
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: "get('dep')"}
	w := mkWorkflow("target", r) // 'dep' not defined - should NOT error
	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestExtractActionIDRefs_DotGet_Valid(t *testing.T) {
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: `get("dep.field")`}
	dep := mkResource("dep")
	w := mkWorkflow("target", r, dep)

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestExtractActionIDRefs_OutputFunc(t *testing.T) {
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: "output('dep')"}
	dep := mkResource("dep")
	w := mkWorkflow("target", r, dep)

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestExtractActionIDRefs_TemplatePattern(t *testing.T) {
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: "value: {{ dep.result }}"}
	dep := mkResource("dep")
	w := mkWorkflow("target", r, dep)

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestExtractActionIDRefs_NoMatch(t *testing.T) {
	r := mkResource("target")
	r.Run.Chat = &domain.ChatConfig{Prompt: "no refs here at all"}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

// Combined: unreachable + bad ref in same workflow

func TestExtractActionIDRefs_BuiltinVarsNotFlagged(t *testing.T) {
	r := mkResource("target")
	// All kdeps/Jinja2 built-in objects - none should be flagged as unknown actionIds.
	r.Run.Chat = &domain.ChatConfig{
		Prompt: `{{ request.method }} {{ request.path }} {{ request.ip }} {{ loop.index }} {{ loop.first }} {{ error.message }} {{ item.value }}`,
	}
	w := mkWorkflow("target", r)

	wa := validator.AnalyzeWorkflow(w)
	assert.Empty(t, wa.Errors())
}

func TestAnalyzeWorkflow_CombinedIssues(t *testing.T) {
	target := mkResource("target")
	target.Run.Chat = &domain.ChatConfig{Prompt: "output('gone')"}
	orphan := mkResource("orphan")
	w := mkWorkflow("target", target, orphan)

	wa := validator.AnalyzeWorkflow(w)
	assert.Len(t, wa.Errors(), 1)   // bad ref
	assert.Len(t, wa.Warnings(), 1) // orphan
}
