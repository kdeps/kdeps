//go:build !js

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewServeCmd_Flags(t *testing.T) {
	cmd := newServeCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.Use != "serve <path>" {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}
	for _, flagName := range []string{"model", "backend", "base-url", "system"} {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("expected flag --%s", flagName)
		}
	}
}

func TestNewServeCmd_RequiresOneArg(t *testing.T) {
	cmd := newServeCmd()
	// Zero args should error.
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("expected error for zero args")
	}
	// Two args should error.
	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Error("expected error for two args")
	}
	// Exactly one arg should be accepted.
	if err := cmd.Args(cmd, []string{"workflow.yaml"}); err != nil {
		t.Errorf("unexpected error for one arg: %v", err)
	}
}

func TestRunServeCmd_NonExistentPath(t *testing.T) {
	err := runServeCmd("/nonexistent/path", &serveFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "serve: path not found")
}

func TestRunServeCmd_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	err := runServeCmd(tmpDir, &serveFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow or agency files found")
}

func TestRunServeCmd_Success(t *testing.T) {
	// Pin the agent away from local backends: the default file backend would
	// download and boot a real llamafile server inside the test.
	t.Setenv("KDEPS_AGENT_BACKEND", "openai")
	t.Setenv("KDEPS_AGENT_BASE_URL", "http://127.0.0.1:1")
	tmpDir := t.TempDir()

	// Create a minimal valid workflow.yaml that passes schema validation.
	workflowContent := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-serve
  targetActionId: action
settings: {}
resources:
  - actionId: action
    name: Test Action
`
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(workflowPath, []byte(workflowContent), 0644))

	// Redirect stdin to a pipe.  Write one input line then close the writer so
	// the REPL scanner hits EOF after a single iteration.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })
	os.Stdin = r

	_, err = w.WriteString("hello\n")
	require.NoError(t, err)
	w.Close()

	err = runServeCmd(tmpDir, &serveFlags{Debug: true})
	// The REPL runs one loop iteration.  The engine execution will fail because
	// no LLM backend is available -- the REPL prints the error to stderr and
	// continues.  The next scanner.Scan() returns false (EOF) so it exits
	// cleanly with nil error.
	assert.NoError(t, err)
}
