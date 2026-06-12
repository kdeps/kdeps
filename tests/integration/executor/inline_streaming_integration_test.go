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

package executor_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func TestInlineStreamingResponse_Integration(t *testing.T) {
	engine := executor.NewEngine(nil)
	workflow := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata: domain.WorkflowMetadata{
			Name:    "inline-stream",
			Version: "1.0.0",
		},
	}
	ctx, err := executor.NewExecutionContext(workflow)
	require.NoError(t, err)

	resource := &domain.Resource{
		ActionID: "main",
		Name:     "Main",
		Before: []domain.InlineResource{
			{Expr: "set('n', 1)"},
		},
		After: []domain.InlineResource{
			{Expr: "set('n', 2)"},
		},
		APIResponse: &domain.APIResponseConfig{
			Success:  true,
			Response: map[string]interface{}{"n": "{{ get('n') }}"},
		},
	}

	result, err := engine.ExecuteResource(resource, ctx)
	require.NoError(t, err)

	// A single apiResponse map, evaluated after the before: step and before
	// the after: step - API clients never receive per-step snapshot slices.
	resp, ok := result.(map[string]interface{})
	require.True(t, ok, "apiResponse primary with inline must return one apiResponse map")
	assert.Equal(t, true, resp["success"])
	data, ok := resp["data"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "1", fmt.Sprintf("%v", data["n"]), "response sees before: state, not after: state")
}
