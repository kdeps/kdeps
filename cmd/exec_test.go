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

//go:build !js

package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorkflowInDir_Found(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(""), 0600))
	p, err := resolveWorkflowInDir(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "workflow.yaml"), p)
}

func TestResolveWorkflowInDir_Agency(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agency.yaml"), []byte(""), 0600))
	p, err := resolveWorkflowInDir(dir)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "agency.yaml"), p)
}

func TestResolveWorkflowInDir_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := resolveWorkflowInDir(dir)
	assert.Error(t, err)
}

func TestKdepsAgentsDir_Default(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_AGENTS_DIR"))
	dir, err := kdepsAgentsDir()
	require.NoError(t, err)
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".kdeps", "agents"), dir)
}

func TestKdepsAgentsDir_Override(t *testing.T) {
	t.Setenv("KDEPS_AGENTS_DIR", "/override/agents")
	dir, err := kdepsAgentsDir()
	require.NoError(t, err)
	assert.Equal(t, "/override/agents", dir)
}

func TestRunInstalledAgent_NotInstalled(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("KDEPS_AGENTS_DIR", tmp)
	cmd := newExecCmd()
	err := runInstalledAgent(cmd, "ghost-agent", &RunFlags{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not installed")
}

func TestRunInstalledAgent_Success(t *testing.T) {
	agentsDir := t.TempDir()
	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)

	agentName := "test-agent"
	agentDir := filepath.Join(agentsDir, agentName)
	require.NoError(t, os.MkdirAll(agentDir, 0755))

	wfContent := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-agent
  version: "1.0.0"
  targetActionId: test-action
settings:
  agentSettings:
    timezone: "UTC"
resources:
  - actionId: test-action
    name: Test Action
    apiResponse:
      success: true
`
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yaml"), []byte(wfContent), 0600))

	cmd := newExecCmd()
	err := runInstalledAgent(cmd, agentName, &RunFlags{})
	// The simple workflow should execute successfully since it is a basic
	// apiResponse workflow with no external dependencies.
	require.NoError(t, err)
}

func TestRunInstalledAgent_DirNoWorkflow(t *testing.T) {
	agentsDir := t.TempDir()
	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)

	agentName := "empty-agent"
	agentDir := filepath.Join(agentsDir, agentName)
	require.NoError(t, os.MkdirAll(agentDir, 0755))

	cmd := newExecCmd()
	err := runInstalledAgent(cmd, agentName, &RunFlags{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow or agency manifest found")
}

func TestExecCmd_NoArgs(t *testing.T) {
	rootCmd := createRootCommand()
	rootCmd.SetArgs([]string{"exec"})
	err := rootCmd.Execute()
	assert.Error(t, err)
}
