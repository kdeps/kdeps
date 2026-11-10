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

package executor_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	executorLLM "github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

// A workflow chat on an OpenRouter free id that the provider rate-limits
// moves to the next free id, as agent mode does.
func TestWorkflowChat_OpenRouterFreeHop(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv(executorLLM.OverrideURLEnv, "")
	t.Setenv("OPENROUTER_API_KEY", "test")
	ids := executorLLM.OpenRouterFreeModelIDs()
	require.GreaterOrEqual(t, len(ids), 2)
	executorLLM.SetFreeModelHop(func(err error) bool { return err != nil })
	t.Cleanup(func() { executorLLM.SetFreeModelHop(func(error) bool { return false }) })

	var mu sync.Mutex
	var asked []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Model string `json:"model"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		mu.Lock()
		asked = append(asked, body.Model)
		mu.Unlock()
		if body.Model == ids[0] {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":{"message":"Provider returned error","code":429}}`))
			return
		}
		raw, _ := json.Marshal(map[string]any{
			"id": "x", "object": "chat.completion", "model": body.Model,
			"choices": []any{map[string]any{"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": "ANSWERED BY " + body.Model}}},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("KDEPS_LLM_BASE_URL", srv.URL)

	wf := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "free", Version: "1.0.0", TargetActionID: "ask"},
		Resources: []*domain.Resource{{
			ActionID: "ask",
			Chat:     &domain.ChatConfig{Model: ids[0], Backend: "openrouter", Role: "user", Prompt: "hi"},
		}},
	}
	engine := executor.NewEngine(t.Context(), slog.Default())
	registry := executor.NewRegistry()
	registry.SetLLMExecutor(executorLLM.NewAdapter(t.Context(), srv.URL))
	engine.SetRegistry(registry)
	out, err := engine.Execute(wf, nil)
	require.NoError(t, err)
	raw, _ := json.Marshal(out)
	assert.Contains(t, string(raw), "ANSWERED BY "+ids[1])
	assert.Contains(t, asked, ids[0])
}
