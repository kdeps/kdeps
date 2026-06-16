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
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"

	"github.com/google/uuid"
	lcemb "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	lcvectorstores "github.com/tmc/langchaingo/vectorstores"
)

// sqlStore is the shared implementation for SQL-backed vector stores (MySQL, PostgreSQL).
// Concrete types embed sqlStore and provide the driver-specific createTableSQL and insertSQL.
type sqlStore struct {
	db             *sql.DB
	tableName      string
	embedder       lcemb.Embedder
	createTableSQL func(table string) string
	insertSQL      func(table string) string
	tag            string // used in error messages
}

func (s *sqlStore) ensureTable(ctx context.Context) error {
	_, execErr := s.db.ExecContext(ctx, s.createTableSQL(s.tableName))
	return execErr
}

func (s *sqlStore) AddDocuments(
	ctx context.Context,
	docs []schema.Document,
	_ ...lcvectorstores.Option,
) ([]string, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	if tableErr := s.ensureTable(ctx); tableErr != nil {
		return nil, fmt.Errorf("%s add_documents: ensure table: %w", s.tag, tableErr)
	}

	texts := make([]string, len(docs))
	for i, d := range docs {
		texts[i] = d.PageContent
	}

	vectors, embedErr := s.embedder.EmbedDocuments(ctx, texts)
	if embedErr != nil {
		return nil, fmt.Errorf("%s add_documents: embed: %w", s.tag, embedErr)
	}
	if len(vectors) != len(docs) {
		return nil, fmt.Errorf("%s: embedder returned wrong number of vectors", s.tag)
	}

	ids := make([]string, len(docs))
	q := s.insertSQL(s.tableName)

	for i, d := range docs {
		ids[i] = uuid.NewString()

		embJSON, marshalErr := json.Marshal(vectors[i])
		if marshalErr != nil {
			return nil, fmt.Errorf("%s: marshal embedding: %w", s.tag, marshalErr)
		}
		metaJSON, marshalErr2 := json.Marshal(d.Metadata)
		if marshalErr2 != nil {
			return nil, fmt.Errorf("%s: marshal metadata: %w", s.tag, marshalErr2)
		}

		if _, execErr := s.db.ExecContext(
			ctx, q, ids[i], texts[i], string(embJSON), string(metaJSON),
		); execErr != nil {
			return nil, fmt.Errorf("%s: insert document: %w", s.tag, execErr)
		}
	}
	return ids, nil
}

func (s *sqlStore) SimilaritySearch(
	ctx context.Context,
	query string,
	numDocuments int,
	_ ...lcvectorstores.Option,
) ([]schema.Document, error) {
	if tableErr := s.ensureTable(ctx); tableErr != nil {
		return nil, fmt.Errorf("%s similarity_search: ensure table: %w", s.tag, tableErr)
	}

	queryVec, embedErr := s.embedder.EmbedQuery(ctx, query)
	if embedErr != nil {
		return nil, fmt.Errorf("%s similarity_search: embed query: %w", s.tag, embedErr)
	}

	if numDocuments <= 0 {
		numDocuments = 5
	}

	rows, queryErr := s.db.QueryContext(
		ctx,
		fmt.Sprintf("SELECT id, content, embedding, metadata FROM %s", s.tableName),
	)
	if queryErr != nil {
		return nil, fmt.Errorf("%s similarity_search: query: %w", s.tag, queryErr)
	}
	defer rows.Close()

	type candidate struct {
		doc   schema.Document
		score float32
	}

	var candidates []candidate
	for rows.Next() {
		var id, content, embStr string
		var metaStr sql.NullString
		if scanErr := rows.Scan(&id, &content, &embStr, &metaStr); scanErr != nil {
			return nil, fmt.Errorf("%s similarity_search: scan: %w", s.tag, scanErr)
		}

		var vec []float32
		if unmarshalErr := json.Unmarshal([]byte(embStr), &vec); unmarshalErr != nil {
			continue
		}

		score := cosineSimilarity(queryVec, vec)

		meta := map[string]interface{}{}
		if metaStr.Valid && metaStr.String != "" {
			_ = json.Unmarshal([]byte(metaStr.String), &meta)
		}

		candidates = append(candidates, candidate{
			doc:   schema.Document{PageContent: content, Metadata: meta, Score: score},
			score: score,
		})
	}
	if rowsErr := rows.Err(); rowsErr != nil {
		return nil, fmt.Errorf("%s similarity_search: rows: %w", s.tag, rowsErr)
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

// cosineSimilarity computes the cosine similarity between two float32 vectors.
func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return float32(dot / denom)
}

// Verify interface compliance for sqlStore via embedding.
var _ lcvectorstores.VectorStore = (*sqlStore)(nil)

// newSQLStore creates a SQL-backed vector store with provided DDL/DML functions.
func newSQLStore(
	driverName, dsn, tableName, tag string,
	createTableSQL func(string) string,
	insertSQL func(string) string,
	embedder lcemb.Embedder,
) (*sqlStore, error) {
	if dsn == "" {
		return nil, fmt.Errorf("vectorstore %s: url (DSN) is required", tag)
	}
	if tableName == "" {
		return nil, errors.New("vectorstore: collection (table name) is required")
	}
	db, openErr := sql.Open(driverName, dsn)
	if openErr != nil {
		return nil, fmt.Errorf("vectorstore %s: open: %w", tag, openErr)
	}
	return &sqlStore{
		db:             db,
		tableName:      tableName,
		embedder:       embedder,
		createTableSQL: createTableSQL,
		insertSQL:      insertSQL,
		tag:            tag,
	}, nil
}
