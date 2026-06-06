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

package sql

import (
	"bytes"
	"context"
	dbsql "database/sql"
	"encoding/csv"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/mattn/go-sqlite3"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
)

func TestFormatSelectResults_MarshalError(t *testing.T) {
	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("injected marshal error")
	}
	e := NewExecutor()
	results := []map[string]interface{}{{"col": "val"}}
	_, err := e.FormatSelectResults(results, "json")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal results")
}

type failCSVWriter struct {
	failOn int
	calls  int
	inner  *csv.Writer
}

func (w *failCSVWriter) Write(record []string) error {
	w.calls++
	if w.calls >= w.failOn {
		return errors.New("csv write failed")
	}
	return w.inner.Write(record)
}

func (w *failCSVWriter) Flush() { w.inner.Flush() }

func (w *failCSVWriter) Error() error { return w.inner.Error() }

func TestGetConnection_OpenError(t *testing.T) {
	orig := sqlOpen
	t.Cleanup(func() { sqlOpen = orig })
	sqlOpen = func(_, _ string) (*dbsql.DB, error) {
		return nil, errors.New("open failed")
	}

	e := NewExecutor()
	_, err := e.getConnection("postgres://invalid", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open database")
}

func TestExecuteSelectQuery_ReadRowsError(t *testing.T) {
	db, err := dbsql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	defer db.Close()

	orig := rowsScanFunc
	t.Cleanup(func() { rowsScanFunc = orig })
	rowsScanFunc = func(_ *dbsql.Rows, _ ...interface{}) error {
		return errors.New("scan failed")
	}

	_, err = db.Exec("CREATE TABLE t (id INT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO t VALUES (1)")
	require.NoError(t, err)

	e := NewExecutor()
	_, err = e.ExecuteSelectQuery(context.Background(), db, "SELECT id FROM t", nil, 10)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read rows")
}

func TestFormatSelectResults_CSVError(t *testing.T) {
	orig := csvNewWriter
	t.Cleanup(func() { csvNewWriter = orig })
	csvNewWriter = func(w io.Writer) csvWriter {
		return &failCSVWriter{inner: csv.NewWriter(w), failOn: 1}
	}

	e := NewExecutor()
	_, err := e.FormatSelectResults([]map[string]interface{}{{"a": 1}}, "csv")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to format as CSV")
}

func TestFormatAsCSV_WriteErrors(t *testing.T) {
	e := NewExecutor()
	data := []map[string]interface{}{{"id": 1, "name": "Alice"}}

	t.Run("header write error", func(t *testing.T) {
		orig := csvNewWriter
		t.Cleanup(func() { csvNewWriter = orig })
		csvNewWriter = func(w io.Writer) csvWriter {
			return &failCSVWriter{inner: csv.NewWriter(w), failOn: 1}
		}
		_, err := e.FormatAsCSV(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write CSV header")
	})

	t.Run("row write error", func(t *testing.T) {
		orig := csvNewWriter
		t.Cleanup(func() { csvNewWriter = orig })
		csvNewWriter = func(w io.Writer) csvWriter {
			return &failCSVWriter{inner: csv.NewWriter(w), failOn: 2}
		}
		_, err := e.FormatAsCSV(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write CSV row")
	})

	t.Run("flush error", func(t *testing.T) {
		orig := csvNewWriter
		t.Cleanup(func() { csvNewWriter = orig })
		csvNewWriter = func(w io.Writer) csvWriter {
			return &flushErrWriter{Writer: csv.NewWriter(w)}
		}
		_, err := e.FormatAsCSV(data)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "CSV writer error")
	})
}

type flushErrWriter struct{ *csv.Writer }

func (f *flushErrWriter) Error() error { return errors.New("csv flush err") }

func TestReadRowsWithLimit_ScanError(t *testing.T) {
	db, err := dbsql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	defer db.Close()

	orig := rowsScanFunc
	t.Cleanup(func() { rowsScanFunc = orig })
	rowsScanFunc = func(_ *dbsql.Rows, _ ...interface{}) error {
		return errors.New("scan failed")
	}

	_, err = db.Exec("CREATE TABLE t (id INT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO t VALUES (1)")
	require.NoError(t, err)

	e := NewExecutor()
	rows, err := db.Query("SELECT id FROM t")
	require.NoError(t, err)

	_, err = e.ReadRowsWithLimit(rows, 10)
	require.Error(t, err)
}

func TestScanRow_Error(t *testing.T) {
	db, err := dbsql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	defer db.Close()

	orig := rowsScanFunc
	t.Cleanup(func() { rowsScanFunc = orig })
	rowsScanFunc = func(_ *dbsql.Rows, _ ...interface{}) error {
		return errors.New("scan row failed")
	}

	_, err = db.Exec("CREATE TABLE t (id INT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO t VALUES (1)")
	require.NoError(t, err)

	e := NewExecutor()
	rows, err := db.Query("SELECT id FROM t")
	require.NoError(t, err)
	defer rows.Close()
	require.True(t, rows.Next())

	_, err = e.scanRow(rows, []string{"id"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scan row")
}

func TestExecuteBatchQuery_ReadRowsError(t *testing.T) {
	db, err := dbsql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	defer db.Close()

	orig := rowsScanFunc
	t.Cleanup(func() { rowsScanFunc = orig })
	rowsScanFunc = func(_ *dbsql.Rows, _ ...interface{}) error {
		return errors.New("batch scan failed")
	}

	_, err = db.Exec("CREATE TABLE t (id INT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO t VALUES (1)")
	require.NoError(t, err)

	tx, err := db.Begin()
	require.NoError(t, err)

	e := NewExecutor()
	ctx, err := executor.NewExecutionContext(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "t"}})
	require.NoError(t, err)
	eval := expression.NewEvaluator(ctx.API)

	_, err = e.executeBatchQuery(ctx, eval, tx, "SELECT id FROM t", `[[]]`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read batch query rows")
}

func TestReadRows_Error(t *testing.T) {
	db, err := dbsql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Skipf("sqlite unavailable: %v", err)
	}
	defer db.Close()

	orig := rowsScanFunc
	t.Cleanup(func() { rowsScanFunc = orig })
	rowsScanFunc = func(_ *dbsql.Rows, _ ...interface{}) error {
		return errors.New("batch scan failed")
	}

	_, err = db.Exec("CREATE TABLE t (id INT)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO t VALUES (1)")
	require.NoError(t, err)

	rows, err := db.Query("SELECT id FROM t")
	require.NoError(t, err)

	e := NewExecutor()
	_, err = e.readRows(rows)
	require.Error(t, err)
}

func TestFormatAsCSV_EmptyResults(t *testing.T) {
	e := NewExecutor()
	out, err := e.FormatAsCSV(nil)
	require.NoError(t, err)
	assert.Equal(t, "", strings.TrimSpace(out))
	_ = bytes.NewBufferString(out)
}
