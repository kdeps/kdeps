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
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/agent"
	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/executor"
	"github.com/kdeps/kdeps/v3/pkg/tools"
)

// subagentTurnStreamer is the main model calling subagent once, and the
// subagent model answering with a report. It records each subagent request.
type subagentTurnStreamer struct {
	mu       sync.Mutex
	main     int
	requests []string
	tools    [][]string
}

func (s *subagentTurnStreamer) StreamChat(
	_ context.Context, cfg *domain.ChatConfig, w io.Writer,
) (string, []domain.StreamedToolCall, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	parts := []string{cfg.Prompt}
	for _, it := range cfg.Scenario {
		parts = append(parts, it.Prompt)
	}
	text := strings.Join(parts, "\n")
	names := []string{}
	for _, t := range cfg.Tools {
		names = append(names, t.Name)
	}
	// The system prompt may be compressed, so the tool list tells the two
	// apart: only the main loop is offered bash_exec.
	if !slices.Contains(names, "bash_exec") {
		s.requests = append(s.requests, text)
		s.tools = append(s.tools, names)
		_, _ = io.WriteString(w, "REPORT: plan.md has no rollback step")
		return "REPORT: plan.md has no rollback step", nil, nil
	}
	s.main++
	if s.main == 1 {
		return "", []domain.StreamedToolCall{{
			ID: "1", Name: "subagent", Arguments: `{"task":"Review plan.md for gaps.","tools":"read_file"}`,
		}}, nil
	}
	_, _ = io.WriteString(w, "done")
	return "done", nil, nil
}

func subagentTestLoop(t *testing.T, s agent.Streamer) *agent.Loop {
	t.Helper()
	reg := tools.NewRegistry()
	agent.RegisterBuiltinTools(context.Background(), reg)
	wf := &domain.Workflow{
		APIVersion: "kdeps.io/v1", Kind: "Workflow",
		Metadata: domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
	}
	return agent.New(t.Context(), executor.NewEngine(context.Background(), nil), wf, reg,
		agent.Config{Model: "test", Streamer: s})
}

// TestSubagent_AgentTurnHandsATaskToAFreshLoop checks a real turn: the model
// calls subagent, the subagent runs with only the task and the tool it was
// given, and the turn finishes.
func TestSubagent_AgentTurnHandsATaskToAFreshLoop(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	s := &subagentTurnStreamer{}
	loop := subagentTestLoop(t, s)

	var buf bytes.Buffer
	result, err := loop.RunStreaming(t.Context(), "please review my plan, earlier-context-marker", &buf)
	require.NoError(t, err)
	assert.Contains(t, result, "done")

	require.Len(t, s.requests, 1, "the subagent ran once")
	assert.Contains(t, s.requests[0], "Review plan.md for gaps.")
	assert.NotContains(t, s.requests[0], "earlier-context-marker", "the subagent never sees the conversation")
	assert.Contains(t, s.tools[0], "read_file")
	assert.NotContains(t, s.tools[0], "write_file")
	assert.NotContains(t, s.tools[0], "subagent")
}

// TestDesignReviewPreset_SetsTheJudgePanel checks the shipped preset puts its
// four reviewers on the session's judge panel.
func TestDesignReviewPreset_SetsTheJudgePanel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Cleanup(agent.ReloadAssets)
	loop := subagentTestLoop(t, &subagentTurnStreamer{})

	require.NoError(t, agent.ApplyPreset("design-review", loop))
	names := []string{}
	for _, j := range loop.Judges() {
		names = append(names, j.Name)
	}
	assert.Equal(t, []string{"scope", "clarity", "failure-modes", "readiness"}, names)
}

// askStreamer calls ask_question once, then would answer "never".
type askStreamer struct{ calls int }

func (s *askStreamer) StreamChat(
	_ context.Context, _ *domain.ChatConfig, w io.Writer,
) (string, []domain.StreamedToolCall, error) {
	s.calls++
	if s.calls == 1 {
		return "", []domain.StreamedToolCall{{
			ID: "1", Name: "ask_question", Arguments: `{"question":"Which mode?","options":["Fast","Thorough"]}`,
		}}, nil
	}
	_, _ = io.WriteString(w, "never")
	return "never", nil, nil
}

// TestAskQuestion_AgentTurnEndsOnTheQuestion checks a real turn that calls
// ask_question ends with the question as its answer.
func TestAskQuestion_AgentTurnEndsOnTheQuestion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	s := &askStreamer{}
	loop := subagentTestLoop(t, s)

	var buf bytes.Buffer
	result, err := loop.RunStreaming(t.Context(), "set up the review", &buf)
	require.NoError(t, err)
	assert.Contains(t, result, "Which mode?")
	assert.Contains(t, result, "2. Thorough")
	assert.NotContains(t, result, "never")
}
