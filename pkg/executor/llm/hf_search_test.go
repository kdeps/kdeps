package llm_test

import (
	"context"
	"os"
	"path/filepath"
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

func TestHFDownloadURL(t *testing.T) {
	got := llm.HFDownloadURL("unsloth/Qwen2.5-VL-7B-Instruct-GGUF", "model-Q4_K_M.gguf")
	assert.Equal(t,
		"https://huggingface.co/unsloth/Qwen2.5-VL-7B-Instruct-GGUF/resolve/main/model-Q4_K_M.gguf",
		got,
	)
}

func TestHFRegisterGGUFEntry_AddsToRegistry(t *testing.T) {
	// Use a temp home dir so we don't touch the real registry.
	t.Cleanup(llm.SetUserHomeDir(t.TempDir()))
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
	t.Cleanup(llm.SetUserHomeDir(t.TempDir()))
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

// TestHFSearchGGUF_CancelledContext covers the HFSearchGGUF wrapper (calls WithBase).
// TestHFRepoFiles_CancelledContext covers the HFRepoFiles wrapper.
func TestHFDownloadGGUF_AlreadyExists(t *testing.T) {
	t.Cleanup(llm.SetUserHomeDir(t.TempDir()))
	llm.ReloadGGUFRegistry()
	t.Cleanup(func() { llm.ReloadGGUFRegistry() })

	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	filename := "AlreadyThere-Q4.gguf"
	dest := filepath.Join(dir, filename)
	require.NoError(t, os.WriteFile(dest, []byte("existing"), 0o600))

	path, alias, err := llm.HFDownloadGGUF(context.Background(), "org/Repo", filename, nil)
	require.NoError(t, err)
	assert.Equal(t, dest, path)
	assert.Equal(t, "AlreadyThere-Q4", alias)
}
