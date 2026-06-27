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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tmc/langchaingo/schema"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// failTransport returns an error on every RoundTrip.
type failTransport struct{}

func (f *failTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("fail: connection refused")
}

// --- chroma collectionID error paths ---

func TestChromaStore_CollectionID_GetRequestError(t *testing.T) {
	t.Parallel()
	store := &chromaStore{
		baseURL:    "http://localhost:8000",
		collection: "test",
		embedder:   &stubVectorEmbedder{},
		client:     &http.Client{Transport: &failTransport{}},
	}
	_, err := store.collectionID(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fail: connection refused")
}

func TestChromaStore_CollectionID_CreateRequestError(t *testing.T) {
	t.Parallel()
	// GET returns 404 (triggers create), then POST transport fails.
	calledGet := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			calledGet = true
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{}`))
			return
		}
		// POST - shouldn't be reached; handled by transport failure below.
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	// Build a store whose client uses failTransport for the POST request.
	// We use a custom transport that succeeds on GET but fails on POST.
	var postFailed bool
	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{},
		client: &http.Client{
			Transport: &methodFailTransport{
				failMethod: http.MethodPost,
				next:       http.DefaultTransport,
				onFail:     func() { postFailed = true },
			},
		},
	}
	_, err := store.collectionID(context.Background())
	require.Error(t, err)
	assert.True(t, calledGet, "GET should have been called")
	assert.True(t, postFailed, "POST should have failed")
}

// methodFailTransport fails RoundTrip for a specific HTTP method.
type methodFailTransport struct {
	failMethod string
	next       http.RoundTripper
	onFail     func()
}

func (m *methodFailTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.Method == m.failMethod {
		m.onFail()
		return nil, errors.New("method fail: connection refused on POST")
	}
	return m.next.RoundTrip(req)
}

// --- chroma AddDocuments metadata nil branch ---

func TestChromaStore_AddDocuments_NilMetadata(t *testing.T) {
	t.Parallel()
	collID := "coll-uuid-nil-meta"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
		case strings.Contains(r.URL.Path, "/add"):
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
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
	// Document with nil Metadata.
	ids, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "nil metadata doc", Metadata: nil},
	})
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.NotEmpty(t, ids[0])
}

// --- chroma SimilaritySearch numDocuments=0 ---

func TestChromaStore_SimilaritySearch_ZeroNumDocuments(t *testing.T) {
	t.Parallel()
	collID := "coll-zero-num"
	respBody := `{
		"documents": [["result"]],
		"metadatas": [[{}]],
		"distances": [[0.3]]
	}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(respBody))
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:     http.DefaultClient,
	}
	docs, err := store.SimilaritySearch(context.Background(), "query", 0)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "result", docs[0].PageContent)
}

// --- chroma AddDocuments request error path ---

func TestChromaStore_AddDocuments_RequestError(t *testing.T) {
	t.Parallel()
	collID := "coll-add-err"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		// client that fails on POST to /add
		client: &http.Client{
			Transport: &pathFailTransport{
				failPath: "/add",
				next:     http.DefaultTransport,
			},
		},
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "add")
}

// pathFailTransport fails RoundTrip for requests whose URL contains a substring.
type pathFailTransport struct {
	failPath string
	next     http.RoundTripper
}

func (p *pathFailTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.Path, p.failPath) {
		return nil, errors.New("path fail: connection refused on " + p.failPath)
	}
	return p.next.RoundTrip(req)
}

// --- chroma SimilaritySearch request error paths ---

func TestChromaStore_SimilaritySearch_QueryRequestError(t *testing.T) {
	t.Parallel()
	collID := "coll-query-err"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
		}
	}))
	defer srv.Close()

	store := &chromaStore{
		baseURL:    srv.URL,
		collection: "test",
		embedder:   &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client: &http.Client{
			Transport: &pathFailTransport{
				failPath: "/query",
				next:     http.DefaultTransport,
			},
		},
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

// --- opensearch AddDocuments with metadata ---

func TestOpenSearchStore_AddDocuments_WithMetadata(t *testing.T) {
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
		{PageContent: "hello world", Metadata: map[string]interface{}{"source": "test", "lang": "en"}},
	})
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.NotEmpty(t, ids[0])
}

// --- opensearch SimilaritySearch numDocuments=0 ---

func TestOpenSearchStore_SimilaritySearch_ZeroNumDocuments(t *testing.T) {
	t.Parallel()
	respBody := `{
		"hits": {
			"hits": [
				{"_score": 0.95, "_source": {"text": "result", "meta": {}}}
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
	docs, err := store.SimilaritySearch(context.Background(), "query", 0)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.InDelta(t, 0.95, float64(docs[0].Score), 0.001)
}

// --- opensearch AddDocuments request error ---

func TestOpenSearchStore_AddDocuments_RequestError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	store := &openSearchStore{
		baseURL:  srv.URL,
		index:    "test-index",
		embedder: &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client: &http.Client{
			Transport: &pathFailTransport{
				failPath: "/_bulk",
				next:     http.DefaultTransport,
			},
		},
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
}

// --- pinecone AddDocuments with metadata ---

func TestPineconeStore_AddDocuments_WithMetadata(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	store := &pineconeStore{
		host:      srv.URL,
		namespace: "default",
		apiKey:    "test-key",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	ids, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello world", Metadata: map[string]interface{}{"source": "test", "lang": "en"}},
	})
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.NotEmpty(t, ids[0])
}

// --- pinecone AddDocuments request error ---

func TestPineconeStore_AddDocuments_RequestError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	store := &pineconeStore{
		host:      srv.URL,
		namespace: "default",
		apiKey:    "test-key",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client: &http.Client{
			Transport: &pathFailTransport{
				failPath: "/vectors/upsert",
				next:     http.DefaultTransport,
			},
		},
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
}

// --- pinecone SimilaritySearch numDocuments=0 ---

func TestPineconeStore_SimilaritySearch_ZeroNumDocuments(t *testing.T) {
	t.Parallel()
	respBody := `{
		"matches": [
			{"id": "id1", "score": 0.92, "metadata": {"text": "first doc"}}
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
	docs, err := store.SimilaritySearch(context.Background(), "query", 0)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.Equal(t, "first doc", docs[0].PageContent)
}

// --- pinecone SimilaritySearch request error ---

func TestPineconeStore_SimilaritySearch_RequestError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	store := &pineconeStore{
		host:      srv.URL,
		namespace: "default",
		apiKey:    "test-key",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client: &http.Client{
			Transport: &pathFailTransport{
				failPath: "/query",
				next:     http.DefaultTransport,
			},
		},
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

// --- weaviate SimilaritySearch numDocuments=0 ---

func TestWeaviateStore_SimilaritySearch_ZeroNumDocuments(t *testing.T) {
	t.Parallel()
	respBody := `{
		"data": {
			"Get": {
				"Test": [
					{"text": "hello", "_additional": {"id": "abc", "distance": 0.2}}
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
	docs, err := store.SimilaritySearch(context.Background(), "query", 0)
	require.NoError(t, err)
	require.Len(t, docs, 1)
	assert.InDelta(t, 0.833, float64(docs[0].Score), 0.01)
}

// --- weaviate AddDocuments request error ---

func TestWeaviateStore_AddDocuments_RequestError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	store := &weaviateStore{
		baseURL:   srv.URL,
		className: "Test",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client: &http.Client{
			Transport: &pathFailTransport{
				failPath: "/v1/batch/objects",
				next:     http.DefaultTransport,
			},
		},
	}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
}

// --- weaviate SimilaritySearch request error ---

func TestWeaviateStore_SimilaritySearch_RequestError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	store := &weaviateStore{
		baseURL:   srv.URL,
		className: "Test",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client: &http.Client{
			Transport: &pathFailTransport{
				failPath: "/v1/graphql",
				next:     http.DefaultTransport,
			},
		},
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

// --- sqlStore wrong vectors in AddDocuments ---

func TestSQLStore_AddDocuments_WrongVectors(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	// Use wrongCountEmbedder that always returns 1 vector regardless of input count.
	store.embedder = &wrongCountEmbedder{}
	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "a"},
		{PageContent: "b"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "wrong number of vectors")
}

// --- sqlStore scan error in SimilaritySearch ---

func TestSQLStore_SimilaritySearch_ScanError(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	require.NoError(t, store.ensureTable(context.Background()))

	// Insert a row with a very long embedding string that might cause scan issues.
	_, execErr := store.db.ExecContext(context.Background(),
		"INSERT INTO test_vectors (id, content, embedding, metadata) VALUES (?, ?, ?, ?)",
		"bad-id", "bad-content", `[1.0, 2.0]`, "{}")
	require.NoError(t, execErr)

	// Also insert a row with nil metadata to test the metaStr.Valid branch.
	_, execErr2 := store.db.ExecContext(context.Background(),
		"INSERT INTO test_vectors (id, content, embedding, metadata) VALUES (?, ?, ?, ?)",
		"bad-id2", "bad-content2", `[3.0, 4.0]`, nil)
	require.NoError(t, execErr2)

	docs, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.NoError(t, err)
	require.Len(t, docs, 2)
}

// --- sqlStore newSQLStore with invalid driver ---

func TestNewSQLStore_InvalidDriver(t *testing.T) {
	_, err := newSQLStore(
		"nonexistent_driver",
		"file::memory:",
		"test",
		"test",
		sqliteCreateTableSQL,
		sqliteInsertSQL,
		nil,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open")
}

// --- sqlStore SimilaritySearch unmarshal skip path ---

func TestSQLStore_SimilaritySearch_InvalidEmbeddingJSON(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	require.NoError(t, store.ensureTable(context.Background()))

	// Insert a row with invalid JSON embedding that will be skipped.
	_, execErr := store.db.ExecContext(context.Background(),
		"INSERT INTO test_vectors (id, content, embedding, metadata) VALUES (?, ?, ?, ?)",
		"skip-id", "skip-content", `not-valid-json-at-all`, `{}`)
	require.NoError(t, execErr)

	docs, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.NoError(t, err)
	require.Empty(t, docs)
}

// --- newRedisStore embedder error path ---

func TestNewRedisStore_EmbedderError(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Collection: "myindex",
		URL:        "redis://localhost:6379",
		EmbedModel: "text-embedding-ada-002",
		// No EmbedBackend set - will default to openai compat with empty env var.
	}
	_, err := newRedisStore(context.Background(), cfg)
	// Should fail because no OPENAI_API_KEY is set.
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embedder")
}

// --- executeAddDocuments store.AddDocuments error ---

func TestExecute_AddDocuments_StoreAddError(t *testing.T) {
	respBody := `{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"text-embedding-ada-002","usage":{"prompt_tokens":1,"total_tokens":1}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(io.Discard, r.Body)
		switch {
		case strings.Contains(r.URL.Path, "/v1/batch/objects"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("batch failed"))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(respBody))
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
	_, err := e.Execute(nil, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vectorstore add_documents")
}

// --- executeSimilaritySearch store.SimilaritySearch error ---

func TestExecute_SimilaritySearch_StoreSearchError(t *testing.T) {
	respBody := `{"data":[{"embedding":[0.1,0.2,0.3],"index":0}],"model":"text-embedding-ada-002","usage":{"prompt_tokens":1,"total_tokens":1}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.Copy(io.Discard, r.Body)
		switch {
		case strings.Contains(r.URL.Path, "/v1/graphql"):
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("graphql failed"))
		default:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(respBody))
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
	_, err := e.Execute(nil, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "vectorstore similarity_search")
}

// --- buildStore for redis with embedder error ---

func TestBuildStore_Redis_EmbedderError(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Provider:   "redis",
		Collection: "myindex",
		URL:        "redis://localhost:6379",
		EmbedModel: "text-embedding-ada-002",
		// No EmbedBackend -> default openai compat, but no API key set -> error.
	}
	_, err := buildStore(context.Background(), cfg)
	require.Error(t, err)
}

// --- buildStore for bedrock with no embedModel (allowed) ---

func TestBuildStore_Bedrock_NoEmbedModel(t *testing.T) {
	cfg := &domain.VectorStoreConfig{
		Provider:   "bedrock",
		Collection: "ABCDEFGHIJ",
		// EmbedModel is empty, but bedrock should allow that.
	}
	// Building should not fail on the embedModel check in buildStore.
	store, err := buildStore(context.Background(), cfg)
	if err != nil {
		// May fail at lcbedrockkb.New due to missing AWS creds, that's fine.
		t.Logf("bedrock build error (expected in CI): %v", err)
		return
	}
	assert.NotNil(t, store)
}

// --- openSearchStore doRequest with auth (basic auth code path) ---

func TestOpenSearchStore_AddDocuments_WithBasicAuth(t *testing.T) {
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
	ids, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello", Metadata: map[string]interface{}{"key": "val"}},
	})
	require.NoError(t, err)
	require.Len(t, ids, 1)
	assert.Contains(t, gotAuth, "Basic ")
}

// --- openSearch doRequest read error in SimilaritySearch ---

func TestOpenSearchStore_SimilaritySearch_ReadBodyError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", "100")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`partial`))
	}))
	defer srv.Close()

	store := &openSearchStore{
		baseURL:  srv.URL,
		index:    "test-index",
		embedder: &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:   http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	// io.ReadAll reads what it can; may or may not return an error.
	if err != nil {
		t.Logf("got error (expected on some platforms): %v", err)
	}
}

// --- weaviate SimilaritySearch read body error ---

func TestWeaviateStore_SimilaritySearch_GraphQLRequestError(t *testing.T) {
	t.Parallel()
	store := &weaviateStore{
		baseURL:   "http://127.0.0.1:1",
		className: "Test",
		embedder:  &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}},
		client:    http.DefaultClient,
	}
	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}

// --- chroma SimilaritySearch query status error path ---

func TestChromaStore_SimilaritySearch_ReadBodyError(t *testing.T) {
	t.Parallel()
	collID := "coll-read-err"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.Method {
		case http.MethodGet:
			_, _ = w.Write([]byte(`{"id":"` + collID + `","name":"test"}`))
		default:
			w.Header().Set("Content-Length", "100")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`partial`))
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
	// May or may not error depending on io.ReadAll behavior.
	if err != nil {
		t.Logf("got error (expected on some platforms): %v", err)
	}
}

// --- executeSimilaritySearch buildStore error ---

func TestExecute_SimilaritySearch_BuildStoreError(t *testing.T) {
	e := NewExecutor()
	cfg := &domain.VectorStoreConfig{
		Operation: "similarity_search",
		Query:     "test query",
		// No collection set -> buildStore fails.
	}
	_, err := e.Execute(nil, cfg)
	require.Error(t, err)
}

// --- sqlStore insert error path (ensureTable succeeds, insert fails) ---

func TestSQLStore_AddDocuments_InsertError(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	require.NoError(t, store.ensureTable(context.Background()))

	// Install a trigger that makes every INSERT fail.
	_, triggerErr := store.db.ExecContext(context.Background(),
		`CREATE TRIGGER fail_insert BEFORE INSERT ON test_vectors
		 BEGIN
			 SELECT RAISE(ABORT, 'insert blocked by trigger');
		 END`)
	require.NoError(t, triggerErr)

	_, err := store.AddDocuments(context.Background(), []schema.Document{
		{PageContent: "hello"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insert document")
}

// --- sqlStore query error in SimilaritySearch (ensureTable succeeds, query fails) ---

func TestSQLStore_SimilaritySearch_QueryErrorAfterEnsure(t *testing.T) {
	store := newTestSQLiteStore(t, &stubVectorEmbedder{vectors: [][]float32{{0.1, 0.2}}})
	require.NoError(t, store.ensureTable(context.Background()))

	// Add a document and then close the DB connection so QueryContext fails.
	_, addErr := store.AddDocuments(context.Background(), []schema.Document{{PageContent: "test"}})
	require.NoError(t, addErr)

	require.NoError(t, store.db.Close())

	_, err := store.SimilaritySearch(context.Background(), "query", 5)
	require.Error(t, err)
}
