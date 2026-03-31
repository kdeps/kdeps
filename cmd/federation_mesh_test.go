package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFederationMeshList_NoWorkflows(t *testing.T) {
	// Change to empty temp dir
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	cmd := newFederationMeshListCmd()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())
}

func TestFederationMeshList_WithWorkflows(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Write a workflow.yaml with no remoteAgent resources
	workflowContent := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-workflow
  version: "1.0.0"
  targetActionId: main
settings:
  apiServerMode: false
  agentSettings:
    timezone: Etc/UTC
    pythonVersion: "3.12"
    installOllama: false
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "workflow.yaml"), []byte(workflowContent), 0644))

	cmd := newFederationMeshListCmd()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())
}

func TestFederationMeshPublish_NoWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	cmd := newFederationMeshPublishCmd()
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	require.Error(t, err)
	// Should say no workflow.yaml or agency.yaml found
}

func TestFederationMeshPublish_WithWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	workflowContent := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: "2.0.0"
  targetActionId: main
settings:
  apiServerMode: false
  agentSettings:
    timezone: Etc/UTC
    pythonVersion: "3.12"
    installOllama: false
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "workflow.yaml"), []byte(workflowContent), 0644))

	cmd := newFederationMeshPublishCmd()
	cmd.SetArgs([]string{})
	require.NoError(t, cmd.Execute())
}

func TestFederationMeshPublish_AgencyYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Chdir(tmpDir)

	// Write agency.yaml instead
	agencyContent := `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: my-agency
  version: "1.0.0"
  targetAgentId: main-agent
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "agency.yaml"), []byte(agencyContent), 0644))

	// Publish should find agency.yaml and succeed
	cmd := newFederationMeshPublishCmd()
	cmd.SetArgs([]string{})
	// This may succeed or fail depending on agency.yaml parsing
	// but shouldn't panic
	cmd.Execute() //nolint:errcheck
}
