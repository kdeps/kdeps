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

package chat

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTestScript writes an executable shell script and returns its path.
func writeTestScript(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+body+"\n"), 0o755))
	return path
}

func TestExecutor_Run_NoWorkflow(t *testing.T) {
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	session := &Session{Dir: t.TempDir()}
	err := exec.Run(context.Background(), session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow")
}

func TestExecutor_ExportK8s_NoWorkflow(t *testing.T) {
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	session := &Session{Dir: t.TempDir()}
	err := exec.ExportK8s(context.Background(), session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow")
}

func TestExecutor_Run_BinaryNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = "/nonexistent/kdeps-binary"

	session := &Session{
		Dir: t.TempDir(),
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{"workflow.yaml": "apiVersion: kdeps.io/v1\n"},
		},
	}

	err := exec.Run(ctx, session)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestNewExecutor_DefaultBin(t *testing.T) {
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	// KDepsBin should be set to the current executable or "kdeps"
	assert.NotEmpty(t, exec.KDepsBin)
}

func TestNewExecutor_FallbackBin(t *testing.T) {
	old := osExecutable
	osExecutable = func() (string, error) { return "", nil }
	defer func() { osExecutable = old }()

	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	assert.Equal(t, "kdeps", exec.KDepsBin)
}

func TestExecutor_ExportK8s_BinaryNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = "/nonexistent/kdeps-binary"

	session := &Session{
		Dir: t.TempDir(),
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{"workflow.yaml": "apiVersion: kdeps.io/v1\n"},
		},
	}

	err := exec.ExportK8s(ctx, session)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestExecutor_Run_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = writeTestScript(t, t.TempDir(), "kdeps-ok", "exit 0")

	session := &Session{
		Dir: t.TempDir(),
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{"workflow.yaml": "apiVersion: kdeps.io/v1\n"},
		},
	}

	err := exec.Run(ctx, session)
	require.NoError(t, err)
}

func TestExecutor_ExportK8s_WriteWorkflowError(t *testing.T) {
	out := &strings.Builder{}
	exec := NewExecutor(out, out)

	// Create a read-only parent so MkdirAll inside WriteWorkflow fails
	tmp := t.TempDir()
	readonlyParent := filepath.Join(tmp, "readonly")
	require.NoError(t, os.MkdirAll(readonlyParent, 0o555))

	session := &Session{
		Dir: filepath.Join(readonlyParent, "subdir"), // does not exist, parent is read-only
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{"workflow.yaml": "apiVersion: kdeps.io/v1\n"},
		},
	}

	err := exec.ExportK8s(context.Background(), session)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not write workflow")
}

func TestExecutor_Run_WriteWorkflowError(t *testing.T) {
	out := &strings.Builder{}
	exec := NewExecutor(out, out)

	tmp := t.TempDir()
	readonlyParent := filepath.Join(tmp, "readonly")
	require.NoError(t, os.MkdirAll(readonlyParent, 0o555))

	session := &Session{
		Dir: filepath.Join(readonlyParent, "subdir"),
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{"workflow.yaml": "apiVersion: kdeps.io/v1\n"},
		},
	}

	err := exec.Run(context.Background(), session)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not write workflow")
}

func TestExecutor_ExportK8s_Success(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = writeTestScript(t, t.TempDir(), "kdeps-ok", "exit 0")

	session := &Session{
		Dir: t.TempDir(),
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{"workflow.yaml": "apiVersion: kdeps.io/v1\n"},
		},
	}

	err := exec.ExportK8s(ctx, session)
	require.NoError(t, err)
}
