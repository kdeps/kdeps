//go:build !js

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

package llm

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/chzyer/readline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func TestRunWithIO_ProcessLineError(t *testing.T) {
	orig := processREPLLineFunc
	t.Cleanup(func() { processREPLLineFunc = orig })
	processREPLLineFunc = func(_ io.Writer, _ *domain.Workflow, _ *executor.Engine, _ string, _ string) (bool, error) {
		return false, errors.New("process failed")
	}

	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t", TargetActionID: "chat"}}
	in := bytes.NewBufferString("hello\n")
	out := &bytes.Buffer{}

	err := RunWithIO(context.Background(), wf, eng, nil, in, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "process failed")
}

func TestReadlineStep_Quit(t *testing.T) {
	rl := newTestReadline(t, "/quit\n")
	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	out := captureStdout(t, func() {
		done, err := readlineStep(rl, wf, eng, "sess")
		assert.True(t, done)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Goodbye!")
}

func TestReadlineStep_Exit(t *testing.T) {
	rl := newTestReadline(t, "/exit\n")
	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	out := captureStdout(t, func() {
		done, err := readlineStep(rl, wf, eng, "sess")
		assert.True(t, done)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Goodbye!")
}

func TestReadlineStep_EmptyLine(t *testing.T) {
	rl := newTestReadline(t, "\n")
	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	done, err := readlineStep(rl, wf, eng, "sess")
	assert.False(t, done)
	assert.NoError(t, err)
}

func TestReadlineStep_WhitespaceLine(t *testing.T) {
	rl := newTestReadline(t, "   \n")
	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	done, err := readlineStep(rl, wf, eng, "sess")
	assert.False(t, done)
	assert.NoError(t, err)
}

func TestReadlineStep_NormalMessage(t *testing.T) {
	var gotMsg string
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, req interface{}) (interface{}, error) {
		rc := req.(*executor.RequestContext)
		gotMsg = rc.Body["message"].(string)
		return "response text", nil
	})
	wf := minimalWorkflow()
	rl := newTestReadline(t, "hello\n")

	out := captureStdout(t, func() {
		done, err := readlineStep(rl, wf, eng, "sess")
		assert.False(t, done)
		assert.NoError(t, err)
	})
	assert.Equal(t, "hello", gotMsg)
	assert.Contains(t, out, "response text")
}

func TestReadlineStep_NilResult(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, nil //nolint:nilnil
	})
	wf := minimalWorkflow()
	rl := newTestReadline(t, "ping\n")

	out := captureStdout(t, func() {
		done, err := readlineStep(rl, wf, eng, "sess")
		assert.False(t, done)
		assert.NoError(t, err)
	})
	// nil result writes an empty line (formatResult returns "").
	assert.Equal(t, "\n", out)
}

func TestReadlineStep_NonStringResult(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return 42, nil
	})
	wf := minimalWorkflow()
	rl := newTestReadline(t, "answer\n")

	out := captureStdout(t, func() {
		done, err := readlineStep(rl, wf, eng, "sess")
		assert.False(t, done)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "42")
}

func TestReadlineStep_EngineError(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return nil, errors.New("server error")
	})
	wf := minimalWorkflow()
	rl := newTestReadline(t, "hello\n")

	out := captureStdout(t, func() {
		done, err := readlineStep(rl, wf, eng, "sess")
		assert.False(t, done)
		assert.NoError(t, err)
	})
	assert.Contains(t, out, "Error:")
}

func TestReadlineStep_EOF(t *testing.T) {
	rIn, wIn, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	wIn.Close() // immediate EOF on read end

	rl, err := readline.NewEx(&readline.Config{
		Stdin:        rIn,
		Stdout:       io.Discard,
		Prompt:       "> ",
		HistoryLimit: 5,
	})
	if err != nil {
		t.Fatalf("readline.NewEx: %v", err)
	}
	t.Cleanup(func() { rl.Close(); rIn.Close() })

	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	done, err := readlineStep(rl, wf, eng, "sess")
	assert.True(t, done)
	assert.NoError(t, err)
}

// TestReadlineStep_UnknownCommand_Fallthrough verifies that an unrecognised
// slash command is forwarded to the engine (dispatchCommand returns false).
func TestReadlineStep_UnknownCommand_Fallthrough(t *testing.T) {
	var engineCalled bool
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		engineCalled = true
		return "llm reply", nil
	})
	wf := minimalWorkflow()
	rl := newTestReadline(t, "/unknowncmd bar\n")

	done, err := readlineStep(rl, wf, eng, "sess")
	assert.False(t, done)
	assert.NoError(t, err)
	assert.True(t, engineCalled, "unknown slash command should fall through to engine")
}

func TestReadlineStep_HelpCommand(t *testing.T) {
	rl := newTestReadline(t, "/help\n")
	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	done, err := readlineStep(rl, wf, eng, "sess")
	assert.False(t, done, "/help should not exit the REPL")
	assert.NoError(t, err)
}

func TestReadlineStep_ListCommand(t *testing.T) {
	rl := newTestReadline(t, "/list\n")
	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	done, err := readlineStep(rl, wf, eng, "sess")
	assert.False(t, done, "/list should not exit the REPL")
	assert.NoError(t, err)
}

func TestReadlineStep_InterruptEmptyLine(t *testing.T) {
	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	rl := &mockReadline{err: readline.ErrInterrupt}
	out := captureStdout(t, func() {
		done, err := readlineStep(rl, wf, eng, "sess")
		assert.True(t, done, "interrupt on empty line should exit")
		assert.NoError(t, err)
	})
	assert.Equal(t, "\n", out, "should print a newline before exiting")
}

func TestReadlineStep_InterruptWithPartialLine(t *testing.T) {
	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	rl := &mockReadline{line: "hello", err: readline.ErrInterrupt}
	done, err := readlineStep(rl, wf, eng, "sess")
	assert.False(t, done, "interrupt with partial input should not exit")
	assert.NoError(t, err)
}

func TestReadlineStep_ReadlineError(t *testing.T) {
	eng := executor.NewEngine(nil)
	wf := minimalWorkflow()

	rl := &mockReadline{err: io.ErrClosedPipe}
	done, err := readlineStep(rl, wf, eng, "sess")
	assert.False(t, done)
	assert.ErrorContains(t, err, "llm repl: read")
}
