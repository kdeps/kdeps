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
	assert.Empty(t, cfg.LLM.OpenAI)
	assert.Empty(t, cfg.LLM.OllamaHost)
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
llm:
  ollama_host: http://localhost:11434
  model: llama3.2
  openai_api_key: sk-test
  anthropic_api_key: ant-test
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	// Unset any pre-existing values so setIfUnset has room to act.
	for _, k := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "OLLAMA_HOST", "KDEPS_DEFAULT_MODEL"} {
		require.NoError(t, os.Unsetenv(k))
	}

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "sk-test", cfg.LLM.OpenAI)
	assert.Equal(t, "http://localhost:11434", cfg.LLM.OllamaHost)
	assert.Equal(t, "llama3.2", cfg.LLM.DefaultModel)

	// Env vars should be populated.
	assert.Equal(t, "sk-test", os.Getenv("OPENAI_API_KEY"))
	assert.Equal(t, "ant-test", os.Getenv("ANTHROPIC_API_KEY"))
	assert.Equal(t, "http://localhost:11434", os.Getenv("OLLAMA_HOST"))
	assert.Equal(t, "llama3.2", os.Getenv("KDEPS_DEFAULT_MODEL"))
}

func TestLoad_EnvVarWins(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := "llm:\n  openai_api_key: from-file\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	t.Setenv("OPENAI_API_KEY", "from-env")

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "from-file", cfg.LLM.OpenAI)             // struct always reflects file
	assert.Equal(t, "from-env", os.Getenv("OPENAI_API_KEY")) // env not overwritten
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
	assert.Contains(t, string(data), "openai_api_key")
}

func TestScaffold_DoesNotOverwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	original := "llm:\n  openai_api_key: keep-me\n"
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
