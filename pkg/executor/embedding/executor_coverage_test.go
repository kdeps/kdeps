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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

// Package embedding – additional coverage tests.
//
// This file closes the gaps in executor_test.go to reach ~100% statement coverage:
//   - evaluateText (nil ctx, no-expression, working expression, eval-error fallback)
//   - TimeoutDuration applied to the HTTP client
//   - openVectorDB failure path (bad DB path)
//   - operationIndex: unmarshalable metadata
//   - operationDelete with metadata-only and no-input/no-metadata configs
//   - Default URL fallbacks for all four backends (no BaseURL → connection error)
//   - Empty-embedding response errors (Ollama, OpenAI, Cohere)
//   - Bad-JSON decode errors for all backends
//   - HTTP-error paths for OpenAI, Cohere, HuggingFace backends
//   - HuggingFace default model fallback (empty Model field)
//   - http.NewRequestWithContext failure paths for all backends (bad BaseURL "://invalid")
//   - parseHuggingFaceResponse edge cases (non-float, unexpected element type)
//   - cosineSimilarity zero-denominator path
//   - Search sort swap (later-inserted doc ranked higher than earlier one)
//   - Search skips malformed embedding rows (invalid JSON in embedding column)
package embedding

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// ─── mock transport ───────────────────────────────────────────────────────────

// roundTripFunc lets us inject a custom transport without a full httptest.Server.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// clientAlwaysError returns a client whose transport always returns an error.
func clientAlwaysError(msg string) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
			return nil, errors.New(msg)
		}),
	}
}

// ─── evaluateText ─────────────────────────────────────────────────────────────

func TestEvaluateText_NoExpression(t *testing.T) {
	e := &Executor{logger: slog.Default(), client: http.DefaultClient}
	// Text without {{ is returned as-is without touching the ctx.
	got := e.evaluateText("plain text", nil)
	assert.Equal(t, "plain text", got)
}

func TestEvaluateText_NilCtx(t *testing.T) {
	e := &Executor{logger: slog.Default(), client: http.DefaultClient}
	// Text with {{ but nil ctx → fall back to raw text.
	got := e.evaluateText("{{ get('something') }}", nil)
	assert.Equal(t, "{{ get('something') }}", got)
}

func TestEvaluateText_NilCtxAPI(t *testing.T) {
	// A non-nil ctx but with API == nil also falls through.
	ctx := &executor.ExecutionContext{}
	e := &Executor{logger: slog.Default(), client: http.DefaultClient}
	got := e.evaluateText("{{ get('x') }}", ctx)
	assert.Equal(t, "{{ get('x') }}", got)
}

func TestEvaluateText_WithExpression(t *testing.T) {
	// Build a real execution context so ctx.API is wired up.
	ctx := makeCtx(t)
	// Seed a value into memory so get('mykey') resolves.
	require.NoError(t, ctx.Memory.Set("mykey", "hello-from-expression"))

	e := &Executor{logger: slog.Default(), client: http.DefaultClient}
	got := e.evaluateText("{{ get('mykey') }}", ctx)
	assert.Equal(t, "hello-from-expression", got)
}

func TestEvaluateText_EvalErrorFallback(t *testing.T) {
	// An expression that fails compilation falls back to the raw text.
	ctx := makeCtx(t)
	e := &Executor{logger: slog.Default(), client: http.DefaultClient}
	// "{{ ??? }}" is a syntax error in the expr language; evaluateText must
	// return the original raw string on error.
	raw := "{{ ??? }}"
	got := e.evaluateText(raw, ctx)
	assert.Equal(t, raw, got)
}

// ─── TimeoutDuration applied ──────────────────────────────────────────────────

func TestExecute_TimeoutDurationApplied(t *testing.T) {
	vec := []float64{0.1, 0.2}
	srv := ollamaEmbedServer(t, vec)
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	cfg := &domain.EmbeddingConfig{
		Model:           "nomic-embed-text",
		BaseURL:         srv.URL,
		Input:           "test",
		DBPath:          tmpDBPath(t, "timeout"),
		TimeoutDuration: "5s",
	}

	result, err := exec.Execute(ctx, cfg)
	require.NoError(t, err)
	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, true, m["success"])
}

// ─── openVectorDB failure ─────────────────────────────────────────────────────

func TestExecute_OpenVectorDB_Failure(t *testing.T) {
	ctx := makeCtx(t)
	exec := NewAdapter(nil)

	cfg := &domain.EmbeddingConfig{
		Model:  "m",
		Input:  "test",
		DBPath: "/nonexistent-coverage-dir/sub/test.db",
	}

	_, err := exec.Execute(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "open vector DB")
}

func TestOpenVectorDB_BadPath(t *testing.T) {
	_, err := openVectorDB("/nonexistent-dir-xyz/test.db", "t")
	require.Error(t, err)
}

// ─── operationIndex: unmarshalable metadata ───────────────────────────────────

func TestExecute_Index_UnmarshalableMetadata(t *testing.T) {
	vec := []float64{0.1, 0.2}
	srv := ollamaEmbedServer(t, vec)
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	// Functions cannot be marshaled to JSON → triggers the marshal-metadata error path.
	cfg := &domain.EmbeddingConfig{
		Model:    "nomic-embed-text",
		BaseURL:  srv.URL,
		Input:    "test with bad metadata",
		DBPath:   tmpDBPath(t, "bad-meta"),
		Metadata: map[string]interface{}{"fn": func() {}},
	}

	_, err := exec.Execute(ctx, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "marshal metadata")
}

// ─── operationDelete branches ─────────────────────────────────────────────────

func TestExecute_Delete_MetadataOnly(t *testing.T) {
	vec := []float64{0.1, 0.2}
	srv := ollamaEmbedServer(t, vec)
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())
	dbPath := tmpDBPath(t, "del-meta")

	// Index a document first.
	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "m",
		BaseURL: srv.URL,
		Input:   "some doc",
		DBPath:  dbPath,
	})
	require.NoError(t, err)

	// Delete with metadata and empty Input → metadata-only branch (line 320-323).
	result, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:     "m",
		BaseURL:   srv.URL,
		Input:     "",
		DBPath:    dbPath,
		Operation: domain.EmbeddingOperationDelete,
		Metadata:  map[string]interface{}{"src": "test"},
	})
	require.NoError(t, err)
	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "delete", m["operation"])
}

func TestExecute_Delete_AllRows(t *testing.T) {
	vec := []float64{0.1, 0.2}
	srv := ollamaEmbedServer(t, vec)
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())
	dbPath := tmpDBPath(t, "del-all")

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "m",
		BaseURL: srv.URL,
		Input:   "doc",
		DBPath:  dbPath,
	})
	require.NoError(t, err)

	// Delete with empty Input and nil Metadata → delete-all branch (line 323-326).
	result, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:     "m",
		DBPath:    dbPath,
		Operation: domain.EmbeddingOperationDelete,
		// Input == "" and Metadata == nil
	})
	require.NoError(t, err)
	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "delete", m["operation"])
	assert.EqualValues(t, 1, m["deleted"])
}

// ─── Ollama backend edge cases ────────────────────────────────────────────────

func TestOllamaEmbed_DefaultURL_NetworkError(t *testing.T) {
	// Omit BaseURL so the code falls through to defaultOllamaURL.
	// The network call will fail, but coverage is achieved for the
	// `if baseURL == ""` branch (lines 367-368).
	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, clientAlwaysError("connection refused"))

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:  "nomic-embed-text",
		Input:  "test",
		DBPath: tmpDBPath(t, "ollama-default-url"),
		// BaseURL intentionally empty → uses defaultOllamaURL
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "connection refused")
}

func TestOllamaEmbed_EmptyEmbeddings(t *testing.T) {
	// Server returns valid JSON but an empty embeddings array.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings": []}`))
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "m",
		Backend: domain.EmbeddingBackendOllama,
		BaseURL: srv.URL,
		Input:   "test",
		DBPath:  tmpDBPath(t, "ollama-empty"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty embeddings")
}

func TestOllamaEmbed_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "m",
		Backend: domain.EmbeddingBackendOllama,
		BaseURL: srv.URL,
		Input:   "test",
		DBPath:  tmpDBPath(t, "ollama-badjson"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestOllamaEmbed_BadBaseURL_NewRequestError(t *testing.T) {
	// A BaseURL of "://invalid" makes http.NewRequestWithContext return a
	// "missing protocol scheme" error (lines 382-383).
	ctx := makeCtx(t)
	exec := NewAdapter(nil)

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "m",
		Backend: domain.EmbeddingBackendOllama,
		BaseURL: "://invalid",
		Input:   "test",
		DBPath:  tmpDBPath(t, "ollama-badurl"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new request")
}

// ─── OpenAI backend edge cases ────────────────────────────────────────────────

func TestOpenAIEmbed_DefaultURL_NetworkError(t *testing.T) {
	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, clientAlwaysError("openai connection refused"))

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "text-embedding-3-small",
		Backend: domain.EmbeddingBackendOpenAI,
		Input:   "test",
		DBPath:  tmpDBPath(t, "openai-default-url"),
		// BaseURL intentionally empty (lines 411-412)
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "openai connection refused")
}

func TestOpenAIEmbed_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "text-embedding-3-small",
		Backend: domain.EmbeddingBackendOpenAI,
		BaseURL: srv.URL,
		APIKey:  "bad-key",
		Input:   "test",
		DBPath:  tmpDBPath(t, "openai-http-error"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 401")
}

func TestOpenAIEmbed_EmptyEmbedding(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data": []}`))
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "text-embedding-3-small",
		Backend: domain.EmbeddingBackendOpenAI,
		BaseURL: srv.URL,
		Input:   "test",
		DBPath:  tmpDBPath(t, "openai-empty"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty embedding")
}

func TestOpenAIEmbed_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "text-embedding-3-small",
		Backend: domain.EmbeddingBackendOpenAI,
		BaseURL: srv.URL,
		Input:   "test",
		DBPath:  tmpDBPath(t, "openai-badjson"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestOpenAIEmbed_BadBaseURL_NewRequestError(t *testing.T) {
	ctx := makeCtx(t)
	exec := NewAdapter(nil)

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "text-embedding-3-small",
		Backend: domain.EmbeddingBackendOpenAI,
		BaseURL: "://invalid",
		Input:   "test",
		DBPath:  tmpDBPath(t, "openai-badurl"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new request")
}

// ─── Cohere backend edge cases ────────────────────────────────────────────────

func TestCohereEmbed_DefaultURL_NetworkError(t *testing.T) {
	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, clientAlwaysError("cohere connection refused"))

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "embed-english-v3.0",
		Backend: domain.EmbeddingBackendCohere,
		Input:   "test",
		DBPath:  tmpDBPath(t, "cohere-default-url"),
		// BaseURL intentionally empty (lines 458-459)
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cohere connection refused")
}

func TestCohereEmbed_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "embed-english-v3.0",
		Backend: domain.EmbeddingBackendCohere,
		BaseURL: srv.URL,
		Input:   "test",
		DBPath:  tmpDBPath(t, "cohere-http-error"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 403")
}

func TestCohereEmbed_EmptyEmbeddings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"embeddings": []}`))
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "embed-english-v3.0",
		Backend: domain.EmbeddingBackendCohere,
		BaseURL: srv.URL,
		Input:   "test",
		DBPath:  tmpDBPath(t, "cohere-empty"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "empty embedding")
}

func TestCohereEmbed_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "embed-english-v3.0",
		Backend: domain.EmbeddingBackendCohere,
		BaseURL: srv.URL,
		Input:   "test",
		DBPath:  tmpDBPath(t, "cohere-badjson"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestCohereEmbed_BadBaseURL_NewRequestError(t *testing.T) {
	ctx := makeCtx(t)
	exec := NewAdapter(nil)

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "embed-english-v3.0",
		Backend: domain.EmbeddingBackendCohere,
		BaseURL: "://invalid",
		Input:   "test",
		DBPath:  tmpDBPath(t, "cohere-badurl"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new request")
}

// ─── HuggingFace backend edge cases ──────────────────────────────────────────

func TestHuggingFaceEmbed_DefaultURLAndModel_NetworkError(t *testing.T) {
	// Omit both BaseURL and Model to exercise the default-URL (lines 505-506)
	// and default-model (lines 509-510) branches.
	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, clientAlwaysError("hf connection refused"))

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Backend: domain.EmbeddingBackendHuggingFace,
		Input:   "test",
		DBPath:  tmpDBPath(t, "hf-defaults"),
		// Model intentionally empty → "sentence-transformers/all-MiniLM-L6-v2"
		// BaseURL intentionally empty → https://api-inference.huggingface.co
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hf connection refused")
}

func TestHuggingFaceEmbed_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "all-MiniLM-L6-v2",
		Backend: domain.EmbeddingBackendHuggingFace,
		BaseURL: srv.URL,
		Input:   "test",
		DBPath:  tmpDBPath(t, "hf-http-error"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 503")
}

func TestHuggingFaceEmbed_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`not-json`))
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "all-MiniLM-L6-v2",
		Backend: domain.EmbeddingBackendHuggingFace,
		BaseURL: srv.URL,
		Input:   "test",
		DBPath:  tmpDBPath(t, "hf-badjson"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "decode response")
}

func TestHuggingFaceEmbed_BadBaseURL_NewRequestError(t *testing.T) {
	ctx := makeCtx(t)
	exec := NewAdapter(nil)

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "all-MiniLM-L6-v2",
		Backend: domain.EmbeddingBackendHuggingFace,
		BaseURL: "://invalid",
		Input:   "test",
		DBPath:  tmpDBPath(t, "hf-badurl"),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new request")
}

// ─── parseHuggingFaceResponse additional edge cases ──────────────────────────

func TestParseHuggingFaceResponse_NonFloatInFlat(t *testing.T) {
	// Flat array but one element is a string, not float64.
	raw := []interface{}{float64(0.1), "not-a-float"}
	_, err := parseHuggingFaceResponse(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-float value in embedding")
}

func TestParseHuggingFaceResponse_NonFloatInNested(t *testing.T) {
	// Nested array but inner element is a string.
	raw := []interface{}{
		[]interface{}{float64(0.1), "bad"},
	}
	_, err := parseHuggingFaceResponse(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "non-float value in nested embedding")
}

func TestParseHuggingFaceResponse_UnexpectedElementType(t *testing.T) {
	// First element is neither float64 nor []interface{}.
	raw := []interface{}{map[string]interface{}{"key": "value"}}
	_, err := parseHuggingFaceResponse(raw)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected response format")
}

// ─── cosineSimilarity zero-denominator ───────────────────────────────────────

func TestCosineSimilarity_ZeroNorm(t *testing.T) {
	// Both vectors are all-zeros → denominator is 0 → returns 0.
	a := []float64{0, 0, 0}
	b := []float64{0, 0, 0}
	got := cosineSimilarity(a, b)
	assert.Equal(t, float64(0), got)
}

// ─── Search sort swap ─────────────────────────────────────────────────────────

func TestExecute_Search_SortSwap(t *testing.T) {
	// Insert doc1 (low similarity) BEFORE doc2 (high similarity) so that the
	// sort loop must actually swap elements to rank doc2 above doc1.
	// Vectors: index-call-1 → orthogonal, index-call-2 → aligned, search → aligned.
	callCount := 0
	vectors := [][]float64{
		{0, 1, 0}, // doc1: orthogonal to search query → sim=0
		{1, 0, 0}, // doc2: identical to search query → sim=1
		{1, 0, 0}, // search query
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		v := vectors[callCount]
		if callCount < len(vectors)-1 {
			callCount++
		}
		resp := map[string]interface{}{"embeddings": [][]float64{v}}
		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)
	}))
	defer srv.Close()

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())
	dbPath := tmpDBPath(t, "sort-swap")

	// Index doc1 (sim=0) first, then doc2 (sim=1).
	for _, text := range []string{"doc1-low", "doc2-high"} {
		_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
			Model:   "m",
			BaseURL: srv.URL,
			Input:   text,
			DBPath:  dbPath,
		})
		require.NoError(t, err)
	}

	// Search: doc2 should rank first (sort must swap since doc1 was inserted first).
	result, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:     "m",
		BaseURL:   srv.URL,
		Input:     "query",
		DBPath:    dbPath,
		Operation: domain.EmbeddingOperationSearch,
		TopK:      2,
	})
	require.NoError(t, err)

	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	results, ok := m["results"].([]map[string]interface{})
	require.True(t, ok)
	require.Len(t, results, 2)
	assert.Equal(t, "doc2-high", results[0]["text"])
	assert.Equal(t, "doc1-low", results[1]["text"])
}

// ─── Search skips malformed embedding rows ────────────────────────────────────

func TestExecute_Search_SkipsMalformedEmbeddingRow(t *testing.T) {
	// Insert one valid row via the executor and one corrupt row directly into the
	// SQLite DB.  The search should skip the corrupt row and return only the
	// valid one (exercises the `continue` on line 259 of executor.go).
	vec := []float64{1, 0, 0}
	srv := ollamaEmbedServer(t, vec)
	defer srv.Close()

	dbPath := tmpDBPath(t, "malformed-row")

	ctx := makeCtx(t)
	exec := NewAdapterWithClient(nil, srv.Client())

	_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:   "m",
		BaseURL: srv.URL,
		Input:   "valid doc",
		DBPath:  dbPath,
	})
	require.NoError(t, err)

	// Insert a corrupt row directly (embedding column contains invalid JSON).
	db, dbErr := openVectorDB(dbPath, defaultCollection)
	require.NoError(t, dbErr)
	_, insErr := db.ExecContext(
		context.Background(),
		"INSERT INTO embeddings (text, embedding, metadata) VALUES (?, ?, ?)",
		"corrupt", "NOT_VALID_JSON", "{}",
	)
	require.NoError(t, insErr)
	require.NoError(t, db.Close())

	// Search: the malformed row must be skipped; only the valid row is returned.
	result, err := exec.Execute(ctx, &domain.EmbeddingConfig{
		Model:     "m",
		BaseURL:   srv.URL,
		Input:     "query",
		DBPath:    dbPath,
		Operation: domain.EmbeddingOperationSearch,
		TopK:      10,
	})
	require.NoError(t, err)

	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 1, m["count"], "malformed row should have been skipped")
}

// ─── operationSearch: embedding-fetch error path ─────────────────────────────

func TestExecute_Search_EmbeddingFetchError(t *testing.T) {
// Index a doc with a working server, then switch to a 500-returning server
// so the search embedding fetch fails (exercises lines 224-227).
vec := []float64{0.1, 0.2}
goodSrv := ollamaEmbedServer(t, vec)
defer goodSrv.Close()

ctx := makeCtx(t)
dbPath := tmpDBPath(t, "search-embed-fail")

// Index a document using the working server.
exec := NewAdapterWithClient(nil, goodSrv.Client())
_, err := exec.Execute(ctx, &domain.EmbeddingConfig{
Model:   "m",
BaseURL: goodSrv.URL,
Input:   "some doc",
DBPath:  dbPath,
})
require.NoError(t, err)

// Now search with a server that always returns HTTP 500.
badSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
w.WriteHeader(http.StatusInternalServerError)
}))
defer badSrv.Close()

exec2 := NewAdapterWithClient(nil, badSrv.Client())
_, err = exec2.Execute(ctx, &domain.EmbeddingConfig{
Model:     "m",
BaseURL:   badSrv.URL,
Input:     "query",
DBPath:    dbPath,
Operation: domain.EmbeddingOperationSearch,
})
require.Error(t, err)
assert.Contains(t, err.Error(), "get query embedding")
}

// ─── evaluateText: non-string evaluator result ────────────────────────────────

func TestEvaluateText_NonStringResult(t *testing.T) {
// Store an integer in memory; get('key') returns an int (not a string).
// evaluateText must fall through to fmt.Sprintf (line 689).
ctx := makeCtx(t)
require.NoError(t, ctx.Memory.Set("intkey", 42))

e := &Executor{logger: slog.Default(), client: http.DefaultClient}
// Single {{ expr }} returns the raw value (an int); evaluateText converts it.
got := e.evaluateText("{{ get('intkey') }}", ctx)
assert.Equal(t, "42", got)
}
