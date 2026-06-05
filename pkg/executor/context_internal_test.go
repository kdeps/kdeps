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
)

func TestBuildEvaluatorEnv_Basic(t *testing.T) {
	ctx := &ExecutionContext{
		Items: make(map[string]interface{}),
	}
	env := ctx.BuildEvaluatorEnv()
	assert.NotNil(t, env)
	assert.Contains(t, env, "llm")
	assert.Contains(t, env, "python")
	assert.Contains(t, env, "exec")
	assert.Contains(t, env, "item")

	itemMap, ok := env["item"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, itemMap, "values")
}

func TestBuildEvaluatorEnv_WithItem(t *testing.T) {
	ctx := &ExecutionContext{
		Items: map[string]interface{}{
			"item": map[string]interface{}{
				"name":  "test-item",
				"value": 42,
			},
		},
	}
	env := ctx.BuildEvaluatorEnv()
	itemMap, ok := env["item"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, "test-item", itemMap["name"])
	assert.Equal(t, 42, itemMap["value"])
	assert.Contains(t, itemMap, "values")
}

func TestBuildEvaluatorEnv_ItemNotMap(t *testing.T) {
	ctx := &ExecutionContext{
		Items: map[string]interface{}{
			"item": "not-a-map",
		},
	}
	env := ctx.BuildEvaluatorEnv()
	assert.NotNil(t, env)
	itemMap, ok := env["item"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, itemMap, "values")
}

func TestBuildEvaluatorEnv_LLMResponse(t *testing.T) {
	ctx := &ExecutionContext{
		Items: make(map[string]interface{}),
	}
	env := ctx.BuildEvaluatorEnv()
	llmMap, ok := env["llm"].(map[string]interface{})
	require.True(t, ok)
	respFn, ok := llmMap["response"].(func(string) interface{})
	require.True(t, ok)
	result := respFn("nonexistent")
	assert.Nil(t, result)
}

func TestBuildEvaluatorEnv_PythonStdout(t *testing.T) {
	ctx := &ExecutionContext{
		Items: make(map[string]interface{}),
	}
	env := ctx.BuildEvaluatorEnv()
	pyMap, ok := env["python"].(map[string]interface{})
	require.True(t, ok)
	stdoutFn, ok := pyMap["stdout"].(func(string) interface{})
	require.True(t, ok)
	result := stdoutFn("nonexistent")
	assert.Equal(t, "", result)
}

func TestIsNamespacedPath(t *testing.T) {
	assert.True(t, isNamespacedPath("config.llm.provider"))
	assert.True(t, isNamespacedPath("workflow.settings"))
	assert.True(t, isNamespacedPath("resource.myRes.field"))
	assert.True(t, isNamespacedPath("component.myComp.key"))
	assert.True(t, isNamespacedPath("agency.myAgency.key"))
	assert.False(t, isNamespacedPath("plain"))
	assert.False(t, isNamespacedPath(""))
}

func TestGetConfigField_InvalidPath(t *testing.T) {
	ctx := &ExecutionContext{}
	_, err := ctx.GetConfigField("noprefix")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config path")
}

func TestGetConfigField_EmptyAfterDot(t *testing.T) {
	ctx := &ExecutionContext{}
	_, err := ctx.GetConfigField("config.")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid config path")
}

func TestGetConfigField_UnknownNamespace(t *testing.T) {
	ctx := &ExecutionContext{}
	_, err := ctx.GetConfigField("unknown.field")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown namespace")
}

func TestNewExecutionContext_BasicWorkflow(t *testing.T) {
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
	}
	ctx, err := NewExecutionContext(wf)
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
	assert.Equal(t, wf, ctx.Workflow)
}

func TestNewExecutionContext_WithSessionID(t *testing.T) {
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
	}
	ctx, err := NewExecutionContext(wf, "custom-session")
	assert.NoError(t, err)
	assert.NotNil(t, ctx)
}
