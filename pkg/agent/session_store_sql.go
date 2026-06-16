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
	"sync"
	"time"
)

// sqlSessionStore is the shared implementation for SQL-backed session stores.
// Driver-specific details (table names, placeholder style, search operator)
// are injected at construction time.
type sqlSessionStore struct {
	mu           sync.Mutex
	db           *sql.DB
	sessTable    string // e.g. "sessions" or "kdeps_sessions"
	msgTable     string // e.g. "messages" or "kdeps_messages"
	insertSessQL string // INSERT INTO <sess> (id,name,model,turns,created_at) VALUES (...)
	insertMsgQL  string // INSERT INTO <msg> (session_id,role,content,seq) VALUES (...)
	searchLike   string // "LIKE" or "ILIKE"
	ph           string // positional placeholder: "$1" (postgres) or "?" (sqlite)
}

func (s *sqlSessionStore) saveAs(session *Session, name, model string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	id := fmt.Sprintf("session-%d", time.Now().UnixNano())
	now := time.Now().UnixMilli()

	tx, txErr := s.db.BeginTx(ctx, nil)
	if txErr != nil {
		return "", fmt.Errorf("sql session store: begin tx: %w", txErr)
	}
	defer func() { _ = tx.Rollback() }()

	if _, insertErr := tx.ExecContext(
		ctx, s.insertSessQL, id, name, model, session.TurnCount(), now,
	); insertErr != nil {
		return "", fmt.Errorf("sql session store: insert session: %w", insertErr)
	}

	for i, m := range session.Messages() {
		if _, msgErr := tx.ExecContext(ctx, s.insertMsgQL, id, m.Role, m.Content, i); msgErr != nil {
			return "", fmt.Errorf("sql session store: insert message: %w", msgErr)
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return "", fmt.Errorf("sql session store: commit: %w", commitErr)
	}
	return id, nil
}

func (s *sqlSessionStore) load(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	q := fmt.Sprintf( //nolint:gosec // G201: table name is internal constant, not user input
		`SELECT role, content FROM %s WHERE session_id = %s ORDER BY seq`, s.msgTable, s.ph,
	)
	rows, queryErr := s.db.QueryContext(ctx, q, id)
	if queryErr != nil {
		return nil, fmt.Errorf("sql session store: query messages: %w", queryErr)
	}
	defer rows.Close()

	session := NewSession(0)
	for rows.Next() {
		var role, content string
		if scanErr := rows.Scan(&role, &content); scanErr != nil {
			return nil, fmt.Errorf("sql session store: scan message: %w", scanErr)
		}
		session.messages = append(session.messages, sessionMessage{Role: role, Content: content})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("sql session store: rows error: %w", rowsErr)
	}
	return session, nil
}

func (s *sqlSessionStore) loadMeta(id string) (*SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	q := fmt.Sprintf( //nolint:gosec // G201: table name is internal constant, not user input
		`SELECT id, name, model, turns, created_at FROM %s WHERE id = %s`, s.sessTable, s.ph,
	)
	row := s.db.QueryRowContext(ctx, q, id)
	var m SessionMetadata
	scanErr := row.Scan(&m.ID, &m.Name, &m.Model, &m.Turns, &m.CreatedAt)
	if scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return nil, fmt.Errorf("sql session store: session %q not found", id)
		}
		return nil, fmt.Errorf("sql session store: scan metadata: %w", scanErr)
	}
	return &m, nil
}

func (s *sqlSessionStore) listMeta() ([]SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	q := fmt.Sprintf( //nolint:gosec // G201: table name is internal constant, not user input
		`SELECT id, name, model, turns, created_at FROM %s ORDER BY created_at DESC`, s.sessTable,
	)
	rows, queryErr := s.db.QueryContext(ctx, q)
	if queryErr != nil {
		return nil, fmt.Errorf("sql session store: list: %w", queryErr)
	}
	defer rows.Close()

	var metas []SessionMetadata
	for rows.Next() {
		var m SessionMetadata
		if scanErr := rows.Scan(&m.ID, &m.Name, &m.Model, &m.Turns, &m.CreatedAt); scanErr != nil {
			return nil, fmt.Errorf("sql session store: scan: %w", scanErr)
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

func (s *sqlSessionStore) list() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	q := fmt.Sprintf( //nolint:gosec // G201: table name is internal constant, not user input
		`SELECT id FROM %s ORDER BY created_at DESC`, s.sessTable,
	)
	rows, queryErr := s.db.QueryContext(ctx, q)
	if queryErr != nil {
		return nil, fmt.Errorf("sql session store: list: %w", queryErr)
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

func (s *sqlSessionStore) delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	q := fmt.Sprintf( //nolint:gosec // G201: table name is internal constant, not user input
		`DELETE FROM %s WHERE id = %s`, s.sessTable, s.ph,
	)
	if _, deleteErr := s.db.ExecContext(ctx, q, id); deleteErr != nil {
		return fmt.Errorf("sql session store: delete %s: %w", id, deleteErr)
	}
	return nil
}

func (s *sqlSessionStore) searchSessions(text string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	q := fmt.Sprintf( //nolint:gosec // G201: table name is internal constant, not user input
		`SELECT DISTINCT session_id FROM %s WHERE content %s %s ORDER BY session_id DESC`,
		s.msgTable, s.searchLike, s.ph,
	)
	rows, queryErr := s.db.QueryContext(ctx, q, "%"+text+"%")
	if queryErr != nil {
		return nil, fmt.Errorf("sql session store: search: %w", queryErr)
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

func (s *sqlSessionStore) close() error {
	return s.db.Close()
}
