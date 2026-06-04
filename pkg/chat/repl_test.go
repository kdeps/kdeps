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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestREPL(t *testing.T, llmReply string, input string) (*REPL, *strings.Builder, *Session) {
	t.Helper()
	sessionDir := t.TempDir()
	session := &Session{ID: "test", Dir: sessionDir}

	client := &mockLLMClient{reply: llmReply}
	gen := NewGenerator(client, "llama3", "", "", nil)

	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = "" // disable actual execution in unit tests

	repl := NewREPL(session, gen, exec, strings.NewReader(input), out)
	return repl, out, session
}

func TestREPL_Quit(t *testing.T) {
	repl, out, _ := newTestREPL(t, "", "/quit\n")
	err := repl.Run(context.Background())
	require.NoError(t, err)
	assert.Contains(t, out.String(), "Bye")
}

func TestREPL_QuitAliases(t *testing.T) {
	for _, cmd := range []string{"/exit\n", "/q\n"} {
		repl, out, _ := newTestREPL(t, "", cmd)
		require.NoError(t, repl.Run(context.Background()))
		assert.Contains(t, out.String(), "Bye")
	}
}

func TestREPL_ShowNoWorkflow(t *testing.T) {
	repl, out, _ := newTestREPL(t, "", "/show\n/quit\n")
	require.NoError(t, repl.Run(context.Background()))
	assert.Contains(t, out.String(), "No workflow yet")
}

func TestREPL_RunNoWorkflow(t *testing.T) {
	repl, out, _ := newTestREPL(t, "", "/run\n/quit\n")
	require.NoError(t, repl.Run(context.Background()))
	assert.Contains(t, out.String(), "No workflow yet")
}

func TestREPL_SaveNoWorkflow(t *testing.T) {
	repl, out, _ := newTestREPL(t, "", "/save\n/quit\n")
	require.NoError(t, repl.Run(context.Background()))
	assert.Contains(t, out.String(), "No workflow yet")
}

func TestREPL_ExportNoWorkflow(t *testing.T) {
	repl, out, _ := newTestREPL(t, "", "/export\n/quit\n")
	require.NoError(t, repl.Run(context.Background()))
	assert.Contains(t, out.String(), "No workflow yet")
}

func TestREPL_UnknownCommand(t *testing.T) {
	repl, out, _ := newTestREPL(t, "", "/unknown\n/quit\n")
	require.NoError(t, repl.Run(context.Background()))
	assert.Contains(t, out.String(), "Unknown command")
}

func TestREPL_Reset(t *testing.T) {
	repl, out, session := newTestREPL(t, "", "/reset\n/quit\n")
	session.AddTurn("user", "hello")
	session.Workflow = &GeneratedWorkflow{Files: map[string]string{"workflow.yaml": "x"}}

	require.NoError(t, repl.Run(context.Background()))
	assert.Contains(t, out.String(), "Session reset")
	assert.Empty(t, session.History)
	assert.Nil(t, session.Workflow)
}

func TestREPL_GenerateAndShow(t *testing.T) {
	repl, out, session := newTestREPL(t, validReply, "list files\n/show\n/quit\n")
	require.NoError(t, repl.Run(context.Background()))

	assert.NotNil(t, session.Workflow)
	output := out.String()
	assert.Contains(t, output, "Workflow generated")
	assert.Contains(t, output, "workflow.yaml")
	// /show should print content
	assert.Contains(t, output, "test-agent")
}

func TestREPL_GenerateAndSave(t *testing.T) {
	sessionDir := t.TempDir()
	destDir := filepath.Join(t.TempDir(), "my-workflow")

	session := &Session{ID: "test", Dir: sessionDir}
	client := &mockLLMClient{reply: validReply}
	gen := NewGenerator(client, "llama3", "", "", nil)
	out := &strings.Builder{}
	exec := NewExecutor(out, out)

	input := "do something\n/save " + destDir + "\n/quit\n"
	repl := NewREPL(session, gen, exec, strings.NewReader(input), out)
	require.NoError(t, repl.Run(context.Background()))

	assert.FileExists(t, filepath.Join(destDir, "workflow.yaml"))
	assert.Contains(t, out.String(), "Workflow saved to")
}

func TestREPL_GenerateLLMError(t *testing.T) {
	sessionDir := t.TempDir()
	session := &Session{ID: "test", Dir: sessionDir}

	client := &mockLLMClient{err: assert.AnError}
	gen := NewGenerator(client, "llama3", "", "", nil)
	out := &strings.Builder{}
	exec := NewExecutor(out, out)

	repl := NewREPL(session, gen, exec, strings.NewReader("do something\n/quit\n"), out)
	require.NoError(t, repl.Run(context.Background()))

	// History should be rolled back on error
	assert.Empty(t, session.History)
	assert.Contains(t, out.String(), "Error")
}

func TestREPL_EmptyInput(t *testing.T) {
	repl, _, _ := newTestREPL(t, "", "\n\n\n/quit\n")
	require.NoError(t, repl.Run(context.Background()))
}

func TestREPL_EOFExits(t *testing.T) {
	repl, _, _ := newTestREPL(t, "", "") // EOF immediately
	require.NoError(t, repl.Run(context.Background()))
}

func TestSortedNames(t *testing.T) {
	files := map[string]string{
		"resources/b.yaml": "",
		"workflow.yaml":    "",
		"resources/a.yaml": "",
	}
	names := sortedNames(files)
	assert.Equal(t, []string{"resources/a.yaml", "resources/b.yaml", "workflow.yaml"}, names)
}

func TestREPL_RunWithWorkflow(t *testing.T) {
	sessionDir := t.TempDir()
	scriptDir := t.TempDir()
	script := writeTestScript(t, scriptDir, "kdeps-ok", "exit 0")

	session := &Session{ID: "test", Dir: sessionDir}
	client := &mockLLMClient{reply: validReply}
	gen := NewGenerator(client, "llama3", "", "", nil)
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = script

	repl := NewREPL(session, gen, exec,
		strings.NewReader("do something\n/run\n/quit\n"), out)
	require.NoError(t, repl.Run(context.Background()))

	output := out.String()
	assert.Contains(t, output, "Running workflow...")
	assert.Contains(t, output, "Workflow finished.")
}

func TestREPL_RunWithError(t *testing.T) {
	sessionDir := t.TempDir()
	scriptDir := t.TempDir()
	script := writeTestScript(t, scriptDir, "kdeps-fail", "exit 1")

	session := &Session{ID: "test", Dir: sessionDir}
	client := &mockLLMClient{reply: validReply}
	gen := NewGenerator(client, "llama3", "", "", nil)
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = script

	repl := NewREPL(session, gen, exec,
		strings.NewReader("do something\n/run\n/quit\n"), out)
	require.NoError(t, repl.Run(context.Background()))

	assert.Contains(t, out.String(), "Run failed:")
}

func TestREPL_ExportWithWorkflow(t *testing.T) {
	sessionDir := t.TempDir()
	scriptDir := t.TempDir()
	script := writeTestScript(t, scriptDir, "kdeps-ok", "exit 0")

	session := &Session{ID: "test", Dir: sessionDir}
	client := &mockLLMClient{reply: validReply}
	gen := NewGenerator(client, "llama3", "", "", nil)
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = script

	repl := NewREPL(session, gen, exec,
		strings.NewReader("do something\n/export\n/quit\n"), out)
	require.NoError(t, repl.Run(context.Background()))

	// On success ExportK8s returns silently (no success printed).
	// Assert that no failure message appears.
	assert.NotContains(t, out.String(), "Export failed:")
}

func TestREPL_ExportFailure(t *testing.T) {
	sessionDir := t.TempDir()
	scriptDir := t.TempDir()
	script := writeTestScript(t, scriptDir, "kdeps-fail", "exit 1")

	session := &Session{ID: "test", Dir: sessionDir}
	client := &mockLLMClient{reply: validReply}
	gen := NewGenerator(client, "llama3", "", "", nil)
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = script

	repl := NewREPL(session, gen, exec,
		strings.NewReader("do something\n/export\n/quit\n"), out)
	require.NoError(t, repl.Run(context.Background()))

	assert.Contains(t, out.String(), "Export failed:")
}

func TestPrintEnvVars(t *testing.T) {
	out := &strings.Builder{}
	wf := &GeneratedWorkflow{
		Files: map[string]string{
			"workflow.yaml": "connection: postgres://localhost:5432/mydb",
		},
	}
	printEnvVars(out, wf)
	output := out.String()
	assert.Contains(t, output, "Required environment variables (.env):")
	assert.Contains(t, output, "DATABASE_URL=")
}

func TestPrintEnvVars_Empty(t *testing.T) {
	out := &strings.Builder{}
	wf := &GeneratedWorkflow{
		Files: map[string]string{
			"workflow.yaml": "echo hello",
		},
	}
	printEnvVars(out, wf)
	assert.Empty(t, out.String())
}

func TestREPL_SaveDefaultDest(t *testing.T) {
	// /save with no argument uses "kdeps-workflow"
	sessionDir := t.TempDir()
	session := &Session{
		ID:  "test",
		Dir: sessionDir,
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{"workflow.yaml": "name: x\n"},
		},
	}

	// Change cwd to a temp dir so the default save lands there
	prevDir, _ := os.Getwd()
	saveBase := t.TempDir()
	require.NoError(t, os.Chdir(saveBase))
	defer func() { _ = os.Chdir(prevDir) }()

	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	repl := NewREPL(session, nil, exec, strings.NewReader("/save\n/quit\n"), out)
	require.NoError(t, repl.Run(context.Background()))

	assert.FileExists(t, filepath.Join(saveBase, "kdeps-workflow", "workflow.yaml"))
}

func TestREPL_NoGenerator(t *testing.T) {
	sessionDir := t.TempDir()
	session := &Session{ID: "test", Dir: sessionDir}

	out := &strings.Builder{}
	exec := NewExecutor(out, out)

	repl := NewREPL(session, nil, exec, strings.NewReader("do something\n/quit\n"), out)
	require.NoError(t, repl.Run(context.Background()))
	assert.Contains(t, out.String(), "No LLM configured")
}

func TestREPL_SaveWithError(t *testing.T) {
	sessionDir := t.TempDir()

	session := &Session{
		ID:  "test",
		Dir: sessionDir,
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{"workflow.yaml": "name: x\n"},
		},
	}

	saveDir := t.TempDir()
	readonlyParent := filepath.Join(saveDir, "readonly")
	require.NoError(t, os.MkdirAll(readonlyParent, 0o555))
	dest := filepath.Join(readonlyParent, "subdir")

	out := &strings.Builder{}
	exec := NewExecutor(out, out)

	repl := NewREPL(session, nil, exec,
		strings.NewReader("/save "+dest+"\n/quit\n"), out)
	require.NoError(t, repl.Run(context.Background()))
	assert.Contains(t, out.String(), "Save failed")
}
