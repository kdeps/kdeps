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

package config_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/config"
)

func TestLoad_NoFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KDEPS_CONFIG_PATH", filepath.Join(dir, "missing.yaml"))
	cfg, err := config.Load()
	require.NoError(t, err)
	assert.NotNil(t, cfg)
	p := config.CloudLLMProviders()[0]
	v, err := cfg.GetField("llm." + p.YAMLKey)
	require.NoError(t, err)
	assert.Empty(t, v)
	assert.Empty(t, cfg.LLM.OllamaHost)
}

func TestLoad_ValidFile(t *testing.T) {
	providers := config.CloudLLMProviders()
	require.GreaterOrEqual(t, len(providers), 2)
	primary, secondary := providers[0], providers[1]
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := fmt.Sprintf(`
llm:
  ollama_host: http://localhost:11434
  %s: sk-test
  %s: ant-test
`, primary.YAMLKey, secondary.YAMLKey)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	for _, k := range []string{primary.EnvVar, secondary.EnvVar, "OLLAMA_HOST"} {
		require.NoError(t, os.Unsetenv(k))
	}

	cfg, err := config.Load()
	require.NoError(t, err)
	v, err := cfg.GetField("llm." + primary.YAMLKey)
	require.NoError(t, err)
	assert.Equal(t, "sk-test", v)
	assert.Equal(t, "http://localhost:11434", cfg.LLM.OllamaHost)
	assert.Equal(t, "sk-test", os.Getenv(primary.EnvVar))
	assert.Equal(t, "ant-test", os.Getenv(secondary.EnvVar))
	assert.Equal(t, "http://localhost:11434", os.Getenv("OLLAMA_HOST"))
}

func TestLoad_EnvVarWins(t *testing.T) {
	p := config.CloudLLMProviders()[0]
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := fmt.Sprintf("llm:\n  %s: from-file\n", p.YAMLKey)
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	t.Setenv(p.EnvVar, "from-env")

	cfg, err := config.Load()
	require.NoError(t, err)
	v, err := cfg.GetField("llm." + p.YAMLKey)
	require.NoError(t, err)
	assert.Equal(t, "from-file", v)
	assert.Equal(t, "from-env", os.Getenv(p.EnvVar))
}

func TestLoad_Malformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("llm: [bad yaml\n"), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)

	_, err := config.Load()
	assert.Error(t, err)
}

func TestLoad_UnreadableFile(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses file permissions")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("llm:\n"), 0600))
	require.NoError(t, os.Chmod(path, 0000))
	t.Cleanup(func() { _ = os.Chmod(path, 0600) })
	t.Setenv("KDEPS_CONFIG_PATH", path)

	_, err := config.Load()
	assert.Error(t, err)
}

func TestScaffold_MkdirFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses file permissions")
	}
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", filepath.Join(blocker, "sub", "config.yaml"))

	err := config.Scaffold()
	assert.Error(t, err)
}

func TestLoad_Defaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
defaults:
  timezone: UTC
  python_version: "3.11"
  offline_mode: true
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	require.NoError(t, os.Unsetenv("TZ"))
	require.NoError(t, os.Unsetenv("KDEPS_PYTHON_VERSION"))
	require.NoError(t, os.Unsetenv("KDEPS_OFFLINE_MODE"))

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "UTC", cfg.Defaults.Timezone)
	assert.Equal(t, "3.11", cfg.Defaults.PythonVersion)
	assert.True(t, cfg.Defaults.OfflineMode)
	assert.Equal(t, "UTC", os.Getenv("TZ"))
	assert.Equal(t, "3.11", os.Getenv("KDEPS_PYTHON_VERSION"))
	assert.Equal(t, "true", os.Getenv("KDEPS_OFFLINE_MODE"))
}

func TestScaffold_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	require.NoError(t, config.Scaffold())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	for _, p := range config.CloudLLMProviders() {
		assert.Contains(t, string(data), p.YAMLKey)
	}
}

func TestScaffold_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	original := fmt.Sprintf("llm:\n  %s: keep-me\n", config.CloudLLMProviders()[0].YAMLKey)
	require.NoError(t, os.WriteFile(path, []byte(original), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)

	require.NoError(t, config.Scaffold())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, original, string(data))
}

func TestAgentsDir_Default(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_AGENTS_DIR"))
	dir, err := config.AgentsDir(nil)
	require.NoError(t, err)
	home, _ := os.UserHomeDir()
	assert.Equal(t, filepath.Join(home, ".kdeps", "agents"), dir)
}

func TestAgentsDir_EnvOverride(t *testing.T) {
	t.Setenv("KDEPS_AGENTS_DIR", "/custom/agents")
	dir, err := config.AgentsDir(nil)
	require.NoError(t, err)
	assert.Equal(t, "/custom/agents", dir)
}

func TestLoad_ModelsDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := "llm:\n  models_dir: /custom/models\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	require.NoError(t, os.Unsetenv("KDEPS_MODELS_DIR"))

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "/custom/models", cfg.LLM.ModelsDir)
	assert.Equal(t, "/custom/models", os.Getenv("KDEPS_MODELS_DIR"))
}

func TestLoad_ModelsDirEnvWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := "llm:\n  models_dir: /from-config\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	t.Setenv("KDEPS_MODELS_DIR", "/from-env")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "/from-config", cfg.LLM.ModelsDir) // struct always reflects file
	assert.Equal(t, "/from-env", os.Getenv("KDEPS_MODELS_DIR"))
}

func TestScaffold_ContainsModelsDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	require.NoError(t, config.Scaffold())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "models_dir")
}

func TestLoad_ResourceDefaults_AllFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
resource_defaults:
  chat:
    timeout: "90s"
    context_length: 8192
  http:
    timeout: "45s"
  python:
    timeout: "120s"
  exec:
    timeout: "15s"
  sql:
    timeout: "20s"
    max_rows: 500
  onError:
    action: "retry"
    max_retries: 5
    retry_delay: "2s"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	for _, k := range []string{
		"KDEPS_CHAT_TIMEOUT", "KDEPS_CHAT_CONTEXT_LENGTH",
		"KDEPS_HTTP_TIMEOUT", "KDEPS_PYTHON_TIMEOUT", "KDEPS_EXEC_TIMEOUT",
		"KDEPS_SQL_TIMEOUT", "KDEPS_SQL_MAX_ROWS",
		"KDEPS_ON_ERROR_ACTION", "KDEPS_ON_ERROR_MAX_RETRIES", "KDEPS_ON_ERROR_RETRY_DELAY",
	} {
		require.NoError(t, os.Unsetenv(k))
	}

	cfg, err := config.Load()
	require.NoError(t, err)

	assert.Equal(t, "90s", cfg.ResourceDefaults.Chat.Timeout)
	assert.Equal(t, 8192, cfg.ResourceDefaults.Chat.ContextLength)
	assert.Equal(t, "45s", cfg.ResourceDefaults.HTTP.Timeout)
	assert.Equal(t, "120s", cfg.ResourceDefaults.Python.Timeout)
	assert.Equal(t, "15s", cfg.ResourceDefaults.Exec.Timeout)
	assert.Equal(t, "20s", cfg.ResourceDefaults.SQL.Timeout)
	assert.Equal(t, 500, cfg.ResourceDefaults.SQL.MaxRows)
	assert.Equal(t, "retry", cfg.ResourceDefaults.OnError.Action)
	assert.Equal(t, 5, cfg.ResourceDefaults.OnError.MaxRetries)
	assert.Equal(t, "2s", cfg.ResourceDefaults.OnError.RetryDelay)

	assert.Equal(t, "90s", os.Getenv("KDEPS_CHAT_TIMEOUT"))
	assert.Equal(t, "8192", os.Getenv("KDEPS_CHAT_CONTEXT_LENGTH"))
	assert.Equal(t, "45s", os.Getenv("KDEPS_HTTP_TIMEOUT"))
	assert.Equal(t, "120s", os.Getenv("KDEPS_PYTHON_TIMEOUT"))
	assert.Equal(t, "15s", os.Getenv("KDEPS_EXEC_TIMEOUT"))
	assert.Equal(t, "20s", os.Getenv("KDEPS_SQL_TIMEOUT"))
	assert.Equal(t, "500", os.Getenv("KDEPS_SQL_MAX_ROWS"))
	assert.Equal(t, "retry", os.Getenv("KDEPS_ON_ERROR_ACTION"))
	assert.Equal(t, "5", os.Getenv("KDEPS_ON_ERROR_MAX_RETRIES"))
	assert.Equal(t, "2s", os.Getenv("KDEPS_ON_ERROR_RETRY_DELAY"))
}

func TestLoad_ResourceDefaults_EnvVarsWin(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
resource_defaults:
  chat:
    timeout: "from-file"
  http:
    timeout: "from-file"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	t.Setenv("KDEPS_CHAT_TIMEOUT", "from-env")
	t.Setenv("KDEPS_HTTP_TIMEOUT", "from-env")

	_, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "from-env", os.Getenv("KDEPS_CHAT_TIMEOUT"))
	assert.Equal(t, "from-env", os.Getenv("KDEPS_HTTP_TIMEOUT"))
}

func TestLoad_ResourceDefaults_ZeroValuesNotSet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// context_length and max_rows intentionally zero (not set)
	content := "resource_defaults:\n  chat:\n    timeout: \"30s\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	require.NoError(t, os.Unsetenv("KDEPS_CHAT_CONTEXT_LENGTH"))
	require.NoError(t, os.Unsetenv("KDEPS_SQL_MAX_ROWS"))
	require.NoError(t, os.Unsetenv("KDEPS_ON_ERROR_MAX_RETRIES"))

	_, err := config.Load()
	require.NoError(t, err)
	assert.Empty(t, os.Getenv("KDEPS_CHAT_CONTEXT_LENGTH"))
	assert.Empty(t, os.Getenv("KDEPS_SQL_MAX_ROWS"))
	assert.Empty(t, os.Getenv("KDEPS_ON_ERROR_MAX_RETRIES"))
}

func TestScaffold_ContainsResourceDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	require.NoError(t, config.Scaffold())
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "resource_defaults")
	assert.Contains(t, string(data), "context_length")
	assert.Contains(t, string(data), "max_rows")
	assert.Contains(t, string(data), "onError")
}
