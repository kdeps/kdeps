package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func writeTempConfig(t *testing.T, dir, content string) {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0600))
	t.Setenv("KDEPS_CONFIG_PATH", path)
}

func loadCfg(t *testing.T) *Config {
	t.Helper()
	cfg, err := LoadStruct()
	require.NoError(t, err)
	return cfg
}

func allCloudProviderKeysYAML() string {
	var b strings.Builder
	for _, p := range cloudProvidersList {
		fmt.Fprintf(&b, "  %s: sk-%s\n", p.yamlKey, p.name)
	}
	return b.String()
}

func TestValidate_ValidConfig_NoWarnings(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  ollama_host: http://localhost:11434
  backend: ollama
  openai_api_key: sk-test
defaults:
  timezone: UTC
resource_defaults:
  chat:
    timeout: "60s"
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.Empty(t, warnings)
}

func TestValidate_NoFile_NoWarnings(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KDEPS_CONFIG_PATH", filepath.Join(dir, "nonexistent.yaml"))
	cfg := loadCfg(t)
	assert.Empty(t, cfg.Validate(""))
}

func TestValidate_UnknownTopLevelKey(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  ollama_host: http://localhost:11434
bad_key: true
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "bad_key") && strings.Contains(w, "top-level") {
			found = true
		}
	}
	assert.True(t, found, "expected warning about unknown top-level key bad_key, got: %v", warnings)
}

func TestValidate_UnknownLLMKey(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  ollama_host: http://localhost:11434
  openai_apikey: sk-typo
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "openai_apikey") && strings.Contains(w, "llm") {
			found = true
		}
	}
	assert.True(t, found, "expected warning about unknown llm key openai_apikey, got: %v", warnings)
}

func TestValidate_UnknownDefaultsKey(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
defaults:
  timezone: UTC
  time_zone: UTC
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "time_zone") {
			found = true
		}
	}
	assert.True(t, found, "expected warning about unknown defaults key time_zone, got: %v", warnings)
}

func TestValidate_UnknownResourceDefaultsKey(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
resource_defaults:
  chat:
    timeout: "60s"
  llm:
    timeout: "30s"
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "llm") && strings.Contains(w, "resource_defaults") {
			found = true
		}
	}
	assert.True(t, found, "expected warning about unknown resource_defaults key llm, got: %v", warnings)
}

func TestValidate_BackendWithoutAPIKey(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  backend: openai
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "openai") && strings.Contains(w, "openai_api_key") {
			found = true
		}
	}
	assert.True(t, found, "expected warning about missing openai_api_key, got: %v", warnings)
}

func TestValidate_BackendWithAPIKey_NoWarning(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  backend: openai
  openai_api_key: sk-test
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	for _, w := range warnings {
		assert.NotContains(t, w, "openai_api_key")
	}
}

func TestValidate_BackendOllama_NoWarning(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  backend: ollama
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	for _, w := range warnings {
		assert.NotContains(t, w, "api_key")
	}
}

func TestValidate_InvalidStrategy(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  strategy: random_choice
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "strategy") && strings.Contains(w, "random_choice") {
			found = true
		}
	}
	assert.True(t, found, "expected warning about invalid strategy, got: %v", warnings)
}

func TestValidate_ValidStrategies_NoWarning(t *testing.T) {
	dir := t.TempDir()
	for _, s := range []string{"token_threshold", "fallback", "cost_optimized", "round_robin"} {
		writeTempConfig(t, dir, "llm:\n  strategy: "+s+"\n")
		cfg := loadCfg(t)
		warnings := cfg.Validate("")
		for _, w := range warnings {
			assert.NotContains(t, w, "strategy", "unexpected warning for valid strategy %q: %s", s, w)
		}
	}
}

func TestValidate_BadDuration(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
resource_defaults:
  chat:
    timeout: "not-a-duration"
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "chat.timeout") && strings.Contains(w, "not-a-duration") {
			found = true
		}
	}
	assert.True(t, found, "expected warning about bad duration, got: %v", warnings)
}

func TestValidate_GoodDuration_NoWarning(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
resource_defaults:
  chat:
    timeout: "120s"
  http:
    timeout: "5m"
    retry_backoff: "1s"
    retry_max_backoff: "30s"
  python:
    timeout: "60s"
  exec:
    timeout: "30s"
  sql:
    timeout: "60s"
  onError:
    retry_delay: "2s"
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	for _, w := range warnings {
		assert.NotContains(t, w, "duration", "unexpected duration warning: %s", w)
	}
}

func TestValidate_EmptyAgentProfile(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
agents:
  empty_profile: {}
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.NotEmpty(t, warnings)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "empty_profile") && strings.Contains(w, "no non-empty fields") {
			found = true
		}
	}
	assert.True(t, found, "expected warning about empty agent profile, got: %v", warnings)
}

func TestValidate_AgentProfileWithFields_NoWarning(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  backend: openai
  openai_api_key: sk-global
agents:
  my_agent:
    llm:
      openai_api_key: sk-agent
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	for _, w := range warnings {
		assert.NotContains(t, w, "no non-empty fields")
	}
}

func TestValidate_AgentProfileWithWorkflow(t *testing.T) {
	agentsDir := t.TempDir()
	// Create a mock workflow with metadata.name
	workflowDir := filepath.Join(agentsDir, "my_agent")
	require.NoError(t, os.MkdirAll(workflowDir, 0755))
	workflowYAML := "metadata:\n  name: my_agent\n"
	require.NoError(t, os.WriteFile(filepath.Join(workflowDir, "workflow.yaml"), []byte(workflowYAML), 0644))

	dir := t.TempDir()
	writeTempConfig(t, dir, `
agents:
  my_agent:
    llm:
      openai_api_key: sk-test
  unknown_agent:
    llm:
      backend: openai
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate(agentsDir)

	// unknown_agent should trigger a warning.
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "unknown_agent") && strings.Contains(w, "does not match") {
			found = true
		}
	}
	assert.True(t, found, "expected warning about unreferenced agent, got: %v", warnings)

	// my_agent should NOT trigger a "does not match" warning.
	for _, w := range warnings {
		assert.NotContains(t, w, "my_agent")
	}
}

func TestValidate_MultipleWarnings(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
unknown_top: true
llm:
  bad_llm_key: true
  backend: openai
  strategy: invalid
resource_defaults:
  chat:
    timeout: "bad"
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.GreaterOrEqual(t, len(warnings), 4, "expected at least 4 warnings, got %d: %v", len(warnings), warnings)
}

func TestCollectUnknownKeys(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("a: 1\nb: 2\nc: 3"), &node))
	root := node.Content[0]
	known := map[string]bool{"a": true, "b": true}
	unknown := collectUnknownKeys(root, known)
	assert.Equal(t, []string{"c"}, unknown)
}

func TestFindMappingValue(t *testing.T) {
	var node yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("a:\n  x: 1\nb: 2"), &node))
	root := node.Content[0]
	found := findMappingValue(root, "a")
	require.NotNil(t, found)
	assert.Equal(t, yaml.MappingNode, found.Kind)
	assert.Nil(t, findMappingValue(root, "b"))
	assert.Nil(t, findMappingValue(root, "missing"))
}

func TestHasCloudProviderKey(t *testing.T) {
	assert.False(t, hasCloudProviderKey(LLMKeys{}))
	llm := LLMKeys{}
	cloudProvidersList[0].setLLMKey(&llm, "sk-test")
	assert.True(t, hasCloudProviderKey(llm))
}

func TestGetLLMAPIKey_AllBackends(t *testing.T) {
	cfg := &Config{}
	for _, p := range cloudProvidersList {
		p.setLLMKey(&cfg.LLM, "sk-"+p.name)
	}
	for _, p := range cloudProvidersList {
		assert.Equal(t, "sk-"+p.name, getLLMAPIKey(cfg.LLM, p.name))
	}
	assert.Equal(t, "", getLLMAPIKey(cfg.LLM, "unknown"))
	assert.Equal(t, "", getLLMAPIKey(LLMKeys{}, "openai"))
}

func TestValidate_UnreadableConfig(t *testing.T) {
	// Config was loaded from a file that was then deleted.
	// Validate should handle the missing file gracefully.
	dir := t.TempDir()
	writeTempConfig(t, dir, "llm:\n  openai_api_key: sk-test\n")
	cfg := loadCfg(t)
	require.NoError(t, os.Remove(filepath.Join(dir, "config.yaml")))
	warnings := cfg.Validate("")
	assert.Empty(t, warnings)
}

func TestValidate_ValidateCalledAfterLoad(t *testing.T) {
	// Validate on a config that was loaded from a valid file
	// with all backends set should produce no API key warnings.
	dir := t.TempDir()
	writeTempConfig(t, dir, fmt.Sprintf(`
llm:
  backend: openai
%sresource_defaults:
  chat:
    timeout: "60s"
`, allCloudProviderKeysYAML()))
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	assert.Empty(t, warnings)
}

func TestValidateUnknownKeys_EmptyDoc(t *testing.T) {
	warnings := validateUnknownKeys([]byte(""))
	assert.Empty(t, warnings)
}

func TestValidateUnknownKeys_ScalarRoot(t *testing.T) {
	// YAML that is just a scalar, not a mapping
	warnings := validateUnknownKeys([]byte("just-a-string"))
	assert.Empty(t, warnings)
}

func TestValidateUnknownKeys_MalformedYAML(t *testing.T) {
	warnings := validateUnknownKeys([]byte("{{invalid"))
	assert.Empty(t, warnings)
}

func TestCollectWorkflowNames_EmptyDir(t *testing.T) {
	assert.Nil(t, collectWorkflowNames(""))
}

func TestCollectWorkflowNames_NonExistentDir(t *testing.T) {
	assert.Nil(t, collectWorkflowNames("/nonexistent/path/12345"))
}

func TestCollectWorkflowNames_EmptyDirNoFiles(t *testing.T) {
	dir := t.TempDir()
	assert.Nil(t, collectWorkflowNames(dir))
}

func TestCollectWorkflowNames_DirWithNonDirEntries(t *testing.T) {
	dir := t.TempDir()
	// Create a file (not a directory) in agents dir
	require.NoError(t, os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("hello"), 0644))
	assert.Nil(t, collectWorkflowNames(dir))
}

func TestCollectWorkflowNames_NoWorkflowYAML(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "some_agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	// Create a file that is NOT workflow.yaml
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "README.md"), []byte("readme"), 0644))
	assert.Nil(t, collectWorkflowNames(dir))
}

func TestCollectWorkflowNames_WorkflowWithEmptyName(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "empty_agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yaml"),
		[]byte("metadata:\n  name: \"\"\n"), 0644))
	assert.Nil(t, collectWorkflowNames(dir))
}

func TestCollectWorkflowNames_WorkflowWithoutMetadata(t *testing.T) {
	dir := t.TempDir()
	agentDir := filepath.Join(dir, "no_meta_agent")
	require.NoError(t, os.MkdirAll(agentDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(agentDir, "workflow.yaml"),
		[]byte("resources:\n  - name: test\n"), 0644))
	assert.Nil(t, collectWorkflowNames(dir))
}

func TestValidate_AgentProfileNoWorkflows(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
agents:
  some_agent:
    llm:
      openai_api_key: sk-test
`)
	cfg := loadCfg(t)
	// AgentsDir is empty string — no workflow discovery, so no "does not match" warning
	warnings := cfg.Validate("")
	for _, w := range warnings {
		assert.NotContains(t, w, "does not match any installed workflow")
	}
}

func TestValidate_BackendAllProvidersAPIKeyCheck(t *testing.T) {
	for _, p := range cloudProvidersList {
		dir := t.TempDir()
		writeTempConfig(t, dir, "llm:\n  backend: "+p.name+"\n")
		cfg := loadCfg(t)
		warnings := cfg.Validate("")
		found := false
		for _, w := range warnings {
			if strings.Contains(w, p.name) && strings.Contains(w, "not set") {
				found = true
			}
		}
		assert.True(t, found, "expected warning for backend %q without API key, got: %v", p.name, warnings)
	}
}

func TestValidate_UnknownBackendNoAPIKeyWarning(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  backend: unknown_provider
`)
	cfg := loadCfg(t)
	warnings := cfg.Validate("")
	// Unknown backend is not in cloudProviders, so no API key warning
	for _, w := range warnings {
		assert.NotContains(t, w, "not set")
	}
}

func TestIsEmptyAgentProfile(t *testing.T) {
	assert.True(t, isEmptyAgentProfile(Config{}))
	llm := LLMKeys{}
	cloudProvidersList[0].setLLMKey(&llm, "sk-test")
	assert.False(t, isEmptyAgentProfile(Config{LLM: llm}))
	assert.False(t, isEmptyAgentProfile(Config{Defaults: Defaults{Timezone: "UTC"}}))
	assert.False(t, isEmptyAgentProfile(Config{ResourceDefaults: ResourceDefaults{
		Chat: ChatDefaults{Timeout: "60s"},
	}}))
	assert.False(t, isEmptyAgentProfile(Config{ResourceDefaults: ResourceDefaults{
		Chat: ChatDefaults{Streaming: true},
	}}))
}
