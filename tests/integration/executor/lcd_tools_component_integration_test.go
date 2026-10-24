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

package executor_test

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

func requirePython(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err = exec.LookPath("python"); err != nil {
			t.Skip("no python interpreter")
		}
	}
}

// The tools written in YAML over python: give the same answer to the agent
// and to a workflow component: both reach the one declared block.
func TestLCDTools_ComponentMatchesAgentTool(t *testing.T) {
	requirePython(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("KDEPS_COMPONENT_DIR", t.TempDir())
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"query":{"pages":[{"index":1,"title":"Go","extract":"A language."}]}}`)
	}))
	t.Cleanup(srv.Close)
	t.Setenv("KDEPS_WIKIPEDIA_URL", srv.URL)

	dbPath := filepath.Join(t.TempDir(), "pets.db")
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE pets (name TEXT); INSERT INTO pets VALUES ('rex')`)
	require.NoError(t, err)
	require.NoError(t, db.Close())

	reg := kdepstools.NewRegistry()
	agent.RegisterBuiltinTools(t.Context(), reg)
	for _, tc := range []struct {
		tool, with string
		args       map[string]any
		want       string
	}{
		{"calculator", "    expression: \"2 + 3 * 4\"\n", map[string]any{"expression": "2 + 3 * 4"}, "14"},
		{"wikipedia", "    query: \"golang\"\n", map[string]any{"query": "golang"}, "Go\nA language."},
		{"sql_list_tables", "    db_path: \"" + filepath.ToSlash(dbPath) + "\"\n",
			map[string]any{"db_path": filepath.ToSlash(dbPath)}, "pets"},
		{"sql_query", "    db_path: \"" + filepath.ToSlash(dbPath) + "\"\n    query: \"SELECT name FROM pets\"\n",
			map[string]any{"db_path": filepath.ToSlash(dbPath), "query": "SELECT name FROM pets"}, "name\nrex"},
	} {
		got, toolErr := reg.Get(tc.tool).Execute(tc.args)
		require.NoError(t, toolErr, tc.tool)
		require.Equal(t, tc.want, got, tc.tool)
		comp, compErr := runComponentValue(t, tc.tool, tc.with)
		require.NoError(t, compErr, tc.tool)
		require.Equal(t, tc.want, comp, tc.tool)
	}
}
