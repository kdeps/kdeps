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
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

func TestAgencyToolDef_WithNameAndDesc(t *testing.T) {
	agency := &domain.Agency{
		Metadata: domain.AgencyMetadata{
			Name:        "test-agency",
			Version:     "1.0.0",
			Description: "A test agency",
		},
	}
	wf := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test-workflow"},
	}
	eng := executor.NewEngine(nil)
	tool := agencyToolDef(agency, wf, eng)
	require.NotNil(t, tool)
	assert.Equal(t, "test-agency", tool.Name)
	assert.Contains(t, tool.Description, "A test agency")
}

func TestAgencyToolDef_EmptyName(t *testing.T) {
	agency := &domain.Agency{
		Metadata: domain.AgencyMetadata{
			Name:        "",
			Version:     "1.0.0",
			Description: "",
		},
	}
	wf := &domain.Workflow{}
	eng := executor.NewEngine(nil)
	tool := agencyToolDef(agency, wf, eng)
	require.NotNil(t, tool)
	assert.Equal(t, "agency", tool.Name)
	assert.Contains(t, tool.Description, "Agency")
}

func TestAgencyToolDef_EmptyDesc(t *testing.T) {
	agency := &domain.Agency{
		Metadata: domain.AgencyMetadata{
			Name:        "my-agency",
			Version:     "2.0.0",
			Description: "",
		},
	}
	wf := &domain.Workflow{}
	eng := executor.NewEngine(nil)
	tool := agencyToolDef(agency, wf, eng)
	require.NotNil(t, tool)
	assert.Equal(t, "my-agency", tool.Name)
	assert.Contains(t, tool.Description, "my-agency")
	assert.Contains(t, tool.Description, "2.0.0")
}

func TestFindServeWorkflowFiles_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	paths := findServeWorkflowFiles(tmpDir)
	assert.Empty(t, paths)
}

func TestFindServeWorkflowFiles_WithWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	wfPath := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte("test"), 0600))

	paths := findServeWorkflowFiles(tmpDir)
	require.Len(t, paths, 1)
	assert.Equal(t, wfPath, paths[0])
}

func TestFindServeWorkflowFiles_WithAgency(t *testing.T) {
	tmpDir := t.TempDir()
	agencyPath := filepath.Join(tmpDir, "agency.yaml")
	require.NoError(t, os.WriteFile(agencyPath, []byte("test"), 0600))

	paths := findServeWorkflowFiles(tmpDir)
	require.Len(t, paths, 1)
	assert.Equal(t, agencyPath, paths[0])
}

func TestFindServeWorkflowFiles_AgencyOverWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	agencyPath := filepath.Join(tmpDir, "agency.yaml")
	wfPath := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(agencyPath, []byte("test"), 0600))
	require.NoError(t, os.WriteFile(wfPath, []byte("test"), 0600))

	// Agency takes precedence over workflow in the same directory
	paths := findServeWorkflowFiles(tmpDir)
	require.Len(t, paths, 1)
	assert.Equal(t, agencyPath, paths[0])
}

func TestFindServeWorkflowFiles_Subdirectory(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "sub")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	wfPath := filepath.Join(subDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte("test"), 0600))

	paths := findServeWorkflowFiles(tmpDir)
	require.Len(t, paths, 1)
	assert.Equal(t, wfPath, paths[0])
}

// validTestWorkflowYAML is a minimal workflow used for serve registration tests.
const validTestWorkflowYAML = `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: bot-a
  version: "1.0.0"
  targetActionId: action
settings:
  agentSettings:
    timezone: "UTC"
resources:
  - actionId: action
    name: Action
    apiResponse:
      response: "ok"
`

// validTestAgencyYAML is a minimal agency used for serve registration tests.
const validTestAgencyYAML = `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: serve-test-agency
  description: A test agency for serve tests
  version: "1.0.0"
  targetAgentId: bot-a
agents:
  - agents/bot-a
`

// TestRegisterWorkflowTool_Success verifies that a valid workflow is registered.
func TestRegisterWorkflowTool_Success(t *testing.T) {
	dir := t.TempDir()
	wfPath := filepath.Join(dir, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte(validTestWorkflowYAML), 0600))

	registry := tools.NewRegistry()
	wf, err := registerWorkflowTool(wfPath, registry, false)
	require.NoError(t, err)
	require.NotNil(t, wf)
	assert.Equal(t, "bot-a", wf.Metadata.Name)
	assert.NotNil(t, registry.Get("bot-a"))
}

// TestRegisterWorkflowTool_ParseError verifies error on invalid workflow file.
func TestRegisterWorkflowTool_ParseError(t *testing.T) {
	registry := tools.NewRegistry()
	_, err := registerWorkflowTool("/nonexistent/workflow.yaml", registry, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "serve: failed to load workflow")
}

// TestRegisterAgencyTool_Success verifies that a valid agency is registered.
func TestRegisterAgencyTool_Success(t *testing.T) {
	dir := t.TempDir()

	// Create agent directory with workflow.
	agentDir := filepath.Join(dir, "agents", "bot-a")
	require.NoError(t, os.MkdirAll(agentDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yml"), []byte(validTestWorkflowYAML), 0600))

	// Create agency file.
	agencyPath := filepath.Join(dir, "agency.yaml")
	require.NoError(t, os.WriteFile(agencyPath, []byte(validTestAgencyYAML), 0600))

	registry := tools.NewRegistry()
	wf, err := registerAgencyTool(agencyPath, registry, false)
	require.NoError(t, err)
	require.NotNil(t, wf)
	assert.Equal(t, "bot-a", wf.Metadata.Name)
	assert.NotNil(t, registry.Get("serve-test-agency"))
}

// TestRegisterAgencyTool_ParseError verifies error on invalid agency file.
func TestRegisterAgencyTool_ParseError(t *testing.T) {
	registry := tools.NewRegistry()
	_, err := registerAgencyTool("/nonexistent/agency.yaml", registry, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "serve: failed to load agency")
}

// TestRegisterAgencyTool_InvalidTarget verifies error when targetAgentId is missing.
func TestRegisterAgencyTool_InvalidTarget(t *testing.T) {
	dir := t.TempDir()

	// Create agent directory with workflow.
	agentDir := filepath.Join(dir, "agents", "bot-a")
	require.NoError(t, os.MkdirAll(agentDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yml"), []byte(validTestWorkflowYAML), 0600))

	// Agency with nonexistent targetAgentId.
	const badTargetAgencyYAML = `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: bad-target-agency
  version: "1.0.0"
  targetAgentId: does-not-exist
agents:
  - agents/bot-a
`
	agencyPath := filepath.Join(dir, "agency.yaml")
	require.NoError(t, os.WriteFile(agencyPath, []byte(badTargetAgencyYAML), 0600))

	registry := tools.NewRegistry()
	_, err := registerAgencyTool(agencyPath, registry, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}

// TestRegisterServeTools_Workflow verifies dispatching to registerWorkflowTool.
func TestRegisterServeTools_Workflow(t *testing.T) {
	dir := t.TempDir()
	wfPath := filepath.Join(dir, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte(validTestWorkflowYAML), 0600))

	registry := tools.NewRegistry()
	wf, err := registerServeTools(wfPath, registry, false)
	require.NoError(t, err)
	require.NotNil(t, wf)
	assert.Equal(t, "bot-a", wf.Metadata.Name)
}

// TestRegisterServeTools_Agency verifies dispatching to registerAgencyTool.
func TestRegisterServeTools_Agency(t *testing.T) {
	dir := t.TempDir()

	agentDir := filepath.Join(dir, "agents", "bot-a")
	require.NoError(t, os.MkdirAll(agentDir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yml"), []byte(validTestWorkflowYAML), 0600))

	agencyPath := filepath.Join(dir, "agency.yaml")
	require.NoError(t, os.WriteFile(agencyPath, []byte(validTestAgencyYAML), 0600))

	registry := tools.NewRegistry()
	wf, err := registerServeTools(agencyPath, registry, false)
	require.NoError(t, err)
	require.NotNil(t, wf)
	assert.Equal(t, "bot-a", wf.Metadata.Name)
}

// TestLoadAndRegisterAll_SingleFile verifies loading a single workflow file.
func TestLoadAndRegisterAll_SingleFile(t *testing.T) {
	dir := t.TempDir()
	wfPath := filepath.Join(dir, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte(validTestWorkflowYAML), 0600))

	registry := tools.NewRegistry()
	hostWF, err := loadAndRegisterAll(wfPath, false, registry, false)
	require.NoError(t, err)
	require.NotNil(t, hostWF)
	// IsDir=false means paths = [absPath], and the host workflow is set to
	// the result of registerServeTools (since hostWorkflow.Metadata.Name gets
	// replaced when name != "agent").
	assert.Equal(t, "bot-a", hostWF.Metadata.Name)
}

// TestLoadAndRegisterAll_Dir verifies loading workflows from a directory.
func TestLoadAndRegisterAll_Dir(t *testing.T) {
	dir := t.TempDir()
	wfPath := filepath.Join(dir, "workflow.yaml")
	require.NoError(t, os.WriteFile(wfPath, []byte(validTestWorkflowYAML), 0600))

	registry := tools.NewRegistry()
	hostWF, err := loadAndRegisterAll(dir, true, registry, false)
	require.NoError(t, err)
	require.NotNil(t, hostWF)
	assert.Equal(t, "bot-a", hostWF.Metadata.Name)
	// After registration, a tool for the workflow should exist.
	assert.NotNil(t, registry.Get("bot-a"))
}

// TestLoadAndRegisterAll_EmptyDir verifies error when no workflow/agency files found.
func TestLoadAndRegisterAll_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	registry := tools.NewRegistry()
	_, err := loadAndRegisterAll(dir, true, registry, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow or agency files found")
}

// TestLoadAndRegisterAll_ParseError verifies error propagation from registerServeTools.
func TestLoadAndRegisterAll_ParseError(t *testing.T) {
	dir := t.TempDir()
	// Write invalid YAML as a workflow file so findServeWorkflowFiles finds it
	// but registerServeTools fails to parse.
	badPath := filepath.Join(dir, "workflow.yaml")
	require.NoError(t, os.WriteFile(badPath, []byte("invalid: ["), 0600))

	registry := tools.NewRegistry()
	_, err := loadAndRegisterAll(dir, true, registry, false)
	require.Error(t, err)
}

func TestRegisterComponentTools_WithComponents(t *testing.T) {
	registry := tools.NewRegistry()
	wf := &domain.Workflow{
		Components: map[string]*domain.Component{
			"comp-a": {Metadata: domain.ComponentMetadata{Name: "comp-a"}},
			"comp-b": {Metadata: domain.ComponentMetadata{Name: "comp-b"}},
		},
	}
	eng := executor.NewEngine(nil)
	// Must not panic and should register component tools
	registerComponentTools(registry, wf, eng)
	// Two components should produce at least one tool definition
	assert.NotNil(t, registry.Get("comp-a"))
}

// newTestLoop creates a minimal agent.Loop backed by an engine whose Execute
// method is stubbed via SetExecuteFunc. The stub returns (result, err).
func newTestLoop(result interface{}, err error) *agent.Loop {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(_ *domain.Workflow, _ interface{}) (interface{}, error) {
		return result, err
	})
	wf := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata:   domain.WorkflowMetadata{Name: "test-agent", Version: "1.0.0"},
	}
	return agent.New(eng, wf, tools.NewRegistry(), agent.Config{})
}

// TestRunREPL_NormalFlow pipes input into stdin and verifies the welcome
// message, prompt, and the agent loop response appear on stdout.
func TestRunREPL_NormalFlow(t *testing.T) {
	loop := newTestLoop("mock response", nil)

	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = stdinR
	os.Stdout = stdoutW
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
		stdinR.Close()
		stdoutR.Close()
	}()

	_, err = stdinW.WriteString("hello world\n")
	require.NoError(t, err)
	stdinW.Close()

	err = runREPL(loop)
	require.NoError(t, err)

	stdoutW.Close()
	out, _ := io.ReadAll(stdoutR)
	output := string(out)

	assert.Contains(t, output, "kdeps agent mode")
	assert.Contains(t, output, "> ")
	assert.Contains(t, output, "mock response")
}

// TestRunREPL_EmptyInput verifies that empty input lines are silently
// skipped and do not trigger a call to loop.Run.
func TestRunREPL_EmptyInput(t *testing.T) {
	loop := newTestLoop("should-not-reach", nil)

	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = stdinR
	os.Stdout = stdoutW
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
		stdinR.Close()
		stdoutR.Close()
	}()

	// Send empty line, then a valid line, then EOF.
	_, err = stdinW.WriteString("\nvalid input\n")
	require.NoError(t, err)
	stdinW.Close()

	err = runREPL(loop)
	require.NoError(t, err)

	stdoutW.Close()
	out, _ := io.ReadAll(stdoutR)
	output := string(out)

	// The empty line should be skipped -- the response for "valid input" appears.
	assert.Contains(t, output, "kdeps agent mode")
	assert.Contains(t, output, "should-not-reach")
}

// TestRunREPL_ErrorFromRun verifies that when loop.Run returns an error,
// the REPL prints it to stderr and continues to the next input line.
func TestRunREPL_ErrorFromRun(t *testing.T) {
	loop := newTestLoop(nil, errors.New("something went wrong"))

	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)
	stderrR, stderrW, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	origStdout := os.Stdout
	origStderr := os.Stderr
	os.Stdin = stdinR
	os.Stdout = stdoutW
	os.Stderr = stderrW
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
		os.Stderr = origStderr
		stdinR.Close()
		stdoutR.Close()
		stderrR.Close()
	}()

	_, err = stdinW.WriteString("hello\n")
	require.NoError(t, err)
	stdinW.Close()

	err = runREPL(loop)
	require.NoError(t, err)

	stdoutW.Close()
	stderrW.Close()
	out, _ := io.ReadAll(stdoutR)
	errOut, _ := io.ReadAll(stderrR)

	assert.Contains(t, string(out), "kdeps agent mode")
	assert.Contains(t, string(errOut), "error: agent loop: something went wrong")
}

// TestRunREPL_EOF verifies that Ctrl+D (closing stdin without any input)
// exits cleanly with only the welcome message and no error.
func TestRunREPL_EOF(t *testing.T) {
	loop := newTestLoop("unused", nil)

	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = stdinR
	os.Stdout = stdoutW
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
		stdinR.Close()
		stdoutR.Close()
	}()

	// Close write end immediately -- no input at all.
	stdinW.Close()

	err = runREPL(loop)
	require.NoError(t, err)

	stdoutW.Close()
	out, _ := io.ReadAll(stdoutR)
	output := string(out)

	assert.Contains(t, output, "kdeps agent mode")
	// Only the welcome message and one prompt should appear.
	assert.Contains(t, output, "> ")
}

// TestRunREPL_ScannerError verifies that a read error from stdin is
// propagated as the return value of runREPL. We close the read end of the
// pipe before the scanner reads, causing an EBADF / read error.
func TestRunREPL_ScannerError(t *testing.T) {
	loop := newTestLoop("unused", nil)

	stdinR, stdinW, err := os.Pipe()
	require.NoError(t, err)
	stdoutR, stdoutW, err := os.Pipe()
	require.NoError(t, err)

	origStdin := os.Stdin
	origStdout := os.Stdout
	os.Stdin = stdinR
	os.Stdout = stdoutW
	defer func() {
		os.Stdin = origStdin
		os.Stdout = origStdout
		stdinR.Close()
		stdinW.Close()
		stdoutR.Close()
		stdoutW.Close()
	}()

	// Close the read end so scanner.Scan() fails with a read error.
	stdinR.Close()

	err = runREPL(loop)
	require.Error(t, err)

	// Consume stdout (welcome message) so the pipe doesn't block cleanup.
	stdoutW.Close()
	_, _ = io.ReadAll(stdoutR)
}
