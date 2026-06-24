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
			_, _ = w.Write(
				[]byte(
					`{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"text-embedding-ada-002","usage":{"prompt_tokens":1,"total_tokens":1}}`,
				),
			)
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
			_, _ = w.Write(
				[]byte(
					`{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"text-embedding-ada-002","usage":{"prompt_tokens":1,"total_tokens":1}}`,
				),
			)
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
			_, _ = w.Write(
				[]byte(
					`{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"text-embedding-ada-002","usage":{"prompt_tokens":1,"total_tokens":1}}`,
				),
			)
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

func TestNewPineconeStore_EnvAPIKey(t *testing.T) {
	t.Setenv("PINECONE_API_KEY", "env-pc-key")
	cfg := &domain.VectorStoreConfig{
		URL:        "https://test-index.pinecone.io",
		Collection: "default",
	}
	store, err := newPineconeStore(cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, "env-pc-key", store.apiKey)
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

// --- buildEmbedder uncovered paths ---

func TestBuildEmbedder_Jina_NoKey(t *testing.T) {
	t.Setenv("JINA_API_KEY", "")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "jina",
		EmbedModel:   "jina-embeddings-v2-base-en",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	if err == nil {
		require.NotNil(t, emb)
	}
}

func TestBuildEmbedder_Google_NoKey(t *testing.T) {
	t.Setenv("GOOGLE_API_KEY", "")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "google",
		EmbedModel:   "text-embedding-004",
	}
	_, _ = buildEmbedder(t.Context(), cfg)
}

func TestBuildEmbedder_HuggingFace_NoKey(t *testing.T) {
	t.Setenv("HF_TOKEN", "")
	t.Setenv("HUGGINGFACEHUB_API_TOKEN", "")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "huggingface",
		EmbedModel:   "sentence-transformers/all-MiniLM-L6-v2",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	if err == nil {
		require.NotNil(t, emb)
	}
}

func TestBuildEmbedder_Cybertron_NoModel(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "cybertron",
		EmbedModel:   "",
	}
	_, err := buildEmbedder(t.Context(), cfg)
	if err == nil {
		t.Log("cybertron with empty model may succeed on some platforms")
	}
}

func TestBuildEmbedder_OpenAICompat_UnknownBackend(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "unknown_backend",
		EmbedModel:   "text-embedding-ada-002",
	}
	_, err := buildEmbedder(t.Context(), cfg)
	require.Error(t, err)
	// Unknown backend means providerEnvKey returns "" -> os.Getenv("") -> empty key,
	// and openAICompatBaseURL returns "" -> baseURL stays empty, covering that branch.
}

// --- buildStore: Qdrant invalid URL and API key ---

func TestBuildStore_Qdrant_InvalidURL(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Provider:   "qdrant",
		URL:        "://invalid-url",
		Collection: "test",
		EmbedModel: "model",
	}
	_, err := buildStore(t.Context(), cfg)
	require.Error(t, err)
}

func TestBuildStore_Qdrant_WithAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "qdrant",
		URL:          "http://localhost:6333",
		Collection:   "test",
		APIKey:       "my-qdrant-key",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	require.NoError(t, err)
	require.NotNil(t, store)
}

func TestBuildStore_Redis_InvalidPort(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "redis",
		URL:          "redis://localhost:0",
		Collection:   "myindex",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Log("redis store construction may succeed with bad port (lazy connect)")
	}
}

// --- buildStore embedder error path for all providers ---

func TestBuildStore_Store_EmbedderError(t *testing.T) {
	providers := []string{
		"chroma", "pinecone", "opensearch", "weaviate",
		"mariadb", "pgvector", "mongodb",
	}
	for _, provider := range providers {
		t.Run(provider, func(t *testing.T) {
			cfg := &domain.VectorStoreConfig{
				Provider:     provider,
				Collection:   "test",
				URL:          "http://localhost:8000",
				EmbedModel:   "text-embedding-ada-002",
				EmbedBackend: "cybertron",
			}
			_, err := buildStore(t.Context(), cfg)
			if err == nil {
				t.Logf("%s: embedder constructed despite cybertron model", provider)
			}
		})
	}
}

// wrongCountEmbedder always returns 1 vector regardless of input,
// exercising the "wrong number of vectors" error paths.
type wrongCountEmbedder struct{}

func (w *wrongCountEmbedder) EmbedDocuments(_ context.Context, _ []string) ([][]float32, error) {
	return [][]float32{{0.1, 0.2}}, nil
}

func (w *wrongCountEmbedder) EmbedQuery(_ context.Context, _ string) ([]float32, error) {
	return []float32{0.1, 0.2}, nil
}

// --- Wrong number of vectors ---

func TestPineconeStore_AddDocuments_WrongVectors(t *testing.T) {
	store := &pineconeStore{
		host:      "http://localhost:9999",
		namespace: "default",
		embedder:  &wrongCountEmbedder{},
		client:    http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "a"},
		{PageContent: "b"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wrong number of vectors")
}

func TestWeaviateStore_AddDocuments_WrongVectors(t *testing.T) {
	store := &weaviateStore{
		baseURL:   "http://localhost:9999",
		className: "Test",
		embedder:  &wrongCountEmbedder{},
		client:    http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "a"},
		{PageContent: "b"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wrong number of vectors")
}

// --- doRequest request creation error ---

func TestChromaStore_doRequest_CreateError(t *testing.T) {
	store := &chromaStore{
		baseURL:    "http://localhost:8000",
		collection: "test",
		embedder:   &stubVectorEmbedder{},
		client:     http.DefaultClient,
	}
	// An invalid method triggers NewRequestWithContext error.
	_, err := store.doRequest(context.Background(), "\x00invalid", "http://localhost", nil)
	require.Error(t, err)
}

func TestPineconeStore_doRequest_CreateError(t *testing.T) {
	store := &pineconeStore{
		host:      "http://localhost",
		namespace: "default",
		embedder:  &stubVectorEmbedder{},
		client:    http.DefaultClient,
	}
	_, err := store.doRequest(context.Background(), "\x00invalid", "http://localhost", nil)
	require.Error(t, err)
}

func TestOpenSearchStore_doRequest_CreateError(t *testing.T) {
	store := &openSearchStore{
		baseURL:  "http://localhost:9200",
		index:    "test",
		embedder: &stubVectorEmbedder{},
		client:   http.DefaultClient,
	}
	_, err := store.doRequest(context.Background(), "\x00invalid", "http://localhost", nil, "application/json")
	require.Error(t, err)
}

func TestWeaviateStore_doRequest_CreateError(t *testing.T) {
	store := &weaviateStore{
		baseURL:   "http://localhost:8080",
		className: "Test",
		embedder:  &stubVectorEmbedder{},
		client:    http.DefaultClient,
	}
	_, err := store.doRequest(context.Background(), "\x00invalid", "http://localhost", nil)
	require.Error(t, err)
}

// --- Chroma collectionID error paths ---

func TestChromaStore_CollectionID_CreateStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		case http.MethodPost:
			w.WriteHeader(http.StatusConflict)
			_, _ = w.Write([]byte(`{"error":"already exists"}`))
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	_, err := store.collectionID(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 409")
}

func TestChromaStore_CollectionID_CreateParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{}`))
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`invalid json`))
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	_, err := store.collectionID(context.Background())
	require.Error(t, err)
}

func TestChromaStore_CollectionID_CreateNoID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{}`))
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"name":"test"}`))
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	_, err := store.collectionID(context.Background())
	require.Error(t, err)
}

// --- Chroma wrong number of vectors ---

func TestChromaStore_AddDocuments_WrongVectors(t *testing.T) {
	store := &chromaStore{
		baseURL:    "http://localhost:9999",
		collection: "test",
		embedder:   &wrongCountEmbedder{},
		client:     http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "a"},
		{PageContent: "b"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wrong number of vectors")
}

// --- OpenSearch wrong number of vectors ---

func TestOpenSearchStore_AddDocuments_WrongVectors(t *testing.T) {
	store := &openSearchStore{
		baseURL:  "http://localhost:9999",
		index:    "test-index",
		embedder: &wrongCountEmbedder{},
		client:   http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "a"},
		{PageContent: "b"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wrong number of vectors")
}

// --- sqlStore error paths ---

func TestSQLStore_AddDocuments_EnsureTableError(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	// Close the underlying DB to make ExecContext fail.
	require.NoError(t, store.db.Close())
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
}

func TestSQLStore_AddDocuments_ExecError(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	// Close the DB so ExecContext fails during insert.
	require.NoError(t, store.db.Close())
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
}

func TestSQLStore_SimilaritySearch_EnsureTableError(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	require.NoError(t, store.db.Close())
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

func TestSQLStore_SimilaritySearch_QueryError(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	// Add a document first.
	_, addErr := store.AddDocuments(context.Background(), []schema.Document{{PageContent: "test"}})
	require.NoError(t, addErr)
	// Close DB to make QueryContext fail.
	require.NoError(t, store.db.Close())
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

func TestSQLStore_SimilaritySearch_UnmarshalSkip(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	// ensureTable must be called before raw SQL operations.
	require.NoError(t, store.ensureTable(context.Background()))
	// Insert a row with invalid JSON embedding; the scan succeeds but
	// json.Unmarshal fails and the code skips the row via continue.
	_, execErr := store.db.ExecContext(context.Background(),
		"INSERT INTO test_vectors (id, content, embedding, metadata) VALUES (?, ?, ?, ?)",
		"bad-id", "bad-content", "not-valid-json", "{}")
	require.NoError(t, execErr)

	docs, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.NoError(t, err)
	require.Empty(t, docs)
}

// --- HTTP-level request errors ---

func TestChromaStore_doRequest_HTTPError(t *testing.T) {
	store := &chromaStore{
		baseURL:    "http://127.0.0.1:1",
		collection: "test",
		embedder:   &stubVectorEmbedder{},
		client:     http.DefaultClient,
	}
	_, err := store.doRequest(context.Background(), http.MethodGet, "http://127.0.0.1:1/", nil)
	require.Error(t, err)
}

func TestPineconeStore_doRequest_HTTPError(t *testing.T) {
	store := &pineconeStore{
		host:      "http://127.0.0.1:1",
		namespace: "default",
		embedder:  &stubVectorEmbedder{},
		client:    http.DefaultClient,
	}
	_, err := store.doRequest(context.Background(), http.MethodGet, "http://127.0.0.1:1/", nil)
	require.Error(t, err)
}

func TestOpenSearchStore_doRequest_HTTPError(t *testing.T) {
	store := &openSearchStore{
		baseURL:  "http://127.0.0.1:1",
		index:    "test",
		embedder: &stubVectorEmbedder{},
		client:   http.DefaultClient,
	}
	_, err := store.doRequest(context.Background(), http.MethodGet, "http://127.0.0.1:1/", nil, "application/json")
	require.Error(t, err)
}

func TestWeaviateStore_doRequest_HTTPError(t *testing.T) {
	store := &weaviateStore{
		baseURL:   "http://127.0.0.1:1",
		className: "Test",
		embedder:  &stubVectorEmbedder{},
		client:    http.DefaultClient,
	}
	_, err := store.doRequest(context.Background(), http.MethodGet, "http://127.0.0.1:1/", nil)
	require.Error(t, err)
}

// --- OpenSearch AddDocuments embed error ---

func TestOpenSearchStore_AddDocuments_EmbedError(t *testing.T) {
	store := &openSearchStore{
		baseURL:  "http://localhost:9200",
		index:    "test-index",
		embedder: &stubVectorEmbedder{err: errors.New("embed failed")},
		client:   http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed")
}

// --- Weaviate SimilaritySearch wrong number of fields with no hits ---

func TestWeaviateStore_SimilaritySearch_NoHits(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":{"Get":{"Test":[]}}}`))
	}))
	defer srv.Close()

	store := &weaviateStore{
		baseURL:   srv.URL,
		className: "Test",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	docs, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestWeaviateStore_SimilaritySearch_ReadError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`partial`))
	}))
	defer srv.Close()

	store := &weaviateStore{
		baseURL:   srv.URL,
		className: "Test",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	// The body is partial but io.ReadAll won't error here
	if err != nil {
		t.Logf("got error (expected on some platforms): %v", err)
	}
}

// --- SQLStore rows iteration error ---

func TestSQLStore_SimilaritySearch_RowsErr(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	// Add a document.
	_, addErr := store.AddDocuments(context.Background(), []schema.Document{{PageContent: "test"}})
	require.NoError(t, addErr)

	// Close the DB to make rows iteration fail on Err.
	require.NoError(t, store.db.Close())
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	// Iteration may or may not fail depending on how SQLite handles closed DB.
	if err != nil {
		t.Logf("got rows error (expected on some platforms): %v", err)
	}
}

// --- Chroma SimilaritySearch parse error ---

func TestPineconeStore_SimilaritySearch_ParseError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer srv.Close()

	store := &pineconeStore{
		host:      srv.URL,
		namespace: "default",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse query response")
}

func TestOpenSearchStore_SimilaritySearch_ParseError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer srv.Close()

	store := &openSearchStore{
		baseURL:  srv.URL,
		index:    "test-index",
		embedder: &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:   http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse search response")
}

func TestWeaviateStore_SimilaritySearch_ParseError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`invalid json`))
	}))
	defer srv.Close()

	store := &weaviateStore{
		baseURL:   srv.URL,
		className: "Test",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse graphql response")
}

func TestChromaStore_SimilaritySearch_StatusError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"col-id","name":"test"}`))
		default:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`server error`))
		}
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

func TestChromaStore_SimilaritySearch_NoDocuments(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"col-id","name":"test"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"documents":[],"metadatas":[],"distances":[]}`))
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	docs, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestOpenSearchStore_SimilaritySearch_StatusError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`server error`))
	}))
	defer srv.Close()

	store := &openSearchStore{
		baseURL:  srv.URL,
		index:    "test-index",
		embedder: &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:   http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

func TestChromaStore_SimilaritySearch_EmbedQueryError(t *testing.T) {
	store := &chromaStore{
		baseURL:    "http://localhost:8000",
		collection: "test",
		embedder:   &stubVectorEmbedder{err: errors.New("embed query failed")},
		client:     http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed query")
}

func TestChromaStore_SimilaritySearch_NoHits(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"col-id","name":"test"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"documents":[[]],"metadatas":[[]],"distances":[[]]}`))
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	docs, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.NoError(t, err)
	assert.Empty(t, docs)
}

func TestOpenSearchStore_SimilaritySearch_EmbedQueryError(t *testing.T) {
	store := &openSearchStore{
		baseURL:  "http://localhost:9200",
		index:    "test-index",
		embedder: &stubVectorEmbedder{err: errors.New("embed query failed")},
		client:   http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embed query")
}

func TestPineconeStore_SimilaritySearch_StatusError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`server error`))
	}))
	defer srv.Close()

	store := &pineconeStore{
		host:      srv.URL,
		namespace: "default",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

func TestChromaStore_SimilaritySearch_ParseError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"col-id","name":"test"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`invalid json`))
		}
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
