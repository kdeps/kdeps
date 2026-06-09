package config

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// HealthStatus represents the result of a health check.
type HealthStatus string

const (
	HealthPass HealthStatus = "PASS"
	HealthWarn HealthStatus = "WARN"
	HealthFail HealthStatus = "FAIL"
)

// DI variables — overridable for testing. osGetenv/osUserHomeDir kept
// (no afero equivalent for home dir / env vars). AppFS defined in config.go.

//nolint:gochecknoglobals // test-replaceable
var osGetenv = os.Getenv

//nolint:gochecknoglobals // test-replaceable
var execLookPath = exec.LookPath

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

type doctorRunner struct {
	checks  []HealthCheck
	healthy bool
}

func (r *doctorRunner) add(name string, status HealthStatus, msg string) {
	r.checks = append(r.checks, HealthCheck{Name: name, Status: status, Message: msg})
	if status == HealthFail {
		r.healthy = false
	}
}

// RunDoctor runs all health checks and returns a report.
func RunDoctor(cfg *Config) *DoctorReport {
	r := &doctorRunner{healthy: true}
	r.configFile()
	r.configValidation(cfg)
	r.ollama(cfg)
	r.python()
	r.backendKey(cfg)
	r.agents(cfg)
	r.criticalEnv()
	return &DoctorReport{Checks: r.checks, Healthy: r.healthy}
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

func (r *doctorRunner) configFile() {
	path, _ := Path()
	if path == "" {
		r.add("Config file", HealthFail, "cannot determine config path")
		return
	}
	if _, err := AppFS.Stat(path); os.IsNotExist(err) {
		r.add("Config file", HealthWarn,
			fmt.Sprintf("%s not found — run 'kdeps edit' to create", path))
		return
	}
	r.add("Config file", HealthPass, path)
}

func (r *doctorRunner) configValidation(cfg *Config) {
	if cfg == nil {
		r.add("Config validation", HealthWarn, "no config loaded")
		return
	}
	warnings := cfg.Validate("")
	if len(warnings) == 0 {
		r.add("Config validation", HealthPass, "no warnings")
		return
	}
	r.add("Config validation", HealthWarn,
		fmt.Sprintf("%d warning(s): %s", len(warnings), warnings[0]))
}

func (r *doctorRunner) ollama(cfg *Config) {
	backend := effectiveBackend(cfg)
	if backend != "" && backend != ollamaBackendStr {
		r.add("Ollama", HealthPass, fmt.Sprintf("skipped — backend is %s", backend))
		return
	}

	addr := ollamaDialAddr(cfg)
	dialer := &net.Dialer{Timeout: ollamaDialTimeout}
	conn, err := dialer.DialContext(context.Background(), "tcp", addr)
	if err != nil {
		r.add("Ollama", HealthWarn, fmt.Sprintf("not reachable at %s — %v", addr, err))
		return
	}
	_ = conn.Close()
	r.add("Ollama", HealthPass, fmt.Sprintf("reachable at %s", addr))
}

func (r *doctorRunner) python() {
	if _, err := execLookPath("python3"); err == nil {
		r.add("Python", HealthPass, "python3 available")
		return
	}
	if _, err := execLookPath("python"); err == nil {
		r.add("Python", HealthPass, "python available")
		return
	}
	r.add("Python", HealthWarn, "python not found in PATH — python resources will fail")
}

func (r *doctorRunner) backendKey(cfg *Config) {
	if cfg == nil {
		return
	}
	backend := effectiveBackend(cfg)
	if backend == "" || backend == ollamaBackendStr {
		r.add("Backend/API key", HealthPass,
			fmt.Sprintf("backend=%s (no API key needed)", backendOrDefault(backend)))
		return
	}

	key := getLLMAPIKey(cfg.LLM, backend)
	if key == "" {
		if p, ok := cloudProviders[backend]; ok {
			key = osGetenv(p.envVar)
		}
	}

	if key != "" {
		r.add("Backend/API key", HealthPass,
			fmt.Sprintf("backend=%s, key=%s", backend, maskAPIKey(key)))
		return
	}
	r.add("Backend/API key", HealthWarn,
		fmt.Sprintf("backend=%s but no API key set for %s",
			backend, providerYAMLKey(backend)))
}

func (r *doctorRunner) agents(cfg *Config) {
	agentsDir, err := AgentsDir(cfg)
	if err != nil {
		r.add("Agents", HealthWarn, fmt.Sprintf("cannot resolve agents dir: %v", err))
		return
	}
	entries, readErr := afero.ReadDir(AppFS, agentsDir)
	if readErr != nil {
		r.add("Agents", HealthPass, fmt.Sprintf("no agents installed (%s)", agentsDir))
		return
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			count++
		}
	}
	if count == 0 {
		r.add("Agents", HealthPass, fmt.Sprintf("no agents installed (%s)", agentsDir))
		return
	}
	r.add("Agents", HealthPass, fmt.Sprintf("%d agent(s) installed (%s)", count, agentsDir))
}

func (r *doctorRunner) criticalEnv() {
	critical := []string{"OLLAMA_HOST", "KDEPS_DEFAULT_BACKEND", "KDEPS_LLM_MODELS", "TZ"}
	for _, p := range cloudProvidersList {
		if p.doctorSpotCheck {
			critical = append(critical, p.envVar)
		}
	}
	missing := make([]string, 0)
	for _, v := range critical {
		if osGetenv(v) == "" {
			missing = append(missing, v)
		}
	}
	switch {
	case len(missing) == 0:
		r.add("Env vars", HealthPass, "all critical vars set")
	case len(missing) <= envWarnThreshold:
		r.add("Env vars", HealthWarn, fmt.Sprintf("missing: %v", missing))
	default:
		r.add("Env vars", HealthPass,
			fmt.Sprintf("%d critical vars not set (config file provides defaults)",
				len(missing)))
	}
}

// effectiveBackend returns the configured LLM backend from config or env.
func effectiveBackend(cfg *Config) string {
	if cfg != nil && cfg.LLM.Backend != "" {
		return cfg.LLM.Backend
	}
	if b := osGetenv("KDEPS_DEFAULT_BACKEND"); b != "" {
		return b
	}
	return ""
}

// ollamaDialAddr resolves the TCP address used to probe Ollama reachability.
func ollamaDialAddr(cfg *Config) string {
	host := osGetenv("OLLAMA_HOST")
	if host == "" && cfg != nil {
		host = cfg.LLM.OllamaHost
	}
	if host == "" {
		host = "http://localhost:11434"
	}

	addr := stripURLScheme(host)
	if _, _, err := net.SplitHostPort(addr); err != nil {
		addr = net.JoinHostPort(addr, defaultOllamaPort)
	}
	return addr
}

func stripURLScheme(host string) string {
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	return host
}

// maskAPIKey returns a partially redacted API key for display.
func maskAPIKey(key string) string {
	if len(key) <= minKeyMaskLen {
		return "****"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

func backendOrDefault(backend string) string {
	if backend == "" {
		return ollamaBackendStr
	}
	return backend
}
