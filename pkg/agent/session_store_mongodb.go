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
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// MongoDBSessionStore persists conversation sessions to a MongoDB collection.
// Each session is a single document with embedded messages array.
// Implements the same API as SQLiteSessionStore.
type MongoDBSessionStore struct {
	mu     sync.Mutex
	client *mongo.Client
	coll   *mongo.Collection
}

// mongoSessionDoc is the BSON document stored for each session.
type mongoSessionDoc struct {
	ID        string            `bson:"_id"`
	Name      string            `bson:"name"`
	Model     string            `bson:"model"`
	Turns     int               `bson:"turns"`
	CreatedAt int64             `bson:"created_at"`
	Messages  []mongoSessionMsg `bson:"messages"`
}

// mongoSessionMsg is a single message stored in the session document.
type mongoSessionMsg struct {
	Role    string `bson:"role"`
	Content string `bson:"content"`
	Seq     int    `bson:"seq"`
}

// NewMongoDBSessionStore connects to MongoDB and returns a session store.
// uri is the MongoDB connection string (e.g. "mongodb://localhost:27017").
// dbName is the database to use (defaults to "kdeps" if empty).
// collName is the collection to use (defaults to "sessions" if empty).
func NewMongoDBSessionStore(
	ctx context.Context,
	uri, dbName, collName string,
) (*MongoDBSessionStore, error) {
	if uri == "" {
		return nil, errors.New("mongodb session store: uri is required")
	}
	if dbName == "" {
		dbName = "kdeps"
	}
	if collName == "" {
		collName = "sessions"
	}

	clientOpts := options.Client().ApplyURI(uri)
	client, connErr := mongo.Connect(ctx, clientOpts)
	if connErr != nil {
		return nil, fmt.Errorf("mongodb session store: connect: %w", connErr)
	}

	coll := client.Database(dbName).Collection(collName)
	return &MongoDBSessionStore{client: client, coll: coll}, nil
}

// Close disconnects the MongoDB client.
func (s *MongoDBSessionStore) Close(ctx context.Context) error {
	return s.client.Disconnect(ctx)
}

// SaveAs persists the session to MongoDB with an optional name and model tag.
// Returns the generated session ID.
func (s *MongoDBSessionStore) SaveAs(session *Session, name, model string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := fmt.Sprintf("session-%d", time.Now().UnixNano())
	msgs := session.Messages()
	mongoMsgs := make([]mongoSessionMsg, len(msgs))
	for i, m := range msgs {
		mongoMsgs[i] = mongoSessionMsg{Role: m.Role, Content: m.Content, Seq: i}
	}

	doc := mongoSessionDoc{
		ID:        id,
		Name:      name,
		Model:     model,
		Turns:     session.TurnCount(),
		CreatedAt: time.Now().UnixMilli(),
		Messages:  mongoMsgs,
	}

	ctx := context.Background()
	if _, insertErr := s.coll.InsertOne(ctx, doc); insertErr != nil {
		return "", fmt.Errorf("mongodb session store: insert: %w", insertErr)
	}
	return id, nil
}

// Save persists the session without a name or model tag.
func (s *MongoDBSessionStore) Save(session *Session) (string, error) {
	return s.SaveAs(session, "", "")
}

// Load loads a session from MongoDB by ID.
func (s *MongoDBSessionStore) Load(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	var doc mongoSessionDoc
	findErr := s.coll.FindOne(ctx, bson.M{"_id": id}).Decode(&doc)
	if findErr != nil {
		if errors.Is(findErr, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("mongodb session store: session %q not found", id)
		}
		return nil, fmt.Errorf("mongodb session store: find: %w", findErr)
	}

	session := NewSession(0)
	for _, m := range doc.Messages {
		session.messages = append(
			session.messages,
			SessionMessage{Role: m.Role, Content: m.Content},
		)
	}
	return session, nil
}

// LoadMeta returns metadata for a single session by ID.
func (s *MongoDBSessionStore) LoadMeta(id string) (*SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	opts := options.FindOne().SetProjection(bson.M{"messages": 0})
	var doc mongoSessionDoc
	findErr := s.coll.FindOne(ctx, bson.M{"_id": id}, opts).Decode(&doc)
	if findErr != nil {
		if errors.Is(findErr, mongo.ErrNoDocuments) {
			return nil, fmt.Errorf("mongodb session store: session %q not found", id)
		}
		return nil, fmt.Errorf("mongodb session store: find meta: %w", findErr)
	}
	return &SessionMetadata{
		ID:        doc.ID,
		Name:      doc.Name,
		Model:     doc.Model,
		Turns:     doc.Turns,
		CreatedAt: doc.CreatedAt,
	}, nil
}

// ListMeta returns metadata for all sessions, newest first.
func (s *MongoDBSessionStore) ListMeta() ([]SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	findOpts := options.Find().
		SetProjection(bson.M{"messages": 0}).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, findErr := s.coll.Find(ctx, bson.M{}, findOpts)
	if findErr != nil {
		return nil, fmt.Errorf("mongodb session store: list meta: %w", findErr)
	}
	defer cursor.Close(ctx)

	var metas []SessionMetadata
	for cursor.Next(ctx) {
		var doc mongoSessionDoc
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}
		metas = append(metas, SessionMetadata{
			ID:        doc.ID,
			Name:      doc.Name,
			Model:     doc.Model,
			Turns:     doc.Turns,
			CreatedAt: doc.CreatedAt,
		})
	}
	if cursorErr := cursor.Err(); cursorErr != nil {
		return nil, fmt.Errorf("mongodb session store: list meta cursor: %w", cursorErr)
	}
	return metas, nil
}

// List returns all stored session IDs, newest first.
func (s *MongoDBSessionStore) List() ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	findOpts := options.Find().
		SetProjection(bson.M{"_id": 1}).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, findErr := s.coll.Find(ctx, bson.M{}, findOpts)
	if findErr != nil {
		return nil, fmt.Errorf("mongodb session store: list: %w", findErr)
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}
		ids = append(ids, doc.ID)
	}
	if cursorErr := cursor.Err(); cursorErr != nil {
		return nil, fmt.Errorf("mongodb session store: list cursor: %w", cursorErr)
	}
	return ids, nil
}

// Delete removes a session from the collection.
func (s *MongoDBSessionStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()
	res, deleteErr := s.coll.DeleteOne(ctx, bson.M{"_id": id})
	if deleteErr != nil {
		return fmt.Errorf("mongodb session store: delete %s: %w", id, deleteErr)
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("mongodb session store: session %q not found", id)
	}
	return nil
}

// SearchSessions returns session IDs whose messages contain the given text.
// Results are ordered newest first.
func (s *MongoDBSessionStore) SearchSessions(text string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ctx := context.Background()

	// Use a regex filter on the messages.content field.
	filter := bson.M{
		"messages": bson.M{
			"$elemMatch": bson.M{
				"content": bson.M{"$regex": escapeMongoRegex(text), "$options": "i"},
			},
		},
	}
	findOpts := options.Find().
		SetProjection(bson.M{"_id": 1}).
		SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, findErr := s.coll.Find(ctx, filter, findOpts)
	if findErr != nil {
		return nil, fmt.Errorf("mongodb session store: search: %w", findErr)
	}
	defer cursor.Close(ctx)

	var ids []string
	for cursor.Next(ctx) {
		var doc struct {
			ID string `bson:"_id"`
		}
		if decodeErr := cursor.Decode(&doc); decodeErr != nil {
			continue
		}
		ids = append(ids, doc.ID)
	}
	if cursorErr := cursor.Err(); cursorErr != nil {
		return nil, fmt.Errorf("mongodb session store: search cursor: %w", cursorErr)
	}
	return ids, nil
}

// escapeMongoRegex escapes special regex characters in a plain text search string.
func escapeMongoRegex(s string) string {
	specialChars := `\.+*?()|[]{}^$`
	var b strings.Builder
	for _, c := range s {
		if strings.ContainsRune(specialChars, c) {
			b.WriteRune('\\')
		}
		b.WriteRune(c)
	}
	return b.String()
}
