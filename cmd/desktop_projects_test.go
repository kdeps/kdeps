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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package cmd_test

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/cmd"
	"github.com/kdeps/kdeps/v2/pkg/desktop"
	"github.com/kdeps/kdeps/v2/pkg/events"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := l.Addr().(*net.TCPAddr).Port
	require.NoError(t, l.Close())
	return port
}

func writeProjectFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}

// helloProject writes a workflow that answers "hi <who>" and returns it.
func helloProject(t *testing.T, dir string, port int) desktop.Project {
	t.Helper()
	path := filepath.Join(dir, "workflow.yaml")
	writeProjectFile(t, path, `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: hello
  version: "1.0.0"
  targetActionId: respond
settings:
  apiServer:
    hostIp: 127.0.0.1
    portNum: `+strconv.Itoa(port)+`
    routes:
      - path: /api/v1/hello
        methods: [GET, POST]
`)
	writeProjectFile(t, filepath.Join(dir, "resources", "respond.yaml"), `actionId: respond
name: Respond
apiResponse:
  success: true
  response:
    msg: "hi {{ get('who') }}"
`)
	return desktop.Project{Path: path, Dir: dir, Kind: desktop.ProjectWorkflow, Name: "hello"}
}

func isolateConfig(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("KDEPS_CONFIG_PATH", filepath.Join(t.TempDir(), "config.yaml"))
}

// syncBuffer collects a child process's output; exec writes stdout and
// stderr from separate goroutines.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

// cliRunner starts this test binary as the kdeps CLI (see TestMain).
func cliRunner() desktop.Runner { return cmd.NewDesktopRunner(os.Args[0], cmd.DesktopCLIFlag) }

func waitDone(t *testing.T, proc *desktop.Process) error {
	t.Helper()
	select {
	case err := <-proc.Done:
		return err
	case <-time.After(60 * time.Second):
		t.Fatal("process did not exit")
		return nil
	}
}

func TestDesktopRunner_StartServesAPI(t *testing.T) {
	isolateConfig(t)
	t.Setenv("KDEPS_API_AUTH_TOKEN", "test-token")
	ws := t.TempDir()
	p := helloProject(t, filepath.Join(ws, "hello"), freePort(t))
	logs := &syncBuffer{}
	proc, err := cliRunner().Start(t.Context(), p, ws, nil, logs)
	require.NoError(t, err)

	addr := regexp.MustCompile(`Starting HTTP server on (\S+)`)
	deadline := time.Now().Add(60 * time.Second)
	var m []string
	for m == nil && time.Now().Before(deadline) {
		m = addr.FindStringSubmatch(logs.String())
		time.Sleep(100 * time.Millisecond)
	}
	require.NotNil(t, m, logs.String())
	assert.True(
		t,
		strings.HasPrefix(logs.String(), "$ kdeps run hello\n"),
		"the relative dir is run, like on the command line",
	)

	var resp *http.Response
	for time.Now().Before(deadline) {
		req, reqErr := http.NewRequestWithContext(
			t.Context(),
			http.MethodGet,
			"http://"+m[1]+"/api/v1/hello?who=srv",
			nil,
		)
		require.NoError(t, reqErr)
		req.Header.Set("Authorization", "Bearer test-token")
		if resp, err = http.DefaultClient.Do(req); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, string(body))
	assert.Contains(t, string(body), "hi srv")

	require.NoError(t, proc.Stop())
	assert.NoError(t, waitDone(t, proc), "a stopped run exits cleanly")
}

func TestDesktopRunner_StartSingleRunWithEnv(t *testing.T) {
	isolateConfig(t)
	ws := t.TempDir()
	dir := filepath.Join(ws, "finder")
	writeProjectFile(t, filepath.Join(dir, "workflow.yaml"), `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: finder
  targetActionId: r
settings:
  agentSettings:
    pythonVersion: "3.12"
`)
	writeProjectFile(t, filepath.Join(dir, "resources", "r.yaml"), `actionId: r
name: r
apiResponse:
  success: true
  response: "root={{ env('HOUSE_FINDER_ROOT') }}"
`)
	p := desktop.Project{
		Path: filepath.Join(dir, "workflow.yaml"),
		Dir:  dir,
		Kind: desktop.ProjectWorkflow,
		Name: "finder",
	}
	logs := &syncBuffer{}
	proc, err := cliRunner().Start(t.Context(), p, ws, []string{"HOUSE_FINDER_ROOT=" + dir}, logs)
	require.NoError(t, err)
	require.NoError(t, waitDone(t, proc), logs.String())
	out := logs.String()
	assert.True(
		t,
		strings.HasPrefix(out, "$ HOUSE_FINDER_ROOT=... kdeps run finder\n"),
		"env names are shown, values are not",
	)
	assert.Contains(t, out, "root="+dir, "the env reaches the run")
}

func TestDesktopRunner_StartFailureExits(t *testing.T) {
	isolateConfig(t)
	ws := t.TempDir()
	dir := filepath.Join(ws, "broken")
	writeProjectFile(t, filepath.Join(dir, "workflow.yaml"), "metadata: [oops\n")
	p := desktop.Project{Path: filepath.Join(dir, "workflow.yaml"), Dir: dir, Kind: desktop.ProjectWorkflow}
	logs := &syncBuffer{}
	proc, err := cliRunner().Start(t.Context(), p, ws, nil, logs)
	require.NoError(t, err)
	assert.Error(t, waitDone(t, proc), "a failed kdeps run reports its exit")
	assert.Contains(t, logs.String(), "yaml", "the reason reaches the log, not only the exit status")
}

func TestDesktopRunner_StartRefusals(t *testing.T) {
	_, err := cliRunner().Start(t.Context(), desktop.Project{Kind: desktop.ProjectComponent}, "", nil, io.Discard)
	assert.ErrorContains(t, err, "inputs")
	_, err = cmd.NewDesktopRunner().
		Start(t.Context(), desktop.Project{Kind: desktop.ProjectWorkflow}, "", nil, io.Discard)
	assert.ErrorContains(t, err, "no kdeps CLI")
	_, err = cmd.NewDesktopRunner(filepath.Join(t.TempDir(), "missing")).Start(
		t.Context(), desktop.Project{Kind: desktop.ProjectWorkflow, Dir: "/x"}, t.TempDir(), nil, io.Discard)
	require.Error(t, err)
}

func TestDesktopRunner_Component(t *testing.T) {
	isolateConfig(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "component.yaml")
	writeProjectFile(t, path, `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: greeter
  description: Greets someone
  version: "1.0.0"
  targetActionId: greet
interface:
  inputs:
    - name: who
      type: string
      required: true
resources:
  - actionId: greet
    name: Greet
    apiResponse:
      success: true
      response: "hello {{ input('who') }}"
`)
	p := desktop.Project{Path: path, Dir: dir, Kind: desktop.ProjectComponent, Name: "greeter"}
	r := cliRunner()
	progress := events.NewChanEmitter(64)
	out, err := r.RunComponent(t.Context(), p, map[string]any{"who": "ada"}, progress)
	require.NoError(t, err)
	assert.Contains(t, out, "hello ada")
	var names []events.EventName
	for len(progress.C()) > 0 {
		names = append(names, (<-progress.C()).Event)
	}
	assert.Contains(t, names, events.EventResourceCompleted)

	tool, err := r.Tool(p)
	require.NoError(t, err)
	assert.Equal(t, "greeter", tool.Name)
	assert.Equal(t, "Greets someone", tool.Description)
	assert.Contains(t, tool.Parameters, "who")

	_, err = r.RunComponent(t.Context(), desktop.Project{Kind: desktop.ProjectWorkflow}, nil, events.NopEmitter{})
	assert.ErrorContains(t, err, "only a component")
	_, err = r.RunComponent(
		t.Context(),
		desktop.Project{Path: filepath.Join(dir, "x", "component.yaml"), Kind: desktop.ProjectComponent},
		nil,
		events.NopEmitter{},
	)
	require.Error(t, err)
}

func TestDesktopRunner_WorkflowTool(t *testing.T) {
	isolateConfig(t)
	p := helloProject(t, t.TempDir(), freePort(t))
	tool, err := cliRunner().Tool(p)
	require.NoError(t, err)
	assert.Equal(t, "hello", tool.Name)
	got, err := tool.Execute(map[string]any{"who": "cy"})
	require.NoError(t, err)
	assert.Contains(t, got, "hi cy")
}

func TestDesktopRunner_AgencyTool(t *testing.T) {
	isolateConfig(t)
	dir := t.TempDir()
	helloProject(t, filepath.Join(dir, "agents", "hello"), freePort(t))
	path := filepath.Join(dir, "agency.yaml")
	writeProjectFile(t, path, `apiVersion: kdeps.io/v1
kind: Agency
metadata:
  name: team
  description: A team
  targetAgentId: hello
agents:
  - agents/hello
`)
	tool, err := cliRunner().Tool(desktop.Project{Path: path, Dir: dir, Kind: desktop.ProjectAgency, Name: "team"})
	require.NoError(t, err)
	assert.Equal(t, "team", tool.Name)
	got, err := tool.Execute(map[string]any{"who": "team"})
	require.NoError(t, err)
	assert.Contains(t, got, "hi team")
}

func TestDesktopRunner_BadConfig(t *testing.T) {
	isolateConfig(t)
	writeProjectFile(t, os.Getenv("KDEPS_CONFIG_PATH"), "llm: [oops\n")
	p := helloProject(t, t.TempDir(), freePort(t))
	_, err := cliRunner().Tool(p)
	assert.ErrorContains(t, err, "config.yaml")
}

func TestDesktopRunner_ToolErrors(t *testing.T) {
	isolateConfig(t)
	r := cliRunner()
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken", "workflow.yaml")
	writeProjectFile(t, broken, "metadata: [oops\n")
	_, err := r.Tool(desktop.Project{Path: broken, Kind: desktop.ProjectWorkflow})
	require.Error(t, err)
	_, err = r.Tool(desktop.Project{Kind: "mystery"})
	assert.ErrorContains(t, err, "unknown project kind")
	_, err = r.Tool(desktop.Project{Path: filepath.Join(dir, "x", "agency.yaml"), Kind: desktop.ProjectAgency})
	require.Error(t, err)
	_, err = r.Tool(desktop.Project{Path: filepath.Join(dir, "x", "component.yaml"), Kind: desktop.ProjectComponent})
	require.Error(t, err)
}
