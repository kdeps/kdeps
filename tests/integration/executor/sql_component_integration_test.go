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
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The sql component runs statements against a named connection with the
// results the sql: resource returned (SQLite here; the other databases use
// the same code with their driver).
func TestSQLComponent_SQLite(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not installed (python resources)")
	}
	t.Setenv("HOME", t.TempDir())
	t.Setenv("KDEPS_COMPONENT_DIR", t.TempDir())
	db := filepath.ToSlash(filepath.Join(t.TempDir(), "app.db"))
	cfgPath := filepath.Join(t.TempDir(), "config.yaml")
	require.NoError(
		t,
		os.WriteFile(cfgPath, []byte("sql_connections:\n  main:\n    connection: \"sqlite://"+db+
			"\"\n  mem:\n    connection: \"file::memory:?cache=shared\"\n"), 0o600),
	)
	t.Setenv("KDEPS_CONFIG_PATH", cfgPath)
	call := func(with string) (interface{}, error) {
		t.Helper()
		return runComponentValue(t, "sql", "    connectionName: main\n"+with)
	}
	run := func(with string) interface{} {
		t.Helper()
		out, err := call(with)
		require.NoError(t, err)
		return out
	}

	run("    query: CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT, status TEXT)\n")
	ins, ok := run("    query: \"INSERT INTO users (name, status) VALUES ($1, $2)\"\n    params: [ann, active]\n").(map[string]interface{})
	require.True(t, ok)
	assert.EqualValues(t, 1, ins["rowsAffected"])
	assert.EqualValues(t, 1, ins["lastInsertID"])
	assert.Equal(t, true, ins["success"])

	tx := run("    transaction: true\n    queries:\n" +
		"      - {query: \"INSERT INTO users (name, status) VALUES (?, ?)\", params: [bob, inactive]}\n" +
		"      - {query: \"INSERT INTO users (name, status) VALUES (?, 'active')\", paramsBatch: [[cy], [di]]}\n" +
		"      - {query: \"SELECT count(*) AS n FROM users\"}\n")
	results, ok := tx.([]interface{})
	require.True(t, ok, "%T", tx)
	require.Len(t, results, 3)
	count, ok := results[2].([]interface{})
	require.True(t, ok)
	assert.EqualValues(t, 4, count[0].(map[string]interface{})["n"])

	one, ok := run("    query: \"SELECT name FROM users WHERE status = $1 AND id = $2\"\n    params: [active, 1]\n").(map[string]interface{})
	require.True(t, ok, "one row is the row itself")
	assert.Equal(t, "ann", one["name"])

	many, ok := run("    query: \"SELECT id, name FROM users WHERE status = 'active' ORDER BY id\"\n    maxRows: 2\n").(map[string]interface{})
	require.True(t, ok)
	assert.EqualValues(t, 2, many["rowsAffected"])
	assert.Equal(t, []interface{}{"id", "name"}, many["columns"])

	asJSON := run("    query: \"SELECT name FROM users ORDER BY id\"\n    format: json\n")
	assert.JSONEq(t, `[{"name": "ann"}, {"name": "bob"}, {"name": "cy"}, {"name": "di"}]`, asJSON.(string))
	asCSV := run("    query: \"SELECT id, name FROM users WHERE id < 3 ORDER BY id\"\n    format: csv\n")
	assert.Equal(t, "id,name\n1,ann\n2,bob\n", asCSV)

	mem, err := runComponentValue(t, "sql", "    connectionName: mem\n    query: \"SELECT 1 AS one\"\n")
	require.NoError(t, err)
	assert.EqualValues(t, 1, mem.(map[string]interface{})["one"], "file: URIs are SQLite URIs, not file names")

	_, err = call("    query: \"SELECT * FROM missing\"\n")
	require.ErrorContains(t, err, "query execution failed")
	_, err = runComponentValue(t, "sql", "    connectionName: nope\n    query: SELECT 1\n")
	require.ErrorContains(t, err, "sql connection 'nope' not found")
}
