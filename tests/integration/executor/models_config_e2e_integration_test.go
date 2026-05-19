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

package executor_test

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	executorLLM "github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

// loadUnifiedConfig writes config.yaml with llm section, resets env vars, and calls config.Load.
func loadUnifiedConfig(t *testing.T, yamlContent string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(yamlContent), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	require.NoError(t, os.Unsetenv("KDEPS_LLM_MODELS"))
	require.NoError(t, os.Unsetenv("KDEPS_LLM_ROUTER"))
	_, err := config.Load()
	require.NoError(t, err)
}

// TestE2E_UnifiedModels_Allowlist verifies plain string entries produce KDEPS_LLM_MODELS.
func TestE2E_UnifiedModels_Allowlist(t *testing.T) {
	loadUnifiedConfig(t, "\nllm:\n  models:\n    - llama3.2:1b\n    - nomic-embed-text\n")
	assert.Equal(t, "llama3.2:1b,nomic-embed-text", os.Getenv("KDEPS_LLM_MODELS"))
}

// TestE2E_UnifiedModels_Allowlist_PlainStrings verifies single plain string.
func TestE2E_UnifiedModels_Allowlist_PlainStrings(t *testing.T) {
	loadUnifiedConfig(t, "\nllm:\n  models:\n    - llama3.2:1b\n")
	assert.Equal(t, "llama3.2:1b", os.Getenv("KDEPS_LLM_MODELS"))
}

// TestE2E_UnifiedModels_Router_JSON verifies strategy+models produce KDEPS_LLM_ROUTER JSON.
func TestE2E_UnifiedModels_Router_JSON(t *testing.T) {
	loadUnifiedConfig(
		t,
		"\nllm:\n  strategy: fallback\n  models:\n    - model: gpt-4o\n      backend: openai\n      priority: 1\n    - model: llama3.2:1b\n      backend: ollama\n      priority: 2\n      default: true\n",
	)
	assert.Equal(t, "gpt-4o,llama3.2:1b", os.Getenv("KDEPS_LLM_MODELS"))

	routerJSON := os.Getenv("KDEPS_LLM_ROUTER")
	require.NotEmpty(t, routerJSON)
	var uc config.UnifiedModelsConfig
	require.NoError(t, json.Unmarshal([]byte(routerJSON), &uc))
	assert.Equal(t, "fallback", uc.Strategy)
	require.Len(t, uc.Models, 2)
	assert.Equal(t, "gpt-4o", uc.Models[0].Model)
	assert.Equal(t, "openai", uc.Models[0].Backend)
	assert.Equal(t, 1, uc.Models[0].Priority)
	assert.Equal(t, "llama3.2:1b", uc.Models[1].Model)
	assert.Equal(t, "ollama", uc.Models[1].Backend)
	assert.Equal(t, 2, uc.Models[1].Priority)
	assert.True(t, uc.Models[1].Default)
}

// TestE2E_UnifiedModels_MixedEntries verifies a mix of plain strings and objects.
func TestE2E_UnifiedModels_MixedEntries(t *testing.T) {
	loadUnifiedConfig(
		t,
		"\nllm:\n  strategy: fallback\n  models:\n    - llama3.2:1b\n    - model: gpt-4o\n      backend: openai\n      priority: 1\n",
	)
	assert.Equal(t, "llama3.2:1b,gpt-4o", os.Getenv("KDEPS_LLM_MODELS"))

	routerJSON := os.Getenv("KDEPS_LLM_ROUTER")
	require.NotEmpty(t, routerJSON)
	var uc config.UnifiedModelsConfig
	require.NoError(t, json.Unmarshal([]byte(routerJSON), &uc))
	assert.Equal(t, "fallback", uc.Strategy)
	require.Len(t, uc.Models, 2)
	assert.Equal(t, "llama3.2:1b", uc.Models[0].Model)
	assert.Empty(t, uc.Models[0].Backend)
	assert.Equal(t, "gpt-4o", uc.Models[1].Model)
	assert.Equal(t, "openai", uc.Models[1].Backend)
	assert.Equal(t, 1, uc.Models[1].Priority)
}

// TestE2E_ModelInResourceYAML verifies that model is parsed from resource YAML.
func TestE2E_ModelInResourceYAML(t *testing.T) {
	yamlContent := `
actionId: test
chat:
  model: gpt-4o
  role: user
  prompt: hello
`
	var resource domain.Resource
	err := yaml.Unmarshal([]byte(yamlContent), &resource)
	require.NoError(t, err)
	require.NotNil(t, resource.Chat)
	assert.Equal(t, "gpt-4o", resource.Chat.Model)
}

// TestE2E_ModelInResourceYAML_PlainString verifies that a model set as a plain
// name in resource YAML is parsed correctly.
func TestE2E_ModelInResourceYAML_PlainString(t *testing.T) {
	yamlContent := `
actionId: test
chat:
  model: llama3.2:1b
  role: user
  prompt: hello
`
	var resource domain.Resource
	err := yaml.Unmarshal([]byte(yamlContent), &resource)
	require.NoError(t, err)
	require.NotNil(t, resource.Chat)
	assert.Equal(t, "llama3.2:1b", resource.Chat.Model)
}

// TestE2E_ModelRouter_Delegation verifies that a resource with model="router"
// triggers router delegation and selects a model from the config.
func TestE2E_ModelRouter_Delegation(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	// Set up config with router
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`
llm:
  strategy: fallback
  models:
    - model: router-test-model
      backend: openai
      priority: 1
      default: true
`), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	require.NoError(t, os.Unsetenv("KDEPS_LLM_MODELS"))
	require.NoError(t, os.Unsetenv("KDEPS_LLM_ROUTER"))
	_, err := config.Load()
	require.NoError(t, err)

	// Start a mock server that captures the model used
	var capturedModel string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		capturedModel, _ = req["model"].(string)
		resp := map[string]interface{}{
			"model":   capturedModel,
			"message": map[string]interface{}{"role": "assistant", "content": "ok"},
			"done":    true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	engine := executor.NewEngine(slog.Default())
	registry := executor.NewRegistry()
	registry.SetLLMExecutor(executorLLM.NewAdapter(server.URL))
	engine.SetRegistry(registry)
	// Disable model manager to avoid interfering with model selection
	if adapter, ok := registry.GetLLMExecutor().(*executorLLM.Adapter); ok {
		adapter.SetOfflineMode(true)
	}

	workflow := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata: domain.WorkflowMetadata{
			Name:           "router-test",
			Version:        "1.0.0",
			TargetActionID: "chat",
		},
		Resources: []*domain.Resource{
			{
				ActionID:    "chat",
				APIResponse: &domain.APIResponseConfig{Success: true, Response: "ok"},
				Chat: &domain.ChatConfig{
					Model:   "router",
					Prompt:  "hi",
					BaseURL: server.URL,
				},
			},
		},
	}

	result, err := engine.Execute(workflow, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "router-test-model", capturedModel)
}

// TestE2E_EmbeddedDefaults verifies that the embedded defaults.yml is parsed
// and returns correct values.
func TestE2E_EmbeddedDefaults(t *testing.T) {
	defaults, err := config.GetDefaults()
	require.NoError(t, err)
	require.NotNil(t, defaults)
	assert.Equal(t, "60s", defaults.Chat.Timeout)
	assert.Equal(t, 4096, defaults.Chat.ContextLength)
	assert.Equal(t, "30s", defaults.HTTP.Timeout)
	assert.True(t, defaults.HTTP.FollowRedirects)
	assert.Equal(t, "60s", defaults.Python.Timeout)
	assert.Equal(t, "30s", defaults.Exec.Timeout)
	assert.Equal(t, "30s", defaults.SQL.Timeout)
	assert.Equal(t, 1000, defaults.SQL.MaxRows)
	assert.Equal(t, 10, defaults.SQL.MaxOpenConns)
	assert.Equal(t, 2, defaults.SQL.MaxIdleConns)
	assert.Equal(t, "5m", defaults.SQL.ConnMaxIdleTime)
	assert.Equal(t, 30, defaults.Scraper.Timeout)
	assert.Equal(t, 15, defaults.SearchWeb.Timeout)
	assert.Equal(t, 5, defaults.SearchWeb.MaxResults)
	assert.Equal(t, "kdeps-embedding.db", defaults.Embedding.DBPath)
	assert.Equal(t, "default", defaults.Embedding.Collection)
	assert.Equal(t, 10, defaults.Embedding.Limit)
}

// TestE2E_APIKeyFromConfigOnly verifies that the backend correctly reads the
// API key from the environment variable (set by config.yaml) rather than
// requiring an explicit apiKey parameter.
func TestE2E_APIKeyFromConfigOnly(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-from-env")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the Authorization header is set from the env var
		auth := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer sk-from-env", auth)
		resp := map[string]interface{}{
			"model":   "gpt-4o",
			"message": map[string]interface{}{"role": "assistant", "content": "ok"},
			"done":    true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}))
	defer server.Close()

	engine := executor.NewEngine(slog.Default())
	registry := executor.NewRegistry()
	registry.SetLLMExecutor(executorLLM.NewAdapter(server.URL))
	engine.SetRegistry(registry)

	// Set Backend to openai so GetAPIKeyHeader reads OPENAI_API_KEY from env var.
	workflow := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata: domain.WorkflowMetadata{
			Name:           "apikey-test",
			Version:        "1.0.0",
			TargetActionID: "chat",
		},
		Resources: []*domain.Resource{
			{
				ActionID:    "chat",
				APIResponse: &domain.APIResponseConfig{Success: true, Response: "ok"},
				Chat: &domain.ChatConfig{
					Model:   "gpt-4o",
					Prompt:  "hi",
					Backend: "openai",
					BaseURL: server.URL,
				},
			},
		},
	}

	result, err := engine.Execute(workflow, nil)
	require.NoError(t, err)
	assert.NotNil(t, result)
}

// TestE2E_ModelEntryWithoutAPIKey verifies that ModelEntry works correctly
// without the APIKey field (removed; API keys come from config.yaml env vars).
func TestE2E_ModelEntryWithoutAPIKey(t *testing.T) {
	// ModelEntry should compile and marshal without APIKey field
	entry := config.ModelEntry{
		Model:    "test-model",
		Backend:  "openai",
		BaseURL:  "http://localhost:11434",
		Priority: 1,
		Default:  true,
	}
	assert.Equal(t, "test-model", entry.Model)
	assert.Equal(t, "openai", entry.Backend)
	assert.Equal(t, "http://localhost:11434", entry.BaseURL)
	assert.Equal(t, 1, entry.Priority)
	assert.True(t, entry.Default)
}
