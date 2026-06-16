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
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

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
