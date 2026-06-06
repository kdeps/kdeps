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

//go:build !js

package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"

	_ "github.com/mattn/go-sqlite3" // SQLite driver for database connectivity
)

// sqlOpen is a test hook for database/sql.Open.
//
//nolint:gochecknoglobals // overridden in tests to inject Open errors
var sqlOpen = sql.Open

// MemoryStorage provides persistent key-value storage using SQLite.
type MemoryStorage struct {
	DB   *sql.DB
	mu   sync.RWMutex
	path string
}

func resolveMemoryDBPath(dbPath string) string {
	if dbPath != "" {
		return dbPath
	}
	if envPath := os.Getenv("KDEPS_MEMORY_DB_PATH"); envPath != "" {
		return envPath
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return filepath.Join(homeDir, ".kdeps", "memory.db")
}

func ensureMemoryDBDirectory(dbPath string) error {
	if dbPath == ":memory:" {
		return nil
	}
	dir := filepath.Dir(dbPath)
	if dir == "." || dir == "/" {
		return nil
	}
	return os.MkdirAll(dir, 0750)
}

func decodeMemoryValue(valueStr string) interface{} {
	var value interface{}
	if unmarshalErr := json.Unmarshal([]byte(valueStr), &value); unmarshalErr != nil {
		return valueStr
	}
	return value
}

// NewMemoryStorage creates a new memory storage.
func NewMemoryStorage(dbPath string) (*MemoryStorage, error) {
	kdeps_debug.Log("enter: NewMemoryStorage")
	dbPath = resolveMemoryDBPath(dbPath)

	if err := ensureMemoryDBDirectory(dbPath); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	db, err := sqlOpen("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	storage := &MemoryStorage{
		DB:   db,
		path: dbPath,
	}

	if initErr := storage.initSchema(); initErr != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", initErr)
	}

	return storage, nil
}

// initSchema initializes the database schema.
func (m *MemoryStorage) initSchema() error {
	kdeps_debug.Log("enter: initSchema")
	query := `
	CREATE TABLE IF NOT EXISTS memory (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_memory_updated_at ON memory(updated_at);
	`
	_, err := m.DB.ExecContext(context.Background(), query)
	return err
}

// Get retrieves a value from memory.
func (m *MemoryStorage) Get(key string) (interface{}, bool) {
	kdeps_debug.Log("enter: Get")
	m.mu.RLock()
	defer m.mu.RUnlock()

	var valueStr string
	err := m.DB.QueryRowContext(context.Background(), "SELECT value FROM memory WHERE key = ?", key).
		Scan(&valueStr)
	if err == sql.ErrNoRows {
		return nil, false
	}
	if err != nil {
		return nil, false
	}

	return decodeMemoryValue(valueStr), true
}

// Set stores a value in memory.
func (m *MemoryStorage) Set(key string, value interface{}) error {
	kdeps_debug.Log("enter: Set")
	m.mu.Lock()
	defer m.mu.Unlock()

	valueBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	query := `
	INSERT INTO memory (key, value, updated_at)
	VALUES (?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(key) DO UPDATE SET
		value = excluded.value,
		updated_at = CURRENT_TIMESTAMP
	`
	_, err = m.DB.ExecContext(context.Background(), query, key, string(valueBytes))
	return err
}

// Delete removes a value from memory.
func (m *MemoryStorage) Delete(key string) error {
	kdeps_debug.Log("enter: Delete")
	m.mu.Lock()
	defer m.mu.Unlock()

	_, err := m.DB.ExecContext(context.Background(), "DELETE FROM memory WHERE key = ?", key)
	return err
}

// Close closes the database connection.
func (m *MemoryStorage) Close() error {
	kdeps_debug.Log("enter: Close")
	return m.DB.Close()
}
