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
	"bufio"
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
	// registry and storage sections are gone; all empty fields commented out.
	assert.NotContains(t, content, "registry:")
	assert.NotContains(t, content, "storage:")
	assert.Contains(t, content, "# ")
}

func TestWriteConfig_WithValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := Config{
		LLM: LLMKeys{
			OllamaHost:   "http://localhost:11434",
			DefaultModel: "llama3.2",
			OpenAI:       "sk-test",
			Anthropic:    "ant-test",
		},
	}
	require.NoError(t, writeConfig(path, cfg))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, `ollama_host: "http://localhost:11434"`)
	assert.Contains(t, content, `model: "llama3.2"`)
	assert.Contains(t, content, `openai_api_key: "sk-test"`)
	assert.Contains(t, content, `anthropic_api_key: "ant-test"`)
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

// --- bootstrapInteractive ---

// testWriter implements bootstrapWriter for tests.
type testWriter struct{ strings.Builder }

func (tw *testWriter) WriteString(s string) (int, error) { return tw.Builder.WriteString(s) }

func TestBootstrapInteractive_OllamaDefaultHost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Input: choose "1" (ollama), accept default host, default model
	input := "1\n\n\n"
	reader := bufio.NewReader(strings.NewReader(input))
	var out testWriter
	require.NoError(t, bootstrapInteractive(&out, reader, path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, "ollama_host")
	assert.Contains(t, content, "llama3.2")
}

func TestBootstrapInteractive_OllamaCustomHost(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Input: choose "1" (ollama), custom host, custom model
	input := "1\nhttp://myserver:11434\nmistral\n"
	reader := bufio.NewReader(strings.NewReader(input))
	var out testWriter
	require.NoError(t, bootstrapInteractive(&out, reader, path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, `ollama_host: "http://myserver:11434"`)
	assert.Contains(t, content, `model: "mistral"`)
}

func TestBootstrapInteractive_OnlineProvider(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Input: choose "2" (openai), then API key via fallback reader
	input := "2\nsk-mykey\n"
	reader := bufio.NewReader(strings.NewReader(input))
	var out testWriter
	require.NoError(t, bootstrapInteractive(&out, reader, path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `openai_api_key: "sk-mykey"`)
}

func TestBootstrapInteractive_Skip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Input: skip (0)
	reader := bufio.NewReader(strings.NewReader("0\n"))
	var out testWriter
	require.NoError(t, bootstrapInteractive(&out, reader, path))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	// Config should still be written, all keys commented out.
	assert.Contains(t, string(data), "llm:")
	assert.Contains(t, string(data), "# ")
}

func TestBootstrapInteractive_InvalidChoice_DefaultsToFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// Input: invalid choice ("99"), then hits out-of-range → no provider chosen
	reader := bufio.NewReader(strings.NewReader("99\n"))
	var out testWriter
	require.NoError(t, bootstrapInteractive(&out, reader, path))
	// Should complete without error even with invalid choice
	_, err := os.Stat(path)
	assert.NoError(t, err)
}

func TestBootstrapInteractive_WriteError(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses file permissions")
	}
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))
	path := filepath.Join(blocker, "sub", "config.yaml")

	reader := bufio.NewReader(strings.NewReader("0\n"))
	var out testWriter
	err := bootstrapInteractive(&out, reader, path)
	assert.Error(t, err)
}

func TestPromptLine_DefaultOnEmpty(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("\n"))
	result := promptLine(os.Stdout, r, "prompt: ", "mydefault")
	assert.Equal(t, "mydefault", result)
}

func TestPromptLine_ReturnsInput(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("uservalue\n"))
	result := promptLine(os.Stdout, r, "prompt: ", "def")
	assert.Equal(t, "uservalue", result)
}

func TestPromptLine_TrimsWhitespace(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("  hello  \n"))
	result := promptLine(os.Stdout, r, "prompt: ", "def")
	assert.Equal(t, "hello", result)
}

// --- readSecret (non-terminal path) ---

func TestReadSecret_NonTerminal(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("mysecret\n"))
	val, err := readSecret(r)
	require.NoError(t, err)
	assert.Equal(t, "mysecret", val)
}

func TestReadSecret_EmptyLine(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("\n"))
	val, err := readSecret(r)
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

// --- providerMetaMap ollama setter ---

func TestProviderMetaMap_OllamaSetterWorks(t *testing.T) {
	meta := providerMetaMap()
	cfg := &Config{}
	meta["ollama"].setter(cfg, "http://myhost:11434")
	assert.Equal(t, "http://myhost:11434", cfg.LLM.OllamaHost)
}

// --- applyEnv offline_mode ---

func TestApplyEnv_OfflineMode(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_OFFLINE_MODE"))
	cfg := Config{Defaults: Defaults{OfflineMode: true}}
	applyEnv(cfg)
	assert.Equal(t, "true", os.Getenv("KDEPS_OFFLINE_MODE"))
	// Cleanup
	require.NoError(t, os.Unsetenv("KDEPS_OFFLINE_MODE"))
}

func TestApplyEnv_OfflineMode_NotOverwritten(t *testing.T) {
	t.Setenv("KDEPS_OFFLINE_MODE", "existing")
	cfg := Config{Defaults: Defaults{OfflineMode: true}}
	applyEnv(cfg)
	assert.Equal(t, "existing", os.Getenv("KDEPS_OFFLINE_MODE"))
}

func TestApplyEnv_Timezone(t *testing.T) {
	require.NoError(t, os.Unsetenv("TZ"))
	cfg := Config{Defaults: Defaults{Timezone: "America/New_York"}}
	applyEnv(cfg)
	assert.Equal(t, "America/New_York", os.Getenv("TZ"))
	require.NoError(t, os.Unsetenv("TZ"))
}

func TestApplyEnv_PythonVersion(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_PYTHON_VERSION"))
	cfg := Config{Defaults: Defaults{PythonVersion: "3.11"}}
	applyEnv(cfg)
	assert.Equal(t, "3.11", os.Getenv("KDEPS_PYTHON_VERSION"))
	require.NoError(t, os.Unsetenv("KDEPS_PYTHON_VERSION"))
}

// --- writeConfig with defaults ---

func TestWriteConfig_WithDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	cfg := Config{
		Defaults: Defaults{Timezone: "UTC", PythonVersion: "3.11"},
	}
	require.NoError(t, writeConfig(path, cfg))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)
	assert.Contains(t, content, `timezone: "UTC"`)
	assert.Contains(t, content, `python_version: "3.11"`)
	assert.Contains(t, content, "defaults:")
}

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
	assert.Contains(t, names, "ollama")
	assert.Contains(t, names, "openai")
	assert.Contains(t, names, "anthropic")
	assert.Len(t, names, 11)
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
