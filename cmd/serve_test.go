//go:build !js

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd_AgentLoopFlags(t *testing.T) {
	cmd := NewRootCmd()
	require.NotNil(t, cmd)
	assert.Equal(t, "kdeps [path]", cmd.Use)
	for _, flagName := range []string{"model", "backend", "base-url", "system"} {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("expected flag --%s on root command", flagName)
		}
	}
}

func TestRootCmd_AcceptsOptionalPath(t *testing.T) {
	cmd := NewRootCmd()
	// Zero args: OK (model-only mode).
	require.NoError(t, cmd.Args(cmd, []string{}))
	// One arg: OK (load tools from path).
	require.NoError(t, cmd.Args(cmd, []string{"workflow.yaml"}))
	// Two args: error.
	require.Error(t, cmd.Args(cmd, []string{"a", "b"}))
}

func TestRunAgentLoopCmd_NonExistentPath(t *testing.T) {
	err := runAgentLoopCmd("/nonexistent/path", &agentLoopFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "path not found")
}

func TestRunAgentLoopCmd_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	err := runAgentLoopCmd(tmpDir, &agentLoopFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow or agency files found")
}

func TestRunAgentLoopCmd_NoPath(t *testing.T) {
	t.Setenv("KDEPS_AGENT_BACKEND", "openai")
	t.Setenv("KDEPS_AGENT_BASE_URL", "http://127.0.0.1:1")

	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })
	os.Stdin = r

	_, err = w.WriteString("hello\n")
	require.NoError(t, err)
	w.Close()

	// No path: model-only mode, loop starts and processes one message then exits on EOF.
	err = runAgentLoopCmd("", &agentLoopFlags{Debug: true})
	assert.NoError(t, err)
}

func TestRunAgentLoopCmd_WithPath(t *testing.T) {
	t.Setenv("KDEPS_AGENT_BACKEND", "openai")
	t.Setenv("KDEPS_AGENT_BASE_URL", "http://127.0.0.1:1")
	tmpDir := t.TempDir()

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

	r, w, err := os.Pipe()
	require.NoError(t, err)
	oldStdin := os.Stdin
	t.Cleanup(func() { os.Stdin = oldStdin })
	os.Stdin = r

	_, err = w.WriteString("hello\n")
	require.NoError(t, err)
	w.Close()

	err = runAgentLoopCmd(tmpDir, &agentLoopFlags{Debug: true})
	assert.NoError(t, err)
}
