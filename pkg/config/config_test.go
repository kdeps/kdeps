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
	assert.Empty(t, cfg.Registry.URL)
}

func TestLoad_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
llm:
  openai_api_key: sk-test
  anthropic_api_key: ant-test
registry:
  url: https://custom.registry.example
  token: tok-abc
storage:
  agents_dir: /tmp/agents
  components_dir: /tmp/components
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
	// Unset any pre-existing values so setIfUnset has room to act.
	for _, k := range []string{"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "KDEPS_REGISTRY_URL",
		"KDEPS_REGISTRY_TOKEN", "KDEPS_AGENTS_DIR", "KDEPS_COMPONENT_DIR"} {
		t.Setenv(k, "") // mark as set-but-empty; we need to actually unset
		require.NoError(t, os.Unsetenv(k))
	}

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "sk-test", cfg.LLM.OpenAI)
	assert.Equal(t, "https://custom.registry.example", cfg.Registry.URL)
	assert.Equal(t, "/tmp/agents", cfg.Storage.AgentsDir)

	// Env vars should be populated.
	assert.Equal(t, "sk-test", os.Getenv("OPENAI_API_KEY"))
	assert.Equal(t, "ant-test", os.Getenv("ANTHROPIC_API_KEY"))
	assert.Equal(t, "https://custom.registry.example", os.Getenv("KDEPS_REGISTRY_URL"))
	assert.Equal(t, "tok-abc", os.Getenv("KDEPS_REGISTRY_TOKEN"))
	assert.Equal(t, "/tmp/agents", os.Getenv("KDEPS_AGENTS_DIR"))
	assert.Equal(t, "/tmp/components", os.Getenv("KDEPS_COMPONENT_DIR"))
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

func TestAgentsDir_ConfigOverride(t *testing.T) {
	require.NoError(t, os.Unsetenv("KDEPS_AGENTS_DIR"))
	cfg := &config.Config{Storage: config.StorageConfig{AgentsDir: "/cfg/agents"}}
	dir, err := config.AgentsDir(cfg)
	require.NoError(t, err)
	assert.Equal(t, "/cfg/agents", dir)
}
