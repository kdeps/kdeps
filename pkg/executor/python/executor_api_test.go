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

package python_test

import (
	"os"
	execpkg "os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	pythonexecutor "github.com/kdeps/kdeps/v2/pkg/executor/python"
)

// recordingUVManager captures the venv request and returns a fixed path.
type recordingUVManager struct {
	version   string
	packages  []string
	venvName  string
	venvPath  string
	pythonBin string
}

func (m *recordingUVManager) EnsureVenv(
	pythonVersion string,
	packages []string,
	_ string,
	venvName string,
) (string, error) {
	m.version = pythonVersion
	m.packages = packages
	m.venvName = venvName
	return m.venvPath, nil
}

func (m *recordingUVManager) GetPythonPath(_ string) (string, error) {
	return m.pythonBin, nil
}

// echoCommand replaces the python process with echo for the given output.
func echoCommand(output string) func(name string, arg ...string) *execpkg.Cmd {
	return func(_ string, _ ...string) *execpkg.Cmd {
		return execpkg.Command("echo", output)
	}
}

func TestExecutor_Run_InlineScriptParsesJSON(t *testing.T) {
	mgr := &recordingUVManager{venvPath: "/mock/venv", pythonBin: "/mock/venv/bin/python"}
	exec := pythonexecutor.NewExecutor(mgr)
	exec.SetExecCommandForTesting(echoCommand(`{"ok":true,"n":2}`))

	result, err := exec.Run(&domain.PythonConfig{Script: "print('hi')"}, "", nil)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{"ok": true, "n": float64(2)}, result)
	assert.Equal(t, "3.12", mgr.version)
}

func TestExecutor_Run_PlainTextOutput(t *testing.T) {
	mgr := &recordingUVManager{venvPath: "/mock/venv", pythonBin: "/mock/venv/bin/python"}
	exec := pythonexecutor.NewExecutor(mgr)
	exec.SetExecCommandForTesting(echoCommand("hello"))

	result, err := exec.Run(&domain.PythonConfig{Script: "print('hello')"}, "", nil)
	require.NoError(t, err)
	assert.Equal(t, "hello\n", result)
}

func TestExecutor_Run_ExplicitVersionPackagesVenv(t *testing.T) {
	mgr := &recordingUVManager{venvPath: "/mock/venv", pythonBin: "/mock/venv/bin/python"}
	exec := pythonexecutor.NewExecutor(mgr)
	exec.SetExecCommandForTesting(echoCommand("ok"))

	_, err := exec.Run(
		&domain.PythonConfig{Script: "print('x')", VenvName: "shard"},
		"3.11",
		[]string{"pandas"},
	)
	require.NoError(t, err)
	assert.Equal(t, "3.11", mgr.version)
	assert.Equal(t, []string{"pandas"}, mgr.packages)
	assert.Equal(t, "shard", mgr.venvName)
}

func TestExecutor_Run_VersionFromEnv(t *testing.T) {
	t.Setenv("KDEPS_PYTHON_VERSION", "3.13")
	mgr := &recordingUVManager{venvPath: "/mock/venv", pythonBin: "/mock/venv/bin/python"}
	exec := pythonexecutor.NewExecutor(mgr)
	exec.SetExecCommandForTesting(echoCommand("ok"))

	_, err := exec.Run(&domain.PythonConfig{Script: "print('x')"}, "", nil)
	require.NoError(t, err)
	assert.Equal(t, "3.13", mgr.version)
}

func TestExecutor_Run_ScriptFileRelativeToWorkingDir(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "job.py"), []byte("print(1)\n"), 0o600))

	var gotArgs []string
	mgr := &recordingUVManager{venvPath: "/mock/venv", pythonBin: "/mock/venv/bin/python"}
	exec := pythonexecutor.NewExecutor(mgr)
	exec.SetExecCommandForTesting(func(_ string, arg ...string) *execpkg.Cmd {
		gotArgs = arg
		return execpkg.Command("echo", "done")
	})

	_, err := exec.Run(&domain.PythonConfig{ScriptFile: "job.py", Args: []string{"--x"}}, "", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{filepath.Join(dir, "job.py"), "--x"}, gotArgs)
}

func TestExecutor_Run_ScriptFileAbsolute(t *testing.T) {
	path := filepath.Join(t.TempDir(), "job.py")
	require.NoError(t, os.WriteFile(path, []byte("print(1)\n"), 0o600))

	var gotArgs []string
	mgr := &recordingUVManager{venvPath: "/mock/venv", pythonBin: "/mock/venv/bin/python"}
	exec := pythonexecutor.NewExecutor(mgr)
	exec.SetExecCommandForTesting(func(_ string, arg ...string) *execpkg.Cmd {
		gotArgs = arg
		return execpkg.Command("echo", "done")
	})

	_, err := exec.Run(&domain.PythonConfig{ScriptFile: path}, "", nil)
	require.NoError(t, err)
	assert.Equal(t, []string{path}, gotArgs)
}

func TestExecutor_Run_NoScript(t *testing.T) {
	exec := pythonexecutor.NewExecutor(&MockUVManager{})
	_, err := exec.Run(&domain.PythonConfig{}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no script or scriptFile specified")
}

func TestExecutor_Run_EnsureVenvError(t *testing.T) {
	exec := pythonexecutor.NewExecutor(&ErrorEnsureVenvManager{})
	_, err := exec.Run(&domain.PythonConfig{Script: "print(1)"}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to ensure venv")
}

func TestExecutor_Run_GetPythonPathError(t *testing.T) {
	exec := pythonexecutor.NewExecutor(&ErrorGetPythonPathManager{})
	_, err := exec.Run(&domain.PythonConfig{Script: "print(1)"}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get python path")
}

func TestExecutor_Run_FailingCommand(t *testing.T) {
	mgr := &recordingUVManager{venvPath: "/mock/venv", pythonBin: "/mock/venv/bin/python"}
	exec := pythonexecutor.NewExecutor(mgr)
	exec.SetExecCommandForTesting(func(_ string, _ ...string) *execpkg.Cmd {
		return execpkg.Command("false")
	})

	_, err := exec.Run(&domain.PythonConfig{Script: "raise SystemExit(1)"}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "python execution failed")
}

func TestExecutor_Run_OutputLimit(t *testing.T) {
	t.Setenv("KDEPS_PYTHON_MAX_OUTPUT_BYTES", "3")
	mgr := &recordingUVManager{venvPath: "/mock/venv", pythonBin: "/mock/venv/bin/python"}
	exec := pythonexecutor.NewExecutor(mgr)
	exec.SetExecCommandForTesting(echoCommand("far more than three bytes"))

	_, err := exec.Run(&domain.PythonConfig{Script: "print('x')"}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds output limit")
}

func TestExecutor_Run_Timeout(t *testing.T) {
	mgr := &recordingUVManager{venvPath: "/mock/venv", pythonBin: "/mock/venv/bin/python"}
	exec := pythonexecutor.NewExecutor(mgr)
	exec.SetExecCommandForTesting(func(_ string, _ ...string) *execpkg.Cmd {
		return execpkg.Command("sleep", "5")
	})

	_, err := exec.Run(&domain.PythonConfig{Script: "time.sleep(5)", Timeout: "10ms"}, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "python execution failed")
}
