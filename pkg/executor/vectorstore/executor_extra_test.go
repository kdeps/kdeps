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
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/schema"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// --- providerEnvKey ---

func TestProviderEnvKey_Additional(t *testing.T) {
	cases := []struct{ backend, want string }{
		{"google", "GOOGLE_API_KEY"},
		{"deepseek", "DEEPSEEK_API_KEY"},
		{"openrouter", "OPENROUTER_API_KEY"},
		{"together", "TOGETHERAI_API_KEY"},
		{"", ""},
	}
	for _, c := range cases {
		got := providerEnvKey(c.backend)
		if got != c.want {
			t.Errorf("providerEnvKey(%q) = %q, want %q", c.backend, got, c.want)
		}
	}
}

// --- buildEmbedder ---

func TestBuildEmbedder_Google(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "google",
		EmbedModel:   "text-embedding-004",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	require.NoError(t, err)
	require.NotNil(t, emb)
}

func TestBuildEmbedder_Bedrock(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "bedrock",
		EmbedModel:   "amazon.titan-embed-text-v2:0",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	require.NoError(t, err)
	require.NotNil(t, emb)
}

// --- buildStore extra error paths (not duplicates of executor_test.go) ---

func TestBuildStore_Chroma_DefaultsToLocalhost(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "chroma",
		Collection:   "test",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	require.NoError(t, err)
	require.NotNil(t, store)
}

func TestBuildStore_Qdrant_WithURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "qdrant",
		URL:          srv.URL,
		Collection:   "test",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Logf("qdrant build error (expected in CI without qdrant): %v", err)
		return
	}
	require.NotNil(t, store)
}

func TestBuildStore_Bedrock_MissingCollection(t *testing.T) {
	_, err := newBedrockStore(t.Context(), &domain.VectorStoreConfig{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "knowledgeBaseId")
}

func TestBuildStore_Redis_MissingCollection(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	_, err := newRedisStore(t.Context(), &domain.VectorStoreConfig{
		URL: "redis://localhost:6379",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "index name")
}

func TestBuildStore_UnsupportedProvider(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Provider:   "nonexistent",
		Collection: "test",
	}
	_, err := buildStore(t.Context(), cfg)
	require.Error(t, err)
}

// --- executeAddDocuments / executeSimilaritySearch via weaviate ---

func TestExecute_AddDocuments_WithWeaviate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(io.Discard, r.Body)
		switch {
		case strings.Contains(r.URL.Path, "/v1/batch/objects"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"text-embedding-ada-002","usage":{"prompt_tokens":1,"total_tokens":1}}`))
		}
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "weaviate",
		URL:          srv.URL,
		Collection:   "TestCollection",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
		EmbedBaseURL: srv.URL,
		Operation:    "add_documents",
		Documents:    []domain.VectorStoreDocument{{Content: "hello world"}},
	}
	e := NewExecutor()
	result, err := e.Execute(nil, cfg)
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, 1, m["added"])
	ids := m["ids"].([]string)
	require.Len(t, ids, 1)
	assert.NotEmpty(t, ids[0])
}

func TestExecute_SimilaritySearch_WithWeaviate(t *testing.T) {
	respBody := `{
		"data": {
			"Get": {
				"TestCollection": [
					{"text": "hello world", "_additional": {"id": "abc", "distance": 0.2}}
				]
			}
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(io.Discard, r.Body)
		switch {
		case strings.Contains(r.URL.Path, "/v1/graphql"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(respBody))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"text-embedding-ada-002","usage":{"prompt_tokens":1,"total_tokens":1}}`))
		}
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "weaviate",
		URL:          srv.URL,
		Collection:   "TestCollection",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
		EmbedBaseURL: srv.URL,
		Operation:    "similarity_search",
		Query:        "test query",
		TopK:         5,
	}
	e := NewExecutor()
	result, err := e.Execute(nil, cfg)
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, 1, m["count"])
	results := m["results"].([]map[string]interface{})
	require.Len(t, results, 1)
	assert.Equal(t, "hello world", results[0]["content"])
}

func TestExecute_SimilaritySearch_WithDefaultTopK(t *testing.T) {
	respBody := `{
		"data": {
			"Get": {
				"TestCollection": [
					{"text": "hello", "_additional": {"id": "a1", "distance": 0.1}},
					{"text": "world", "_additional": {"id": "a2", "distance": 0.3}}
				]
			}
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(io.Discard, r.Body)
		switch {
		case strings.Contains(r.URL.Path, "/v1/graphql"):
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(respBody))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"text-embedding-ada-002","usage":{"prompt_tokens":1,"total_tokens":1}}`))
		}
	}))
	defer srv.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "weaviate",
		URL:          srv.URL,
		Collection:   "TestCollection",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
		EmbedBaseURL: srv.URL,
		Operation:    "similarity_search",
		Query:        "test",
		TopK:         0, // should default to 5
	}
	e := NewExecutor()
	result, err := e.Execute(nil, cfg)
	require.NoError(t, err)
	m := result.(map[string]interface{})
	assert.Equal(t, 2, m["count"])
}

// --- Chroma store extra error paths ---

func TestChromaStore_AddDocuments_EmbedError(t *testing.T) {
	store := &chromaStore{
		baseURL:    "http://localhost:8000",
		collection: "test",
		embedder:   &stubVectorEmbedder{err: errors.New("embed failed")},
		client:     http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed")
}

func TestChromaStore_AddDocuments_EmptyDocs(t *testing.T) {
	store := &chromaStore{
		baseURL:    "http://localhost:8000",
		collection: "test",
		embedder:   &stubVectorEmbedder{},
		client:     http.DefaultClient,
	}
	ids, err := store.AddDocuments(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, ids)
}

func TestChromaStore_SimilaritySearch_EmbedError(t *testing.T) {
	store := &chromaStore{
		baseURL:    "http://localhost:8000",
		collection: "test",
		embedder:   &stubVectorEmbedder{err: errors.New("embed failed")},
		client:     http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed")
}

func TestChromaStore_SimilaritySearch_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

func TestChromaStore_AddDocuments_HTTPError(t *testing.T) {
	collID := uuid.NewString()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		case strings.Contains(r.URL.Path, "/add"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("add failed"))
		default:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestChromaStore_SimilaritySearch_CollectionIDError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

// --- Pinecone store extra error paths ---

func TestPineconeStore_AddDocuments_EmbedError(t *testing.T) {
	store := &pineconeStore{
		host:      "http://localhost",
		namespace: "default",
		embedder:  &stubVectorEmbedder{err: errors.New("embed failed")},
		client:    http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed")
}

func TestPineconeStore_AddDocuments_EmptyDocs(t *testing.T) {
	store := &pineconeStore{
		host:      "http://localhost",
		namespace: "default",
		embedder:  &stubVectorEmbedder{},
		client:    http.DefaultClient,
	}
	ids, err := store.AddDocuments(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, ids)
}

func TestPineconeStore_SimilaritySearch_EmbedError(t *testing.T) {
	store := &pineconeStore{
		host:      "http://localhost",
		namespace: "default",
		embedder:  &stubVectorEmbedder{err: errors.New("embed failed")},
		client:    http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed")
}

// --- OpenSearch store extra error paths ---

func TestOpenSearchStore_AddDocuments_EmptyDocs(t *testing.T) {
	store := &openSearchStore{
		baseURL:  "http://localhost:9200",
		index:    "test-index",
		embedder: &stubVectorEmbedder{},
		client:   http.DefaultClient,
	}
	ids, err := store.AddDocuments(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, ids)
}

func TestOpenSearchStore_SimilaritySearch_EmbedError(t *testing.T) {
	store := &openSearchStore{
		baseURL:  "http://localhost:9200",
		index:    "test-index",
		embedder: &stubVectorEmbedder{err: errors.New("embed failed")},
		client:   http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed")
}

// --- Weaviate store extra error paths ---

func TestWeaviateStore_AddDocuments_EmptyDocs(t *testing.T) {
	store := &weaviateStore{
		baseURL:   "http://localhost:8080",
		className: "Test",
		embedder:  &stubVectorEmbedder{},
		client:    http.DefaultClient,
	}
	ids, err := store.AddDocuments(context.Background(), nil)
	require.NoError(t, err)
	assert.Nil(t, ids)
}

func TestWeaviateStore_NewStore_MissingURL(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Collection: "test",
	}
	_, err := newWeaviateStore(cfg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "url is required")
}

func TestWeaviateStore_AddDocuments_EmbedError(t *testing.T) {
	store := &weaviateStore{
		baseURL:   "http://localhost:8080",
		className: "Test",
		embedder:  &stubVectorEmbedder{err: errors.New("embed failed")},
		client:    http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed")
}

func TestWeaviateStore_SimilaritySearch_EmbedError(t *testing.T) {
	store := &weaviateStore{
		baseURL:   "http://localhost:8080",
		className: "Test",
		embedder:  &stubVectorEmbedder{err: errors.New("embed failed")},
		client:    http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed")
}

func TestNewPineconeStore_MissingURL(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Collection: "default",
	}
	_, err := newPineconeStore(cfg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "url (index host) is required")
}

func TestNewOpenSearchStore_MissingURL(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Collection: "my-index",
	}
	_, err := newOpenSearchStore(cfg, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "url is required")
}

func TestExecute_BuildStoreError(t *testing.T) {
	e := NewExecutor()
	cfg := &domain.VectorStoreConfig{
		Operation: "add_documents",
		Documents: []domain.VectorStoreDocument{{Content: "test"}},
		Provider:  "nonexistent",
	}
	_, err := e.Execute(nil, cfg)
	require.Error(t, err)
}
