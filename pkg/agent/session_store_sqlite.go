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
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3" // sqlite3 driver
)

// SQLiteSessionStore persists conversation sessions to a SQLite database.
// Provides the same API as SessionStore plus SearchSessions for content queries.
type SQLiteSessionStore struct {
	mu   sync.Mutex
	db   *sql.DB
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

	store := &SQLiteSessionStore{db: db, path: dbPath}
	if migrateErr := store.migrate(); migrateErr != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite session store: migrate: %w", migrateErr)
	}
	return store, nil
}

func (s *SQLiteSessionStore) migrate() error {
	ctx := context.Background()
	if _, err := s.db.ExecContext(ctx, createSessionsSQL); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, createMessagesSQL)
	return err
}

// Close closes the underlying database connection.
func (s *SQLiteSessionStore) Close() error {
	return s.db.Close()
}

// SaveAs persists the session to SQLite with an optional name and model tag.
// Returns the generated session ID.
func (s *SQLiteSessionStore) SaveAs(session *Session, name, model string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	id := fmt.Sprintf("session-%d", time.Now().UnixNano())
	now := time.Now().UnixMilli()

	tx, txErr := s.db.BeginTx(ctx, nil)
	if txErr != nil {
		return "", fmt.Errorf("sqlite session store: begin tx: %w", txErr)
	}
	defer func() { _ = tx.Rollback() }()

	_, insertErr := tx.ExecContext(ctx,
		`INSERT INTO sessions(id, name, model, turns, created_at) VALUES(?, ?, ?, ?, ?)`,
		id, name, model, session.TurnCount(), now,
	)
	if insertErr != nil {
		return "", fmt.Errorf("sqlite session store: insert session: %w", insertErr)
	}

	for i, m := range session.Messages() {
		_, msgErr := tx.ExecContext(ctx,
			`INSERT INTO messages(session_id, role, content, seq) VALUES(?, ?, ?, ?)`,
			id, m.Role, m.Content, i,
		)
		if msgErr != nil {
			return "", fmt.Errorf("sqlite session store: insert message: %w", msgErr)
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return "", fmt.Errorf("sqlite session store: commit: %w", commitErr)
	}
	return id, nil
}

// Save persists the session without a name or model tag.
func (s *SQLiteSessionStore) Save(session *Session) (string, error) {
	return s.SaveAs(session, "", "")
}

// Load loads a session from SQLite by ID.
func (s *SQLiteSessionStore) Load(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	rows, queryErr := s.db.QueryContext(ctx,
		`SELECT role, content FROM messages WHERE session_id = ? ORDER BY seq`,
		id,
	)
	if queryErr != nil {
		return nil, fmt.Errorf("sqlite session store: query messages: %w", queryErr)
	}
	defer rows.Close()

	session := NewSession(0)
	for rows.Next() {
		var role, content string
		if scanErr := rows.Scan(&role, &content); scanErr != nil {
			return nil, fmt.Errorf("sqlite session store: scan message: %w", scanErr)
		}
		session.messages = append(session.messages, sessionMessage{Role: role, Content: content})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("sqlite session store: rows error: %w", rowsErr)
	}
	return session, nil
}

// LoadMeta returns metadata for a single session by ID.
func (s *SQLiteSessionStore) LoadMeta(id string) (*SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, model, turns, created_at FROM sessions WHERE id = ?`, id,
	)
	var m SessionMetadata
	scanErr := row.Scan(&m.ID, &m.Name, &m.Model, &m.Turns, &m.CreatedAt)
	if scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return nil, fmt.Errorf("sqlite session store: session %q not found", id)
		}
		return nil, fmt.Errorf("sqlite session store: scan metadata: %w", scanErr)
	}
	return &m, nil
}

// ListMeta returns metadata for all sessions, newest first.
func (s *SQLiteSessionStore) ListMeta() ([]SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	rows, queryErr := s.db.QueryContext(ctx,
		`SELECT id, name, model, turns, created_at FROM sessions ORDER BY created_at DESC`,
	)
	if queryErr != nil {
		return nil, fmt.Errorf("sqlite session store: list: %w", queryErr)
	}
	defer rows.Close()

	var metas []SessionMetadata
	for rows.Next() {
		var m SessionMetadata
		if scanErr := rows.Scan(&m.ID, &m.Name, &m.Model, &m.Turns, &m.CreatedAt); scanErr != nil {
			return nil, fmt.Errorf("sqlite session store: scan: %w", scanErr)
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

// List returns all stored session IDs, newest first.
func (s *SQLiteSessionStore) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	rows, queryErr := s.db.QueryContext(ctx,
		`SELECT id FROM sessions ORDER BY created_at DESC`,
	)
	if queryErr != nil {
		return nil, fmt.Errorf("sqlite session store: list: %w", queryErr)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// Delete removes a session and its messages from the database.
func (s *SQLiteSessionStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	_, deleteErr := s.db.ExecContext(ctx,
		`DELETE FROM sessions WHERE id = ?`, id,
	)
	if deleteErr != nil {
		return fmt.Errorf("sqlite session store: delete %s: %w", id, deleteErr)
	}
	return nil
}

// SearchSessions returns session IDs whose messages contain the given text.
// Results are ordered newest first. JSONL store has no equivalent.
func (s *SQLiteSessionStore) SearchSessions(text string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	rows, queryErr := s.db.QueryContext(ctx, `
		SELECT DISTINCT session_id
		  FROM messages
		 WHERE content LIKE ?
		 ORDER BY session_id DESC
	`, "%"+text+"%")
	if queryErr != nil {
		return nil, fmt.Errorf("sqlite session store: search: %w", queryErr)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			return nil, scanErr
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
