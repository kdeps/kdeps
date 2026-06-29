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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
	lcemb "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	lcvectorstores "github.com/tmc/langchaingo/vectorstores"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// pineconeStore implements vectorstores.VectorStore using Pinecone's REST API.
// URL must be the index host (e.g. "https://my-index-abc123.svc.us-east-1.pinecone.io").
// Does not use go-pinecone; all calls are net/http.
type pineconeStore struct {
	host      string
	namespace string
	apiKey    string
	embedder  lcemb.Embedder
	client    *http.Client
}

func newPineconeStore(cfg *domain.VectorStoreConfig, embedder lcemb.Embedder) (*pineconeStore, error) {
	if cfg.URL == "" {
		return nil, errors.New("vectorstore pinecone: url (index host) is required")
	}
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("PINECONE_API_KEY")
	}
	return &pineconeStore{
		host:      cfg.URL,
		namespace: cfg.Collection,
		apiKey:    apiKey,
		embedder:  embedder,
		client:    http.DefaultClient,
	}, nil
}

var _ lcvectorstores.VectorStore = (*pineconeStore)(nil)

func (s *pineconeStore) AddDocuments(
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
		return nil, fmt.Errorf("pinecone add_documents: embed: %w", embedErr)
	}
	if len(vectors) != len(docs) {
		return nil, errors.New("pinecone: embedder returned wrong number of vectors")
	}

	type pineconeVector struct {
		ID       string                 `json:"id"`
		Values   []float32              `json:"values"`
		Metadata map[string]interface{} `json:"metadata,omitempty"`
	}

	ids := make([]string, len(docs))
	pVectors := make([]pineconeVector, len(docs))
	for i, d := range docs {
		ids[i] = uuid.NewString()
		meta := map[string]interface{}{"text": texts[i]}
		for k, v := range d.Metadata {
			meta[k] = v
		}
		pVectors[i] = pineconeVector{
			ID:       ids[i],
			Values:   vectors[i],
			Metadata: meta,
		}
	}

	payload, marshalErr := json.Marshal(map[string]interface{}{
		"vectors":   pVectors,
		"namespace": s.namespace,
	})
	if marshalErr != nil {
		return nil, fmt.Errorf("pinecone: marshal upsert payload: %w", marshalErr)
	}

	resp, upsertErr := s.doRequest(ctx, http.MethodPost, s.host+"/vectors/upsert", payload)
	if upsertErr != nil {
		return nil, upsertErr
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("pinecone: upsert: status %d: %s", resp.StatusCode, string(b))
	}
	return ids, nil
}

func (s *pineconeStore) SimilaritySearch(
	ctx context.Context,
	query string,
	numDocuments int,
	_ ...lcvectorstores.Option,
) ([]schema.Document, error) {
	vector, embedErr := embedQuery(ctx, s.embedder, "pinecone", query)
	if embedErr != nil {
		return nil, embedErr
	}

	numDocuments = normalizeTopK(numDocuments)

	payload, marshalErr := json.Marshal(map[string]interface{}{
		"vector":          vector,
		"topK":            numDocuments,
		"namespace":       s.namespace,
		"includeMetadata": true,
	})
	if marshalErr != nil {
		return nil, fmt.Errorf("pinecone: marshal query payload: %w", marshalErr)
	}

	resp, queryErr := s.doRequest(ctx, http.MethodPost, s.host+"/query", payload)
	if queryErr != nil {
		return nil, queryErr
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("pinecone: read query response: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pinecone: query: status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Matches []struct {
			ID       string                 `json:"id"`
			Score    float32                `json:"score"`
			Metadata map[string]interface{} `json:"metadata"`
		} `json:"matches"`
	}
	if unmarshalErr := json.Unmarshal(body, &result); unmarshalErr != nil {
		return nil, fmt.Errorf("pinecone: parse query response: %w", unmarshalErr)
	}

	docs := make([]schema.Document, len(result.Matches))
	for i, m := range result.Matches {
		text, _ := m.Metadata["text"].(string)
		meta := make(map[string]interface{}, len(m.Metadata))
		for k, v := range m.Metadata {
			if k != "text" {
				meta[k] = v
			}
		}
		docs[i] = schema.Document{
			PageContent: text,
			Metadata:    meta,
			Score:       m.Score,
		}
	}
	return docs, nil
}

func (s *pineconeStore) doRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, reqErr := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if reqErr != nil {
		return nil, fmt.Errorf("pinecone: create request: %w", reqErr)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Api-Key", s.apiKey)
	resp, doErr := s.client.Do(req)
	if doErr != nil {
		return nil, fmt.Errorf("pinecone: %s %s: %w", method, url, doErr)
	}
	return resp, nil
}
