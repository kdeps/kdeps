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

package agent

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// The go toolchain is on PATH on every platform the tests run on, so
// `go env <VAR>` stands in for a real tool program.

func TestCallDeclaredExecTool_RendersArgsAndResult(t *testing.T) {
	s := &execToolSpec{
		Command: "go",
		Args:    []string{"env", "{{.args.var}}", "{{.args.missing}}"}, // an empty arg is dropped
		Result:  "os={{trimSuffix .stdout \"\\n\"}}",
	}
	out, err := callDeclaredExecTool(context.Background(), "goenv", s, map[string]any{"var": "GOOS"})
	require.NoError(t, err)
	assert.Equal(t, "os="+runtime.GOOS, out)

	s.Result = ""
	out, err = callDeclaredExecTool(context.Background(), "goenv", s, map[string]any{"var": "GOARCH"})
	require.NoError(t, err)
	assert.Equal(t, runtime.GOARCH, out, "stdout trimmed when there is no result template")
}

func TestCallDeclaredExecTool_Errors(t *testing.T) {
	ctx := context.Background()
	_, err := callDeclaredExecTool(ctx, "t", &execToolSpec{Command: "go", Args: []string{"no-such-subcommand"}}, nil)
	require.ErrorContains(t, err, "t: go: exit status", "a failing run reports its exit and stderr")

	_, err = callDeclaredExecTool(ctx, "t", &execToolSpec{Command: "kdeps-no-such-program", Install: "pkg"}, nil)
	require.ErrorContains(t, err, "t: kdeps-no-such-program not found in PATH (install pkg)")

	t.Setenv("KDEPS_ALLOW_ROOT", "")
	_, err = callDeclaredExecTool(ctx, "t", &execToolSpec{Command: "go", Paths: []string{"file"}},
		map[string]any{"file": "/"})
	require.ErrorContains(t, err, "t: operation on root directory", "paths are checked against the boundary")

	_, err = callDeclaredExecTool(ctx, "t", &execToolSpec{Command: "go", Args: []string{"{{fromJSON .args.x}}"}},
		map[string]any{"x": "not json"})
	require.ErrorContains(t, err, "not valid JSON")
}

func TestExecToolSpec_ValidateAndTimeout(t *testing.T) {
	require.NoError(t, (&execToolSpec{Command: "tesseract", Args: []string{"{{.args.file}}"}}).validate())
	require.ErrorContains(t, (&execToolSpec{}).validate(), "command is required")
	require.ErrorContains(t, (&execToolSpec{Command: "x", Timeout: "soon"}).validate(), "timeout")
	require.ErrorContains(t, (&execToolSpec{Command: "x", Args: []string{"{{.args"}}).validate(), "exec:")

	assert.Equal(t, defaultExecToolTimeout, (&execToolSpec{}).timeout())
	assert.Equal(t, 2*time.Second, (&execToolSpec{Timeout: "2s"}).timeout())
}

func TestValidateToolDefinition_ExecBlock(t *testing.T) {
	require.NoError(t, validateToolDefinition([]byte("version: 1.0.0\ndescription: d\nexec:\n  command: go\n"), "t"))
	require.ErrorContains(t, validateToolDefinition(
		[]byte("version: 1.0.0\ndescription: d\nexec:\n  args: [x]\n"), "t"), "command is required")
	require.ErrorContains(t, validateToolDefinition(
		[]byte("version: 1.0.0\ndescription: d\nexec:\n  command: go\nhttp:\n  method: GET\n  url: https://x\n"), "t"),
		"not both")
}

func TestDeclaredExecExecute_Defaults(t *testing.T) {
	d := toolDefinition{
		Parameters: map[string]domain.ToolParam{"var": {Type: "string", Default: "GOOS"}},
		Exec:       &execToolSpec{Command: "go", Args: []string{"env", "{{.args.var}}"}},
	}
	out, err := declaredExecExecute(context.Background(), "goenv", d)(nil)
	require.NoError(t, err)
	assert.Equal(t, runtime.GOOS, out, "a declared default fills a missing argument")
}
