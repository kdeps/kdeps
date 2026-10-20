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

package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func TestApplyOverrideURL(t *testing.T) {
	cfg := &domain.ChatConfig{Backend: BackendFile, Model: "llama3.2:1b"}
	t.Setenv(OverrideURLEnv, "")
	applyOverrideURL(cfg)
	assert.Equal(t, BackendFile, cfg.Backend, "unset: the resource's backend stands")

	t.Setenv(OverrideURLEnv, " http://127.0.0.1:9/ ")
	applyOverrideURL(cfg)
	assert.Equal(t, backendOpenAI, cfg.Backend)
	assert.Equal(t, "http://127.0.0.1:9", cfg.BaseURL)
	assert.Equal(t, "llama3.2:1b", cfg.Model, "the model name is sent as is")
	applyOverrideURL(nil)
}

func TestExecute_OverrideURLTakesEveryChatCall(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	var calls atomic.Int32
	var model string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		model, _ = body["model"].(string)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "x", "object": "chat.completion", "model": model,
			"choices": []any{map[string]any{"index": 0, "finish_reason": "stop",
				"message": map[string]any{"role": "assistant", "content": "OVERRIDDEN"}}},
		})
	}))
	defer srv.Close()
	t.Setenv(OverrideURLEnv, srv.URL)

	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	// A resource that names the local file backend and a real model: with the
	// override it never downloads or serves that model.
	out, err := NewExecutor(
		"",
	).Execute(ctx, &domain.ChatConfig{Backend: BackendFile, Model: "llama3.2:1b", Prompt: "hi"})
	require.NoError(t, err)
	assert.Equal(t, int32(1), calls.Load())
	assert.Equal(t, "llama3.2:1b", model)
	data, _ := json.Marshal(out)
	assert.Contains(t, string(data), "OVERRIDDEN")
}

func TestStreamChat_OverrideURLTakesTheAgentLoopsCalls(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("OPENAI_API_KEY", "")
	var path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write(
			[]byte(
				`data: {"id":"x","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{"role":"assistant","content":"STREAMED"}}]}` + "\n\n",
			),
		)
		_, _ = w.Write(
			[]byte(
				`data: {"id":"x","object":"chat.completion.chunk","model":"m","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\ndata: [DONE]\n\n",
			),
		)
	}))
	defer srv.Close()
	t.Setenv(OverrideURLEnv, srv.URL)

	var out strings.Builder
	text, _, err := NewExecutor("").StreamChat(t.Context(),
		&domain.ChatConfig{Backend: "anthropic", Model: "claude-x", Prompt: "hi"}, &out)
	require.NoError(t, err, "no key needed: a stub or mock takes the call")
	assert.Equal(t, "/v1/chat/completions", path)
	assert.Contains(t, text, "STREAMED")
}

// An upstream failure from the m365 bridge arrives as a normal content delta
// (see pkg/executor/llm/m365/server.go's producedError case). StreamChat must
// report it as an error, so the agent loop's transient-error retry runs
// instead of saving "[m365 error: ...]" as the assistant's answer.
func TestStreamChat_M365UpstreamErrorIsAnError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte(
			`data: {"id":"x","object":"chat.completion.chunk","model":"m","choices":[` +
				`{"index":0,"delta":{"role":"assistant","content":"[m365 error: m365: read frame: ` +
				`read: connection reset by peer]"}}]}` + "\n\n",
		))
		_, _ = w.Write([]byte(
			`data: {"id":"x","object":"chat.completion.chunk","model":"m","choices":[` +
				`{"index":0,"delta":{},"finish_reason":"stop"}]}` + "\n\ndata: [DONE]\n\n",
		))
	}))
	defer srv.Close()
	t.Setenv(OverrideURLEnv, srv.URL)

	var out strings.Builder
	text, toolCalls, err := NewExecutor("").StreamChat(t.Context(),
		&domain.ChatConfig{Backend: "m365", Model: "claude-sonnet", Prompt: "hi"}, &out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection reset by peer")
	assert.Empty(t, text)
	assert.Empty(t, toolCalls)
}
