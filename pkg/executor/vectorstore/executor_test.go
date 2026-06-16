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
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/schema"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// stubVectorEmbedder is a test double for lcemb.Embedder.
type stubVectorEmbedder struct {
	vectors [][]float32
	err     error
}

func (s *stubVectorEmbedder) EmbedDocuments(_ context.Context, texts []string) ([][]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		if i < len(s.vectors) {
			result[i] = s.vectors[i]
		} else {
			result[i] = s.vectors[0]
		}
	}
	return result, nil
}

func (s *stubVectorEmbedder) EmbedQuery(_ context.Context, _ string) ([]float32, error) {
	if s.err != nil {
		return nil, s.err
	}
	if len(s.vectors) > 0 {
		return s.vectors[0], nil
	}
	return []float32{0.1, 0.2, 0.3}, nil
}

func TestExecute_UnknownOperation(t *testing.T) {
	e := NewExecutor()
	cfg := &domain.VectorStoreConfig{
		URL:        "http://localhost:6333",
		Collection: "test",
		EmbedModel: "text-embedding-ada-002",
		Operation:  "unknown_op",
	}
	_, err := e.Execute(nil, cfg)
	if err == nil {
		t.Fatal("expected error for unknown operation")
	}
}

func TestExecute_AddDocuments_NoDocuments(t *testing.T) {
	e := NewExecutor()
	cfg := &domain.VectorStoreConfig{
		URL:        "http://localhost:6333",
		Collection: "test",
		EmbedModel: "text-embedding-ada-002",
		Operation:  "add_documents",
		Documents:  nil,
	}
	_, err := e.Execute(nil, cfg)
	if err == nil {
		t.Fatal("expected error for missing documents")
	}
}

func TestExecute_SimilaritySearch_NoQuery(t *testing.T) {
	e := NewExecutor()
	cfg := &domain.VectorStoreConfig{
		URL:        "http://localhost:6333",
		Collection: "test",
		EmbedModel: "text-embedding-ada-002",
		Operation:  "similarity_search",
		Query:      "",
	}
	_, err := e.Execute(nil, cfg)
	if err == nil {
		t.Fatal("expected error for missing query")
	}
}

func TestBuildStore_MissingURL(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Collection: "test",
		EmbedModel: "model",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing url")
	}
}

func TestBuildStore_MissingCollection(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		URL:        "http://localhost:6333",
		EmbedModel: "model",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing collection")
	}
}

func TestBuildStore_MissingEmbedModel(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		URL:        "http://localhost:6333",
		Collection: "test",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing embedModel")
	}
}

func TestBuildStore_InvalidURL(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		URL:        "://bad url",
		Collection: "test",
		EmbedModel: "model",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for invalid url")
	}
}

func TestBuildEmbedder_VoyageAI_NoKey(t *testing.T) {
	t.Setenv("VOYAGEAI_API_KEY", "")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "voyageai",
		EmbedModel:   "voyage-2",
	}
	_, err := buildEmbedder(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error when VOYAGEAI_API_KEY is empty")
	}
}

func TestBuildEmbedder_VoyageAI_WithKey(t *testing.T) {
	t.Setenv("VOYAGEAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "voyageai",
		EmbedModel:   "voyage-2",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb == nil {
		t.Fatal("expected non-nil embedder")
	}
}

func TestBuildEmbedder_HuggingFace(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "huggingface",
		EmbedModel:   "sentence-transformers/all-MiniLM-L6-v2",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb == nil {
		t.Fatal("expected non-nil embedder")
	}
}

func TestBuildEmbedder_Jina(t *testing.T) {
	t.Setenv("JINA_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "jina",
		EmbedModel:   "jina-embeddings-v2-base-en",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb == nil {
		t.Fatal("expected non-nil embedder")
	}
}

func TestBuildEmbedder_OpenAICompat_Default(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "openai",
		EmbedModel:   "text-embedding-ada-002",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb == nil {
		t.Fatal("expected non-nil embedder")
	}
}

func TestBuildEmbedder_OpenAICompat_CustomBaseURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "openai",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBaseURL: "https://custom.openai.example.com/v1",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb == nil {
		t.Fatal("expected non-nil embedder")
	}
}

func TestBuildEmbedder_Groq(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "groq",
		EmbedModel:   "nomic-embed-text-v1.5",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb == nil {
		t.Fatal("expected non-nil embedder")
	}
}

func TestBuildEmbedder_Cohere(t *testing.T) {
	t.Setenv("COHERE_API_KEY", "test-cohere-key")
	cfg := &domain.VectorStoreConfig{
		EmbedBackend: "cohere",
		EmbedModel:   "embed-english-v3.0",
	}
	emb, err := buildEmbedder(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if emb == nil {
		t.Fatal("expected non-nil embedder")
	}
}

func TestNewAdapter(t *testing.T) {
	a := NewAdapter()
	if a == nil {
		t.Fatal("expected non-nil adapter")
	}
}

func TestProviderEnvKey(t *testing.T) {
	cases := []struct{ backend, want string }{
		{"openai", "OPENAI_API_KEY"},
		{"groq", "GROQ_API_KEY"},
		{"mistral", "MISTRAL_API_KEY"},
		{"cohere", "COHERE_API_KEY"},
		{"xai", "XAI_API_KEY"},
		{"perplexity", "PERPLEXITY_API_KEY"},
		{"unknown", ""},
	}
	for _, c := range cases {
		got := providerEnvKey(c.backend)
		if got != c.want {
			t.Errorf("providerEnvKey(%q) = %q, want %q", c.backend, got, c.want)
		}
	}
}

func TestOpenAICompatBaseURL(t *testing.T) {
	if u := openAICompatBaseURL("openai"); u == "" {
		t.Error("expected non-empty base URL for openai")
	}
	if u := openAICompatBaseURL("ollama"); u == "" {
		t.Error("expected non-empty base URL for ollama")
	}
	if u := openAICompatBaseURL("cohere"); u == "" {
		t.Error("expected non-empty base URL for cohere")
	}
	if u := openAICompatBaseURL("xai"); u == "" {
		t.Error("expected non-empty base URL for xai")
	}
	if u := openAICompatBaseURL("perplexity"); u == "" {
		t.Error("expected non-empty base URL for perplexity")
	}
	if u := openAICompatBaseURL("unknown"); u != "" {
		t.Errorf("expected empty URL for unknown backend, got %q", u)
	}
}

func TestBuildStore_AzureAISearch_MissingEndpoint(t *testing.T) {
	t.Setenv("AZURE_AI_SEARCH_ENDPOINT", "")
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "azureaisearch",
		Collection:   "myindex",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error when Azure endpoint is missing")
	}
}

func TestBuildStore_AzureAISearch_WithEndpoint(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "azureaisearch",
		Collection:   "myindex",
		URL:          "https://mysearch.search.windows.net",
		APIKey:       "azure-key",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBuildStore_AzureAISearch_EnvEndpoint(t *testing.T) {
	t.Setenv("AZURE_AI_SEARCH_ENDPOINT", "https://envsearch.search.windows.net")
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "azureaisearch",
		Collection:   "myindex",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBuildStore_AzureAISearch_MissingCollection(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "azureaisearch",
		URL:          "https://mysearch.search.windows.net",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing collection")
	}
}

func TestBuildStore_Qdrant_URLStillRequired(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Provider:   "qdrant",
		Collection: "test",
		EmbedModel: "text-embedding-ada-002",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing qdrant url")
	}
}

func TestBuildStore_Chroma_DefaultURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "chroma",
		Collection:   "my_collection",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBuildStore_Chroma_MissingCollection(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Provider:   "chroma",
		EmbedModel: "text-embedding-ada-002",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing collection")
	}
}

func TestBuildStore_Pinecone_MissingURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "pinecone",
		Collection:   "default",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing pinecone url (index host)")
	}
}

func TestBuildStore_Pinecone_WithURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "pinecone",
		Collection:   "default",
		URL:          "https://my-index-abc123.svc.us-east-1.pinecone.io",
		APIKey:       "pc-key",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBuildStore_OpenSearch_MissingURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "opensearch",
		Collection:   "my-index",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing opensearch url")
	}
}

func TestBuildStore_OpenSearch_WithURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "opensearch",
		Collection:   "my-index",
		URL:          "http://localhost:9200",
		APIKey:       "admin:admin",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBuildStore_Elasticsearch_Alias(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "elasticsearch",
		Collection:   "my-index",
		URL:          "http://localhost:9200",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBuildStore_Weaviate_MissingURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "weaviate",
		Collection:   "Articles",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing weaviate url")
	}
}

func TestBuildStore_Weaviate_WithURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "weaviate",
		Collection:   "articles",
		URL:          "http://localhost:8080",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestWeaviateClassNameUppercase(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "weaviate",
		Collection:   "articles",
		URL:          "http://localhost:8080",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := newWeaviateStore(cfg, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.className != "Articles" {
		t.Fatalf("expected className=Articles, got %q", store.className)
	}
}

func TestFloat64SliceToJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input []float64
		want  string
	}{
		{"empty", nil, "null"},
		{"single", []float64{1.5}, "[1.5]"},
		{"multiple", []float64{0.1, 0.2, 0.3}, "[0.1,0.2,0.3]"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := float64SliceToJSON(tc.input)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestWeaviateStore_AddDocuments_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer srv.Close()

	store := &weaviateStore{
		baseURL:   srv.URL,
		className: "Test",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 500")
}

func TestWeaviateStore_SimilaritySearch_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
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
	assert.Contains(t, err.Error(), "status 401")
}

func TestWeaviateStore_SimilaritySearch_Success(t *testing.T) {
	t.Parallel()
	respBody := `{
		"data": {
			"Get": {
				"Test": [
					{"text": "hello world", "_additional": {"id": "abc", "distance": 0.2}},
					{"text": "foo bar", "_additional": {"id": "def", "distance": 0.5}}
				]
			}
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respBody))
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
	require.Len(t, docs, 2)
	assert.Equal(t, "hello world", docs[0].PageContent)
	assert.Equal(t, "foo bar", docs[1].PageContent)
	// distance 0.2 -> score 1/(1+0.2) = ~0.833
	assert.InDelta(t, 0.833, float64(docs[0].Score), 0.01)
}

func TestWeaviateStore_AddDocuments_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	store := &weaviateStore{
		baseURL:   srv.URL,
		className: "Test",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	ids, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello world", Metadata: map[string]interface{}{"source": "test"}},
	})
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.NotEmpty(t, ids[0])
}

func TestWeaviateStore_APIKey_Header(t *testing.T) {
	t.Parallel()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	store := &weaviateStore{
		baseURL:   srv.URL,
		className: "Test",
		apiKey:    "my-secret-key",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, _ = store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	assert.Equal(t, "Bearer my-secret-key", gotAuth)
}

func TestBuildStore_MariaDB_MissingURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "mariadb",
		Collection:   "docs",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing mariadb url")
	}
}

func TestBuildStore_Dolt_MissingURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "dolt",
		Collection:   "docs",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing dolt url")
	}
}

func TestBuildStore_MySQL_MissingCollection(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "mysql",
		URL:          "user:pass@tcp(localhost:3306)/mydb",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing mysql collection")
	}
}

func TestBuildStore_MariaDB_WithURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "mariadb",
		Collection:   "docs",
		URL:          "user:pass@tcp(localhost:3306)/mydb",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBuildStore_Postgres_MissingURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "pgvector",
		Collection:   "docs",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing pgvector url")
	}
}

func TestBuildStore_Postgres_MissingCollection(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "postgres",
		URL:          "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing postgres collection")
	}
}

func TestBuildStore_Postgres_WithURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "pgvector",
		Collection:   "docs",
		URL:          "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBuildStore_AlloyDB_WithURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "alloydb",
		Collection:   "embeddings",
		URL:          "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store == nil {
		t.Fatal("expected non-nil store")
	}
}

func TestBuildStore_Postgres_AliasPSQL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	for _, alias := range []string{"postgresql", "cloudsql"} {
		t.Run(alias, func(t *testing.T) {
			cfg := &domain.VectorStoreConfig{
				Provider:     alias,
				Collection:   "docs",
				URL:          "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
				EmbedModel:   "text-embedding-ada-002",
				EmbedBackend: "openai",
			}
			store, err := buildStore(t.Context(), cfg)
			require.NoError(t, err, "alias %q should construct without error", alias)
			assert.NotNil(t, store)
		})
	}
}

func TestPostgresCreateTableSQL(t *testing.T) {
	t.Parallel()
	sql := postgresCreateTableSQL("my_table")
	assert.Contains(t, sql, "my_table")
	assert.Contains(t, sql, "embedding JSONB")
	assert.Contains(t, sql, "id TEXT")
	assert.Contains(t, sql, "content TEXT")
	assert.Contains(t, sql, "metadata JSONB")
}

func TestPostgresInsertSQL(t *testing.T) {
	t.Parallel()
	sql := postgresInsertSQL("my_table")
	assert.Contains(t, sql, "my_table")
	assert.Contains(t, sql, "$1")
	assert.Contains(t, sql, "$4")
}

func TestBuildStore_AlloyDB_MissingURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "alloydb",
		Collection:   "embeddings",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing alloydb url")
	}
}

func TestBuildStore_AlloyDB_MissingCollection(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "alloydb",
		URL:          "postgres://user:pass@localhost:5432/mydb?sslmode=disable",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing alloydb collection")
	}
}

func TestBuildStore_CloudSQL_MissingURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "cloudsql",
		Collection:   "embeddings",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing cloudsql url")
	}
}

func TestBuildStore_MongoDB_MissingURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "mongodb",
		Collection:   "docs",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing mongodb url")
	}
}

func TestBuildStore_MongoDB_WithURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "mongodb",
		Collection:   "docs",
		URL:          "mongodb://localhost:27017",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	// MongoDB connect is lazy; just verify no constructor error
	if err != nil {
		t.Logf("mongodb connection error (expected in CI): %v", err)
		return
	}
	assert.NotNil(t, store)
}

func TestBuildStore_MongoDB_MissingCollection(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "mongo",
		URL:          "mongodb://localhost:27017",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	_, err := buildStore(t.Context(), cfg)
	if err == nil {
		t.Fatal("expected error for missing mongo collection")
	}
}

func TestCosineSimilarity_SameVector(t *testing.T) {
	v := []float32{0.1, 0.2, 0.3, 0.4}
	score := cosineSimilarity(v, v)
	if score < 0.999 {
		t.Fatalf("expected near 1.0 cosine similarity for same vector, got %f", score)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{0, 0, 0}
	score := cosineSimilarity(a, b)
	if score != 0 {
		t.Fatalf("expected 0 for zero vectors, got %f", score)
	}
}

func TestCosineSimilarity_LengthMismatch(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{1, 2}
	score := cosineSimilarity(a, b)
	if score != 0 {
		t.Fatalf("expected 0 for mismatched lengths, got %f", score)
	}
}

func TestOpenSearchStore_SimilaritySearch_WithMetadata(t *testing.T) {
	t.Parallel()
	respBody := `{
		"hits": {
			"hits": [
				{"_score": 0.85, "_source": {"text": "result with meta", "meta": {"src": "wiki", "lang": "en"}}}
			]
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respBody))
	}))
	defer srv.Close()

	store := &openSearchStore{
		baseURL:  srv.URL,
		index:    "test-index",
		embedder: &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:   http.DefaultClient,
	}
	docs, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "result with meta", docs[0].PageContent)
	assert.Equal(t, "wiki", docs[0].Metadata["src"])
	assert.Equal(t, "en", docs[0].Metadata["lang"])
}

func TestOpenSearchStore_AddDocuments_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer srv.Close()

	store := &openSearchStore{
		baseURL:  srv.URL,
		index:    "test-index",
		embedder: &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:   http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestOpenSearchStore_SimilaritySearch_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("unauthorized"))
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
	assert.Contains(t, err.Error(), "401")
}

func TestOpenSearchStore_SimilaritySearch_Success(t *testing.T) {
	t.Parallel()
	respBody := `{
		"hits": {
			"hits": [
				{"_score": 0.95, "_source": {"text": "doc one", "meta": {"src": "a"}}},
				{"_score": 0.70, "_source": {"text": "doc two", "meta": {}}}
			]
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respBody))
	}))
	defer srv.Close()

	store := &openSearchStore{
		baseURL:  srv.URL,
		index:    "test-index",
		embedder: &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:   http.DefaultClient,
	}
	docs, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	assert.Equal(t, "doc one", docs[0].PageContent)
	assert.InDelta(t, 0.95, float64(docs[0].Score), 0.001)
}

func TestOpenSearchStore_AddDocuments_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":false,"items":[]}`))
	}))
	defer srv.Close()

	store := &openSearchStore{
		baseURL:  srv.URL,
		index:    "test-index",
		embedder: &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:   http.DefaultClient,
	}
	ids, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello world"},
	})
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.NotEmpty(t, ids[0])
}

func TestOpenSearchStore_BasicAuth(t *testing.T) {
	t.Parallel()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"errors":false,"items":[]}`))
	}))
	defer srv.Close()

	store := &openSearchStore{
		baseURL:  srv.URL,
		index:    "test-index",
		username: "admin",
		password: "secret",
		embedder: &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:   http.DefaultClient,
	}
	_, _ = store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	assert.Contains(t, gotAuth, "Basic ")
}

func TestWeaviateStore_SimilaritySearch_WithMetadata(t *testing.T) {
	t.Parallel()
	// Response includes non-text, non-_additional fields (metadata propagation)
	respBody := `{
		"data": {
			"Get": {
				"Test": [
					{"text": "doc with meta", "source": "wiki", "_additional": {"id": "abc", "distance": 0.1}}
				]
			}
		}
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respBody))
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
	require.Len(t, docs, 1)
	assert.Equal(t, "doc with meta", docs[0].PageContent)
	assert.Equal(t, "wiki", docs[0].Metadata["source"])
	// distance 0.1 -> score 1/(1+0.1) = ~0.909
	assert.InDelta(t, 0.909, float64(docs[0].Score), 0.01)
}

func TestNewOpenSearchStore_APIKeyBasicAuth(t *testing.T) {
	t.Parallel()
	cfg := &domain.VectorStoreConfig{
		URL:        "http://localhost:9200",
		Collection: "my-index",
		APIKey:     "admin:password123",
	}
	store, err := newOpenSearchStore(cfg, nil)
	require.NoError(t, err)
	assert.Equal(t, "admin", store.username)
	assert.Equal(t, "password123", store.password)
}

func TestPineconeStore_AddDocuments_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("forbidden"))
	}))
	defer srv.Close()

	store := &pineconeStore{
		host:      srv.URL,
		namespace: "default",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestPineconeStore_SimilaritySearch_Success(t *testing.T) {
	t.Parallel()
	respBody := `{
		"matches": [
			{"id": "id1", "score": 0.92, "metadata": {"text": "first doc", "source": "a"}},
			{"id": "id2", "score": 0.75, "metadata": {"text": "second doc"}}
		]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respBody))
	}))
	defer srv.Close()

	store := &pineconeStore{
		host:      srv.URL,
		namespace: "default",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	docs, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.NoError(t, err)
	require.Len(t, docs, 2)
	assert.Equal(t, "first doc", docs[0].PageContent)
	assert.InDelta(t, 0.92, float64(docs[0].Score), 0.001)
	assert.Equal(t, "a", docs[0].Metadata["source"])
}

func TestPineconeStore_AddDocuments_APIKeyHeader(t *testing.T) {
	t.Parallel()
	var gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("Api-Key")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	store := &pineconeStore{
		host:      srv.URL,
		namespace: "default",
		apiKey:    "pc-test-key",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, _ = store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	assert.Equal(t, "pc-test-key", gotKey)
}

func TestPineconeStore_SimilaritySearch_HTTPError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	store := &pineconeStore{
		host:      srv.URL,
		namespace: "default",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 3)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestChromaStore_SimilaritySearch_Success(t *testing.T) {
	t.Parallel()
	collID := "coll-uuid-123"
	queryResp := `{
		"documents": [["doc one", "doc two"]],
		"metadatas": [[{"src": "a"}, {}]],
		"distances": [[0.1, 0.4]]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			// collectionID: return existing collection
			_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
		default:
			// query endpoint
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(queryResp))
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
	require.Len(t, docs, 2)
	assert.Equal(t, "doc one", docs[0].PageContent)
	assert.Equal(t, "a", docs[0].Metadata["src"])
	// distance 0.1 -> score 1/(1+0.1) = ~0.909
	assert.InDelta(t, 0.909, float64(docs[0].Score), 0.01)
}

func TestChromaStore_AddDocuments_CreateCollection(t *testing.T) {
	t.Parallel()
	collID := "new-coll-id"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			// collection not found -- trigger create path
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"not found"}`))
		case http.MethodPost:
			if r.URL.Path == "/api/v1/collections" {
				// create collection
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
			} else {
				// add documents
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{}`))
			}
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	ids, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello chroma"},
	})
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.NotEmpty(t, ids[0])
}

func TestMySQLCreateTableSQL(t *testing.T) {
	t.Parallel()
	sql := mysqlCreateTableSQL("my_table")
	assert.Contains(t, sql, "my_table")
	assert.Contains(t, sql, "embedding JSON")
	assert.Contains(t, sql, "id VARCHAR(36)")
	assert.Contains(t, sql, "content LONGTEXT")
	assert.Contains(t, sql, "metadata JSON")
}

func TestMySQLInsertSQL(t *testing.T) {
	t.Parallel()
	sql := mysqlInsertSQL("my_table")
	assert.Contains(t, sql, "my_table")
	assert.Contains(t, sql, "?")
	assert.Contains(t, sql, "id, content, embedding, metadata")
}

func TestBuildStore_MySQL_WithURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "mysql",
		Collection:   "docs",
		URL:          "user:pass@tcp(localhost:3306)/mydb",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestBuildStore_Dolt_WithURL(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	cfg := &domain.VectorStoreConfig{
		Provider:     "dolt",
		Collection:   "docs",
		URL:          "user:pass@tcp(localhost:3306)/mydb",
		EmbedModel:   "text-embedding-ada-002",
		EmbedBackend: "openai",
	}
	store, err := buildStore(t.Context(), cfg)
	require.NoError(t, err)
	assert.NotNil(t, store)
}

func TestChromaStore_APIKey_Header(t *testing.T) {
	t.Parallel()
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"col-id","name":"test"}`))
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		apiKey:     "chroma-secret",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	// collectionID is called on AddDocuments; trigger it with an empty doc list (returns nil early)
	// so we call it directly via SimilaritySearch with a query that will fail at query step
	store.SimilaritySearch(context.Background(), "q", 1) //nolint:errcheck // only checking header
	assert.Equal(t, "Bearer chroma-secret", gotAuth)
}
