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

package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

func TestEvaluateData_MapFieldError(t *testing.T) {
	e := NewExecutor()
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	eval := expression.NewEvaluator(ctx.API)

	_, err = e.evaluateData(eval, ctx, map[string]interface{}{
		"bad": "{{{",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to evaluate data field bad")
}

func TestPrepareRequest_DefaultMethod(t *testing.T) {
	e := NewExecutor()
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	eval := expression.NewEvaluator(ctx.API)

	_, method, _, err := e.prepareRequest(eval, ctx, &domain.HTTPClientConfig{URL: "http://example.com"}, nil)
	require.NoError(t, err)
	assert.Equal(t, http.MethodGet, method)
}
