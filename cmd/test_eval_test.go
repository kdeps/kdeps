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

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	executorLLM "github.com/kdeps/kdeps/v3/pkg/executor/llm"
)

// evalJudgeServer is an OpenAI-compatible mock that answers every chat with
// reply.
func evalJudgeServer(t *testing.T, reply string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			_, _ = w.Write([]byte(`{"data":[{"id":"judge"}]}`))
			return
		}
		raw, _ := json.Marshal(map[string]any{
			"id": "x", "object": "chat.completion", "model": "judge",
			"choices": []any{map[string]any{"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": reply}}},
		})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(raw)
	}))
	t.Cleanup(srv.Close)
	t.Setenv(executorLLM.OverrideURLEnv, srv.URL)
}

func TestEvalReview(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	tc := domain.TestCase{
		Name:       "tone",
		Request:    domain.TestRequest{Path: "/ask"},
		JudgeModel: &domain.EvalModel{Name: "judge", Backend: "openai"},
	}
	spec := domain.Judge{Name: "polite", Criteria: "The answer is polite."}

	evalJudgeServer(t, `{"approved": false, "feedback": "too curt"}`)
	v := evalReview(setupEngine(nil, false))(t.Context(), tc, spec, "no")
	assert.False(t, v.Approved)
	assert.Equal(t, "too curt", v.Feedback)

	evalJudgeServer(t, `{"approved": true, "feedback": ""}`)
	v = evalReview(setupEngine(nil, false))(t.Context(), tc, spec, "hello, thanks")
	assert.True(t, v.Approved)

	t.Setenv(executorLLM.OverrideURLEnv, "http://127.0.0.1:1")
	v = evalReview(setupEngine(nil, false))(t.Context(), tc, spec, "hi")
	assert.False(t, v.Approved, "a review that cannot run rejects")
	assert.NotEmpty(t, v.Feedback)
}
