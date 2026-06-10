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
	"io"
	"os"
	"testing"

	"github.com/chzyer/readline"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// newTestReadline pipes input into a *readline.Instance so that the first
// Readline() call returns the provided input line.
func newTestReadline(t *testing.T, input string) *readline.Instance {
	t.Helper()
	rIn, wIn, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	if _, err = io.WriteString(wIn, input); err != nil {
		t.Fatalf("write input: %v", err)
	}
	wIn.Close()

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
	return rl
}

// captureStdout replaces os.Stdout with a pipe for the duration of f and
// returns any output written to os.Stdout during the call.
func captureStdout(t *testing.T, f func()) string {
	t.Helper()
	pr, pw, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = pw

	f()

	pw.Close()
	os.Stdout = orig

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, pr)
	pr.Close()
	return buf.String()
}

// minimalWorkflow returns a bare *domain.Workflow that passes the assertions
// in readlineStep (non-nil workflow with Metadata.Name / TargetActionID).
func minimalWorkflow() *domain.Workflow {
	return &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:           "test",
			TargetActionID: "chat",
		},
	}
}

// mockReadline returns the configured line and error on each Readline() call.
type mockReadline struct {
	line string
	err  error
}

func (m *mockReadline) Readline() (string, error) {
	return m.line, m.err
}
