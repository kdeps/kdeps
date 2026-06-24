// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package agent

import (
	"context"
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// mockMongoColl implements mongoCollection for testing.
type mockMongoColl struct {
	insertOneResult *mongo.InsertOneResult
	insertOneErr    error

	findOneDecodeDoc *mongoSessionDoc
	findOneDecodeErr error

	findAllDecodeDocs []mongoSessionDoc
	findAllDecodeErr  error

	findIDsDecodeIDs []string
	findIDsDecodeErr error

	deleteResult *mongo.DeleteResult
	deleteErr    error
}

func (m *mockMongoColl) InsertOne(
	_ context.Context,
	_ interface{},
	_ ...*options.InsertOneOptions,
) (*mongo.InsertOneResult, error) {
	return m.insertOneResult, m.insertOneErr
}

func (m *mockMongoColl) FindOneDecode(
	_ context.Context,
	_ interface{},
	result interface{},
	_ ...*options.FindOneOptions,
) error {
	if m.findOneDecodeErr != nil {
		return m.findOneDecodeErr
	}
	if m.findOneDecodeDoc != nil {
		*(result.(*mongoSessionDoc)) = *m.findOneDecodeDoc
	}
	return nil
}

func (m *mockMongoColl) FindAllDecode(
	_ context.Context,
	_ interface{},
	results interface{},
	_ ...*options.FindOptions,
) error {
	if m.findAllDecodeErr != nil {
		return m.findAllDecodeErr
	}
	*(results.(*[]mongoSessionDoc)) = m.findAllDecodeDocs
	return nil
}

func (m *mockMongoColl) FindIDsDecode(
	_ context.Context,
	_ interface{},
	ids *[]string,
	_ ...*options.FindOptions,
) error {
	if m.findIDsDecodeErr != nil {
		return m.findIDsDecodeErr
	}
	*ids = append(*ids, m.findIDsDecodeIDs...)
	return nil
}

func (m *mockMongoColl) DeleteOne(
	_ context.Context,
	_ interface{},
	_ ...*options.DeleteOptions,
) (*mongo.DeleteResult, error) {
	return m.deleteResult, m.deleteErr
}

func newMockStore(coll *mockMongoColl) *MongoDBSessionStore {
	return newMongoDBSessionStoreWithColl(nil, coll)
}

func TestMongoDBSessionStore_SaveAs_MockSuccess(t *testing.T) {
	coll := &mockMongoColl{
		insertOneResult: &mongo.InsertOneResult{InsertedID: "test-id"},
	}
	store := newMockStore(coll)

	session := NewSession(0)
	session.Append("hello", "world")

	id, err := store.SaveAs(session, "my-session", "llama3")
	if err != nil {
		t.Fatalf("SaveAs failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}
}

func TestMongoDBSessionStore_SaveAs_InsertError(t *testing.T) {
	coll := &mockMongoColl{
		insertOneErr: errors.New("insert failed"),
	}
	store := newMockStore(coll)

	session := NewSession(0)
	session.Append("hello", "world")

	_, err := store.SaveAs(session, "test", "model")
	if err == nil {
		t.Fatal("expected error from insert")
	}
}

func TestMongoDBSessionStore_Save_Delegates(t *testing.T) {
	coll := &mockMongoColl{
		insertOneResult: &mongo.InsertOneResult{InsertedID: "saved"},
	}
	store := newMockStore(coll)

	session := NewSession(0)
	id, err := store.Save(session)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}
	if id == "" {
		t.Fatal("expected non-empty session ID")
	}
}

func TestMongoDBSessionStore_Load_Success(t *testing.T) {
	coll := &mockMongoColl{
		findOneDecodeDoc: &mongoSessionDoc{
			ID:    "session-1",
			Name:  "test",
			Model: "llama3",
			Turns: 1,
			Messages: []mongoSessionMsg{
				{Role: "user", Content: "hi", Seq: 0},
				{Role: "assistant", Content: "hello", Seq: 1},
			},
		},
	}
	store := newMockStore(coll)

	session, err := store.Load("session-1")
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if session.TurnCount() != 1 {
		t.Errorf("expected 1 turn, got %d", session.TurnCount())
	}
}

func TestMongoDBSessionStore_Load_NotFound(t *testing.T) {
	coll := &mockMongoColl{
		findOneDecodeErr: mongo.ErrNoDocuments,
	}
	store := newMockStore(coll)

	_, err := store.Load("nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestMongoDBSessionStore_Load_OtherError(t *testing.T) {
	coll := &mockMongoColl{
		findOneDecodeErr: errors.New("connection refused"),
	}
	store := newMockStore(coll)

	_, err := store.Load("any-id")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMongoDBSessionStore_LoadMeta_Success(t *testing.T) {
	coll := &mockMongoColl{
		findOneDecodeDoc: &mongoSessionDoc{
			ID:        "session-1",
			Name:      "named-session",
			Model:     "gpt-4",
			Turns:     5,
			CreatedAt: 1234567890,
		},
	}
	store := newMockStore(coll)

	meta, err := store.LoadMeta("session-1")
	if err != nil {
		t.Fatalf("LoadMeta failed: %v", err)
	}
	if meta.Name != "named-session" {
		t.Errorf("expected Name=named-session, got %q", meta.Name)
	}
	if meta.Model != "gpt-4" {
		t.Errorf("expected Model=gpt-4, got %q", meta.Model)
	}
	if meta.Turns != 5 {
		t.Errorf("expected Turns=5, got %d", meta.Turns)
	}
}

func TestMongoDBSessionStore_LoadMeta_NotFound(t *testing.T) {
	coll := &mockMongoColl{
		findOneDecodeErr: mongo.ErrNoDocuments,
	}
	store := newMockStore(coll)

	_, err := store.LoadMeta("nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestMongoDBSessionStore_LoadMeta_OtherError(t *testing.T) {
	coll := &mockMongoColl{
		findOneDecodeErr: errors.New("timeout"),
	}
	store := newMockStore(coll)

	_, err := store.LoadMeta("any-id")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMongoDBSessionStore_List_Success(t *testing.T) {
	coll := &mockMongoColl{
		findIDsDecodeIDs: []string{"id-1", "id-2", "id-3"},
	}
	store := newMockStore(coll)

	ids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 3 {
		t.Errorf("expected 3 ids, got %d", len(ids))
	}
}

func TestMongoDBSessionStore_List_Empty(t *testing.T) {
	coll := &mockMongoColl{}
	store := newMockStore(coll)

	ids, err := store.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 ids, got %d", len(ids))
	}
}

func TestMongoDBSessionStore_List_Error(t *testing.T) {
	coll := &mockMongoColl{
		findIDsDecodeErr: errors.New("find failed"),
	}
	store := newMockStore(coll)

	_, err := store.List()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMongoDBSessionStore_ListMeta_Success(t *testing.T) {
	coll := &mockMongoColl{
		findAllDecodeDocs: []mongoSessionDoc{
			{ID: "a", Name: "first", Turns: 1, CreatedAt: 100},
			{ID: "b", Name: "second", Turns: 2, CreatedAt: 200},
		},
	}
	store := newMockStore(coll)

	metas, err := store.ListMeta()
	if err != nil {
		t.Fatalf("ListMeta failed: %v", err)
	}
	if len(metas) != 2 {
		t.Errorf("expected 2 metas, got %d", len(metas))
	}
	if metas[0].Name != "first" {
		t.Errorf("expected first meta Name=first, got %q", metas[0].Name)
	}
}

func TestMongoDBSessionStore_ListMeta_Error(t *testing.T) {
	coll := &mockMongoColl{
		findAllDecodeErr: errors.New("find failed"),
	}
	store := newMockStore(coll)

	_, err := store.ListMeta()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMongoDBSessionStore_Delete_Success(t *testing.T) {
	coll := &mockMongoColl{
		deleteResult: &mongo.DeleteResult{DeletedCount: 1},
	}
	store := newMockStore(coll)

	err := store.Delete("session-1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestMongoDBSessionStore_Delete_NotFound(t *testing.T) {
	coll := &mockMongoColl{
		deleteResult: &mongo.DeleteResult{DeletedCount: 0},
	}
	store := newMockStore(coll)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for not found")
	}
}

func TestMongoDBSessionStore_Delete_Error(t *testing.T) {
	coll := &mockMongoColl{
		deleteErr: errors.New("delete failed"),
	}
	store := newMockStore(coll)

	err := store.Delete("any-id")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMongoDBSessionStore_SearchSessions_Success(t *testing.T) {
	coll := &mockMongoColl{
		findIDsDecodeIDs: []string{"match-1", "match-2"},
	}
	store := newMockStore(coll)

	ids, err := store.SearchSessions("hello")
	if err != nil {
		t.Fatalf("SearchSessions failed: %v", err)
	}
	if len(ids) != 2 {
		t.Errorf("expected 2 ids, got %d", len(ids))
	}
}

func TestMongoDBSessionStore_SearchSessions_NoMatches(t *testing.T) {
	coll := &mockMongoColl{}
	store := newMockStore(coll)

	ids, err := store.SearchSessions("nonexistent text")
	if err != nil {
		t.Fatalf("SearchSessions failed: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("expected 0 ids, got %d", len(ids))
	}
}

func TestMongoDBSessionStore_SearchSessions_Error(t *testing.T) {
	coll := &mockMongoColl{
		findIDsDecodeErr: errors.New("search failed"),
	}
	store := newMockStore(coll)

	_, err := store.SearchSessions("query")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMongoDBSessionStore_SearchSessions_SpecialChars(t *testing.T) {
	coll := &mockMongoColl{
		findIDsDecodeIDs: []string{"id-1"},
	}
	store := newMockStore(coll)

	ids, err := store.SearchSessions("hello.world(query)+test")
	if err != nil {
		t.Fatalf("SearchSessions with special chars failed: %v", err)
	}
	if len(ids) != 1 {
		t.Errorf("expected 1 id, got %d", len(ids))
	}
}

func TestMongoDBSessionStore_Close(_ *testing.T) {
	// Close with nil client should not panic (from mock)
	store := newMockStore(&mockMongoColl{})
	// Close only works with real client; with nil it's a no-op coverage test
	_ = store.Close(context.Background())
}

// Ensure *realMongoColl satisfies mongoCollection interface.
var _ mongoCollection = (*realMongoColl)(nil)

// Ensure bson import is used (kept for filter construction in source).
var _ = bson.M{}

// --- realMongoColl wrapper coverage via cancelled context ---

func TestRealMongoColl_CoverageViaCancelledContext(t *testing.T) {
	store, err := NewMongoDBSessionStore(context.Background(),
		"mongodb://127.0.0.1:27099/?serverSelectionTimeoutMS=100&connectTimeoutMS=100",
		"kdeps", "test")
	if err != nil {
		t.Fatalf("NewMongoDBSessionStore: %v", err)
	}

	cancelledCtx, cancel2 := context.WithCancel(context.Background())
	cancel2()

	// Exercise all methods through the realMongoColl wrapper.
	// All should fail fast with cancelled context.
	session := NewSession(0)
	session.Append("hello", "world")

	_, _ = store.SaveAs(session, "test", "model")
	_, _ = store.Save(session)
	_, _ = store.Load("any-id")
	_, _ = store.LoadMeta("any-id")
	_, _ = store.List()
	_, _ = store.ListMeta()
	_ = store.Delete("any-id")
	_, _ = store.SearchSessions("query")

	_ = store.Close(cancelledCtx)
}
