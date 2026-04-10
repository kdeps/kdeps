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

// Internal tests for unexported helpers in bootstrap.go and config.go.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- writeConfig ---

func TestWriteConfig_AllEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, writeConfig(path, Config{}))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "llm:")
	assert.Contains(t, content, "registry:")
	assert.Contains(t, content, "storage:")
	// All empty fields should be commented out.
	assert.Contains(t, content, "# ")
}

func TestWriteConfig_WithValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := Config{
		LLM:      LLMKeys{OpenAI: "sk-test", Anthropic: "ant-test"},
		Registry: RegistryConfig{URL: "https://r.example.com", Token: "tok"},
		Storage:  StorageConfig{AgentsDir: "/agents", ComponentsDir: "/comps"},
	}
	require.NoError(t, writeConfig(path, cfg))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, `openai_api_key: "sk-test"`)
	assert.Contains(t, content, `anthropic_api_key: "ant-test"`)
	assert.Contains(t, content, `url: "https://r.example.com"`)
	assert.Contains(t, content, `token: "tok"`)
	assert.Contains(t, content, `agents_dir: "/agents"`)
	assert.Contains(t, content, `components_dir: "/comps"`)
}

func TestWriteConfig_MkdirError(t *testing.T) {
	// Use a path where the parent is an existing file (cannot mkdir).
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))
	path := filepath.Join(blocker, "sub", "config.yaml")
	err := writeConfig(path, Config{})
	assert.Error(t, err)
}

func TestWriteConfig_QuotesInValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := Config{LLM: LLMKeys{OpenAI: `key"with"quotes`}}
	require.NoError(t, writeConfig(path, cfg))
	data, _ := os.ReadFile(path)
	assert.Contains(t, string(data), `key\"with\"quotes`)
}

// --- appendField ---

func TestAppendField_Empty(t *testing.T) {
	var lines []string
	appendField(&lines, "  mykey", "")
	assert.Equal(t, []string{`#   mykey: ""`}, lines)
}

func TestAppendField_NonEmpty(t *testing.T) {
	var lines []string
	appendField(&lines, "  mykey", "myval")
	assert.Equal(t, []string{`  mykey: "myval"`}, lines)
}

// --- yamlQuote ---

func TestYamlQuote_Plain(t *testing.T) {
	assert.Equal(t, `"hello"`, yamlQuote("hello"))
}

func TestYamlQuote_WithQuotes(t *testing.T) {
	assert.Equal(t, `"he said \"hi\""`, yamlQuote(`he said "hi"`))
}

func TestYamlQuote_Empty(t *testing.T) {
	assert.Equal(t, `""`, yamlQuote(""))
}

// --- dirOf ---

func TestDirOf_WithSeparator(t *testing.T) {
	assert.Equal(t, "/home/user/.kdeps", dirOf("/home/user/.kdeps/config.yaml"))
}

func TestDirOf_NoSeparator(t *testing.T) {
	assert.Equal(t, ".", dirOf("config.yaml"))
}

func TestDirOf_Root(t *testing.T) {
	assert.Equal(t, "", dirOf("/config.yaml"))
}

// --- providerNames ---

func TestProviderNames_NonEmpty(t *testing.T) {
	names := providerNames()
	assert.NotEmpty(t, names)
	assert.Contains(t, names, "openai")
	assert.Contains(t, names, "anthropic")
	assert.Len(t, names, 10)
}

// --- providerMetaMap ---

func TestProviderMetaMap_HasAllProviders(t *testing.T) {
	meta := providerMetaMap()
	for _, name := range providerNames() {
		assert.Contains(t, meta, name, "missing provider %s", name)
	}
}

func TestProviderMetaMap_SetterWorks(t *testing.T) {
	meta := providerMetaMap()
	cfg := &Config{}
	meta["openai"].setter(cfg, "sk-xyz")
	assert.Equal(t, "sk-xyz", cfg.LLM.OpenAI)

	meta["anthropic"].setter(cfg, "ant-xyz")
	assert.Equal(t, "ant-xyz", cfg.LLM.Anthropic)

	meta["google"].setter(cfg, "ggl-xyz")
	assert.Equal(t, "ggl-xyz", cfg.LLM.Google)

	meta["groq"].setter(cfg, "grq-xyz")
	assert.Equal(t, "grq-xyz", cfg.LLM.Groq)

	meta["deepseek"].setter(cfg, "ds-xyz")
	assert.Equal(t, "ds-xyz", cfg.LLM.DeepSeek)

	meta["openrouter"].setter(cfg, "or-xyz")
	assert.Equal(t, "or-xyz", cfg.LLM.OpenRouter)

	meta["cohere"].setter(cfg, "co-xyz")
	assert.Equal(t, "co-xyz", cfg.LLM.Cohere)

	meta["mistral"].setter(cfg, "ms-xyz")
	assert.Equal(t, "ms-xyz", cfg.LLM.Mistral)

	meta["together"].setter(cfg, "tg-xyz")
	assert.Equal(t, "tg-xyz", cfg.LLM.Together)

	meta["perplexity"].setter(cfg, "pp-xyz")
	assert.Equal(t, "pp-xyz", cfg.LLM.Perplexity)
}

// --- Bootstrap via writeConfig path (integration through public API) ---

func TestBootstrap_WritesAllProviders(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	// Non-interactive env → Scaffold() is called.
	require.NoError(t, Bootstrap(os.Stdout))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	// Scaffold template must mention all providers.
	for _, p := range []string{
		"openai_api_key", "anthropic_api_key", "google_api_key",
		"groq_api_key", "deepseek_api_key", "openrouter_api_key",
	} {
		assert.True(t, strings.Contains(content, p), "missing %s in template", p)
	}
}

// --- Path ---

func TestPath_EnvOverride(t *testing.T) {
	t.Setenv("KDEPS_CONFIG_PATH", "/tmp/myconfig.yaml")
	p, err := Path()
	require.NoError(t, err)
	assert.Equal(t, "/tmp/myconfig.yaml", p)
}

func TestPath_Default(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_CONFIG_PATH"))
	p, err := Path()
	require.NoError(t, err)
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".kdeps", "config.yaml"), p)
}
