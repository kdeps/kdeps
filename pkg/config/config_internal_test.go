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

// Internal tests for unexported helpers in config.go and defaults.go.
package config

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// --- hasRoutingMeta ---

func TestHasRoutingMeta_Empty(t *testing.T) {
	assert.False(t, hasRoutingMeta(ModelList{}))
}

func TestHasRoutingMeta_NoMeta(t *testing.T) {
	assert.False(t, hasRoutingMeta(ModelList{{Model: "gpt-4"}}))
}

func TestHasRoutingMeta_WithBackend(t *testing.T) {
	assert.True(t, hasRoutingMeta(ModelList{{Model: "gpt-4", Backend: "openai"}}))
}

func TestHasRoutingMeta_WithBaseURL(t *testing.T) {
	assert.True(t, hasRoutingMeta(ModelList{{Model: "gpt-4", BaseURL: "http://custom:8080"}}))
}

func TestHasRoutingMeta_Mixed(t *testing.T) {
	assert.True(t, hasRoutingMeta(ModelList{
		{Model: "llama3.2"},
		{Model: "gpt-4", Backend: "openai"},
	}))
}

// --- applyRouterEnv ---

func TestApplyRouterEnv_StrategySet(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_LLM_ROUTER"))
	applyRouterEnv(LLMKeys{
		Strategy: "round_robin",
		Models:   ModelList{{Model: "gpt-4"}, {Model: "claude-sonnet"}},
	})
	val := os.Getenv("KDEPS_LLM_ROUTER")
	assert.Contains(t, val, `"strategy":"round_robin"`)
	assert.Contains(t, val, `"models"`)
	require.NoError(t, os.Unsetenv("KDEPS_LLM_ROUTER"))
}

func TestApplyRouterEnv_RoutingMeta(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_LLM_ROUTER"))
	applyRouterEnv(LLMKeys{
		Models: ModelList{{Model: "gpt-4", Backend: "openai"}},
	})
	val := os.Getenv("KDEPS_LLM_ROUTER")
	assert.Contains(t, val, `"backend":"openai"`)
	require.NoError(t, os.Unsetenv("KDEPS_LLM_ROUTER"))
}

func TestApplyRouterEnv_PlainModels(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_LLM_ROUTER"))
	applyRouterEnv(LLMKeys{
		Models: ModelList{{Model: "llama3.2"}},
	})
	assert.Empty(t, os.Getenv("KDEPS_LLM_ROUTER"))
}

func TestApplyRouterEnv_Empty(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_LLM_ROUTER"))
	applyRouterEnv(LLMKeys{})
	assert.Empty(t, os.Getenv("KDEPS_LLM_ROUTER"))
}

func TestApplyRouterEnv_EnvAlreadySet(t *testing.T) {
	t.Setenv("KDEPS_LLM_ROUTER", "already-set")
	applyRouterEnv(LLMKeys{Strategy: "round_robin", Models: ModelList{{Model: "gpt-4"}}})
	assert.Equal(t, "already-set", os.Getenv("KDEPS_LLM_ROUTER"))
}

// --- knownConfigEnvVars ---

func TestKnownConfigEnvVars_NotEmpty(t *testing.T) {
	vars := knownConfigEnvVars()
	assert.NotEmpty(t, vars)
	assert.Contains(t, vars, "TZ")
	assert.Contains(t, vars, "KDEPS_LLM_ROUTER")
	assert.Contains(t, vars, "OPENAI_API_KEY")
	assert.Contains(t, vars, "OLLAMA_HOST")
	assert.Contains(t, vars, "KDEPS_CHAT_TIMEOUT")
	assert.Contains(t, vars, "KDEPS_API_AUTH_TOKEN")
}

// --- GetDefaults ---

func TestGetDefaults_ReturnsValues(t *testing.T) {
	d, err := GetDefaults()
	require.NoError(t, err)
	require.NotNil(t, d)
	assert.Equal(t, "60s", d.Chat.Timeout)
	assert.Equal(t, "30s", d.HTTP.Timeout)
	assert.Equal(t, "60s", d.Python.Timeout)
	assert.Equal(t, "30s", d.Exec.Timeout)
	assert.Equal(t, "30s", d.SQL.Timeout)
	assert.Equal(t, 4096, d.Chat.ContextLength)
	assert.Equal(t, 1000, d.SQL.MaxRows)
}

func TestGetDefaults_SyncOnce(t *testing.T) {
	d1, err1 := GetDefaults()
	d2, err2 := GetDefaults()
	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Same(t, d1, d2, "GetDefaults should return the same pointer via sync.Once")
}

// --- TimeoutDuration ---

func TestChatTimeoutDuration_Valid(t *testing.T) {
	d := (&ChatExecutorDefaults{Timeout: "30s"}).TimeoutDuration()
	assert.Equal(t, 30*time.Second, d)
}

func TestChatTimeoutDuration_Invalid(t *testing.T) {
	d := (&ChatExecutorDefaults{Timeout: "bad"}).TimeoutDuration()
	assert.Equal(t, 60*time.Second, d)
}

func TestChatTimeoutDuration_Empty(t *testing.T) {
	d := (&ChatExecutorDefaults{}).TimeoutDuration()
	assert.Equal(t, 60*time.Second, d)
}

func TestHTTPTimeoutDuration_Valid(t *testing.T) {
	d := (&HTTPExecutorDefaults{Timeout: "45s"}).TimeoutDuration()
	assert.Equal(t, 45*time.Second, d)
}

func TestHTTPTimeoutDuration_Invalid(t *testing.T) {
	d := (&HTTPExecutorDefaults{Timeout: "bad"}).TimeoutDuration()
	assert.Equal(t, 30*time.Second, d)
}

func TestHTTPTimeoutDuration_Empty(t *testing.T) {
	d := (&HTTPExecutorDefaults{}).TimeoutDuration()
	assert.Equal(t, 30*time.Second, d)
}

func TestPythonTimeoutDuration_Valid(t *testing.T) {
	d := (&PythonExecutorDefaults{Timeout: "120s"}).TimeoutDuration()
	assert.Equal(t, 120*time.Second, d)
}

func TestPythonTimeoutDuration_Invalid(t *testing.T) {
	d := (&PythonExecutorDefaults{Timeout: "bad"}).TimeoutDuration()
	assert.Equal(t, 60*time.Second, d)
}

func TestPythonTimeoutDuration_Empty(t *testing.T) {
	d := (&PythonExecutorDefaults{}).TimeoutDuration()
	assert.Equal(t, 60*time.Second, d)
}

func TestExecTimeoutDuration_Valid(t *testing.T) {
	d := (&ExecExecutorDefaults{Timeout: "15s"}).TimeoutDuration()
	assert.Equal(t, 15*time.Second, d)
}

func TestExecTimeoutDuration_Invalid(t *testing.T) {
	d := (&ExecExecutorDefaults{Timeout: "bad"}).TimeoutDuration()
	assert.Equal(t, 30*time.Second, d)
}

func TestExecTimeoutDuration_Empty(t *testing.T) {
	d := (&ExecExecutorDefaults{}).TimeoutDuration()
	assert.Equal(t, 30*time.Second, d)
}

func TestSQLTimeoutDuration_Valid(t *testing.T) {
	d := (&SQLExecutorDefaults{Timeout: "20s"}).TimeoutDuration()
	assert.Equal(t, 20*time.Second, d)
}

func TestSQLTimeoutDuration_Invalid(t *testing.T) {
	d := (&SQLExecutorDefaults{Timeout: "bad"}).TimeoutDuration()
	assert.Equal(t, 30*time.Second, d)
}

func TestSQLTimeoutDuration_Empty(t *testing.T) {
	d := (&SQLExecutorDefaults{}).TimeoutDuration()
	assert.Equal(t, 30*time.Second, d)
}

func TestConnMaxIdleTimeDuration_Valid(t *testing.T) {
	d := (&SQLExecutorDefaults{ConnMaxIdleTime: "10m"}).ConnMaxIdleTimeDuration()
	assert.Equal(t, 10*time.Minute, d)
}

func TestConnMaxIdleTimeDuration_Invalid(t *testing.T) {
	d := (&SQLExecutorDefaults{ConnMaxIdleTime: "bad"}).ConnMaxIdleTimeDuration()
	assert.Equal(t, 5*time.Minute, d)
}

func TestConnMaxIdleTimeDuration_Empty(t *testing.T) {
	d := (&SQLExecutorDefaults{}).ConnMaxIdleTimeDuration()
	assert.Equal(t, 5*time.Minute, d)
}

// --- setIfUnset ---

func TestSetIfUnset_SetsEnv(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_TEST_SET"))
	setIfUnset("KDEPS_TEST_SET", "value")
	assert.Equal(t, "value", os.Getenv("KDEPS_TEST_SET"))
	require.NoError(t, os.Unsetenv("KDEPS_TEST_SET"))
}

func TestSetIfUnset_DoesNotOverwrite(t *testing.T) {
	t.Setenv("KDEPS_TEST_NOCLOBBER", "existing")
	setIfUnset("KDEPS_TEST_NOCLOBBER", "new")
	assert.Equal(t, "existing", os.Getenv("KDEPS_TEST_NOCLOBBER"))
}

func TestSetIfUnset_EmptyValue(t *testing.T) {
	t.Setenv("KDEPS_TEST_EMPTY", "keep-me")
	setIfUnset("KDEPS_TEST_EMPTY", "")
	assert.Equal(t, "keep-me", os.Getenv("KDEPS_TEST_EMPTY"))
}

// --- mergeConfig ---

func ptr[T any](v T) *T { return &v }

func TestMergeConfig_LLMFields(t *testing.T) {
	dst := &Config{}
	src := &Config{
		LLM: LLMKeys{
			OllamaHost: "http://ollama:11434",
			Backend:    "openai",
			BaseURL:    "http://proxy:8080",
			Strategy:   "round_robin",
			Models:     ModelList{{Model: "gpt-4"}, {Model: "claude"}},
			ModelsDir:  "/custom/models",
			OpenAI:     "sk-openai",
			Anthropic:  "ant-key",
			Google:     "ggl-key",
			Cohere:     "co-key",
			Mistral:    "ms-key",
			Together:   "tg-key",
			Perplexity: "pp-key",
			Groq:       "grq-key",
			DeepSeek:   "ds-key",
			OpenRouter: "or-key",
		},
	}
	mergeConfig(dst, src)
	assert.Equal(t, "http://ollama:11434", dst.LLM.OllamaHost)
	assert.Equal(t, "openai", dst.LLM.Backend)
	assert.Equal(t, "http://proxy:8080", dst.LLM.BaseURL)
	assert.Equal(t, "round_robin", dst.LLM.Strategy)
	assert.Len(t, dst.LLM.Models, 2)
	assert.Equal(t, "/custom/models", dst.LLM.ModelsDir)
	assert.Equal(t, "sk-openai", dst.LLM.OpenAI)
	assert.Equal(t, "ant-key", dst.LLM.Anthropic)
	assert.Equal(t, "ggl-key", dst.LLM.Google)
	assert.Equal(t, "co-key", dst.LLM.Cohere)
	assert.Equal(t, "ms-key", dst.LLM.Mistral)
	assert.Equal(t, "tg-key", dst.LLM.Together)
	assert.Equal(t, "pp-key", dst.LLM.Perplexity)
	assert.Equal(t, "grq-key", dst.LLM.Groq)
	assert.Equal(t, "ds-key", dst.LLM.DeepSeek)
	assert.Equal(t, "or-key", dst.LLM.OpenRouter)
}

func TestMergeConfig_Defaults(t *testing.T) {
	dst := &Config{}
	src := &Config{
		Defaults: Defaults{
			Timezone:      "America/New_York",
			PythonVersion: "3.12",
			OfflineMode:   true,
		},
	}
	mergeConfig(dst, src)
	assert.Equal(t, "America/New_York", dst.Defaults.Timezone)
	assert.Equal(t, "3.12", dst.Defaults.PythonVersion)
	assert.True(t, dst.Defaults.OfflineMode)
}

func TestMergeConfig_ChatDefaults(t *testing.T) {
	dst := &Config{}
	src := &Config{
		ResourceDefaults: ResourceDefaults{
			Chat: ChatDefaults{
				Timeout:          "90s",
				ContextLength:    8192,
				Streaming:        true,
				Temperature:      ptr(0.7),
				MaxTokens:        ptr(4096),
				TopP:             ptr(0.9),
				FrequencyPenalty: ptr(0.1),
				PresencePenalty:  ptr(0.2),
			},
		},
	}
	mergeConfig(dst, src)
	assert.Equal(t, "90s", dst.ResourceDefaults.Chat.Timeout)
	assert.Equal(t, 8192, dst.ResourceDefaults.Chat.ContextLength)
	assert.True(t, dst.ResourceDefaults.Chat.Streaming)
	assert.Equal(t, 0.7, *dst.ResourceDefaults.Chat.Temperature)
	assert.Equal(t, 4096, *dst.ResourceDefaults.Chat.MaxTokens)
	assert.Equal(t, 0.9, *dst.ResourceDefaults.Chat.TopP)
	assert.Equal(t, 0.1, *dst.ResourceDefaults.Chat.FrequencyPenalty)
	assert.Equal(t, 0.2, *dst.ResourceDefaults.Chat.PresencePenalty)
}

func TestMergeConfig_ChatMaxTokensZero(t *testing.T) {
	dst := &Config{ResourceDefaults: ResourceDefaults{Chat: ChatDefaults{MaxTokens: ptr(100)}}}
	src := &Config{ResourceDefaults: ResourceDefaults{Chat: ChatDefaults{MaxTokens: ptr(0)}}}
	mergeConfig(dst, src)
	// ptr(0) with *rd.Chat.MaxTokens > 0 guard: should NOT overwrite
	assert.Equal(t, 100, *dst.ResourceDefaults.Chat.MaxTokens)
}

func TestMergeConfig_HTTPDefaults(t *testing.T) {
	dst := &Config{}
	src := &Config{
		ResourceDefaults: ResourceDefaults{
			HTTP: HTTPDefaults{
				Timeout:          "45s",
				FollowRedirects:  true,
				Proxy:            "http://proxy:8080",
				RetryMaxAttempts: 3,
				RetryBackoff:     "1s",
				RetryMaxBackoff:  "30s",
				RetryOn:          "429,503",
			},
		},
	}
	mergeConfig(dst, src)
	assert.Equal(t, "45s", dst.ResourceDefaults.HTTP.Timeout)
	assert.True(t, dst.ResourceDefaults.HTTP.FollowRedirects)
	assert.Equal(t, "http://proxy:8080", dst.ResourceDefaults.HTTP.Proxy)
	assert.Equal(t, 3, dst.ResourceDefaults.HTTP.RetryMaxAttempts)
	assert.Equal(t, "1s", dst.ResourceDefaults.HTTP.RetryBackoff)
	assert.Equal(t, "30s", dst.ResourceDefaults.HTTP.RetryMaxBackoff)
	assert.Equal(t, "429,503", dst.ResourceDefaults.HTTP.RetryOn)
}

func TestMergeConfig_PythonExecSQLDefaults(t *testing.T) {
	dst := &Config{}
	src := &Config{
		ResourceDefaults: ResourceDefaults{
			Python: PythonDefaults{Timeout: "120s"},
			Exec:   ExecDefaults{Timeout: "15s"},
			SQL:    SQLDefaults{Timeout: "20s", MaxRows: 500},
		},
	}
	mergeConfig(dst, src)
	assert.Equal(t, "120s", dst.ResourceDefaults.Python.Timeout)
	assert.Equal(t, "15s", dst.ResourceDefaults.Exec.Timeout)
	assert.Equal(t, "20s", dst.ResourceDefaults.SQL.Timeout)
	assert.Equal(t, 500, dst.ResourceDefaults.SQL.MaxRows)
}

func TestMergeConfig_OnError(t *testing.T) {
	dst := &Config{}
	src := &Config{
		ResourceDefaults: ResourceDefaults{
			OnError: OnErrorDefaults{
				Action:     "retry",
				MaxRetries: 5,
				RetryDelay: "2s",
			},
		},
	}
	mergeConfig(dst, src)
	assert.Equal(t, "retry", dst.ResourceDefaults.OnError.Action)
	assert.Equal(t, 5, dst.ResourceDefaults.OnError.MaxRetries)
	assert.Equal(t, "2s", dst.ResourceDefaults.OnError.RetryDelay)
}

func TestMergeConfig_ConnectionMaps(t *testing.T) {
	dst := &Config{}
	src := &Config{
		HTTPConnections:   map[string]HTTPConnectionConfig{"myhttp": {Proxy: "http://p"}},
		SearchConnections: map[string]SearchConnectionConfig{"mysearch": {APIKey: "sk-search"}},
		SMTPConnections:   map[string]SMTPConnectionConfig{"mysmtp": {Host: "smtp.example.com"}},
		IMAPConnections:   map[string]IMAPConnectionConfig{"myimap": {Host: "imap.example.com"}},
		SQLConnections: map[string]SQLConnectionConfig{
			"mysql": {Connection: "postgres://localhost/db"},
		},
	}
	mergeConfig(dst, src)
	assert.Equal(t, "http://p", dst.HTTPConnections["myhttp"].Proxy)
	assert.Equal(t, "sk-search", dst.SearchConnections["mysearch"].APIKey)
	assert.Equal(t, "smtp.example.com", dst.SMTPConnections["mysmtp"].Host)
	assert.Equal(t, "imap.example.com", dst.IMAPConnections["myimap"].Host)
	assert.Equal(t, "postgres://localhost/db", dst.SQLConnections["mysql"].Connection)
}

func TestMergeConfig_ConnectionMapsNilDst(t *testing.T) {
	// dst maps are nil, should be initialized by mergeConfig
	dst := &Config{}
	src := &Config{
		HTTPConnections:   map[string]HTTPConnectionConfig{"h": {}},
		SearchConnections: map[string]SearchConnectionConfig{"s": {}},
		SMTPConnections:   map[string]SMTPConnectionConfig{"m": {}},
		IMAPConnections:   map[string]IMAPConnectionConfig{"i": {}},
		SQLConnections:    map[string]SQLConnectionConfig{"q": {}},
	}
	mergeConfig(dst, src)
	assert.NotNil(t, dst.HTTPConnections)
	assert.NotNil(t, dst.SearchConnections)
	assert.NotNil(t, dst.SMTPConnections)
	assert.NotNil(t, dst.IMAPConnections)
	assert.NotNil(t, dst.SQLConnections)
}

func TestMergeConfig_ConnectionMapsMergeIntoExisting(t *testing.T) {
	dst := &Config{
		HTTPConnections: map[string]HTTPConnectionConfig{"existing": {Proxy: "keep"}},
	}
	src := &Config{
		HTTPConnections: map[string]HTTPConnectionConfig{"new": {Proxy: "added"}},
	}
	mergeConfig(dst, src)
	assert.Equal(t, "keep", dst.HTTPConnections["existing"].Proxy)
	assert.Equal(t, "added", dst.HTTPConnections["new"].Proxy)
}

func TestMergeConfig_BotConnections(t *testing.T) {
	dst := &Config{}
	src := &Config{
		BotConnections: &BotConnectionConfig{
			Discord: &DiscordConnectionConfig{BotToken: "discord-token"},
		},
	}
	mergeConfig(dst, src)
	require.NotNil(t, dst.BotConnections)
	require.NotNil(t, dst.BotConnections.Discord)
	assert.Equal(t, "discord-token", dst.BotConnections.Discord.BotToken)
}

func TestMergeConfig_BotConnectionsNilSrc(t *testing.T) {
	dst := &Config{BotConnections: &BotConnectionConfig{}}
	src := &Config{}
	mergeConfig(dst, src)
	require.NotNil(t, dst.BotConnections)
}

func TestMergeConfig_APIAuthToken(t *testing.T) {
	dst := &Config{}
	src := &Config{APIAuthToken: "my-secret-token"}
	mergeConfig(dst, src)
	assert.Equal(t, "my-secret-token", dst.APIAuthToken)
}

func TestMergeConfig_EmptySrcNoOverwrite(t *testing.T) {
	dst := &Config{
		LLM: LLMKeys{
			OllamaHost: "http://original:11434",
			OpenAI:     "sk-original",
		},
		Defaults: Defaults{
			Timezone: "America/Chicago",
		},
		ResourceDefaults: ResourceDefaults{
			Chat: ChatDefaults{Timeout: "30s"},
			HTTP: HTTPDefaults{Timeout: "15s"},
		},
	}
	src := &Config{} // all zero values
	mergeConfig(dst, src)
	// dst values must be preserved
	assert.Equal(t, "http://original:11434", dst.LLM.OllamaHost)
	assert.Equal(t, "sk-original", dst.LLM.OpenAI)
	assert.Equal(t, "America/Chicago", dst.Defaults.Timezone)
	assert.Equal(t, "30s", dst.ResourceDefaults.Chat.Timeout)
	assert.Equal(t, "15s", dst.ResourceDefaults.HTTP.Timeout)
}

// --- runPythonCheck ---

func TestRunPythonCheck_Python3Available(t *testing.T) {
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not in PATH")
	}
	r := &doctorRunner{healthy: true}
	r.python()
	assert.True(t, r.healthy)
	assert.Equal(t, HealthPass, r.checks[0].Status)
	assert.Contains(t, r.checks[0].Name, "Python")
}

func TestRunPythonCheck_OnlyPython(t *testing.T) {
	if _, err := exec.LookPath("python3"); err == nil {
		t.Skip("python3 is in PATH, cannot test python-only path on this machine")
	}
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("python not in PATH either, skipping")
	}
	r := &doctorRunner{healthy: true}
	r.python()
	assert.True(t, r.healthy)
	assert.Equal(t, HealthPass, r.checks[0].Status)
}

func TestRunPythonCheck_Neither(t *testing.T) {
	// Use a temp dir as PATH so no python/python3 binary is found.
	dir := t.TempDir()
	t.Setenv("PATH", dir)

	r := &doctorRunner{healthy: true}
	r.python()
	require.GreaterOrEqual(t, len(r.checks), 1)
	assert.Equal(t, HealthWarn, r.checks[0].Status)
	assert.Contains(t, r.checks[0].Message, "python not found")
	assert.True(t, r.healthy)
}

func TestRunPythonCheck_OnlyPython_Shim(t *testing.T) {
	// Place a shim python binary (without python3) in a temp PATH.
	dir := t.TempDir()
	pythonBin := filepath.Join(dir, "python")
	require.NoError(t, os.WriteFile(pythonBin, []byte("#!/bin/sh\nexit 0"), 0755))
	t.Setenv("PATH", dir)

	r := &doctorRunner{healthy: true}
	r.python()
	require.Len(t, r.checks, 1)
	assert.Equal(t, HealthPass, r.checks[0].Status)
	assert.Contains(t, r.checks[0].Message, "python available")
	assert.True(t, r.healthy)
}

// --- LoadWithAgent (integration through real file) ---

func TestLoadWithAgent_NoAgentName(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, "llm:\n  openai_api_key: sk-base\n")
	cfg, err := LoadWithAgent("")
	require.NoError(t, err)
	assert.Equal(t, "sk-base", cfg.LLM.OpenAI)
}

func TestLoadWithAgent_KnownAgent(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  openai_api_key: sk-base
  anthropic_api_key: ant-base
agents:
  my_agent:
    llm:
      openai_api_key: sk-agent
`)
	// Unset known env vars so agent merge takes effect.
	for _, key := range knownConfigEnvVars() {
		_ = os.Unsetenv(key)
	}
	cfg, err := LoadWithAgent("my_agent")
	require.NoError(t, err)
	assert.Equal(t, "sk-agent", cfg.LLM.OpenAI)
	assert.Equal(t, "ant-base", cfg.LLM.Anthropic) // unchanged by agent
	// Env should reflect the merged value.
	assert.Equal(t, "sk-agent", os.Getenv("OPENAI_API_KEY"))
}

func TestLoadWithAgent_UnknownAgent(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  openai_api_key: sk-base
agents:
  other_agent:
    llm:
      openai_api_key: sk-other
`)
	for _, key := range knownConfigEnvVars() {
		_ = os.Unsetenv(key)
	}
	cfg, err := LoadWithAgent("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "sk-base", cfg.LLM.OpenAI)
}

func TestLoadWithAgent_NoAgents(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, "llm:\n  openai_api_key: sk-base\n")
	for _, key := range knownConfigEnvVars() {
		_ = os.Unsetenv(key)
	}
	cfg, err := LoadWithAgent("some_agent")
	require.NoError(t, err)
	assert.Equal(t, "sk-base", cfg.LLM.OpenAI)
}

func TestLoadWithAgent_PreservesExplicitAPIAuthToken(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, "llm:\n  openai_api_key: sk-base\n")
	t.Setenv("KDEPS_API_AUTH_TOKEN", "from-shell")
	for _, key := range knownConfigEnvVars() {
		if key != "KDEPS_API_AUTH_TOKEN" {
			_ = os.Unsetenv(key)
		}
	}
	_, err := LoadWithAgent("workflow-without-agent-profile")
	require.NoError(t, err)
	assert.Equal(t, "from-shell", os.Getenv("KDEPS_API_AUTH_TOKEN"))
}

func TestLoadWithAgent_LoadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("{bad yaml\n"), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	_, err := LoadWithAgent("my_agent")
	assert.Error(t, err)
}

// --- LoadStructWithAgent ---

func TestLoadStructWithAgent_KnownAgent(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  openai_api_key: sk-base
agents:
  my_agent:
    llm:
      openai_api_key: sk-agent
      anthropic_api_key: ant-agent
`)
	cfg, err := LoadStructWithAgent("my_agent")
	require.NoError(t, err)
	assert.Equal(t, "sk-agent", cfg.LLM.OpenAI)
	assert.Equal(t, "ant-agent", cfg.LLM.Anthropic)
}

func TestLoadStructWithAgent_UnknownAgent(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  openai_api_key: sk-base
agents:
  other:
    llm:
      openai_api_key: sk-other
`)
	cfg, err := LoadStructWithAgent("nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "sk-base", cfg.LLM.OpenAI)
}

func TestLoadStructWithAgent_NoAgentName(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, "llm:\n  openai_api_key: sk-base\n")
	cfg, err := LoadStructWithAgent("")
	require.NoError(t, err)
	assert.Equal(t, "sk-base", cfg.LLM.OpenAI)
}

func TestLoadStructWithAgent_NoAgentsInCfg(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, "llm:\n  openai_api_key: sk-base\n")
	cfg, err := LoadStructWithAgent("my_agent")
	require.NoError(t, err)
	assert.Equal(t, "sk-base", cfg.LLM.OpenAI)
}

func TestLoadStructWithAgent_LoadError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("bad: [yaml\n"), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	_, err := LoadStructWithAgent("my_agent")
	assert.Error(t, err)
}

// --- UnmarshalYAML on ModelList ---

func TestModelList_UnmarshalYAML_NonSequence(t *testing.T) {
	var ml ModelList
	err := yaml.Unmarshal([]byte("scalar-value"), &ml)
	assert.Error(t, err)
}

func TestModelList_UnmarshalYAML_InvalidEntry(t *testing.T) {
	var ml ModelList
	err := yaml.Unmarshal([]byte("- [nested, sequence]"), &ml)
	assert.Error(t, err)
}

func TestModelList_UnmarshalYAML_FullEntry(t *testing.T) {
	var ml ModelList
	err := yaml.Unmarshal([]byte("- model: gpt-4\n  backend: openai"), &ml)
	require.NoError(t, err)
	require.Len(t, ml, 1)
	assert.Equal(t, "gpt-4", ml[0].Model)
	assert.Equal(t, "openai", ml[0].Backend)
}

// --- applyResourceDefaults ---

func TestApplyResourceDefaults_AllPointerFields(t *testing.T) {
	// Unset everything so setIfUnset has room to act.
	for _, k := range []string{
		"KDEPS_CHAT_STREAMING", "KDEPS_CHAT_TEMPERATURE",
		"KDEPS_CHAT_MAX_TOKENS", "KDEPS_CHAT_TOP_P",
		"KDEPS_CHAT_FREQUENCY_PENALTY", "KDEPS_CHAT_PRESENCE_PENALTY",
		"KDEPS_CHAT_MAX_OUTPUT_BYTES",
		"KDEPS_HTTP_FOLLOW_REDIRECTS", "KDEPS_HTTP_RETRY_MAX_ATTEMPTS",
		"KDEPS_HTTP_MAX_RESPONSE_BYTES",
		"KDEPS_PYTHON_MAX_OUTPUT_BYTES",
		"KDEPS_EXEC_MAX_OUTPUT_BYTES",
	} {
		require.NoError(t, os.Unsetenv(k))
	}
	t.Cleanup(func() {
		for _, k := range []string{
			"KDEPS_CHAT_STREAMING", "KDEPS_CHAT_TEMPERATURE",
			"KDEPS_CHAT_MAX_TOKENS", "KDEPS_CHAT_TOP_P",
			"KDEPS_CHAT_FREQUENCY_PENALTY", "KDEPS_CHAT_PRESENCE_PENALTY",
			"KDEPS_CHAT_MAX_OUTPUT_BYTES",
			"KDEPS_HTTP_FOLLOW_REDIRECTS", "KDEPS_HTTP_RETRY_MAX_ATTEMPTS",
			"KDEPS_HTTP_MAX_RESPONSE_BYTES",
			"KDEPS_PYTHON_MAX_OUTPUT_BYTES",
			"KDEPS_EXEC_MAX_OUTPUT_BYTES",
		} {
			os.Unsetenv(k)
		}
	})

	rd := ResourceDefaults{
		Chat: ChatDefaults{
			Streaming:        true,
			Temperature:      ptr(0.7),
			MaxTokens:        ptr(4096),
			TopP:             ptr(0.9),
			FrequencyPenalty: ptr(0.1),
			PresencePenalty:  ptr(0.2),
			MaxOutputBytes:   1048576,
		},
		HTTP: HTTPDefaults{
			FollowRedirects:  true,
			RetryMaxAttempts: 3,
			MaxResponseBytes: 10485760,
		},
		Python: PythonDefaults{
			MaxOutputBytes: 1048576,
		},
		Exec: ExecDefaults{
			MaxOutputBytes: 1048576,
		},
	}
	applyResourceDefaults(rd)

	assert.Equal(t, "true", os.Getenv("KDEPS_CHAT_STREAMING"))
	assert.Equal(t, "0.7", os.Getenv("KDEPS_CHAT_TEMPERATURE"))
	assert.Equal(t, "4096", os.Getenv("KDEPS_CHAT_MAX_TOKENS"))
	assert.Equal(t, "0.9", os.Getenv("KDEPS_CHAT_TOP_P"))
	assert.Equal(t, "0.1", os.Getenv("KDEPS_CHAT_FREQUENCY_PENALTY"))
	assert.Equal(t, "0.2", os.Getenv("KDEPS_CHAT_PRESENCE_PENALTY"))
	assert.Equal(t, "1048576", os.Getenv("KDEPS_CHAT_MAX_OUTPUT_BYTES"))
	assert.Equal(t, "true", os.Getenv("KDEPS_HTTP_FOLLOW_REDIRECTS"))
	assert.Equal(t, "3", os.Getenv("KDEPS_HTTP_RETRY_MAX_ATTEMPTS"))
	assert.Equal(t, "10485760", os.Getenv("KDEPS_HTTP_MAX_RESPONSE_BYTES"))
	assert.Equal(t, "1048576", os.Getenv("KDEPS_PYTHON_MAX_OUTPUT_BYTES"))
	assert.Equal(t, "1048576", os.Getenv("KDEPS_EXEC_MAX_OUTPUT_BYTES"))
}

// --- applyEnv models list ---

func TestApplyEnv_ModelsList(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_LLM_MODELS"))
	require.NoError(t, os.Unsetenv("KDEPS_LLM_ROUTER"))
	t.Cleanup(func() {
		os.Unsetenv("KDEPS_LLM_MODELS")
		os.Unsetenv("KDEPS_LLM_ROUTER")
	})

	cfg := Config{
		LLM: LLMKeys{
			Models: ModelList{
				{Model: "gpt-4"},
				{Model: "claude-sonnet"},
				{Model: "llama3.2"},
			},
		},
	}
	applyEnv(cfg)

	assert.Equal(t, "gpt-4,claude-sonnet,llama3.2", os.Getenv("KDEPS_LLM_MODELS"))
}

func TestScaffold_StatError(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	AppFS = afero.NewMemMapFs()

	origGetenv := osGetenv
	t.Cleanup(func() { osGetenv = origGetenv })
	osGetenv = func(key string) string {
		if key == "KDEPS_CONFIG_PATH" {
			return t.TempDir() + "/.kdeps/config.yaml"
		}
		return ""
	}

	err := Scaffold()
	assert.NoError(t, err)
}

func TestAgentsDir_EnvVar(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	memFS := afero.NewMemMapFs()
	AppFS = memFS

	tmpDir := "/tmp/agents-test"
	agentsDir := filepath.Join(tmpDir, "agents")
	_ = memFS.MkdirAll(agentsDir, 0750)

	origGetenv := osGetenv
	t.Cleanup(func() { osGetenv = origGetenv })
	osGetenv = func(key string) string {
		if key == "KDEPS_AGENTS_DIR" {
			return agentsDir
		}
		return ""
	}

	dir, err := AgentsDir(&Config{})
	require.NoError(t, err)
	assert.Equal(t, agentsDir, dir)
}

func TestScaffold_PathError(t *testing.T) {
	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home")
	}
	origGetenv := osGetenv
	t.Cleanup(func() { osGetenv = origGetenv })
	osGetenv = func(_ string) string { return "" }

	err := Scaffold()
	assert.NoError(t, err) // non-fatal
}

func TestScaffold_AlreadyExists(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	memFS := afero.NewMemMapFs()
	AppFS = memFS

	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) { return "/fakehome", nil }
	origGetenv := osGetenv
	t.Cleanup(func() { osGetenv = origGetenv })
	osGetenv = func(_ string) string { return "" }

	// Create the config file so it already exists
	configDir := "/fakehome/.kdeps"
	_ = memFS.MkdirAll(configDir, 0750)
	_ = afero.WriteFile(memFS, configDir+"/config.yaml", []byte("llm: {}"), 0600)

	err := Scaffold()
	assert.NoError(t, err)
}

func TestAgentsDir_HomeError(t *testing.T) {
	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) { return "", errors.New("no home") }
	origGetenv := osGetenv
	t.Cleanup(func() { osGetenv = origGetenv })
	osGetenv = func(_ string) string { return "" }

	_, err := AgentsDir(&Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "home directory")
}

type errorFS struct{ afero.Fs }

func (e errorFS) Open(_ string) (afero.File, error) { return nil, errors.New("permission denied") }

func TestLoad_ReadError(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	AppFS = errorFS{afero.NewMemMapFs()}

	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) { return "/fakehome", nil }
	origGetenv := osGetenv
	t.Cleanup(func() { osGetenv = origGetenv })
	osGetenv = func(_ string) string { return "" }

	_, err := load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestLoad_ParseError(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	memFS := afero.NewMemMapFs()
	AppFS = memFS

	configPath := "/fakehome/.kdeps/config.yaml"
	_ = memFS.MkdirAll("/fakehome/.kdeps", 0750)
	_ = afero.WriteFile(memFS, configPath, []byte("invalid: {{{yaml"), 0600)

	origHome := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = origHome })
	osUserHomeDir = func() (string, error) { return "/fakehome", nil }
	origGetenv := osGetenv
	t.Cleanup(func() { osGetenv = origGetenv })
	osGetenv = func(_ string) string { return "" }

	_, err := load()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parse")
}

func TestConfigEnvVar_APIKeys(t *testing.T) {
	for _, p := range cloudProvidersList {
		env, ok := configEnvVar("llm." + p.yamlKey)
		require.True(t, ok, "backend %s", p.name)
		assert.Equal(t, p.envVar, env)
	}
}
