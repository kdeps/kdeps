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

package agent_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/api"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

// claimThenActStreamer first claims an action in prose with no tool call,
// then (once nudged) makes the real call, then answers.
type claimThenActStreamer struct {
	calls   int
	prompts []string
}

func (s *claimThenActStreamer) StreamChat(
	_ context.Context, cfg *domain.ChatConfig, w io.Writer,
) (string, []domain.StreamedToolCall, error) {
	defer func() { s.calls++ }()
	s.prompts = append(s.prompts, cfg.Prompt)
	switch s.calls {
	case 0:
		return "Created [notes.md](https://example.invalid/notes.md) containing hello kdeps", nil, nil
	case 1:
		return "", []domain.StreamedToolCall{{ID: "1", Name: "write_note", Arguments: `{}`}}, nil
	default:
		_, _ = io.WriteString(w, "wrote notes.md")
		return "wrote notes.md", nil, nil
	}
}

// A model that reports an action it never performed is pushed back to make
// the real call, on any backend: the tool runs and the turn ends on the real
// result.
func TestUnbackedClaim_DrawsARealToolCall(t *testing.T) {
	writes := 0
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name: "write_note", Description: "writes the note", Parameters: map[string]domain.ToolParam{},
		Execute: func(_ map[string]any) (string, error) { writes++; return "ok", nil },
	})
	ms := &claimThenActStreamer{}
	wf := &domain.Workflow{
		APIVersion: "kdeps.io/v1", Kind: "Workflow",
		Metadata: domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
	}
	require.NoError(t, agent.SetEfficiencyEnabled(false))
	loop := agent.New(executor.NewEngine(nil), wf, reg, agent.Config{Model: "test", Streamer: ms, MaxToolRounds: 10})

	var buf bytes.Buffer
	got, err := loop.RunStreaming(context.Background(), "create notes.md", &buf)
	require.NoError(t, err)
	assert.Equal(t, 1, writes, "the nudged model made the real call")
	assert.Equal(t, "wrote notes.md", got)
	require.GreaterOrEqual(t, len(ms.prompts), 2)
	assert.True(t, strings.Contains(ms.prompts[1], "You made no tool call this turn"),
		"the second call carries the nudge")
}

// A caught refusal is saved in the model's record under ~/.kdeps, and /model
// record shows it through the kdeps API; record clear deletes it.
func TestModelRecord_RefusalReachesSlashModelRecord(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name: "write_note", Description: "writes the note", Parameters: map[string]domain.ToolParam{},
		Execute: func(_ map[string]any) (string, error) { return "ok", nil },
	})
	wf := &domain.Workflow{
		APIVersion: "kdeps.io/v1", Kind: "Workflow",
		Metadata: domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
	}
	require.NoError(t, agent.SetEfficiencyEnabled(false))
	loop := agent.New(executor.NewEngine(nil), wf, reg,
		agent.Config{Model: "itmodel", Backend: "openai", Streamer: &claimThenActStreamer{}, MaxToolRounds: 10})
	var buf bytes.Buffer
	_, err := loop.RunStreaming(context.Background(), "create notes.md", &buf)
	require.NoError(t, err)

	repl := agent.NewREPL(t.Context(), loop)
	repl.RegisterCommandAPI()
	t.Cleanup(func() { api.SetSessionOps(nil) })
	out, err := api.CallCommand(t.Context(), "model", map[string]any{"args": []string{"record"}})
	require.NoError(t, err)
	assert.Contains(t, out, "openai/itmodel: 1 caught refusals")
	assert.Contains(t, out, "Created [notes.md]")

	out, err = api.CallCommand(t.Context(), "model", map[string]any{"args": []string{"record", "clear"}})
	require.NoError(t, err)
	assert.Contains(t, out, "Cleared the refusal record of openai/itmodel")
	out, err = api.CallCommand(t.Context(), "model", map[string]any{"args": []string{"record"}})
	require.NoError(t, err)
	assert.Contains(t, out, "no refusals on record")
}
