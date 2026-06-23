//go:build !js

package llm_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

// hfServerStub sets up a fake HuggingFace API server for the duration of fn.
// It patches the HF constants via a round-tripper override on the test's
// http.DefaultClient. Since hfRequest creates its own client, we need an httptest
// server and redirect calls via a custom RoundTripper injected in tests.
//
// Because hfRequest constructs its own *http.Client, we test the exported
// helper functions directly using a test server whose URL is injected by
// setting the HF_TOKEN env var (not applicable here); instead we test the
// downstream helpers that are mockable.

func TestHFGGUFFiles_FiltersToGGUF(t *testing.T) {
	files := []llm.HFFileEntry{
		{Filename: "model-Q4.gguf", Size: 1024},
		{Filename: "model-Q8.GGUF", Size: 2048},
		{Filename: "config.json", Size: 100},
		{Filename: "README.md", Size: 50},
	}
	got := llm.HFGGUFFiles(files)
	require.Len(t, got, 2)
	assert.Equal(t, "model-Q4.gguf", got[0].Filename)
	assert.Equal(t, "model-Q8.GGUF", got[1].Filename)
}

func TestHFGGUFFiles_EmptySlice(t *testing.T) {
	got := llm.HFGGUFFiles(nil)
	assert.Nil(t, got)
}

func TestHFDownloadURL(t *testing.T) {
	got := llm.HFDownloadURL("unsloth/Qwen2.5-VL-7B-Instruct-GGUF", "model-Q4_K_M.gguf")
	assert.Equal(t,
		"https://huggingface.co/unsloth/Qwen2.5-VL-7B-Instruct-GGUF/resolve/main/model-Q4_K_M.gguf",
		got,
	)
}

func TestHFSearchGGUF_APIResponse(t *testing.T) {
	results := []llm.HFModelResult{
		{
			ID:        "unsloth/Qwen2.5-VL-7B-Instruct-GGUF",
			Downloads: 50000,
			Likes:     300,
			Siblings: []llm.HFFileEntry{
				{Filename: "Qwen2.5-VL-7B-Instruct-Q4_K_M.gguf", Size: 4 * 1024 * 1024 * 1024},
				{Filename: "Qwen2.5-VL-7B-Instruct-Q2_K_M.gguf", Size: 2 * 1024 * 1024 * 1024},
			},
		},
		{ID: "bartowski/Llama-3.2-3B-Instruct-GGUF", Downloads: 30000, Likes: 200},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "filter=gguf")
		assert.Contains(t, r.URL.RawQuery, "search=qwen")
		assert.Contains(t, r.URL.RawQuery, "full=true") // siblings requested
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(results)
	}))
	defer srv.Close()
	got, err := llm.HFSearchGGUFWithBase(context.Background(), srv.URL+"/api/models", "qwen", 5)
	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, "unsloth/Qwen2.5-VL-7B-Instruct-GGUF", got[0].ID)
	assert.Equal(t, 50000, got[0].Downloads)
	gguf := llm.HFGGUFFiles(got[0].Siblings)
	require.Len(t, gguf, 2)
	assert.Equal(t, "Qwen2.5-VL-7B-Instruct-Q4_K_M.gguf", gguf[0].Filename)
}

func TestHFRepoFiles_APIResponse(t *testing.T) {
	info := llm.HFRepoInfo{
		ID: "unsloth/Qwen2.5-VL-7B-Instruct-GGUF",
		Siblings: []llm.HFFileEntry{
			{Filename: "Qwen2.5-VL-7B-Instruct-Q4_K_M.gguf", Size: 4 * 1024 * 1024 * 1024},
			{Filename: "config.json", Size: 1000},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.True(t, strings.HasSuffix(r.URL.Path, "/unsloth/Qwen2.5-VL-7B-Instruct-GGUF"))
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(info)
	}))
	defer srv.Close()
	got, err := llm.HFRepoFilesWithBase(
		context.Background(),
		srv.URL+"/api/models",
		"unsloth/Qwen2.5-VL-7B-Instruct-GGUF",
	)
	require.NoError(t, err)
	assert.Equal(t, "unsloth/Qwen2.5-VL-7B-Instruct-GGUF", got.ID)
	require.Len(t, got.Siblings, 2)
}

func TestHFSearchGGUF_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	_, err := llm.HFSearchGGUFWithBase(context.Background(), srv.URL+"/api/models", "llama", 5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}

func TestHFRegisterGGUFEntry_AddsToRegistry(t *testing.T) {
	// Use a temp home dir so we don't touch the real registry.
	t.Setenv("HOME", t.TempDir())
	llm.ReloadGGUFRegistry()
	t.Cleanup(func() { llm.ReloadGGUFRegistry() })

	entry := llm.GGUFEntry{
		Alias:    "test-hf-model",
		URL:      "https://huggingface.co/testorg/TestModel-GGUF/resolve/main/TestModel-Q4.gguf",
		Filename: "TestModel-Q4.gguf",
		Repo:     "testorg/TestModel-GGUF",
	}
	require.NoError(t, llm.HFRegisterGGUFEntry(entry))

	found := false
	for _, e := range llm.ListGGUFMappings() {
		if e.Alias == "test-hf-model" {
			found = true
			assert.Equal(t, "testorg/TestModel-GGUF", e.Repo)
		}
	}
	assert.True(t, found, "registered entry not found in registry")
}

func TestHFRegisterGGUFEntry_UpdatesExisting(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	llm.ReloadGGUFRegistry()
	t.Cleanup(func() { llm.ReloadGGUFRegistry() })

	e1 := llm.GGUFEntry{Alias: "update-test", URL: "http://old", Filename: "old.gguf", Repo: "org/old"}
	require.NoError(t, llm.HFRegisterGGUFEntry(e1))

	e2 := llm.GGUFEntry{Alias: "update-test", URL: "http://new", Filename: "new.gguf", Repo: "org/new"}
	require.NoError(t, llm.HFRegisterGGUFEntry(e2))

	var found llm.GGUFEntry
	for _, e := range llm.ListGGUFMappings() {
		if e.Alias == "update-test" {
			found = e
		}
	}
	assert.Equal(t, "org/new", found.Repo)
}

func TestHFSearchGGUF_FallbackWithoutFilter(t *testing.T) {
	fallbackResult := []llm.HFModelResult{
		{
			ID:        "cyankiwi/MyModel-GGUF",
			Downloads: 10,
			Likes:     1,
			Siblings: []llm.HFFileEntry{
				{Filename: "MyModel-Q4_K_M.gguf", Size: 1 << 30},
			},
		},
		{
			ID:       "cyankiwi/NoGGUF",
			Siblings: []llm.HFFileEntry{{Filename: "model.bin", Size: 100}},
		},
	}
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.RawQuery, "filter=gguf") {
			_ = json.NewEncoder(w).Encode([]llm.HFModelResult{})
			return
		}
		_ = json.NewEncoder(w).Encode(fallbackResult)
	}))
	defer srv.Close()

	got, err := llm.HFSearchGGUFWithBase(context.Background(), srv.URL+"/api/models", "cyankiwi", 20)
	require.NoError(t, err)
	assert.Equal(t, 2, callCount, "expected two API calls (tagged + untagged)")
	require.Len(t, got, 1, "only repos with .gguf siblings should be returned")
	assert.Equal(t, "cyankiwi/MyModel-GGUF", got[0].ID)
}

func TestHFSearchGGUF_FallbackHTTPError(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if strings.Contains(r.URL.RawQuery, "filter=gguf") {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode([]llm.HFModelResult{})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	got, err := llm.HFSearchGGUFWithBase(context.Background(), srv.URL+"/api/models", "cyankiwi", 20)
	require.NoError(t, err)
	assert.Empty(t, got)
}

// TestHFSearchGGUF_CancelledContext covers the HFSearchGGUF wrapper (calls WithBase).
func TestHFSearchGGUF_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := llm.HFSearchGGUF(ctx, "qwen", 5)
	assert.Error(t, err) // network call fails on cancelled context
}

// TestHFRepoFiles_CancelledContext covers the HFRepoFiles wrapper.
func TestHFRepoFiles_CancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := llm.HFRepoFiles(ctx, "owner/repo")
	assert.Error(t, err)
}

func TestHFDownloadGGUF_AlreadyExists(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	llm.ReloadGGUFRegistry()
	t.Cleanup(func() { llm.ReloadGGUFRegistry() })

	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	filename := "AlreadyThere-Q4.gguf"
	dest := dir + "/" + filename
	require.NoError(t, os.WriteFile(dest, []byte("existing"), 0o600))

	path, alias, err := llm.HFDownloadGGUF(context.Background(), "org/Repo", filename, nil)
	require.NoError(t, err)
	assert.Equal(t, dest, path)
	assert.Equal(t, "AlreadyThere-Q4", alias)
}
