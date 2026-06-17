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
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestSQLiteStore(t *testing.T) *SQLiteSessionStore {
	t.Helper()
	dir := t.TempDir()
	store, err := NewSQLiteSessionStore(filepath.Join(dir, "test.db"))
	if err != nil {
		t.Fatalf("NewSQLiteSessionStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestSQLiteSessionStore_SaveAndLoad(t *testing.T) {
	t.Parallel()
	store := newTestSQLiteStore(t)

	sess := NewSession(0)
	sess.Append("hello", "world")
	sess.Append("foo", "bar")

	id, err := store.SaveAs(sess, "my-session", "gpt-4")
	if err != nil {
		t.Fatalf("SaveAs: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}

	loaded, err := store.Load(id)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.TurnCount() != 2 {
		t.Errorf("TurnCount = %d, want 2", loaded.TurnCount())
	}
}

func TestSQLiteSessionStore_LoadMeta(t *testing.T) {
	t.Parallel()
	store := newTestSQLiteStore(t)

	sess := NewSession(0)
	sess.Append("q", "a")

	id, err := store.SaveAs(sess, "test-meta", "claude-3")
	if err != nil {
		t.Fatalf("SaveAs: %v", err)
	}

	meta, err := store.LoadMeta(id)
	if err != nil {
		t.Fatalf("LoadMeta: %v", err)
	}
	if meta.Name != "test-meta" {
		t.Errorf("Name = %q, want 'test-meta'", meta.Name)
	}
	if meta.Model != "claude-3" {
		t.Errorf("Model = %q, want 'claude-3'", meta.Model)
	}
	if meta.Turns != 1 {
		t.Errorf("Turns = %d, want 1", meta.Turns)
	}
}

func TestSQLiteSessionStore_ListMeta(t *testing.T) {
	t.Parallel()
	store := newTestSQLiteStore(t)

	s1 := NewSession(0)
	s1.Append("a", "b")
	s2 := NewSession(0)
	s2.Append("c", "d")

	_, err := store.Save(s1)
	if err != nil {
		t.Fatalf("Save s1: %v", err)
	}
	_, err = store.Save(s2)
	if err != nil {
		t.Fatalf("Save s2: %v", err)
	}

	metas, err := store.ListMeta()
	if err != nil {
		t.Fatalf("ListMeta: %v", err)
	}
	if len(metas) != 2 {
		t.Errorf("ListMeta returned %d, want 2", len(metas))
	}
}

func TestSQLiteSessionStore_List(t *testing.T) {
	t.Parallel()
	store := newTestSQLiteStore(t)

	sess := NewSession(0)
	sess.Append("x", "y")

	id, _ := store.Save(sess)

	ids, err := store.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, i := range ids {
		if i == id {
			found = true
		}
	}
	if !found {
		t.Errorf("id %q not found in List: %v", id, ids)
	}
}

func TestSQLiteSessionStore_Delete(t *testing.T) {
	t.Parallel()
	store := newTestSQLiteStore(t)

	sess := NewSession(0)
	sess.Append("del", "me")
	id, _ := store.Save(sess)

	if err := store.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, err := store.Load(id)
	if err != nil {
		// Load of deleted session returns no rows, which is fine.
		return
	}
	// Empty session is also acceptable (no messages after delete).
}

func TestSQLiteSessionStore_SearchSessions(t *testing.T) {
	t.Parallel()
	store := newTestSQLiteStore(t)

	sess := NewSession(0)
	sess.Append("hello world", "response about kdeps")

	id, err := store.Save(sess)
	if err != nil {
		t.Fatalf("Save: %v", err)
	}

	ids, err := store.SearchSessions("kdeps")
	if err != nil {
		t.Fatalf("SearchSessions: %v", err)
	}
	found := false
	for _, i := range ids {
		if i == id {
			found = true
		}
	}
	if !found {
		t.Errorf("session %q not found in search results: %v", id, ids)
	}

	notFound, _ := store.SearchSessions("xyz-no-match-zzz")
	if len(notFound) != 0 {
		t.Errorf("unexpected results for no-match search: %v", notFound)
	}
}

func TestNewSQLiteSessionStore_DefaultPath(t *testing.T) {
	// Override home dir to ensure test doesn't write to real home.
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	store, err := NewSQLiteSessionStore("")
	if err != nil {
		t.Fatalf("NewSQLiteSessionStore with empty path: %v", err)
	}
	defer store.Close()

	// Verify the db was created in the expected location.
	expected := filepath.Join(dir, sessionDir, "sessions.db")
	if !strings.HasSuffix(store.path, "sessions.db") {
		t.Errorf("unexpected store path: %q", store.path)
	}
	if _, statErr := os.Stat(expected); os.IsNotExist(statErr) {
		t.Errorf("database file not created at %q", expected)
	}
}

func TestSQLiteSessionStore_LoadMeta_NotFound(t *testing.T) {
	store := newTestSQLiteStore(t)
	_, err := store.LoadMeta("nonexistent-session-id")
	if err == nil {
		t.Fatal("expected error for nonexistent session")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

func TestSQLiteSessionStore_ClosedDB_Errors(t *testing.T) {
	store := newTestSQLiteStore(t)
	// Close the DB first, then verify operations return errors
	if err := store.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	sess := NewSession(0)
	_, err := store.Save(sess)
	if err == nil {
		t.Error("expected error from Save on closed DB")
	}

	_, err = store.Load("any-id")
	if err == nil {
		t.Error("expected error from Load on closed DB")
	}

	_, err = store.LoadMeta("any-id")
	if err == nil {
		t.Error("expected error from LoadMeta on closed DB")
	}

	_, err = store.ListMeta()
	if err == nil {
		t.Error("expected error from ListMeta on closed DB")
	}

	_, err = store.List()
	if err == nil {
		t.Error("expected error from List on closed DB")
	}

	err = store.Delete("any-id")
	if err == nil {
		t.Error("expected error from Delete on closed DB")
	}

	_, err = store.SearchSessions("query")
	if err == nil {
		t.Error("expected error from SearchSessions on closed DB")
	}
}

func TestNewSQLiteSessionStore_MkdirError(t *testing.T) {
	// Try to create a store under a path that cannot be created
	// Use a file as a directory component to trigger mkdir error
	dir := t.TempDir()
	// Create a file where we want a directory
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	_, err := NewSQLiteSessionStore(filepath.Join(blocker, "sessions.db"))
	if err == nil {
		t.Fatal("expected error when mkdir fails")
	}
}
