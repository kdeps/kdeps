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
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/executor"
)

func withLoopRunner(t *testing.T, fn LoopRunner) {
	t.Helper()
	prev := currentLoopRunner()
	SetLoopRunner(fn)
	t.Cleanup(func() { SetLoopRunner(prev) })
}

func loopExecCtx(t *testing.T) *executor.ExecutionContext {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	ctx, err := executor.NewExecutionContext(
		t.Context(),
		&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "assistant"}},
	)
	require.NoError(t, err)
	return ctx
}

func TestExecuteLoop_HandsTheResolvedRequestToTheRunner(t *testing.T) {
	var got LoopRequest
	withLoopRunner(t, func(req LoopRequest) (string, error) {
		got = req
		return "all done", nil
	})
	out, err := NewExecutor("").Execute(loopExecCtx(t), &domain.ChatConfig{
		Model:  "llama3.2:1b",
		Prompt: "fix the {{ 'failing' }} test",
		Loop: &domain.ChatLoopConfig{
			Tools:     []string{"read_file", "bash_exec"},
			MaxRounds: 12,
			Session:   "{{ 'user-' + '42' }}",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "all done", resultText(out))
	assert.Equal(t, "fix the failing test", got.Prompt)
	assert.Equal(t, "llama3.2:1b", got.Model)
	assert.Equal(t, []string{"read_file", "bash_exec"}, got.Tools)
	assert.Equal(t, 12, got.MaxRounds)
	assert.Equal(t, "user-42", got.Session)
	assert.Equal(t, "assistant", got.Workflow.Metadata.Name)
	assert.NotNil(t, got.Ctx)
}

func TestExecuteLoop_LiteralSessionAndNone(t *testing.T) {
	var sessions []string
	withLoopRunner(t, func(req LoopRequest) (string, error) {
		sessions = append(sessions, req.Session)
		return "ok", nil
	})
	ex := NewExecutor("")
	ctx := loopExecCtx(t)
	_, err := ex.Execute(ctx, &domain.ChatConfig{
		Model: "m", Prompt: "p", Loop: &domain.ChatLoopConfig{Session: "fixed"},
	})
	require.NoError(t, err)
	_, err = ex.Execute(ctx, &domain.ChatConfig{Model: "m", Prompt: "p", Loop: &domain.ChatLoopConfig{}})
	require.NoError(t, err)
	assert.Equal(t, []string{"fixed", ""}, sessions)
}

func TestExecuteLoop_RunnerError(t *testing.T) {
	withLoopRunner(t, func(LoopRequest) (string, error) { return "", errors.New("model down") })
	_, err := NewExecutor("").Execute(loopExecCtx(t), &domain.ChatConfig{
		Model: "m", Prompt: "p", Loop: &domain.ChatLoopConfig{},
	})
	require.EqualError(t, err, "chat loop: model down")
}

func TestExecuteLoop_NoRunnerInstalled(t *testing.T) {
	withLoopRunner(t, nil)
	_, err := NewExecutor("").Execute(loopExecCtx(t), &domain.ChatConfig{
		Model: "m", Prompt: "p", Loop: &domain.ChatLoopConfig{},
	})
	require.EqualError(t, err, "chat loop: no agent loop runner is installed")
}
