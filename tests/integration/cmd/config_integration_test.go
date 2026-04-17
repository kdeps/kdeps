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

// Package cmd_test - integration tests for the config package and kdeps edit command.
package cmd_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

// TestConfig_Integration_ScaffoldAndLoad verifies that Scaffold() writes a
// valid YAML file that Load() can parse, and that no registry/storage keys are
// present (they were removed in v2.0).
func TestConfig_Integration_ScaffoldAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	require.NoError(t, config.Scaffold())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	// Must contain llm and defaults sections.
	assert.Contains(t, content, "llm:")
	assert.Contains(t, content, "defaults:")
	assert.Contains(t, content, "ollama_host")
	assert.Contains(t, content, "model:")

	// Must NOT contain removed sections.
	assert.NotContains(t, content, "registry:")
	assert.NotContains(t, content, "storage:")
	assert.NotContains(t, content, "agents_dir")
	assert.NotContains(t, content, "components_dir")

	// Load must parse without error.
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
}

// TestConfig_Integration_DefaultsToEnvVars verifies that values in the
// defaults: section of config.yaml are propagated to the expected env vars.
func TestConfig_Integration_DefaultsToEnvVars(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	content := `
llm:
  model: llama3.2
  ollama_host: http://localhost:11434
defaults:
  timezone: Pacific/Auckland
  python_version: "3.10"
  offline_mode: true
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	for _, k := range []string{"TZ", "KDEPS_PYTHON_VERSION", "KDEPS_OFFLINE_MODE",
		"KDEPS_DEFAULT_MODEL", "OLLAMA_HOST"} {
		require.NoError(t, os.Unsetenv(k))
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "llama3.2", cfg.LLM.DefaultModel)
	assert.Equal(t, "http://localhost:11434", cfg.LLM.OllamaHost)
	assert.Equal(t, "Pacific/Auckland", cfg.Defaults.Timezone)
	assert.Equal(t, "3.10", cfg.Defaults.PythonVersion)
	assert.True(t, cfg.Defaults.OfflineMode)

	assert.Equal(t, "Pacific/Auckland", os.Getenv("TZ"))
	assert.Equal(t, "3.10", os.Getenv("KDEPS_PYTHON_VERSION"))
	assert.Equal(t, "true", os.Getenv("KDEPS_OFFLINE_MODE"))
	assert.Equal(t, "llama3.2", os.Getenv("KDEPS_DEFAULT_MODEL"))
	assert.Equal(t, "http://localhost:11434", os.Getenv("OLLAMA_HOST"))
}

// TestConfig_Integration_LLMKeys verifies all 10 online provider API keys
// plus the two Ollama fields round-trip correctly.
func TestConfig_Integration_LLMKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	content := `
llm:
  ollama_host: http://myserver:11434
  model: qwen2.5:7b
  openai_api_key: sk-openai
  anthropic_api_key: ant-1
  google_api_key: ggl-1
  cohere_api_key: co-1
  mistral_api_key: ms-1
  together_api_key: tg-1
  perplexity_api_key: pp-1
  groq_api_key: grq-1
  deepseek_api_key: ds-1
  openrouter_api_key: or-1
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	for _, k := range []string{
		"OLLAMA_HOST", "KDEPS_DEFAULT_MODEL",
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "GOOGLE_API_KEY", "COHERE_API_KEY",
		"MISTRAL_API_KEY", "TOGETHER_API_KEY", "PERPLEXITY_API_KEY", "GROQ_API_KEY",
		"DEEPSEEK_API_KEY", "OPENROUTER_API_KEY",
	} {
		require.NoError(t, os.Unsetenv(k))
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "http://myserver:11434", cfg.LLM.OllamaHost)
	assert.Equal(t, "qwen2.5:7b", cfg.LLM.DefaultModel)
	assert.Equal(t, "sk-openai", cfg.LLM.OpenAI)
	assert.Equal(t, "ant-1", cfg.LLM.Anthropic)
	assert.Equal(t, "ggl-1", cfg.LLM.Google)
	assert.Equal(t, "co-1", cfg.LLM.Cohere)
	assert.Equal(t, "ms-1", cfg.LLM.Mistral)
	assert.Equal(t, "tg-1", cfg.LLM.Together)
	assert.Equal(t, "pp-1", cfg.LLM.Perplexity)
	assert.Equal(t, "grq-1", cfg.LLM.Groq)
	assert.Equal(t, "ds-1", cfg.LLM.DeepSeek)
	assert.Equal(t, "or-1", cfg.LLM.OpenRouter)

	// Verify env vars set.
	assert.Equal(t, "sk-openai", os.Getenv("OPENAI_API_KEY"))
	assert.Equal(t, "qwen2.5:7b", os.Getenv("KDEPS_DEFAULT_MODEL"))
	assert.Equal(t, "http://myserver:11434", os.Getenv("OLLAMA_HOST"))
}

// TestConfig_Integration_ModelsDir verifies that models_dir in config.yaml is
// propagated to KDEPS_MODELS_DIR.
func TestConfig_Integration_ModelsDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)
	require.NoError(t, os.Unsetenv("KDEPS_MODELS_DIR"))

	custom := filepath.Join(dir, "mymodels")
	content := "llm:\n  models_dir: " + custom + "\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, custom, cfg.LLM.ModelsDir)
	assert.Equal(t, custom, os.Getenv("KDEPS_MODELS_DIR"))
}

// TestConfig_Integration_ScaffoldContainsModelsDir checks that the scaffold
// template includes the models_dir field.
func TestConfig_Integration_ScaffoldContainsModelsDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	require.NoError(t, config.Scaffold())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "models_dir")
}

// TestConfig_Integration_ScaffoldContents checks the actual commented template
// output has the right structure - no registry/storage, ollama at the top.
func TestConfig_Integration_ScaffoldContents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	require.NoError(t, config.Scaffold())

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	lines := strings.Split(content, "\n")
	var llmIdx, defaultsIdx int
	for i, l := range lines {
		if strings.TrimSpace(l) == "llm:" {
			llmIdx = i
		}
		if strings.TrimSpace(l) == "defaults:" {
			defaultsIdx = i
		}
	}
	assert.Greater(t, llmIdx, 0, "llm: section not found")
	assert.Greater(t, defaultsIdx, llmIdx, "defaults: section must come after llm:")
	assert.NotContains(t, content, "registry:")
	assert.NotContains(t, content, "storage:")
}
