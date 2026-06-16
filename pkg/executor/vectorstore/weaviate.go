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

// weaviateStore implements vectorstores.VectorStore using Weaviate's REST + GraphQL API.
// Collection is used as the Weaviate class name (must start with uppercase letter).
// Does not use weaviate-go-client; all calls are net/http.
type weaviateStore struct {
	baseURL   string
	className string
	apiKey    string
	embedder  lcemb.Embedder
	client    *http.Client
}

func newWeaviateStore(cfg *domain.VectorStoreConfig, embedder lcemb.Embedder) (*weaviateStore, error) {
	if cfg.URL == "" {
		return nil, errors.New("vectorstore weaviate: url is required")
	}
	// Weaviate class names must start with uppercase.
	className := cfg.Collection
	if len(className) > 0 {
		className = strings.ToUpper(className[:1]) + className[1:]
	}
	return &weaviateStore{
		baseURL:   strings.TrimSuffix(cfg.URL, "/"),
		className: className,
		apiKey:    cfg.APIKey,
		embedder:  embedder,
		client:    http.DefaultClient,
	}, nil
}

var _ lcvectorstores.VectorStore = (*weaviateStore)(nil)

func (s *weaviateStore) AddDocuments(
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
		return nil, fmt.Errorf("weaviate add_documents: embed: %w", embedErr)
	}
	if len(vectors) != len(docs) {
		return nil, errors.New("weaviate: embedder returned wrong number of vectors")
	}

	type weaviateObject struct {
		Class      string                 `json:"class"`
		ID         string                 `json:"id"`
		Properties map[string]interface{} `json:"properties"`
		Vector     []float32              `json:"vector"`
	}

	ids := make([]string, len(docs))
	objects := make([]weaviateObject, len(docs))
	for i, d := range docs {
		ids[i] = uuid.NewString()
		props := map[string]interface{}{"text": texts[i]}
		for k, v := range d.Metadata {
			props[k] = v
		}
		objects[i] = weaviateObject{
			Class:      s.className,
			ID:         ids[i],
			Properties: props,
			Vector:     vectors[i],
		}
	}

	payload, marshalErr := json.Marshal(map[string]interface{}{"objects": objects})
	if marshalErr != nil {
		return nil, fmt.Errorf("weaviate: marshal batch payload: %w", marshalErr)
	}

	resp, batchErr := s.doRequest(ctx, http.MethodPost, s.baseURL+"/v1/batch/objects", payload)
	if batchErr != nil {
		return nil, batchErr
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("weaviate: batch objects: status %d: %s", resp.StatusCode, string(b))
	}
	return ids, nil
}

func (s *weaviateStore) SimilaritySearch(
	ctx context.Context,
	query string,
	numDocuments int,
	_ ...lcvectorstores.Option,
) ([]schema.Document, error) {
	vector, embedErr := s.embedder.EmbedQuery(ctx, query)
	if embedErr != nil {
		return nil, fmt.Errorf("weaviate similarity_search: embed query: %w", embedErr)
	}

	if numDocuments <= 0 {
		numDocuments = 5
	}

	// Build float64 slice for JSON serialization of the vector.
	vectorJSON := make([]float64, len(vector))
	for i, v := range vector {
		vectorJSON[i] = float64(v)
	}

	gqlQuery := fmt.Sprintf(`{
  Get {
    %s(
      nearVector: { vector: %s }
      limit: %d
    ) {
      text
      _additional { id distance }
    }
  }
}`, s.className, float64SliceToJSON(vectorJSON), numDocuments)

	payload, marshalErr := json.Marshal(map[string]string{"query": gqlQuery})
	if marshalErr != nil {
		return nil, fmt.Errorf("weaviate: marshal graphql payload: %w", marshalErr)
	}

	resp, gqlErr := s.doRequest(ctx, http.MethodPost, s.baseURL+"/v1/graphql", payload)
	if gqlErr != nil {
		return nil, gqlErr
	}
	defer resp.Body.Close()
	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("weaviate: read graphql response: %w", readErr)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("weaviate: graphql: status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data struct {
			Get map[string][]map[string]interface{} `json:"Get"`
		} `json:"data"`
	}
	if unmarshalErr := json.Unmarshal(body, &result); unmarshalErr != nil {
		return nil, fmt.Errorf("weaviate: parse graphql response: %w", unmarshalErr)
	}

	items := result.Data.Get[s.className]
	docs := make([]schema.Document, 0, len(items))
	for _, item := range items {
		text, _ := item["text"].(string)
		meta := map[string]interface{}{}
		for k, v := range item {
			if k != "text" && k != "_additional" {
				meta[k] = v
			}
		}
		var score float32
		if add, addOK := item["_additional"].(map[string]interface{}); addOK {
			if dist, distOK := add["distance"].(float64); distOK {
				score = float32(1.0 / (1.0 + dist))
			}
		}
		docs = append(docs, schema.Document{
			PageContent: text,
			Metadata:    meta,
			Score:       score,
		})
	}
	return docs, nil
}

// float64SliceToJSON renders a float64 slice as a compact JSON array string.
func float64SliceToJSON(v []float64) string {
	b, _ := json.Marshal(v)
	return string(b)
}

func (s *weaviateStore) doRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, reqErr := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if reqErr != nil {
		return nil, fmt.Errorf("weaviate: create request: %w", reqErr)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}
	resp, doErr := s.client.Do(req)
	if doErr != nil {
		return nil, fmt.Errorf("weaviate: %s %s: %w", method, url, doErr)
	}
	return resp, nil
}
