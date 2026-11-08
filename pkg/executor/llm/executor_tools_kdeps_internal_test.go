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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/api"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func registerEchoTool(t *testing.T) {
	t.Helper()
	api.RegisterTool(api.ToolSpec{
		Name:        "kt_echo",
		Description: "Echo the text.",
		Parameters:  map[string]domain.ToolParam{"text": {Type: "string", Required: true}},
		Call: func(_ context.Context, args map[string]any) (string, error) {
			s, _ := args["text"].(string)
			return "echo:" + s, nil
		},
	})
	api.RegisterTool(api.ToolSpec{
		Name:      "kt_session",
		AgentOnly: true,
		Call:      func(context.Context, map[string]any) (string, error) { return "", nil },
	})
}

func TestKdepsTools_ResolvesAPITool(t *testing.T) {
	registerEchoTool(t)
	_, err := api.Start(t.Context(), "", "127.0.0.1:0")
	require.NoError(t, err)

	in := []domain.Tool{
		{Name: "kt_echo"},
		{Name: "own", Script: "res"},
		{Name: "kt_echo", Description: "Mine."},
	}
	out, err := kdepsTools(t.Context(), in)
	require.NoError(t, err)
	require.Len(t, out, 3)

	assert.Equal(t, "Echo the text.", out[0].Description)
	assert.Contains(t, out[0].Parameters, "text")
	require.NotNil(t, out[0].Execute)
	got, err := out[0].Execute(map[string]any{"text": "hi"})
	require.NoError(t, err)
	assert.Equal(t, "echo:hi", got)

	assert.Nil(t, out[1].Execute, "a script tool stays a resource tool")
	assert.Equal(t, "Mine.", out[2].Description, "the YAML description wins")
	assert.Nil(t, in[0].Execute, "the workflow config is not changed")
}

func TestKdepsTools_Errors(t *testing.T) {
	registerEchoTool(t)
	_, err := kdepsTools(t.Context(), []domain.Tool{{Name: "kt_missing"}})
	require.ErrorContains(t, err, `chat tool "kt_missing": no script, mcp or kdeps tool`)

	_, err = kdepsTools(t.Context(), []domain.Tool{{Name: "kt_session"}})
	require.ErrorContains(t, err, "works only in an agent session")
}

// TestExecute_KdepsToolRoundTrip: the model calls a kdeps tool by name and
// answers from its result.
func TestExecute_KdepsToolRoundTrip(t *testing.T) {
	registerEchoTool(t)
	_, err := api.Start(t.Context(), "", "127.0.0.1:0")
	require.NoError(t, err)
	t.Setenv("HOME", t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Messages []map[string]any `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		reply := `{"name": "kt_echo", "arguments": {"text": "hi"}}`
		if last := body.Messages[len(body.Messages)-1]; last["role"] == "tool" {
			reply = fmt.Sprintf("GOT %v", last["content"])
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(withOpenAIChoices(map[string]any{
			"model":   "m",
			"message": map[string]any{"role": "assistant", "content": reply},
			"done":    true,
		}))
	}))
	t.Cleanup(server.Close)
	ctx, err := executor.NewExecutionContext(
		t.Context(),
		&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}},
	)
	require.NoError(t, err)
	cfg := &domain.ChatConfig{Prompt: "echo hi", Tools: []domain.Tool{{Name: "kt_echo"}}}
	cfg.Model = "llama3.2:1b"
	cfg.BaseURL = server.URL
	out, err := NewExecutor(server.URL).Execute(ctx, cfg)
	require.NoError(t, err)
	assert.Contains(t, resultText(out), "echo:hi")

	cfg.Tools = []domain.Tool{{Name: "kt_missing"}}
	_, err = NewExecutor(server.URL).Execute(ctx, cfg)
	require.ErrorContains(t, err, "kt_missing")
}
