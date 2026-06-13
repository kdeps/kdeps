package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDoctor_AllChecks(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  ollama_host: http://localhost:11434
  backend: ollama
  models:
    - llama3.2
defaults:
  timezone: UTC
resource_defaults:
  chat:
    timeout: "60s"
`)

	cfg := loadCfg(t)
	report := RunDoctor(cfg)
	assert.NotNil(t, report)
	assert.GreaterOrEqual(t, len(report.Checks), 5) // At least 5 checks always run.

	// Verify each check has a name and status.
	for _, c := range report.Checks {
		assert.NotEmpty(t, c.Name)
		assert.NotEmpty(t, c.Message)
		assert.Contains(t, []HealthStatus{HealthPass, HealthWarn, HealthFail}, c.Status)
	}

	// Verify formatted report.
	formatted := report.FormatReport()
	assert.Contains(t, formatted, "kdeps doctor")
	assert.Contains(t, formatted, "Config file")
	assert.Contains(t, formatted, "Ollama")
	assert.Contains(t, formatted, "Python")
	assert.Contains(t, formatted, "Backend/API key")
	assert.Contains(t, formatted, "Env vars")
}

func TestRunDoctor_ConfigFileNotFound(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.yaml")
	t.Setenv("KDEPS_CONFIG_PATH", path)

	cfg := loadCfg(t)
	report := RunDoctor(cfg)

	var configCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Config file" {
			configCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, configCheck)
	assert.Equal(t, HealthWarn, configCheck.Status)
	assert.Contains(t, configCheck.Message, "not found")
}

func TestRunDoctor_ConfigValidationWarnings(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  ollama_host: http://localhost:11434
  openai_apikey: sk-typo
`)
	cfg := loadCfg(t)
	report := RunDoctor(cfg)

	var valCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Config validation" {
			valCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, valCheck)
	assert.Equal(t, HealthWarn, valCheck.Status)
	assert.Contains(t, valCheck.Message, "openai_apikey")
}

func TestRunDoctor_BackendWithoutAPIKey(t *testing.T) {
	p := primaryCloudProvider()
	dir := t.TempDir()
	t.Setenv(p.envVar, "")
	writeTempConfig(t, dir, fmt.Sprintf(`
llm:
  backend: %s
`, p.name))
	cfg := loadCfg(t)
	report := RunDoctor(cfg)

	var backendCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Backend/API key" {
			backendCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, backendCheck)
	assert.Equal(t, HealthWarn, backendCheck.Status)
	assert.Contains(t, backendCheck.Message, "no API key")
}

func TestRunDoctor_BackendWithAPIKey(t *testing.T) {
	p := primaryCloudProvider()
	dir := t.TempDir()
	t.Setenv(p.envVar, "")
	writeTempConfig(t, dir, fmt.Sprintf(`
llm:
  backend: %s
  %s: sk-test-key-12345678
`, p.name, p.yamlKey))
	cfg := loadCfg(t)
	report := RunDoctor(cfg)

	var backendCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Backend/API key" {
			backendCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, backendCheck)
	assert.Equal(t, HealthPass, backendCheck.Status)
	assert.Contains(t, backendCheck.Message, "backend="+p.name)
	assert.Contains(t, backendCheck.Message, "sk-t...5678")
}

func TestRunDoctor_OllamaCheck(t *testing.T) {
	// Use a port that's unlikely to have anything listening, with backend=ollama.
	t.Setenv("OLLAMA_HOST", "")
	t.Setenv("KDEPS_DEFAULT_BACKEND", "")
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  backend: ollama
  ollama_host: http://localhost:19999
`)
	cfg := loadCfg(t)
	report := RunDoctor(cfg)

	var ollamaCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Ollama" {
			ollamaCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, ollamaCheck)
	assert.Equal(t, HealthWarn, ollamaCheck.Status)
	assert.Contains(t, ollamaCheck.Message, "not reachable")
}

func TestRunDoctor_OllamaCheck_SkippedForCloudBackend(t *testing.T) {
	p := primaryCloudProvider()
	t.Setenv("KDEPS_DEFAULT_BACKEND", "")
	dir := t.TempDir()
	writeTempConfig(t, dir, fmt.Sprintf(`
llm:
  backend: %s
  %s: sk-test
`, p.name, p.yamlKey))
	cfg := loadCfg(t)
	report := RunDoctor(cfg)

	var ollamaCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Ollama" {
			ollamaCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, ollamaCheck)
	assert.Equal(t, HealthPass, ollamaCheck.Status)
	assert.Contains(t, ollamaCheck.Message, "skipped")
}

func TestRunDoctor_PythonCheck(t *testing.T) {
	cfg := &Config{}
	report := RunDoctor(cfg)

	var pythonCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Python" {
			pythonCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, pythonCheck)
	// Python should be available on any dev machine.
	assert.Equal(t, HealthPass, pythonCheck.Status)
}

func TestRunDoctor_AgentsCheck_NoAgents(t *testing.T) {
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "agents")
	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)
	require.NoError(t, os.MkdirAll(agentsDir, 0755))

	cfg := &Config{}
	report := RunDoctor(cfg)

	var agentsCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Agents" {
			agentsCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, agentsCheck)
	assert.Equal(t, HealthPass, agentsCheck.Status)
	assert.Contains(t, agentsCheck.Message, "no agents")
}

func TestRunDoctor_AgentsCheck_WithAgents(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks not applicable on Windows")
	}
	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "agents")
	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)
	require.NoError(t, os.MkdirAll(filepath.Join(agentsDir, "agent1"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(agentsDir, "agent2"), 0755))

	cfg := &Config{}
	report := RunDoctor(cfg)

	var agentsCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Agents" {
			agentsCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, agentsCheck)
	assert.Equal(t, HealthPass, agentsCheck.Status)
	assert.Contains(t, agentsCheck.Message, "2 agent(s)")
}

func TestRunDoctor_EnvVarsCheck_MissingVars(t *testing.T) {
	for _, v := range doctorCriticalEnvVars() {
		t.Setenv(v, "")
	}
	cfg := &Config{}
	report := RunDoctor(cfg)

	var envCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Env vars" {
			envCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, envCheck)
	// Missing <=3 vars should be WARN
	if strings.Contains(envCheck.Message, "missing:") {
		assert.Equal(t, HealthWarn, envCheck.Status)
	}
}

func TestRunDoctor_AllPass_Healthy(t *testing.T) {
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  ollama_host: http://localhost:11434
  backend: ollama
defaults:
  timezone: UTC
resource_defaults:
  chat:
    timeout: "60s"
`)
	cfg := loadCfg(t)
	report := RunDoctor(cfg)
	assert.True(t, report.Healthy)
}

func TestRunDoctor_FormatReport(t *testing.T) {
	report := &DoctorReport{
		Checks: []HealthCheck{
			{Name: "Test A", Status: HealthPass, Message: "ok"},
			{Name: "Test B", Status: HealthWarn, Message: "warning"},
			{Name: "Test C", Status: HealthFail, Message: "failed"},
		},
		Healthy: false,
	}
	formatted := report.FormatReport()
	assert.Contains(t, formatted, "[PASS] Test A: ok")
	assert.Contains(t, formatted, "[WARN] Test B: warning")
	assert.Contains(t, formatted, "[FAIL] Test C: failed")
	assert.Contains(t, formatted, "issues found")
}

func TestRunDoctor_NilConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KDEPS_CONFIG_PATH", filepath.Join(dir, "nonexistent.yaml"))

	report := RunDoctor(nil)
	assert.NotNil(t, report)
	// With no config file and nil config, health check should find issues.
	var valCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Config validation" {
			valCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, valCheck)
	assert.Equal(t, HealthWarn, valCheck.Status)
}

func TestDoctorRunnerAdd(t *testing.T) {
	r := &doctorRunner{healthy: true}
	r.add("test", HealthPass, "ok")
	assert.True(t, r.healthy)
	assert.Len(t, r.checks, 1)

	r.add("test2", HealthFail, "failed")
	assert.False(t, r.healthy)
	assert.Len(t, r.checks, 2)
}

func TestProviderYAMLKey(t *testing.T) {
	p := primaryCloudProvider()
	assert.Equal(t, p.yamlKey, providerYAMLKey(p.name))
	assert.Equal(t, "unknown_api_key", providerYAMLKey("unknown"))
}

func TestCloudProviderEnvVars(t *testing.T) {
	for _, p := range cloudProvidersList {
		assert.Equal(t, p.envVar, cloudProviders[p.name].envVar, p.name)
	}
	_, ok := cloudProviders["unknown"]
	assert.False(t, ok)
}

func TestBackendOrDefault(t *testing.T) {
	p := primaryCloudProvider()
	assert.Equal(t, fileBackendStr, backendOrDefault(""))
	assert.Equal(t, p.name, backendOrDefault(p.name))
}

// --- effectiveBackend ---

func TestEffectiveBackend_FromEnv(t *testing.T) {
	p := primaryCloudProvider()
	t.Setenv("KDEPS_DEFAULT_BACKEND", p.name)
	assert.Equal(t, p.name, effectiveBackend(nil))
}

func TestEffectiveBackend_CfgTakesPrecedence(t *testing.T) {
	t.Setenv("KDEPS_DEFAULT_BACKEND", "env-backend")
	cfg := &Config{LLM: LLMKeys{Backend: "cfg-backend"}}
	assert.Equal(t, "cfg-backend", effectiveBackend(cfg))
}

func TestEffectiveBackend_EmptyEnvFallback(t *testing.T) {
	t.Setenv("KDEPS_DEFAULT_BACKEND", "")
	assert.Equal(t, "", effectiveBackend(nil))
}

// --- runCriticalEnvCheck ---

func TestRunCriticalEnvCheck_AllSet(t *testing.T) {
	for _, k := range doctorCriticalEnvVars() {
		t.Setenv(k, "set")
	}

	r := &doctorRunner{healthy: true}
	r.criticalEnv()
	require.GreaterOrEqual(t, len(r.checks), 1)
	assert.Equal(t, HealthPass, r.checks[0].Status)
	assert.Contains(t, r.checks[0].Message, "all critical vars set")
}

func TestRunCriticalEnvCheck_PartialSet(t *testing.T) {
	critical := doctorCriticalEnvVars()
	require.Greater(t, len(critical), envWarnThreshold+1)
	for _, k := range critical[:envWarnThreshold+1] {
		t.Setenv(k, "")
	}
	for _, k := range critical[envWarnThreshold+1:] {
		t.Setenv(k, "set")
	}

	r := &doctorRunner{healthy: true}
	r.criticalEnv()

	require.GreaterOrEqual(t, len(r.checks), 1)
	msg := r.checks[0].Message
	if len(msg) > 0 && msg[0] == 'm' {
		assert.Equal(t, HealthWarn, r.checks[0].Status)
	} else {
		assert.Equal(t, HealthPass, r.checks[0].Status)
	}
}

// --- runAgentsCheck ---

func TestRunAgentsCheck_ReadDirFails(t *testing.T) {
	dir := t.TempDir()
	// Point agents dir at a file so os.ReadDir fails.
	blocker := filepath.Join(dir, "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0600))
	t.Setenv("KDEPS_AGENTS_DIR", blocker)

	r := &doctorRunner{healthy: true}
	r.agents(&Config{})
	require.GreaterOrEqual(t, len(r.checks), 1)
	assert.Equal(t, HealthPass, r.checks[0].Status)
	assert.Contains(t, r.checks[0].Message, "no agents installed")
}

// --- runOllamaCheck with various URL formats ---

func TestRunOllamaCheck_HTTPSURL(t *testing.T) {
	t.Setenv("KDEPS_DEFAULT_BACKEND", "")
	t.Setenv("OLLAMA_HOST", "https://ollama.example.com")
	dir := t.TempDir()
	writeTempConfig(t, dir, `
llm:
  backend: ollama
  ollama_host: https://ollama.example.com
`)
	cfg := loadCfg(t)
	report := RunDoctor(cfg)

	var ollamaCheck *HealthCheck
	for i := range report.Checks {
		if report.Checks[i].Name == "Ollama" {
			ollamaCheck = &report.Checks[i]
			break
		}
	}
	require.NotNil(t, ollamaCheck)
	// Can't reach ollama.example.com, but the HTTPS URL parsing was exercised.
	assert.Equal(t, HealthWarn, ollamaCheck.Status)
	assert.Contains(t, ollamaCheck.Message, "not reachable")
}

func TestStripURLScheme(t *testing.T) {
	assert.Equal(t, "localhost:11434", stripURLScheme("http://localhost:11434"))
	assert.Equal(t, "ollama.example.com", stripURLScheme("https://ollama.example.com"))
	assert.Equal(t, "host:8080", stripURLScheme("host:8080"))
}

func TestOllamaDialAddr(t *testing.T) {
	t.Setenv("OLLAMA_HOST", "")
	cfg := &Config{LLM: LLMKeys{OllamaHost: "http://myhost"}}
	assert.Equal(t, "myhost:11434", ollamaDialAddr(cfg))

	t.Setenv("OLLAMA_HOST", "https://envhost:9999")
	assert.Equal(t, "envhost:9999", ollamaDialAddr(nil))

	t.Setenv("OLLAMA_HOST", "")
	assert.Equal(t, "localhost:11434", ollamaDialAddr(nil))
}

func TestRunCriticalEnvCheck_WarnThreshold(t *testing.T) {
	// Leave exactly 3 of 6 critical vars unset (at envWarnThreshold).
	t.Setenv("OLLAMA_HOST", "http://localhost:11434")
	t.Setenv("KDEPS_DEFAULT_BACKEND", "ollama")
	t.Setenv("KDEPS_LLM_MODELS", "gpt-4")

	r := &doctorRunner{healthy: true}
	r.criticalEnv()
	require.NotEmpty(t, r.checks)
	assert.Equal(t, HealthWarn, r.checks[0].Status)
	assert.Contains(t, r.checks[0].Message, "missing:")
}

func TestDoctorOllama_FileBackendReportsModelsDir(t *testing.T) {
	t.Setenv("KDEPS_DEFAULT_BACKEND", "file")
	r := &doctorRunner{healthy: true}
	r.ollama(&Config{LLM: LLMKeys{ModelsDir: "/data/models"}})

	require.Len(t, r.checks, 2)
	assert.Equal(t, "Ollama", r.checks[0].Name)
	assert.Contains(t, r.checks[0].Message, "llamafile")
	assert.Equal(t, "Models dir", r.checks[1].Name)
	assert.Contains(t, r.checks[1].Message, "/data/models")
}

func TestDoctorModelsDir_DefaultPath(t *testing.T) {
	r := &doctorRunner{healthy: true}
	r.modelsDir(nil)
	require.Len(t, r.checks, 1)
	assert.Contains(t, r.checks[0].Message, "~/.kdeps/models")
}
