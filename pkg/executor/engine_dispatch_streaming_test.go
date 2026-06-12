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

package executor

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

func TestExecuteResource_StreamingInlineResponse(t *testing.T) {
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "wf", Version: "1.0.0"},
	})
	require.NoError(t, err)
	e.SetEvaluatorForTesting(expression.NewEvaluator(ctx.API))

	resource := &domain.Resource{
		ActionID: "stream",
		Name:     "Stream",
		Before: []domain.InlineResource{
			{Expr: "set('step', 1)"},
			{Expr: "set('step', 2)"},
		},
		APIResponse: &domain.APIResponseConfig{
			Success:  true,
			Response: map[string]interface{}{"step": "{{ get('step') }}"},
		},
	}

	result, err := e.ExecuteResource(resource, ctx)
	require.NoError(t, err)

	// A single apiResponse evaluated after all before: steps — JSON API
	// clients must never receive per-step snapshot slices.
	resp, ok := result.(map[string]interface{})
	require.True(t, ok, "response-only primary with inline must return one apiResponse map")
	assert.Equal(t, true, resp["success"])
	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "2", fmt.Sprintf("%v", data["step"]), "response must see the final before-step state")
}

func TestExecuteResource_ChatPlusAPIResponseDoesNotStream(t *testing.T) {
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetLLMExecutor(&covMockExecutor{result: "llm out"})
	e.SetRegistry(reg)

	ctx, err := NewExecutionContext(&domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "wf", Version: "1.0.0"},
	})
	require.NoError(t, err)
	e.SetEvaluatorForTesting(expression.NewEvaluator(ctx.API))

	resource := &domain.Resource{
		ActionID: "combo",
		Name:     "Combo",
		Before: []domain.InlineResource{
			{Expr: "set('x', 1)"},
		},
		Chat: &domain.ChatConfig{Model: "m", Prompt: "p"},
		APIResponse: &domain.APIResponseConfig{
			Success:  true,
			Response: "ok",
		},
	}

	result, err := e.ExecuteResource(resource, ctx)
	require.NoError(t, err)
	_, isSlice := result.([]interface{})
	assert.False(t, isSlice, "chat + apiResponse should return a single apiResponse map")
	resp, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, resp["success"])
}

func TestShouldStreamInlineResponse_Guards(t *testing.T) {
	e := covTestEngine()
	assert.False(t, e.shouldStreamInlineResponse(nil, &ExecutionContext{}))
	assert.False(t, e.shouldStreamInlineResponse(&domain.Resource{}, nil))

	ctx, err := NewExecutionContext(&domain.Workflow{})
	require.NoError(t, err)
	ctx.Items["item"] = 1
	res := &domain.Resource{
		Before:      []domain.InlineResource{{Expr: "set('x', 1)"}},
		APIResponse: &domain.APIResponseConfig{Success: true, Response: "ok"},
	}
	assert.False(t, e.shouldStreamInlineResponse(res, ctx))
}

func TestExecuteResource_StreamingAfterInline(t *testing.T) {
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "wf", Version: "1.0.0"},
	})
	require.NoError(t, err)
	e.SetEvaluatorForTesting(expression.NewEvaluator(ctx.API))

	resource := &domain.Resource{
		ActionID: "stream-after",
		Name:     "Stream After",
		After: []domain.InlineResource{
			{Expr: "set('step', 1)"},
		},
		APIResponse: &domain.APIResponseConfig{
			Success:  true,
			Response: map[string]interface{}{"step": "{{ get('step') }}"},
		},
	}

	result, err := e.ExecuteResource(resource, ctx)
	require.NoError(t, err)
	resp, ok := result.(map[string]interface{})
	require.True(t, ok, "after: steps must not turn the response into a slice")
	assert.Equal(t, true, resp["success"])
}

func TestExecuteResource_StreamingSingleChunk(t *testing.T) {
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "wf", Version: "1.0.0"},
	})
	require.NoError(t, err)
	e.SetEvaluatorForTesting(expression.NewEvaluator(ctx.API))

	resource := &domain.Resource{
		ActionID: "single",
		Name:     "Single",
		APIResponse: &domain.APIResponseConfig{
			Success:  true,
			Response: "only",
		},
	}

	result, err := e.ExecuteResource(resource, ctx)
	require.NoError(t, err)
	_, isSlice := result.([]interface{})
	assert.False(t, isSlice)
}

func TestExecuteInlineStep_NonExpressionInline(t *testing.T) {
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetHTTPExecutor(&covMockExecutor{result: map[string]interface{}{"ok": true}})
	e.SetRegistry(reg)
	ctx, err := NewExecutionContext(&domain.Workflow{})
	require.NoError(t, err)

	resource := &domain.Resource{
		ActionID: "inline-http",
		Name:     "Inline HTTP",
		Before: []domain.InlineResource{
			{HTTPClient: &domain.HTTPClientConfig{Method: "GET", URL: "https://example.com"}},
		},
		APIResponse: &domain.APIResponseConfig{Success: true, Response: "ok"},
	}

	_, err = e.ExecuteResource(resource, ctx)
	require.NoError(t, err)
}

func TestExecuteStreamingInlineResponse_SingleSnapshot(t *testing.T) {
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "wf", Version: "1.0.0"},
	})
	require.NoError(t, err)
	e.SetEvaluatorForTesting(expression.NewEvaluator(ctx.API))

	resource := &domain.Resource{
		APIResponse: &domain.APIResponseConfig{Success: true, Response: "only"},
	}

	result, err := e.executeStreamingInlineResponse(resource, ctx)
	require.NoError(t, err)
	_, isSlice := result.([]interface{})
	assert.False(t, isSlice)
}

func TestExecuteStreamingInlineResponse_BeforeExprError(t *testing.T) {
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{})
	require.NoError(t, err)
	e.SetEvaluatorForTesting(expression.NewEvaluator(ctx.API))

	resource := &domain.Resource{
		Before:      []domain.InlineResource{{Expr: "{{unclosed"}},
		APIResponse: &domain.APIResponseConfig{Success: true, Response: "ok"},
	}

	_, err = e.executeStreamingInlineResponse(resource, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inline before resource")
}

func TestExecuteStreamingInlineResponse_AfterExprError(t *testing.T) {
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{})
	require.NoError(t, err)
	e.SetEvaluatorForTesting(expression.NewEvaluator(ctx.API))

	resource := &domain.Resource{
		After:       []domain.InlineResource{{Expr: "{{unclosed"}},
		APIResponse: &domain.APIResponseConfig{Success: true, Response: "ok"},
	}

	_, err = e.executeStreamingInlineResponse(resource, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inline after resource")
}

func TestExecuteStreamingInlineResponse_FinalSnapshotError(t *testing.T) {
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{})
	require.NoError(t, err)

	resource := &domain.Resource{}

	_, err = e.executeStreamingInlineResponse(resource, ctx)
	require.Error(t, err)
}

func TestExecuteStreamingInlineResponse_AfterSnapshotError(t *testing.T) {
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{})
	require.NoError(t, err)
	e.SetEvaluatorForTesting(expression.NewEvaluator(ctx.API))

	resource := &domain.Resource{
		After: []domain.InlineResource{{Expr: "set('x', 1)"}},
	}

	_, err = e.executeStreamingInlineResponse(resource, ctx)
	require.Error(t, err)
}

func TestExecuteExpressions_NilContextRequiresEvaluator(t *testing.T) {
	e := covTestEngine()
	err := e.executeExpressions([]domain.Expression{{Raw: "1 + 1"}}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "execution context required")
}

func TestExecuteStreamingInlineResponse_SnapshotError(t *testing.T) {
	e := covTestEngine()
	ctx, err := NewExecutionContext(&domain.Workflow{})
	require.NoError(t, err)
	e.SetEvaluatorForTesting(expression.NewEvaluator(ctx.API))

	resource := &domain.Resource{
		Before: []domain.InlineResource{{Expr: "set('x', 1)"}},
	}

	_, err = e.executeStreamingInlineResponse(resource, ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no apiResponse configuration")
}
