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
	"strings"

	"github.com/google/uuid"
	lcemb "github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/schema"
	lcvectorstores "github.com/tmc/langchaingo/vectorstores"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

const basicAuthPartCount = 2

// openSearchStore implements vectorstores.VectorStore using the OpenSearch k-NN REST API.
// Compatible with both OpenSearch and Elasticsearch; uses the _bulk and _search endpoints.
// Does not use opensearch-go; all calls are net/http.
type openSearchStore struct {
	baseURL  string
	index    string
	username string
	password string
	embedder lcemb.Embedder
	client   *http.Client
}

func newOpenSearchStore(cfg *domain.VectorStoreConfig, embedder lcemb.Embedder) (*openSearchStore, error) {
	if cfg.URL == "" {
		return nil, errors.New("vectorstore opensearch: url is required")
	}
	s := &openSearchStore{
		baseURL:  strings.TrimSuffix(cfg.URL, "/"),
		index:    cfg.Collection,
		embedder: embedder,
		client:   http.DefaultClient,
	}
	// APIKey treated as "username:password" for basic auth.
	if cfg.APIKey != "" {
		parts := strings.SplitN(cfg.APIKey, ":", basicAuthPartCount)
		if len(parts) == basicAuthPartCount {
			s.username = parts[0]
			s.password = parts[1]
		}
	}
	return s, nil
}

var _ lcvectorstores.VectorStore = (*openSearchStore)(nil)

func (s *openSearchStore) AddDocuments(
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
		return nil, fmt.Errorf("opensearch add_documents: embed: %w", embedErr)
	}
	if len(vectors) != len(docs) {
		return nil, errors.New("opensearch: embedder returned wrong number of vectors")
	}

	// Build NDJSON bulk body.
	var buf bytes.Buffer
	ids := make([]string, len(docs))
	for i, d := range docs {
		ids[i] = uuid.NewString()

		metaLine, _ := json.Marshal(map[string]interface{}{
			"index": map[string]interface{}{"_index": s.index, "_id": ids[i]},
		})
		buf.Write(metaLine)
		buf.WriteByte('\n')

		docMeta := map[string]interface{}{}
		for k, v := range d.Metadata {
			docMeta[k] = v
		}
		docLine, _ := json.Marshal(map[string]interface{}{
			"text":   texts[i],
			"vector": vectors[i],
			"meta":   docMeta,
		})
		buf.Write(docLine)
		buf.WriteByte('\n')
	}

	resp, addErr := s.doRequest(ctx, http.MethodPost, s.baseURL+"/_bulk", buf.Bytes(), "application/x-ndjson")
	if addErr != nil {
		return nil, addErr
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("opensearch: bulk index: status %d: %s", resp.StatusCode, string(b))
	}
	return ids, nil
}

func (s *openSearchStore) SimilaritySearch(
	ctx context.Context,
	query string,
	numDocuments int,
	_ ...lcvectorstores.Option,
) ([]schema.Document, error) {
	vector, embedErr := s.embedder.EmbedQuery(ctx, query)
	if embedErr != nil {
		return nil, fmt.Errorf("opensearch similarity_search: embed query: %w", embedErr)
	}

	if numDocuments <= 0 {
		numDocuments = 5
	}

	payload, marshalErr := json.Marshal(map[string]interface{}{
		"size": numDocuments,
		"query": map[string]interface{}{
			"knn": map[string]interface{}{
				"vector": map[string]interface{}{
					"vector": vector,
					"k":      numDocuments,
				},
			},
		},
	})
	if marshalErr != nil {
		return nil, fmt.Errorf("opensearch: marshal query: %w", marshalErr)
	}

	searchURL := fmt.Sprintf("%s/%s/_search", s.baseURL, s.index)
	resp, searchErr := s.doRequest(ctx, http.MethodPost, searchURL, payload, "application/json")
	if searchErr != nil {
		return nil, searchErr
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("opensearch: read search response: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("opensearch: search: status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Score  float32                `json:"_score"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if unmarshalErr := json.Unmarshal(body, &result); unmarshalErr != nil {
		return nil, fmt.Errorf("opensearch: parse search response: %w", unmarshalErr)
	}

	docs := make([]schema.Document, len(result.Hits.Hits))
	for i, h := range result.Hits.Hits {
		text, _ := h.Source["text"].(string)
		meta := map[string]interface{}{}
		if m, ok := h.Source["meta"].(map[string]interface{}); ok {
			meta = m
		}
		docs[i] = schema.Document{
			PageContent: text,
			Metadata:    meta,
			Score:       h.Score,
		}
	}
	return docs, nil
}

func (s *openSearchStore) doRequest(
	ctx context.Context,
	method, url string,
	body []byte,
	contentType string,
) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, reqErr := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if reqErr != nil {
		return nil, fmt.Errorf("opensearch: create request: %w", reqErr)
	}
	req.Header.Set("Content-Type", contentType)
	if s.username != "" {
		req.SetBasicAuth(s.username, s.password)
	}
	resp, doErr := s.client.Do(req)
	if doErr != nil {
		return nil, fmt.Errorf("opensearch: %s %s: %w", method, url, doErr)
	}
	return resp, nil
}
