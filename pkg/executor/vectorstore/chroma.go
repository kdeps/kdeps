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

	"github.com/google/uuid"
	lcemb "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	lcvectorstores "github.com/tmc/langchaingo/vectorstores"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// chromaStore implements vectorstores.VectorStore using Chroma's HTTP API (v1).
// It does not use amikos-tech/chroma-go; all calls are made via net/http.
type chromaStore struct {
	baseURL    string
	collection string
	apiKey     string
	embedder   lcemb.Embedder
	client     *http.Client
}

func newChromaStore(cfg *domain.VectorStoreConfig, embedder lcemb.Embedder) *chromaStore {
	baseURL := cfg.URL
	if baseURL == "" {
		baseURL = "http://localhost:8000"
	}
	return &chromaStore{
		baseURL:    baseURL,
		collection: cfg.Collection,
		apiKey:     cfg.APIKey,
		embedder:   embedder,
		client:     http.DefaultClient,
	}
}

var _ lcvectorstores.VectorStore = (*chromaStore)(nil)

// collectionID fetches (or creates) the collection and returns its UUID.
func (s *chromaStore) collectionID(ctx context.Context) (string, error) {
	// Try to get existing collection first.
	getURL := fmt.Sprintf("%s/api/v1/collections/%s", s.baseURL, s.collection)
	resp, getErr := s.doRequest(ctx, http.MethodGet, getURL, nil)
	if getErr != nil {
		return "", getErr
	}
	defer resp.Body.Close()
	getBody, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return "", fmt.Errorf("chroma: read response: %w", readErr)
	}
	if resp.StatusCode == http.StatusOK {
		var existing map[string]interface{}
		if json.Unmarshal(getBody, &existing) == nil {
			if id, ok := existing["id"].(string); ok && id != "" {
				return id, nil
			}
		}
	}

	// Collection does not exist; create it.
	createURL := fmt.Sprintf("%s/api/v1/collections", s.baseURL)
	payload, _ := json.Marshal(map[string]string{"name": s.collection})
	resp2, createErr := s.doRequest(ctx, http.MethodPost, createURL, payload)
	if createErr != nil {
		return "", createErr
	}
	defer resp2.Body.Close()
	createBody, readErr2 := io.ReadAll(resp2.Body)
	if readErr2 != nil {
		return "", fmt.Errorf("chroma: read create response: %w", readErr2)
	}
	if resp2.StatusCode != http.StatusOK && resp2.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("chroma: create collection %q: status %d: %s",
			s.collection, resp2.StatusCode, string(createBody))
	}
	var created map[string]interface{}
	if unmarshalErr := json.Unmarshal(createBody, &created); unmarshalErr != nil {
		return "", fmt.Errorf("chroma: parse create response: %w", unmarshalErr)
	}
	id, ok := created["id"].(string)
	if !ok || id == "" {
		return "", errors.New("chroma: create collection returned no id")
	}
	return id, nil
}

func (s *chromaStore) AddDocuments(
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
		return nil, fmt.Errorf("chroma add_documents: embed: %w", embedErr)
	}
	if len(vectors) != len(docs) {
		return nil, errors.New("chroma: embedder returned wrong number of vectors")
	}

	collID, collErr := s.collectionID(ctx)
	if collErr != nil {
		return nil, collErr
	}

	ids := make([]string, len(docs))
	metadatas := make([]map[string]interface{}, len(docs))
	for i := range docs {
		ids[i] = uuid.NewString()
		if docs[i].Metadata != nil {
			metadatas[i] = docs[i].Metadata
		} else {
			metadatas[i] = map[string]interface{}{}
		}
	}

	payload, marshalErr := json.Marshal(map[string]interface{}{
		"ids":        ids,
		"embeddings": vectors,
		"documents":  texts,
		"metadatas":  metadatas,
	})
	if marshalErr != nil {
		return nil, fmt.Errorf("chroma: marshal add payload: %w", marshalErr)
	}

	addURL := fmt.Sprintf("%s/api/v1/collections/%s/add", s.baseURL, collID)
	resp, addErr := s.doRequest(ctx, http.MethodPost, addURL, payload)
	if addErr != nil {
		return nil, addErr
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("chroma: add documents: status %d: %s", resp.StatusCode, string(b))
	}
	return ids, nil
}

func (s *chromaStore) SimilaritySearch(
	ctx context.Context,
	query string,
	numDocuments int,
	_ ...lcvectorstores.Option,
) ([]schema.Document, error) {
	vector, embedErr := s.embedder.EmbedQuery(ctx, query)
	if embedErr != nil {
		return nil, fmt.Errorf("chroma similarity_search: embed query: %w", embedErr)
	}

	collID, collErr := s.collectionID(ctx)
	if collErr != nil {
		return nil, collErr
	}

	if numDocuments <= 0 {
		numDocuments = 5
	}

	payload, marshalErr := json.Marshal(map[string]interface{}{
		"query_embeddings": [][]float32{vector},
		"n_results":        numDocuments,
		"include":          []string{"documents", "metadatas", "distances"},
	})
	if marshalErr != nil {
		return nil, fmt.Errorf("chroma: marshal query payload: %w", marshalErr)
	}

	queryURL := fmt.Sprintf("%s/api/v1/collections/%s/query", s.baseURL, collID)
	resp, queryErr := s.doRequest(ctx, http.MethodPost, queryURL, payload)
	if queryErr != nil {
		return nil, queryErr
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("chroma: read query response: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("chroma: query: status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Documents [][]string                 `json:"documents"`
		Metadatas [][]map[string]interface{} `json:"metadatas"`
		Distances [][]float32                `json:"distances"`
	}
	if unmarshalErr := json.Unmarshal(body, &result); unmarshalErr != nil {
		return nil, fmt.Errorf("chroma: parse query response: %w", unmarshalErr)
	}
	if len(result.Documents) == 0 {
		return nil, nil
	}

	rows := result.Documents[0]
	metas := result.Metadatas[0]
	dists := result.Distances[0]

	docs := make([]schema.Document, len(rows))
	for i, text := range rows {
		meta := map[string]interface{}{}
		if i < len(metas) {
			meta = metas[i]
		}
		var score float32
		if i < len(dists) {
			// Chroma returns L2 distance; convert to similarity (1 / (1 + distance)).
			score = 1.0 / (1.0 + dists[i])
		}
		docs[i] = schema.Document{
			PageContent: text,
			Metadata:    meta,
			Score:       score,
		}
	}
	return docs, nil
}

func (s *chromaStore) doRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, reqErr := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if reqErr != nil {
		return nil, fmt.Errorf("chroma: create request: %w", reqErr)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	resp, doErr := s.client.Do(req)
	if doErr != nil {
		return nil, fmt.Errorf("chroma: %s %s: %w", method, url, doErr)
	}
	return resp, nil
}
