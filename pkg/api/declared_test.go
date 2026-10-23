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
	"net/http"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// registerUnitDeclared serves one test-only declared tool and removes it after.
func registerUnitDeclared(t *testing.T, name string, params map[string]domain.ToolParam, spec DeclaredSpec) {
	t.Helper()
	t.Cleanup(func() {
		toolMu.Lock()
		delete(toolReg, name)
		toolMu.Unlock()
	})
	RegisterDeclaredTool(name, "test declared tool", params, spec)
}

// A declared exec: block runs on its call route, with the declared default
// filling the missing argument.
func TestRegisterDeclaredTool_RunsTheBlock(t *testing.T) {
	registerUnitDeclared(t, "unit_goenv",
		map[string]domain.ToolParam{"var": {Type: "string", Default: "GOOS"}},
		DeclaredSpec{Exec: &ExecToolSpec{Command: "go", Args: []string{"env", "{{.args.var}}"}}})

	s := startTest(t, "tool-token")
	resp := doJSON(t, s.Client(), toolCallURL("unit_goenv"), map[string]any{})
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, runtime.GOOS, mustText(t, resp.Body))
}

func TestRegisterDeclaredTool_RequiredParam(t *testing.T) {
	registerUnitDeclared(t, "unit_req",
		map[string]domain.ToolParam{"q": {Type: "string", Required: true}},
		DeclaredSpec{Exec: &ExecToolSpec{Command: "go", Args: []string{"version"}}})
	_, err := CallTool(t.Context(), "unit_req", map[string]any{})
	require.ErrorContains(t, err, "unit_req: q is required")
}

func TestRegisterDeclaredTool_MissingEnv(t *testing.T) {
	t.Setenv("KDEPS_TEST_DECLARED_KEY", "")
	registerUnitDeclared(t, "unit_env", nil, DeclaredSpec{Exec: &ExecToolSpec{
		Command: "go", Args: []string{"version"},
		Env: map[string][]string{"KEY": {"KDEPS_TEST_DECLARED_KEY"}},
	}})
	_, err := CallTool(t.Context(), "unit_env", map[string]any{})
	require.ErrorContains(t, err, msgEnvUnset)
}

func TestRegisterDeclaredTool_NoBlock(t *testing.T) {
	registerUnitDeclared(t, "unit_none", nil, DeclaredSpec{})
	_, err := CallTool(t.Context(), "unit_none", map[string]any{})
	require.ErrorContains(t, err, "no declared implementation")
}
