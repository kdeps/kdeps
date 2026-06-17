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

package agent

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPostgresSessionStore_RequiresDSN(t *testing.T) {
	t.Parallel()
	_, err := NewPostgresSessionStore("")
	if err == nil {
		t.Fatal("expected error for empty DSN")
	}
	if !strings.Contains(err.Error(), "dsn is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewPostgresSessionStore_InvalidDSN_FailsOnMigrate(t *testing.T) {
	t.Parallel()
	// A well-formed but unreachable DSN should fail at ping/migrate, not at Open.
	_, err := NewPostgresSessionStore("postgres://invalid:invalid@127.0.0.1:15432/nodb?sslmode=disable")
	if err == nil {
		t.Fatal("expected error for unreachable DSN")
	}
}

func TestPostgresSessionStore_ImplementsInterface(t *testing.T) {
	t.Parallel()
	// Compile-time check: *PostgresSessionStore exposes the same methods as SQLiteSessionStore.
	type sessionStoreIface interface {
		Save(session *Session) (string, error)
		SaveAs(session *Session, name, model string) (string, error)
		Load(id string) (*Session, error)
		LoadMeta(id string) (*SessionMetadata, error)
		ListMeta() ([]SessionMetadata, error)
		List() ([]string, error)
		Delete(id string) error
		SearchSessions(text string) ([]string, error)
		Close() error
	}
	var _ sessionStoreIface = (*PostgresSessionStore)(nil)
}

// newSQLiteBackedPostgresStore creates a PostgresSessionStore backed by an in-memory SQLite DB.
// This allows testing the delegating methods without a real PostgreSQL connection.
func newSQLiteBackedPostgresStore(t *testing.T) *PostgresSessionStore {
	t.Helper()
	dbPath := t.TempDir() + "/pg_test.db"
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)

	// Create SQLite-compatible tables
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS kdeps_sessions (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		model TEXT NOT NULL DEFAULT '',
		turns INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL
	)`)
	require.NoError(t, err)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS kdeps_messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		seq INTEGER NOT NULL
	)`)
	require.NoError(t, err)

	return &PostgresSessionStore{
		sql: sqlSessionStore{
			db:           db,
			sessTable:    "kdeps_sessions",
			msgTable:     "kdeps_messages",
			insertSessQL: "INSERT INTO kdeps_sessions(id, name, model, turns, created_at) VALUES(?, ?, ?, ?, ?)",
			insertMsgQL:  "INSERT INTO kdeps_messages(session_id, role, content, seq) VALUES(?, ?, ?, ?)",
			searchLike:   "LIKE",
			ph:           "?",
		},
		dsn: "sqlite-backed-test",
	}
}

func TestPostgresSessionStore_SaveAsAndLoad(t *testing.T) {
	store := newSQLiteBackedPostgresStore(t)
	defer store.Close() //nolint:errcheck

	session := NewSession(0)
	session.messages = []sessionMessage{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}

	id, err := store.SaveAs(session, "test-session", "gpt-4")
	require.NoError(t, err)
	assert.NotEmpty(t, id)

	loaded, err := store.Load(id)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Len(t, loaded.messages, 2)
}

func TestPostgresSessionStore_Save(t *testing.T) {
	store := newSQLiteBackedPostgresStore(t)
	defer store.Close() //nolint:errcheck

	session := NewSession(0)
	id, err := store.Save(session)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestPostgresSessionStore_LoadMeta(t *testing.T) {
	store := newSQLiteBackedPostgresStore(t)
	defer store.Close() //nolint:errcheck

	session := NewSession(0)
	id, err := store.SaveAs(session, "my-session", "model-x")
	require.NoError(t, err)

	meta, err := store.LoadMeta(id)
	require.NoError(t, err)
	require.NotNil(t, meta)
	assert.Equal(t, "my-session", meta.Name)
	assert.Equal(t, "model-x", meta.Model)
}

func TestPostgresSessionStore_ListMeta(t *testing.T) {
	store := newSQLiteBackedPostgresStore(t)
	defer store.Close() //nolint:errcheck

	session := NewSession(0)
	_, err := store.SaveAs(session, "s1", "m1")
	require.NoError(t, err)
	_, err = store.SaveAs(session, "s2", "m2")
	require.NoError(t, err)

	metas, err := store.ListMeta()
	require.NoError(t, err)
	assert.Len(t, metas, 2)
}

func TestPostgresSessionStore_List(t *testing.T) {
	store := newSQLiteBackedPostgresStore(t)
	defer store.Close() //nolint:errcheck

	session := NewSession(0)
	id1, err := store.Save(session)
	require.NoError(t, err)
	id2, err := store.Save(session)
	require.NoError(t, err)

	ids, err := store.List()
	require.NoError(t, err)
	assert.Contains(t, ids, id1)
	assert.Contains(t, ids, id2)
}

func TestPostgresSessionStore_Delete(t *testing.T) {
	store := newSQLiteBackedPostgresStore(t)
	defer store.Close() //nolint:errcheck

	session := NewSession(0)
	id, err := store.Save(session)
	require.NoError(t, err)

	err = store.Delete(id)
	require.NoError(t, err)

	ids, err := store.List()
	require.NoError(t, err)
	assert.NotContains(t, ids, id)
}

func TestPostgresSessionStore_SearchSessions(t *testing.T) {
	store := newSQLiteBackedPostgresStore(t)
	defer store.Close() //nolint:errcheck

	session := NewSession(0)
	session.messages = []sessionMessage{
		{Role: "user", Content: "looking for specific content"},
	}
	id, err := store.SaveAs(session, "s", "m")
	require.NoError(t, err)

	ids, err := store.SearchSessions("specific content")
	require.NoError(t, err)
	assert.Contains(t, ids, id)
}

func TestPostgresSessionStore_Close(t *testing.T) {
	store := newSQLiteBackedPostgresStore(t)
	err := store.Close()
	require.NoError(t, err)
}
