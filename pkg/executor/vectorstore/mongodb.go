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

package vectorstore

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	lcemb "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	lcvectorstores "github.com/tmc/langchaingo/vectorstores"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// mongoStore implements vectorstores.VectorStore using MongoDB via go.mongodb.org/mongo-driver v1.
// Vectors are stored as float32 arrays in BSON documents.
// Cosine similarity is computed in Go (no Atlas Vector Search required).
type mongoStore struct {
	client   *mongo.Client
	coll     *mongo.Collection
	embedder lcemb.Embedder
}

type mongoDoc struct {
	ID        string                 `bson:"_id"`
	Content   string                 `bson:"content"`
	Embedding []float32              `bson:"embedding"`
	Metadata  map[string]interface{} `bson:"metadata"`
}

func newMongoStore(
	ctx context.Context,
	cfg *domain.VectorStoreConfig,
	embedder lcemb.Embedder,
) (*mongoStore, error) {
	if cfg.URL == "" {
		return nil, errors.New("vectorstore mongodb: url is required")
	}
	if cfg.Collection == "" {
		return nil, errors.New("vectorstore mongodb: collection is required")
	}

	clientOpts := options.Client().ApplyURI(cfg.URL)
	client, connErr := mongo.Connect(ctx, clientOpts)
	if connErr != nil {
		return nil, fmt.Errorf("vectorstore mongodb: connect: %w", connErr)
	}

	dbName := cfg.APIKey
	if dbName == "" {
		dbName = "kdeps"
	}

	return &mongoStore{
		client:   client,
		coll:     client.Database(dbName).Collection(cfg.Collection),
		embedder: embedder,
	}, nil
}

var _ lcvectorstores.VectorStore = (*mongoStore)(nil)

func (s *mongoStore) AddDocuments(
	ctx context.Context,
	docs []schema.Document,
	_ ...lcvectorstores.Option,
) ([]string, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	texts := make([]string, len(docs))
	for i, d := range docs {
		texts[i] = d.PageContent
	}

	vectors, embedErr := s.embedder.EmbedDocuments(ctx, texts)
	if embedErr != nil {
		return nil, fmt.Errorf("mongodb add_documents: embed: %w", embedErr)
	}
	if len(vectors) != len(docs) {
		return nil, errors.New("mongodb: embedder returned wrong number of vectors")
	}

	ids := make([]string, len(docs))
	toInsert := make([]interface{}, len(docs))
	for i, d := range docs {
		ids[i] = uuid.NewString()
		toInsert[i] = mongoDoc{
			ID:        ids[i],
			Content:   texts[i],
			Embedding: vectors[i],
			Metadata:  d.Metadata,
		}
	}

	if _, insertErr := s.coll.InsertMany(ctx, toInsert); insertErr != nil {
		return nil, fmt.Errorf("mongodb add_documents: insert: %w", insertErr)
	}
	return ids, nil
}

func (s *mongoStore) SimilaritySearch(
	ctx context.Context,
	query string,
	numDocuments int,
	_ ...lcvectorstores.Option,
) ([]schema.Document, error) {
	queryVec, embedErr := s.embedder.EmbedQuery(ctx, query)
	if embedErr != nil {
		return nil, fmt.Errorf("mongodb similarity_search: embed query: %w", embedErr)
	}

	if numDocuments <= 0 {
		numDocuments = 5
	}

	cursor, findErr := s.coll.Find(ctx, bson.M{})
	if findErr != nil {
		return nil, fmt.Errorf("mongodb similarity_search: find: %w", findErr)
	}
	defer cursor.Close(ctx)

	type candidate struct {
		doc   schema.Document
		score float32
	}

	var candidates []candidate
	for cursor.Next(ctx) {
		var d mongoDoc
		if decodeErr := cursor.Decode(&d); decodeErr != nil {
			continue
		}
		score := cosineSimilarity(queryVec, d.Embedding)
		candidates = append(candidates, candidate{
			doc:   schema.Document{PageContent: d.Content, Metadata: d.Metadata, Score: score},
			score: score,
		})
	}
	if cursorErr := cursor.Err(); cursorErr != nil {
		return nil, fmt.Errorf("mongodb similarity_search: cursor: %w", cursorErr)
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})
	if numDocuments < len(candidates) {
		candidates = candidates[:numDocuments]
	}

	docs := make([]schema.Document, len(candidates))
	for i, c := range candidates {
		docs[i] = c.doc
	}
	return docs, nil
}
