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

//go:build !js

package agent

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
)

// SQLiteSessionStore persists conversation sessions to a SQLite database.
// Provides the same API as SessionStore plus SearchSessions for content queries.
type SQLiteSessionStore struct {
	sql  sqlSessionStore
	path string
}

const (
	createSessionsSQL = `
CREATE TABLE IF NOT EXISTS sessions (
  id         TEXT    PRIMARY KEY,
  name       TEXT    NOT NULL DEFAULT '',
  model      TEXT    NOT NULL DEFAULT '',
  turns      INTEGER NOT NULL DEFAULT 0,
  created_at INTEGER NOT NULL
);`

	createMessagesSQL = `
CREATE TABLE IF NOT EXISTS messages (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT    NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
  role       TEXT    NOT NULL,
  content    TEXT    NOT NULL,
  seq        INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_session ON messages(session_id, seq);`
)

// NewSQLiteSessionStore opens (or creates) a SQLite session database at dbPath.
// If dbPath is empty, uses ~/.kdeps/sessions/sessions.db.
func NewSQLiteSessionStore(dbPath string) (*SQLiteSessionStore, error) {
	if dbPath == "" {
		home, _ := os.UserHomeDir()
		if home != "" {
			dbPath = filepath.Join(home, sessionDir, "sessions.db")
		}
	}

	mkdirErr := os.MkdirAll(filepath.Dir(dbPath), 0750)
	if mkdirErr != nil {
		return nil, fmt.Errorf("sqlite session store: mkdir: %w", mkdirErr)
	}

	db, openErr := sql.Open("sqlite3", dbPath+"?_journal=WAL&_synchronous=NORMAL")
	if openErr != nil {
		return nil, fmt.Errorf("sqlite session store: open: %w", openErr)
	}

	store := &SQLiteSessionStore{
		sql: sqlSessionStore{
			db:           db,
			sessTable:    "sessions",
			msgTable:     "messages",
			insertSessQL: "INSERT INTO sessions(id, name, model, turns, created_at) VALUES(?, ?, ?, ?, ?)",
			insertMsgQL:  "INSERT INTO messages(session_id, role, content, seq) VALUES(?, ?, ?, ?)",
			searchLike:   "LIKE",
			ph:           "?",
		},
		path: dbPath,
	}

	if migrateErr := store.migrate(); migrateErr != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite session store: migrate: %w", migrateErr)
	}
	return store, nil
}

func (s *SQLiteSessionStore) migrate() error {
	ctx := context.Background()
	if _, err := s.sql.db.ExecContext(ctx, createSessionsSQL); err != nil {
		return err
	}
	_, err := s.sql.db.ExecContext(ctx, createMessagesSQL)
	return err
}

// Close closes the underlying database connection.
func (s *SQLiteSessionStore) Close() error {
	return s.sql.close()
}

// SaveAs persists the session to SQLite with an optional name and model tag.
// Returns the generated session ID.
func (s *SQLiteSessionStore) SaveAs(session *Session, name, model string) (string, error) {
	return s.sql.saveAs(session, name, model)
}

// Save persists the session without a name or model tag.
func (s *SQLiteSessionStore) Save(session *Session) (string, error) {
	return s.sql.saveAs(session, "", "")
}

// Load loads a session from SQLite by ID.
func (s *SQLiteSessionStore) Load(id string) (*Session, error) {
	return s.sql.load(id)
}

// LoadMeta returns metadata for a single session by ID.
func (s *SQLiteSessionStore) LoadMeta(id string) (*SessionMetadata, error) {
	return s.sql.loadMeta(id)
}

// ListMeta returns metadata for all sessions, newest first.
func (s *SQLiteSessionStore) ListMeta() ([]SessionMetadata, error) {
	return s.sql.listMeta()
}

// List returns all stored session IDs, newest first.
func (s *SQLiteSessionStore) List() ([]string, error) {
	return s.sql.list()
}

// Delete removes a session and its messages from the database.
func (s *SQLiteSessionStore) Delete(id string) error {
	return s.sql.delete(id)
}

// SearchSessions returns session IDs whose messages contain the given text.
// Results are ordered newest first.
func (s *SQLiteSessionStore) SearchSessions(text string) ([]string, error) {
	return s.sql.searchSessions(text)
}
