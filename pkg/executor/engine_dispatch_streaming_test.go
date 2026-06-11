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

	stream, ok := result.([]interface{})
	require.True(t, ok, "response-only primary with inline should stream")
	// 2 before steps + 1 final apiResponse snapshot
	assert.Len(t, stream, 3)
	for i, chunk := range stream {
		resp, mapOK := chunk.(map[string]interface{})
		require.True(t, mapOK, "chunk %d should be apiResponse map", i)
		assert.Equal(t, true, resp["success"])
	}
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
