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
	"strings"
	"testing"
)

func TestNewMongoDBSessionStore_RequiresURI(t *testing.T) {
	t.Parallel()
	_, err := NewMongoDBSessionStore(context.Background(), "", "", "")
	if err == nil {
		t.Fatal("expected error for empty URI")
	}
	if !strings.Contains(err.Error(), "uri is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewMongoDBSessionStore_DefaultsApplied(t *testing.T) {
	t.Parallel()
	// NewMongoDBSessionStore with a non-reachable URI still constructs the client
	// (mongo.Connect is lazy). Verify it returns a non-nil store.
	store, err := NewMongoDBSessionStore(context.Background(), "mongodb://127.0.0.1:27099", "", "")
	if err != nil {
		t.Fatalf("NewMongoDBSessionStore: %v", err)
	}
	_ = store.Close(context.Background())
}

func TestEscapeMongoRegex(t *testing.T) {
	t.Parallel()
	cases := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"a.b", `a\.b`},
		{"(test)", `\(test\)`},
		{"a+b*c", `a\+b\*c`},
		{"no-special", "no-special"},
	}
	for _, tc := range cases {
		got := escapeMongoRegex(tc.input)
		if got != tc.want {
			t.Errorf("escapeMongoRegex(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestMongoDBSessionStore_StructFields(t *testing.T) {
	t.Parallel()
	doc := mongoSessionDoc{
		ID: "session-123",
		Messages: []mongoSessionMsg{
			{Role: "user", Content: "hi", Seq: 0},
			{Role: "assistant", Content: "hello", Seq: 1},
		},
	}
	if doc.ID != "session-123" {
		t.Errorf("unexpected ID: %q", doc.ID)
	}
	if len(doc.Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(doc.Messages))
	}
}

func TestMongoDBSessionStore_CloseWithBackground(t *testing.T) {
	t.Parallel()
	store, err := NewMongoDBSessionStore(context.Background(), "mongodb://127.0.0.1:27099", "kdeps", "sessions")
	if err != nil {
		t.Fatalf("NewMongoDBSessionStore: %v", err)
	}
	// Close must not panic.
	_ = store.Close(context.Background())
}
