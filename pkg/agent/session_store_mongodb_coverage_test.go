// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package agent

import (
	"context"
	"os"
	"slices"
	"testing"
	"time"
)

// getMongoURI returns a MongoDB URI for testing, or empty string if unavailable.
func getMongoURI() string {
	if uri := os.Getenv("KDEPS_MONGODB_URI"); uri != "" {
		return uri
	}
	return "mongodb://127.0.0.1:27017"
}

// mongoTestConnect attempts to connect to MongoDB and skips if unavailable.
func mongoTestConnect(t *testing.T) *MongoDBSessionStore {
	t.Helper()
	uri := getMongoURI()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	store, err := NewMongoDBSessionStore(ctx, uri, "kdeps_test", "test_sessions")
	if err != nil {
		t.Skipf("Skipping: cannot connect to MongoDB: %v", err)
	}
	// Ping to verify the server is reachable before running tests.
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer pingCancel()
	if pingErr := store.client.Ping(pingCtx, nil); pingErr != nil {
		_ = store.Close(context.Background())
		t.Skipf("Skipping: MongoDB not reachable: %v", pingErr)
	}
	return store
}

// mongoSaveAs creates a session and persists via SaveAs.
func mongoSaveAs(t *testing.T, store *MongoDBSessionStore) string {
	t.Helper()
	session := NewSession(0)
	session.Append("hello", "world")
	id, err := store.SaveAs(session, "test-session", "test-model")
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}
	return id
}

// mongoCreateSession creates and saves a second session via Save.
func mongoCreateSession(t *testing.T, store *MongoDBSessionStore) string {
	t.Helper()
	s2 := NewSession(0)
	s2.Append("q", "a")
	id2, err := store.Save(s2)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if id2 == "" {
		t.Fatal("expected non-empty session ID from Save")
	}
	return id2
}

// TestMongoDBSessionStore_FullCRUD runs full CRUD tests against a real MongoDB.
//
//nolint:gocognit // comprehensive integration test; gocognit counts subtest closures
func TestMongoDBSessionStore_FullCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping MongoDB integration test in short mode")
	}

	store := mongoTestConnect(t)
	defer func() {
		_ = store.Close(context.Background())
	}()

	id := mongoSaveAs(t, store)
	id2 := mongoCreateSession(t, store)

	t.Run("LoadMeta", func(t *testing.T) {
		meta, err := store.LoadMeta(id)
		if err != nil {
			t.Fatalf("LoadMeta failed: %v", err)
		}
		if meta.Name != "test-session" {
			t.Errorf("expected Name=test-session, got %q", meta.Name)
		}
		if meta.Model != "test-model" {
			t.Errorf("expected Model=test-model, got %q", meta.Model)
		}
		if meta.Turns != 1 {
			t.Errorf("expected Turns=1, got %d", meta.Turns)
		}
	})

	t.Run("LoadMetaNotFound", func(t *testing.T) {
		_, err := store.LoadMeta("nonexistent-id-12345")
		if err == nil {
			t.Fatal("expected error for nonexistent session")
		}
	})

	t.Run("Load", func(t *testing.T) {
		loaded, err := store.Load(id)
		if err != nil {
			t.Fatalf("Load failed: %v", err)
		}
		if loaded.TurnCount() != 1 {
			t.Errorf("expected 1 turn, got %d", loaded.TurnCount())
		}
	})

	t.Run("LoadNotFound", func(t *testing.T) {
		_, err := store.Load("nonexistent-id-12345")
		if err == nil {
			t.Fatal("expected error loading nonexistent session")
		}
	})

	t.Run("List", func(t *testing.T) {
		ids, err := store.List()
		if err != nil {
			t.Fatalf("List failed: %v", err)
		}
		if !slices.Contains(ids, id) {
			t.Error("expected saved session ID in List")
		}
	})

	t.Run("ListMeta", func(t *testing.T) {
		metas, err := store.ListMeta()
		if err != nil {
			t.Fatalf("ListMeta failed: %v", err)
		}
		var found bool
		for _, m := range metas {
			if m.ID == id {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected saved session in ListMeta")
		}
	})

	t.Run("SearchSessions", func(t *testing.T) {
		results, err := store.SearchSessions("hello")
		if err != nil {
			t.Fatalf("SearchSessions failed: %v", err)
		}
		if !slices.Contains(results, id) {
			t.Error("expected to find session via SearchSessions")
		}
	})

	t.Run("SearchSessionsSpecialChars", func(t *testing.T) {
		results, err := store.SearchSessions("hello.world")
		if err != nil {
			t.Fatalf("SearchSessions with special chars failed: %v", err)
		}
		if len(results) == 0 {
			t.Log("Search with special chars returned no results (expected if no matching content)")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		if err := store.Delete(id); err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
	})

	t.Run("VerifyDeleted", func(t *testing.T) {
		_, err := store.Load(id)
		if err == nil {
			t.Fatal("expected error loading deleted session")
		}
	})

	t.Run("DeleteNotFound", func(t *testing.T) {
		err := store.Delete("nonexistent-id-12345")
		if err == nil {
			t.Fatal("expected error deleting nonexistent session")
		}
	})

	if id2 != "" {
		_ = store.Delete(id2)
	}
}
