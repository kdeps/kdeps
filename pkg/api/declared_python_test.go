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

package api

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPythonToolSpec_Validate(t *testing.T) {
	require.Error(t, (&PythonToolSpec{}).Validate(), "a script is required")
	require.Error(t, (&PythonToolSpec{Script: "print(1)", Timeout: "soon"}).Validate())
	require.NoError(t, (&PythonToolSpec{Script: "print(1)", Timeout: "5s"}).Validate())
}

func TestPythonToolSpec_ResolveEnv(t *testing.T) {
	s := &PythonToolSpec{Env: map[string][]string{"KEY": {"KDEPS_TEST_PY_KEY"}}}
	t.Setenv("KDEPS_TEST_PY_KEY", "")
	_, ok := s.ResolveEnv()
	assert.False(t, ok, "a required env var that is unset keeps the tool off")
	t.Setenv("KDEPS_TEST_PY_KEY", "v")
	env, ok := s.ResolveEnv()
	require.True(t, ok)
	assert.Equal(t, "v", env["KEY"])
}

func TestCallDeclaredPythonTool_RejectsRootPath(t *testing.T) {
	s := &PythonToolSpec{Script: "print(1)", Paths: []string{"file"}}
	_, err := callDeclaredPythonTool(t.Context(), "py", s, map[string]any{"file": "/"}, nil)
	require.ErrorContains(t, err, "py:")
}

// A script with no packages runs on the system python: no uv, no venv.
func TestCallDeclaredPythonTool_NoPackagesRunsSystemPython(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err = exec.LookPath("python"); err != nil {
			t.Skip("no python interpreter in PATH")
		}
	}
	s := &PythonToolSpec{Script: "import json, sys, os\nprint(json.loads(sys.argv[1])['w'].upper() + os.environ['K'])"}
	out, err := callDeclaredPythonTool(t.Context(), "py", s, map[string]any{"w": "kdeps"}, map[string]string{"K": "!"})
	require.NoError(t, err)
	assert.Equal(t, "KDEPS!", out)

	_, err = callDeclaredPythonTool(
		t.Context(),
		"py",
		&PythonToolSpec{Script: "import sys; sys.exit('nope')"},
		nil,
		nil,
	)
	require.ErrorContains(t, err, "nope")

	// What Windows python prints for "a\nb": the answer has \n line ends everywhere.
	crlf := &PythonToolSpec{Script: "import sys; sys.stdout.buffer.write(b'a\\r\\nb\\r\\n')"}
	out, err = callDeclaredPythonTool(t.Context(), "py", crlf, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, "a\nb", out)
}

func TestCallDeclaredPythonTool_NoInterpreter(t *testing.T) {
	orig := pythonCandidates
	t.Cleanup(func() { pythonCandidates = orig })
	pythonCandidates = []string{"kdeps-no-such-python"}
	_, err := callDeclaredPythonTool(t.Context(), "py", &PythonToolSpec{Script: "print(1)"}, nil, nil)
	require.ErrorContains(t, err, "no python interpreter found")
}
