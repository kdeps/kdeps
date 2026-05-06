package config

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"
)

// HealthStatus represents the result of a health check.
type HealthStatus string

const (
	HealthPass HealthStatus = "PASS"
	HealthWarn HealthStatus = "WARN"
	HealthFail HealthStatus = "FAIL"
)

const (
	ollamaDialTimeout = 2 * time.Second
	defaultOllamaPort = "11434"
	minKeyMaskLen     = 8
	envWarnThreshold  = 3
)

// HealthCheck represents a single health check result.
type HealthCheck struct {
	Name    string
	Status  HealthStatus
	Message string
}

// DoctorReport holds the results of all health checks.
type DoctorReport struct {
	Checks  []HealthCheck
	Healthy bool
}

// RunDoctor runs all health checks and returns a report.
func RunDoctor(cfg *Config) *DoctorReport {
	report := &DoctorReport{Healthy: true}
	checks := &report.Checks

	runConfigFileCheck(checks, &report.Healthy)
	runConfigValidationCheck(checks, cfg, &report.Healthy)
	runOllamaCheck(checks, cfg, &report.Healthy)
	runPythonCheck(checks, &report.Healthy)
	runBackendKeyCheck(checks, cfg, &report.Healthy)
	runAgentsCheck(checks, cfg, &report.Healthy)
	runCriticalEnvCheck(checks, &report.Healthy)

	return report
}

// FormatReport formats the doctor report as a human-readable string.
func (r *DoctorReport) FormatReport() string {
	var b strings.Builder
	b.WriteString("kdeps doctor\n")
	b.WriteString("=============\n\n")

	for _, c := range r.Checks {
		fmt.Fprintf(&b, "  [%s] %s: %s\n", c.Status, c.Name, c.Message)
	}

	b.WriteString("\n")
	if r.Healthy {
		b.WriteString("Overall: healthy\n")
	} else {
		b.WriteString("Overall: issues found — review warnings above\n")
	}
	return b.String()
}

func addCheck(checks *[]HealthCheck, name string, status HealthStatus, msg string, healthy *bool) {
	*checks = append(*checks, HealthCheck{Name: name, Status: status, Message: msg})
	if status == HealthFail {
		*healthy = false
	}
}

func runConfigFileCheck(checks *[]HealthCheck, healthy *bool) {
	path, _ := Path()
	if path == "" {
		addCheck(checks, "Config file", HealthFail, "cannot determine config path", healthy)
		return
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		addCheck(checks, "Config file", HealthWarn,
			fmt.Sprintf("%s not found — run 'kdeps edit' to create", path), healthy)
		return
	}
	addCheck(checks, "Config file", HealthPass, path, healthy)
}

func runConfigValidationCheck(checks *[]HealthCheck, cfg *Config, healthy *bool) {
	if cfg == nil {
		addCheck(checks, "Config validation", HealthWarn, "no config loaded", healthy)
		return
	}
	warnings := cfg.Validate("")
	if len(warnings) == 0 {
		addCheck(checks, "Config validation", HealthPass, "no warnings", healthy)
	} else {
		addCheck(checks, "Config validation", HealthWarn,
			fmt.Sprintf("%d warning(s): %s", len(warnings), warnings[0]), healthy)
	}
}

func runOllamaCheck(checks *[]HealthCheck, cfg *Config, healthy *bool) {
	backend := ollamaEffectiveBackend(cfg)
	if backend != "" && backend != ollamaBackendStr {
		addCheck(checks, "Ollama", HealthPass,
			fmt.Sprintf("skipped — backend is %s", backend), healthy)
		return
	}

	host := os.Getenv("OLLAMA_HOST")
	if host == "" && cfg != nil {
		host = cfg.LLM.OllamaHost
	}
	if host == "" {
		host = "http://localhost:11434"
	}

	addr := host
	if len(addr) > 7 && addr[:7] == "http://" {
		addr = addr[7:]
	} else if len(addr) > 8 && addr[:8] == "https://" {
		addr = addr[8:]
	}
	if _, _, splitErr := net.SplitHostPort(addr); splitErr != nil {
		addr = net.JoinHostPort(addr, defaultOllamaPort)
	}

	dialer := &net.Dialer{Timeout: ollamaDialTimeout}
	conn, err := dialer.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		addCheck(checks, "Ollama", HealthWarn,
			fmt.Sprintf("not reachable at %s — %v", addr, err), healthy)
		return
	}
	_ = conn.Close()
	addCheck(checks, "Ollama", HealthPass, fmt.Sprintf("reachable at %s", addr), healthy)
}

func runPythonCheck(checks *[]HealthCheck, healthy *bool) {
	if _, python3Err := exec.LookPath("python3"); python3Err == nil {
		addCheck(checks, "Python", HealthPass, "python3 available", healthy)
	} else if _, pythonErr := exec.LookPath("python"); pythonErr == nil {
		addCheck(checks, "Python", HealthPass, "python available", healthy)
	} else {
		addCheck(checks, "Python", HealthWarn,
			"python not found in PATH — python resources will fail", healthy)
	}
}

func runBackendKeyCheck(checks *[]HealthCheck, cfg *Config, healthy *bool) {
	if cfg == nil {
		return
	}
	backend := cfg.LLM.Backend
	if backend == "" {
		backend = os.Getenv("KDEPS_DEFAULT_BACKEND")
	}
	if backend == "" || backend == ollamaBackendStr {
		addCheck(checks, "Backend/API key", HealthPass,
			fmt.Sprintf("backend=%s (no API key needed)", backendOrDefault(backend)), healthy)
		return
	}

	key := getLLMAPIKey(cfg.LLM, backend)
	if key == "" {
		key = os.Getenv(backendToEnvVar(backend))
	}

	if key != "" {
		masked := key[:4] + "..." + key[len(key)-4:]
		if len(key) <= minKeyMaskLen {
			masked = "****"
		}
		addCheck(checks, "Backend/API key", HealthPass,
			fmt.Sprintf("backend=%s, key=%s", backend, masked), healthy)
	} else {
		addCheck(checks, "Backend/API key", HealthWarn,
			fmt.Sprintf("backend=%s but no API key set for %s",
				backend, backendToKeyName(backend)), healthy)
	}
}

func runAgentsCheck(checks *[]HealthCheck, cfg *Config, healthy *bool) {
	agentsDir, err := AgentsDir(cfg)
	if err != nil {
		addCheck(checks, "Agents", HealthWarn, fmt.Sprintf("cannot resolve agents dir: %v", err), healthy)
		return
	}
	entries, readErr := os.ReadDir(agentsDir)
	if readErr != nil {
		addCheck(checks, "Agents", HealthPass,
			fmt.Sprintf("no agents installed (%s)", agentsDir), healthy)
		return
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	if count == 0 {
		addCheck(checks, "Agents", HealthPass,
			fmt.Sprintf("no agents installed (%s)", agentsDir), healthy)
	} else {
		addCheck(checks, "Agents", HealthPass,
			fmt.Sprintf("%d agent(s) installed (%s)", count, agentsDir), healthy)
	}
}

func runCriticalEnvCheck(checks *[]HealthCheck, healthy *bool) {
	critical := []string{
		"OLLAMA_HOST", "KDEPS_DEFAULT_BACKEND", "KDEPS_LLM_MODELS",
		"OPENAI_API_KEY", "ANTHROPIC_API_KEY", "TZ",
	}
	missing := make([]string, 0)
	for _, v := range critical {
		if os.Getenv(v) == "" {
			missing = append(missing, v)
		}
	}
	switch {
	case len(missing) == 0:
		addCheck(checks, "Env vars", HealthPass, "all critical vars set", healthy)
	case len(missing) <= envWarnThreshold:
		addCheck(checks, "Env vars", HealthWarn,
			fmt.Sprintf("missing: %v", missing), healthy)
	default:
		addCheck(checks, "Env vars", HealthPass,
			fmt.Sprintf("%d critical vars not set (config file provides defaults)",
				len(missing)), healthy)
	}
}

func ollamaEffectiveBackend(cfg *Config) string {
	if cfg != nil && cfg.LLM.Backend != "" {
		return cfg.LLM.Backend
	}
	if b := os.Getenv("KDEPS_DEFAULT_BACKEND"); b != "" {
		return b
	}
	return "" // effectively ollama
}

func backendOrDefault(backend string) string {
	if backend == "" {
		return ollamaBackendStr
	}
	return backend
}

func backendToKeyName(backend string) string {
	if k, ok := backendToKey[backend]; ok {
		return k
	}
	return backend + "_api_key"
}

func backendToEnvVar(backend string) string {
	envMap := map[string]string{
		"openai":     "OPENAI_API_KEY",
		"anthropic":  "ANTHROPIC_API_KEY",
		"google":     "GOOGLE_API_KEY",
		"cohere":     "COHERE_API_KEY",
		"mistral":    "MISTRAL_API_KEY",
		"together":   "TOGETHER_API_KEY",
		"perplexity": "PERPLEXITY_API_KEY",
		"groq":       "GROQ_API_KEY",
		"deepseek":   "DEEPSEEK_API_KEY",
		"openrouter": "OPENROUTER_API_KEY",
	}
	if v, ok := envMap[backend]; ok {
		return v
	}
	return ""
}
