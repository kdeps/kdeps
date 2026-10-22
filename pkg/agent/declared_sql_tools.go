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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3" // sqlite3 driver for declared sql tools
)

// sqlToolSpec implements a declared tool as one parameterized SQL statement
// against a SQLite database. The query text is fixed in the YAML. Arguments are
// bound as parameters in the order listed in params; they are never spliced
// into the text, so a value cannot change the statement.
type sqlToolSpec struct {
	// DSN names the environment variable that holds the SQLite file path,
	// e.g. KDEPS_STATE_DB. The tool is registered only when it is set.
	DSN string `yaml:"dsn"`
	// Query is one statement with ? placeholders, one per params entry.
	Query string `yaml:"query"`
	// Params are the tool parameter names bound to the placeholders, in order.
	Params []string `yaml:"params,omitempty"`
	// Mode is "rows" (default: the result is a JSON array of row objects) or
	// "exec" (the result is the number of rows changed).
	Mode string `yaml:"mode,omitempty"`
}

// resolveDSN returns the database path from the named environment variable.
func (s *sqlToolSpec) resolveDSN() (string, bool) {
	v := strings.TrimSpace(os.Getenv(s.DSN))
	return v, v != ""
}

// runSQLTool executes spec with args bound to its placeholders.
func runSQLTool(ctx context.Context, name string, spec *sqlToolSpec, args map[string]any) (string, error) {
	path, ok := spec.resolveDSN()
	if !ok {
		return "", fmt.Errorf("%s: %s is not set", name, spec.DSN)
	}
	values := make([]any, 0, len(spec.Params))
	for _, p := range spec.Params {
		v, present := args[p]
		if !present {
			return "", fmt.Errorf("%s: %s is required", name, p)
		}
		values = append(values, v)
	}
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return "", fmt.Errorf("%s: open: %w", name, err)
	}
	defer db.Close()

	if spec.Mode == "exec" {
		res, execErr := db.ExecContext(ctx, spec.Query, values...)
		if execErr != nil {
			return "", fmt.Errorf("%s: %w", name, execErr)
		}
		n, _ := res.RowsAffected()
		return fmt.Sprintf("%d rows changed", n), nil
	}
	rows, qErr := db.QueryContext(ctx, spec.Query, values...)
	if qErr != nil {
		return "", fmt.Errorf("%s: %w", name, qErr)
	}
	defer rows.Close()
	return rowsJSON(rows)
}

// rowsJSON encodes every row as an object keyed by column name.
func rowsJSON(rows *sql.Rows) (string, error) {
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	out := []map[string]any{}
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if scanErr := rows.Scan(ptrs...); scanErr != nil {
			return "", scanErr
		}
		row := make(map[string]any, len(cols))
		for i, c := range cols {
			if b, isBytes := vals[i].([]byte); isBytes {
				row[c] = string(b)
			} else {
				row[c] = vals[i]
			}
		}
		out = append(out, row)
	}
	if rowsErr := rows.Err(); rowsErr != nil && !errors.Is(rowsErr, sql.ErrNoRows) {
		return "", rowsErr
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
