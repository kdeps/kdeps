// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	executorLLM "github.com/kdeps/kdeps/v2/pkg/executor/llm"
	"github.com/kdeps/kdeps/v2/pkg/overflow"
)

// A workflow chat whose tool round overflows the model's context is
// compacted and retried once: the conversation after the system messages is
// summarized, and the model resumes from the summary and the original request.
func TestWorkflowChat_OverflowCompaction(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	overflow.Configure(overflow.Settings{
		SystemPrompt: func() string { return "SUMMARIZER" },
		UserPrompt:   func() string { return "Write the summary." },
	})
	t.Cleanup(func() { overflow.Configure(overflow.Settings{}) })
	var mu sync.Mutex
	var seen []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			_, _ = w.Write([]byte(`{"data":[{"id":"fake"}]}`))
			return
		}
		var body struct {
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		var all strings.Builder
		for _, m := range body.Messages {
			all.WriteString(m.Role + ":" + m.Content + "\n")
		}
		text := all.String()
		mu.Lock()
		seen = append(seen, text)
		mu.Unlock()
		reply := `{"name": "step", "arguments": {}}`
		switch {
		case strings.Contains(text, "SUMMARIZER"):
			reply = "ran step, got a huge result"
		case strings.Contains(text, "CONTEXT OVERFLOW RECOVERY"):
			reply = "RECOVERED"
		case strings.Contains(text, "HUGE"):
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"error":{"message":"This model's maximum context length is 100 tokens",` +
				`"type":"invalid_request_error","code":"context_length_exceeded"}}`))
			return
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

	step := domain.Tool{
		Name:        "step",
		Description: "One step.",
		Execute:     func(map[string]any) (string, error) { return "HUGE result", nil },
	}
	wf := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "overflow", Version: "1.0.0", TargetActionID: "ask"},
		Resources: []*domain.Resource{{
			ActionID: "ask",
			Chat:     &domain.ChatConfig{Model: "fake", Role: "user", Prompt: "go", Tools: []domain.Tool{step}},
		}},
	}
	engine := executor.NewEngine(t.Context(), slog.Default())
	registry := executor.NewRegistry()
	registry.SetLLMExecutor(executorLLM.NewAdapter(t.Context(), srv.URL))
	engine.SetRegistry(registry)

	out, err := engine.Execute(wf, nil)
	require.NoError(t, err)
	raw, _ := json.Marshal(out)
	assert.Contains(t, string(raw), "RECOVERED")
	mu.Lock()
	defer mu.Unlock()
	require.Len(t, seen, 4, "call, overflowing follow-up, summary, retry")
	assert.Contains(t, seen[2], "HUGE result", "the summary sees the tool result")
	assert.Contains(t, seen[3], "Summary of previous work: ran step, got a huge result")
	assert.Contains(t, seen[3], "Original request: go")
	assert.NotContains(t, seen[3], "HUGE")
}
