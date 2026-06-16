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
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
)

// PostgresSessionStore persists conversation sessions to a PostgreSQL database.
// Compatible with AlloyDB, CloudSQL, pgvector, and standard PostgreSQL.
// Implements the same API as SQLiteSessionStore.
type PostgresSessionStore struct {
	sql sqlSessionStore
	dsn string
}

const (
	pgCreateSessionsSQL = `
CREATE TABLE IF NOT EXISTS kdeps_sessions (
  id         TEXT    PRIMARY KEY,
  name       TEXT    NOT NULL DEFAULT '',
  model      TEXT    NOT NULL DEFAULT '',
  turns      INTEGER NOT NULL DEFAULT 0,
  created_at BIGINT  NOT NULL
);`

	pgCreateMessagesSQL = `
CREATE TABLE IF NOT EXISTS kdeps_messages (
  id         BIGSERIAL PRIMARY KEY,
  session_id TEXT      NOT NULL REFERENCES kdeps_sessions(id) ON DELETE CASCADE,
  role       TEXT      NOT NULL,
  content    TEXT      NOT NULL,
  seq        INTEGER   NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_kdeps_messages_session ON kdeps_messages(session_id, seq);`
)

// NewPostgresSessionStore opens a PostgreSQL session database at dsn.
// dsn is a standard PostgreSQL DSN, e.g. "postgres://user:pass@localhost/dbname".
func NewPostgresSessionStore(dsn string) (*PostgresSessionStore, error) {
	if dsn == "" {
		return nil, errors.New("postgres session store: dsn is required")
	}

	db, openErr := sql.Open("postgres", dsn)
	if openErr != nil {
		return nil, fmt.Errorf("postgres session store: open: %w", openErr)
	}

	store := &PostgresSessionStore{
		sql: sqlSessionStore{
			db:           db,
			sessTable:    "kdeps_sessions",
			msgTable:     "kdeps_messages",
			insertSessQL: "INSERT INTO kdeps_sessions(id, name, model, turns, created_at) VALUES($1, $2, $3, $4, $5)",
			insertMsgQL:  "INSERT INTO kdeps_messages(session_id, role, content, seq) VALUES($1, $2, $3, $4)",
			searchLike:   "ILIKE",
			ph:           "$1",
		},
		dsn: dsn,
	}

	if migrateErr := store.migrate(); migrateErr != nil {
		_ = db.Close()
		return nil, fmt.Errorf("postgres session store: migrate: %w", migrateErr)
	}
	return store, nil
}

func (s *PostgresSessionStore) migrate() error {
	ctx := context.Background()
	if _, err := s.sql.db.ExecContext(ctx, pgCreateSessionsSQL); err != nil {
		return err
	}
	_, err := s.sql.db.ExecContext(ctx, pgCreateMessagesSQL)
	return err
}

// Close closes the underlying database connection.
func (s *PostgresSessionStore) Close() error {
	return s.sql.close()
}

// SaveAs persists the session to PostgreSQL with an optional name and model tag.
// Returns the generated session ID.
func (s *PostgresSessionStore) SaveAs(session *Session, name, model string) (string, error) {
	return s.sql.saveAs(session, name, model)
}

// Save persists the session without a name or model tag.
func (s *PostgresSessionStore) Save(session *Session) (string, error) {
	return s.sql.saveAs(session, "", "")
}

// Load loads a session from PostgreSQL by ID.
func (s *PostgresSessionStore) Load(id string) (*Session, error) {
	return s.sql.load(id)
}

// LoadMeta returns metadata for a single session by ID.
func (s *PostgresSessionStore) LoadMeta(id string) (*SessionMetadata, error) {
	return s.sql.loadMeta(id)
}

// ListMeta returns metadata for all sessions, newest first.
func (s *PostgresSessionStore) ListMeta() ([]SessionMetadata, error) {
	return s.sql.listMeta()
}

// List returns all stored session IDs, newest first.
func (s *PostgresSessionStore) List() ([]string, error) {
	return s.sql.list()
}

// Delete removes a session and its messages from the database.
func (s *PostgresSessionStore) Delete(id string) error {
	return s.sql.delete(id)
}

// SearchSessions returns session IDs whose messages contain the given text.
// Results are ordered newest first.
func (s *PostgresSessionStore) SearchSessions(text string) ([]string, error) {
	return s.sql.searchSessions(text)
}
