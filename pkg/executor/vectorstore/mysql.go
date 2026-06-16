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

	_ "github.com/go-sql-driver/mysql" // MySQL/MariaDB/Dolt driver
	"github.com/google/uuid"
	lcemb "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	lcvectorstores "github.com/tmc/langchaingo/vectorstores"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// mysqlStore implements vectorstores.VectorStore using MySQL/MariaDB/Dolt via database/sql.
// Vectors are stored as JSON arrays and cosine similarity is computed in Go.
// This works with any MySQL-compatible server regardless of vector extension availability.
type mysqlStore struct {
	db        *sql.DB
	tableName string
	embedder  lcemb.Embedder
}

func newMySQLStore(cfg *domain.VectorStoreConfig, embedder lcemb.Embedder) (*mysqlStore, error) {
	if cfg.URL == "" {
		return nil, errors.New("vectorstore mysql: url (DSN) is required")
	}
	if cfg.Collection == "" {
		return nil, errors.New("vectorstore mysql: collection (table name) is required")
	}
	db, openErr := sql.Open("mysql", cfg.URL)
	if openErr != nil {
		return nil, fmt.Errorf("vectorstore mysql: open: %w", openErr)
	}
	return &mysqlStore{
		db:        db,
		tableName: cfg.Collection,
		embedder:  embedder,
	}, nil
}

var _ lcvectorstores.VectorStore = (*mysqlStore)(nil)

func (s *mysqlStore) ensureTable(ctx context.Context) error {
	q := fmt.Sprintf(
		`CREATE TABLE IF NOT EXISTS %s (
		id VARCHAR(36) NOT NULL PRIMARY KEY,
		content LONGTEXT NOT NULL,
		embedding JSON NOT NULL,
		metadata JSON
	) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`,
		s.tableName,
	)
	_, execErr := s.db.ExecContext(ctx, q)
	return execErr
}

func (s *mysqlStore) AddDocuments(
	ctx context.Context,
	docs []schema.Document,
	_ ...lcvectorstores.Option,
) ([]string, error) {
	if len(docs) == 0 {
		return nil, nil
	}

	if tableErr := s.ensureTable(ctx); tableErr != nil {
		return nil, fmt.Errorf("mysql add_documents: ensure table: %w", tableErr)
	}

	texts := make([]string, len(docs))
	for i, d := range docs {
		texts[i] = d.PageContent
	}

	vectors, embedErr := s.embedder.EmbedDocuments(ctx, texts)
	if embedErr != nil {
		return nil, fmt.Errorf("mysql add_documents: embed: %w", embedErr)
	}
	if len(vectors) != len(docs) {
		return nil, errors.New("mysql: embedder returned wrong number of vectors")
	}

	ids := make([]string, len(docs))
	q := fmt.Sprintf( //nolint:gosec // G201: table name from validated config, not user input
		"INSERT INTO %s (id, content, embedding, metadata) VALUES (?, ?, ?, ?)",
		s.tableName,
	)

	for i, d := range docs {
		ids[i] = uuid.NewString()

		embJSON, marshalErr := json.Marshal(vectors[i])
		if marshalErr != nil {
			return nil, fmt.Errorf("mysql: marshal embedding: %w", marshalErr)
		}
		metaJSON, marshalErr2 := json.Marshal(d.Metadata)
		if marshalErr2 != nil {
			return nil, fmt.Errorf("mysql: marshal metadata: %w", marshalErr2)
		}

		if _, execErr := s.db.ExecContext(ctx, q, ids[i], texts[i], string(embJSON), string(metaJSON)); execErr != nil {
			return nil, fmt.Errorf("mysql: insert document: %w", execErr)
		}
	}
	return ids, nil
}

func (s *mysqlStore) SimilaritySearch(
	ctx context.Context,
	query string,
	numDocuments int,
	_ ...lcvectorstores.Option,
) ([]schema.Document, error) {
	if tableErr := s.ensureTable(ctx); tableErr != nil {
		return nil, fmt.Errorf("mysql similarity_search: ensure table: %w", tableErr)
	}

	queryVec, embedErr := s.embedder.EmbedQuery(ctx, query)
	if embedErr != nil {
		return nil, fmt.Errorf("mysql similarity_search: embed query: %w", embedErr)
	}

	if numDocuments <= 0 {
		numDocuments = 5
	}

	rows, queryErr := s.db.QueryContext(
		ctx,
		fmt.Sprintf("SELECT id, content, embedding, metadata FROM %s", s.tableName),
	)
	if queryErr != nil {
		return nil, fmt.Errorf("mysql similarity_search: query: %w", queryErr)
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
			return nil, fmt.Errorf("mysql similarity_search: scan: %w", scanErr)
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
		return nil, fmt.Errorf("mysql similarity_search: rows: %w", rowsErr)
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
