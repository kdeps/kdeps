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
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/executor"
	executorLLM "github.com/kdeps/kdeps/v3/pkg/executor/llm"
	"github.com/kdeps/kdeps/v3/pkg/nudge"
)

// A workflow chat with tools that claims an action it never took is pushed
// back with the unbacked-claim nudge, as in agent mode, and answers again.
func TestWorkflowChat_Nudges(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	nudge.Configure(nudge.Settings{
		Note:    func(name string, _ any) string { return "NUDGE " + name },
		Allowed: func(_ string, occurred int) bool { return occurred < 1 },
	})
	t.Cleanup(func() { nudge.Configure(nudge.Settings{}) })
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			_, _ = w.Write([]byte(`{"data":[{"id":"fake"}]}`))
			return
		}
		calls.Add(1)
		var body struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		last := body.Messages[len(body.Messages)-1].Content
		reply := "I've created notes.md for you."
		if strings.HasPrefix(last, "NUDGE nudge-unbacked-claim") {
			reply = "I have not created anything yet."
		}
		raw, _ := json.Marshal(map[string]any{
			"id": "x", "object": "chat.completion", "model": "fake",
			"choices": []any{map[string]any{"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": reply}}},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)
	t.Setenv(executorLLM.OverrideURLEnv, srv.URL)

	tool := domain.Tool{
		Name: "write_note", Description: "Write a note",
		Execute: func(map[string]any) (string, error) { return "ok", nil },
	}
	wf := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "nudges", Version: "1.0.0", TargetActionID: "ask"},
		Resources: []*domain.Resource{{
			ActionID: "ask",
			Chat: &domain.ChatConfig{
				Model: "fake", Role: "user", Prompt: "make notes.md", Tools: []domain.Tool{tool},
			},
		}},
	}
	engine := executor.NewEngine(t.Context(), slog.Default())
	registry := executor.NewRegistry()
	registry.SetLLMExecutor(executorLLM.NewAdapter(t.Context(), srv.URL))
	engine.SetRegistry(registry)
	out, err := engine.Execute(wf, nil)
	require.NoError(t, err)
	raw, _ := json.Marshal(out)
	assert.Contains(t, string(raw), "I have not created anything yet.")
	assert.Equal(t, int32(2), calls.Load(), "one nudge, one more model call")
}
