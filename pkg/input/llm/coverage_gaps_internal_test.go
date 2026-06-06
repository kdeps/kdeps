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
	"github.com/creack/pty"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func TestRun_ReadlineInitFallback(t *testing.T) {
	origNew := readlineNewEx
	t.Cleanup(func() { readlineNewEx = origNew })
	readlineNewEx = func(_ *readline.Config) (*readline.Instance, error) {
		return nil, errors.New("readline init failed")
	}

	origTerm := isStdinTerminal
	t.Cleanup(func() { isStdinTerminal = origTerm })
	isStdinTerminal = func() bool { return true }

	r, w, err := os.Pipe()
	require.NoError(t, err)
	_, err = io.WriteString(w, "/quit\n")
	require.NoError(t, err)
	w.Close()

	origStdin := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = origStdin })

	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t", TargetActionID: "chat"}}

	err = Run(context.Background(), wf, eng, nil)
	require.NoError(t, err)
}

func TestRun_CtxDoneAtLoopStart(t *testing.T) {
	master, slave, err := pty.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = slave
	os.Stdout = slave
	t.Cleanup(func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t", TargetActionID: "chat"}}

	err = Run(ctx, wf, eng, nil)
	require.NoError(t, err)
}

func TestRun_ReadlineStepError(t *testing.T) {
	origStep := readlineStepFunc
	t.Cleanup(func() { readlineStepFunc = origStep })
	readlineStepFunc = func(_ readlineReader, _ *domain.Workflow, _ *executor.Engine, _ string) (bool, error) {
		return false, errors.New("step failed")
	}

	origNew := readlineNewEx
	t.Cleanup(func() { readlineNewEx = origNew })
	readlineNewEx = func(cfg *readline.Config) (*readline.Instance, error) {
		return readline.NewEx(&readline.Config{
			Stdin:  cfg.Stdin,
			Stdout: io.Discard,
			Prompt: "> ",
		})
	}

	origTerm := isStdinTerminal
	t.Cleanup(func() { isStdinTerminal = origTerm })
	isStdinTerminal = func() bool { return true }

	master, slave, err := pty.Open()
	if err != nil {
		t.Skipf("pty unavailable: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = slave
	os.Stdout = slave
	t.Cleanup(func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
	})

	eng := executor.NewEngine(nil)
	wf := &domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t", TargetActionID: "chat"}}

	err = Run(context.Background(), wf, eng, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "step failed")
}

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
