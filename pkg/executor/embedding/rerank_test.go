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

package embedding

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestCohereRerank_MissingQuery(t *testing.T) {
	cfg := &domain.EmbeddingConfig{
		Backend:         "cohere",
		RerankDocuments: []string{"doc1"},
	}
	_, err := cohereRerank(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rerankQuery is required")
}

func TestCohereRerank_MissingDocuments(t *testing.T) {
	cfg := &domain.EmbeddingConfig{
		Backend:     "cohere",
		RerankQuery: "what is AI?",
	}
	_, err := cohereRerank(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rerankDocuments must not be empty")
}

func TestCohereRerank_MissingAPIKey(t *testing.T) {
	t.Setenv("COHERE_API_KEY", "")
	cfg := &domain.EmbeddingConfig{
		Backend:         "cohere",
		RerankQuery:     "what is AI?",
		RerankDocuments: []string{"doc1"},
	}
	_, err := cohereRerank(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "COHERE_API_KEY is not set")
}

func TestCohereRerank_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))

		var req cohereRerankRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "rerank-v3.5", req.Model)
		assert.Equal(t, "what is AI?", req.Query)
		assert.Equal(t, []string{"doc about AI", "doc about cooking"}, req.Documents)

		resp := cohereRerankResponse{
			Results: []cohereRerankItem{
				{Index: 0, RelevanceScore: 0.98, Document: &struct {
					Text string `json:"text"`
				}{Text: "doc about AI"}},
				{Index: 1, RelevanceScore: 0.12, Document: &struct {
					Text string `json:"text"`
				}{Text: "doc about cooking"}},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	// Patch the endpoint for testing.
	orig := cohereRerankEndpointVar
	cohereRerankEndpointVar = srv.URL
	defer func() { cohereRerankEndpointVar = orig }()

	t.Setenv("COHERE_API_KEY", "test-key")

	cfg := &domain.EmbeddingConfig{
		Backend:         "cohere",
		RerankQuery:     "what is AI?",
		RerankDocuments: []string{"doc about AI", "doc about cooking"},
	}
	result, err := cohereRerank(cfg)
	require.NoError(t, err)
	assert.Equal(t, "rerank-v3.5", result["model"])

	results, ok := result["results"].([]RerankResult)
	require.True(t, ok)
	require.Len(t, results, 2)
	assert.Equal(t, 0, results[0].Index)
	assert.InDelta(t, 0.98, results[0].RelevanceScore, 0.001)
	assert.Equal(t, "doc about AI", results[0].Document)
}

func TestCohereRerank_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"invalid api key"}`))
	}))
	defer srv.Close()

	orig := cohereRerankEndpointVar
	cohereRerankEndpointVar = srv.URL
	defer func() { cohereRerankEndpointVar = orig }()

	t.Setenv("COHERE_API_KEY", "bad-key")

	cfg := &domain.EmbeddingConfig{
		Backend:         "cohere",
		RerankQuery:     "query",
		RerankDocuments: []string{"doc1"},
	}
	_, err := cohereRerank(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestCohereRerank_DefaultModel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req cohereRerankRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, cohereDefaultRerankModel, req.Model)
		resp := cohereRerankResponse{Results: []cohereRerankItem{{Index: 0, RelevanceScore: 0.9}}}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	orig := cohereRerankEndpointVar
	cohereRerankEndpointVar = srv.URL
	defer func() { cohereRerankEndpointVar = orig }()

	t.Setenv("COHERE_API_KEY", "test-key")

	cfg := &domain.EmbeddingConfig{
		Backend:         "cohere",
		RerankQuery:     "query",
		RerankDocuments: []string{"doc1"},
		// Model intentionally omitted — should default to cohereDefaultRerankModel
	}
	result, err := cohereRerank(cfg)
	require.NoError(t, err)
	assert.Equal(t, cohereDefaultRerankModel, result["model"])
}

func TestExecutor_Rerank_UnknownOp_ErrorMessage(t *testing.T) {
	e := NewExecutor()
	_, err := e.Execute(nil, &domain.EmbeddingConfig{Operation: "badop"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "rerank")
}
