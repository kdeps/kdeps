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
	"strings"
	"testing"
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
