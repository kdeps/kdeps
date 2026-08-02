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

package config

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
)

//nolint:gochecknoglobals // test-replaceable
var isStdinTerminal = func() bool { return term.IsTerminal(int(os.Stdin.Fd())) }

// providerNames returns the ordered list of supported LLM provider names.
// "ollama" is the local option (no API key needed).
func providerNames() []string {
	return allProviderNames
}

// Bootstrap writes an initial ~/.kdeps/config.yaml by interactively asking the
// user for their LLM provider and API key. It is called automatically on first
// run when the config file does not yet exist and stdin is a terminal.
//
// In non-interactive environments (CI, pipes) it falls back to Scaffold(),
// writing the commented template and returning without prompting.
// Set KDEPS_SKIP_BOOTSTRAP=1 to suppress all bootstrapping (useful in tests
// that override HOME to a temp directory that has no config file).
func Bootstrap(out *os.File) error {
	if os.Getenv("KDEPS_SKIP_BOOTSTRAP") == "1" {
		return nil
	}

	path, err := Path()
	if err != nil {
		return nil //nolint:nilerr // non-fatal
	}
	if _, statErr := AppFS.Stat(path); statErr == nil {
		return nil // config already exists
	}

	if !isStdinTerminal() {
		// Non-interactive: write template and continue silently.
		return Scaffold()
	}

	// Heal a terminal a prior program left in raw mode, so the prompt below
	// reads a line normally instead of echoing "^M"/"^C" on every key.
	sanitizeTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)
	return bootstrapInteractive(out, reader, path)
}

// backendChoice describes one entry in the bootstrap backend-type menu.
type backendChoice struct {
	kind  string // file | gguf | cloud | ollama | router
	label string
	desc  string
}

// backendChoices are the LLM backend types offered during bootstrap, in order.
//
//nolint:gochecknoglobals // static menu
var backendChoices = []backendChoice{
	{fileBackendStr, "llamafile", "local, self-contained, no server install (recommended)"},
	{ggufBackendStr, "gguf", "local GGUF models via llama.cpp"},
	{"cloud", "cloud", "OpenAI, Anthropic, DeepSeek, Groq, xAI, ... (needs an API key)"},
	{ollamaBackendStr, "ollama", "connect to an Ollama server"},
	{"router", "router", "route across multiple models by strategy (advanced)"},
	{m365BackendStr, "m365", "Microsoft 365 Copilot (browser login, no API key)"},
}

// routerStrategies are the selectable LLM router strategies, in menu order.
//
//nolint:gochecknoglobals // static menu
var routerStrategies = []string{
	strategyFallback, strategyRoundRobin, strategyTokenThreshold, strategyCostOptimized,
}

// bootstrapInteractive runs the interactive setup wizard.
// Separated from Bootstrap so it can be tested without a real TTY.
func bootstrapInteractive(out io.StringWriter, reader *bufio.Reader, path string) error {
	w := &fmtWriter{out}

	w.println("")
	w.println("  Welcome to kdeps!")
	w.println("  No configuration found. Let's set up ~/.kdeps/config.yaml.")
	w.println("")
	w.println("  How should kdeps run language models?")
	for i, bt := range backendChoices {
		w.printf("    [%d] %-10s %s\n", i+1, bt.label, bt.desc)
	}
	w.println("    [0] Skip (configure later)")
	w.println("")

	choice := promptLine(out, reader, "  Enter number [1]: ", "1")

	var cfg Config
	if err := configureBackendChoice(out, reader, w, &cfg, choice); err != nil {
		return err
	}

	if writeErr := writeConfig(path, cfg); writeErr != nil {
		return writeErr
	}

	w.println("")
	w.printf("  ✓ Configuration written to %s\n", path)
	w.println("  You can edit it at any time to change models or backends.")
	w.println("")

	applyEnv(cfg)
	return nil
}

// configureBackendChoice dispatches on the backend-type menu number.
func configureBackendChoice(
	out io.StringWriter, reader *bufio.Reader, w *fmtWriter, cfg *Config, choice string,
) error {
	idx := 0
	if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 ||
		idx > len(backendChoices) {
		return nil //nolint:nilerr // invalid/blank menu input means "skip", not an error
	}
	switch backendChoices[idx-1].kind {
	case fileBackendStr:
		configureLlamafile(out, reader, cfg)
	case ggufBackendStr:
		configureGGUF(out, reader, cfg)
	case "cloud":
		return configureCloud(out, reader, w, cfg)
	case ollamaBackendStr:
		configureOllama(out, reader, cfg)
	case m365BackendStr:
		configureM365(out, reader, cfg)
	case "router":
		configureRouter(out, reader, w, cfg)
	}
	return nil
}

// configureLlamafile sets the local file backend and a default model.
func configureLlamafile(out io.StringWriter, reader *bufio.Reader, cfg *Config) {
	cfg.LLM.Backend = fileBackendStr
	model := promptLine(out, reader, "  Default model [llama3.2:1b]: ", "llama3.2:1b")
	setDefaultModel(cfg, model)
}

// configureGGUF sets the gguf backend and an optional default model.
func configureGGUF(out io.StringWriter, reader *bufio.Reader, cfg *Config) {
	cfg.LLM.Backend = ggufBackendStr
	model := promptLine(
		out,
		reader,
		"  Default GGUF model alias or path (blank to set per resource): ",
		"",
	)
	setDefaultModel(cfg, model)
}

// configureCloud picks a cloud provider, its API key, and an optional model.
func configureCloud(out io.StringWriter, reader *bufio.Reader, w *fmtWriter, cfg *Config) error {
	w.println("")
	w.println("  Which cloud provider?")
	for i, p := range cloudProvidersList {
		w.printf("    [%d] %s\n", i+1, p.name)
	}
	choice := promptLine(out, reader, "  Enter number [1]: ", "1")
	idx := 1
	if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 ||
		idx > len(cloudProvidersList) {
		idx = 1
	}
	p := cloudProvidersList[idx-1]
	cfg.LLM.Backend = p.name

	w.printf("\n  Enter your %s API key (input hidden): ", p.name)
	apiKey, readErr := readSecretFunc(reader)
	w.println("")
	if readErr != nil {
		return readErr
	}
	if strings.TrimSpace(apiKey) != "" {
		p.setOnConfig(cfg, strings.TrimSpace(apiKey))
	}

	model := promptLine(out, reader, "  Default model (blank to set per resource): ", "")
	setDefaultModel(cfg, model)
	return nil
}

// configureOllama sets the ollama backend, host, and an optional model.
func configureOllama(out io.StringWriter, reader *bufio.Reader, cfg *Config) {
	cfg.LLM.Backend = ollamaBackendStr
	host := promptLine(
		out,
		reader,
		"  Ollama host URL [http://localhost:11434]: ",
		"http://localhost:11434",
	)
	if strings.TrimSpace(host) != "" {
		cfg.LLM.OllamaHost = strings.TrimSpace(host)
	}
	model := promptLine(
		out,
		reader,
		"  Default model (e.g. llama3.2, blank to set per resource): ",
		"",
	)
	setDefaultModel(cfg, model)
}

// configureM365 sets the m365 backend and an optional default model. No API
// key is collected: M365 Copilot authenticates via a cached browser login
// token, not an environment-variable key.
func configureM365(out io.StringWriter, reader *bufio.Reader, cfg *Config) {
	cfg.LLM.Backend = m365BackendStr
	model := promptLine(
		out,
		reader,
		"  Default model [m365-copilot]: ",
		"m365-copilot",
	)
	setDefaultModel(cfg, model)
}

// configureRouter collects the models to route across and a strategy.
func configureRouter(out io.StringWriter, reader *bufio.Reader, w *fmtWriter, cfg *Config) {
	w.println("")
	w.println("  Router: add the models to route across (blank model name to finish).")
	for {
		model := promptLine(out, reader, "  Model name (blank to finish): ", "")
		if model == "" {
			break
		}
		backend := promptLine(out, reader, "    Backend (blank to auto-detect from model): ", "")
		cfg.LLM.Models = append(cfg.LLM.Models, ModelEntry{Model: model, Backend: backend})
	}
	if len(cfg.LLM.Models) == 0 {
		return
	}
	w.println("  Routing strategy:")
	for i, s := range routerStrategies {
		w.printf("    [%d] %s\n", i+1, s)
	}
	choice := promptLine(out, reader, "  Enter number [1]: ", "1")
	idx := 1
	if _, err := fmt.Sscanf(choice, "%d", &idx); err != nil || idx < 1 ||
		idx > len(routerStrategies) {
		idx = 1
	}
	cfg.LLM.Strategy = routerStrategies[idx-1]
}

// setDefaultModel records a single default model so resources may omit `model:`.
func setDefaultModel(cfg *Config, model string) {
	if strings.TrimSpace(model) != "" {
		cfg.LLM.Models = ModelList{{Model: strings.TrimSpace(model)}}
	}
}

// fmtWriter wraps a WriteString-capable writer for fmt calls.
type fmtWriter struct {
	w io.StringWriter
}

func (f *fmtWriter) println(s string) { _, _ = f.w.WriteString(s + "\n") }
func (f *fmtWriter) printf(format string, args ...interface{}) {
	_, _ = f.w.WriteString(fmt.Sprintf(format, args...))
}

// promptLine prints prompt, reads a line, returns def if the line is blank.
func promptLine(out io.StringWriter, r *bufio.Reader, prompt, def string) string {
	_, _ = out.WriteString(prompt)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// readSecretFunc reads a secret from stdin; overridable for testing.
//
//nolint:gochecknoglobals // test-replaceable
var readSecretFunc = readSecret

// readSecret reads a line from stdin with echo disabled when possible.
func readSecret(fallback *bufio.Reader) (string, error) {
	if isStdinTerminal() {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		return string(b), err
	}
	line, err := fallback.ReadString('\n')
	return strings.TrimSpace(line), err
}

// writeConfig marshals cfg into YAML and writes it to path.
// User-filled values are written uncommented; all other template
// sections are appended as comments so users can discover them.
func writeConfig(path string, cfg Config) error {
	if mkdirErr := AppFS.MkdirAll(dirOf(path), configDirPerm); mkdirErr != nil {
		return fmt.Errorf("create config dir: %w", mkdirErr)
	}

	userFields := buildUserFields(cfg)
	content := userFields + "\n" + configOptionsReference()
	return afero.WriteFile(AppFS, path, []byte(content), configFilePerm)
}

// buildUserFields builds the YAML for fields the user set during bootstrap.
func buildUserFields(cfg Config) string {
	var lines []string
	lines = append(lines, "# kdeps global configuration — ~/.kdeps/config.yaml")
	lines = append(lines, "# Edit at any time. Explicit env vars always take precedence.")
	lines = append(lines, "")
	lines = append(lines, "llm:")
	appendField(&lines, "  backend", cfg.LLM.Backend)
	appendField(&lines, "  ollama_host", cfg.LLM.OllamaHost)
	appendField(&lines, "  models_dir", cfg.LLM.ModelsDir)
	for _, p := range cloudProvidersList {
		appendField(&lines, "  "+p.yamlKey, p.getKey(cfg.LLM))
	}
	appendModelsAndStrategy(&lines, cfg.LLM)
	if cfg.Defaults.Timezone != "" || cfg.Defaults.PythonVersion != "" {
		lines = append(lines, "defaults:")
		appendField(&lines, "  timezone", cfg.Defaults.Timezone)
		appendField(&lines, "  python_version", cfg.Defaults.PythonVersion)
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

func appendField(lines *[]string, key, value string) {
	if value == "" {
		*lines = append(*lines, "# "+key+": \"\"")
	} else {
		*lines = append(*lines, key+": "+yamlQuote(value))
	}
}

// appendModelsAndStrategy serializes llm.strategy and llm.models (a list) under
// the already-emitted `llm:` block.
func appendModelsAndStrategy(lines *[]string, llm LLMKeys) {
	if llm.Strategy != "" {
		*lines = append(*lines, "  strategy: "+yamlQuote(llm.Strategy))
	}
	if len(llm.Models) == 0 {
		return
	}
	data, err := yaml.Marshal(map[string]ModelList{"models": llm.Models})
	if err != nil {
		return
	}
	for _, ln := range strings.Split(strings.TrimRight(string(data), "\n"), "\n") {
		*lines = append(*lines, "  "+ln)
	}
}

func yamlQuote(s string) string {
	// Wrap in double quotes and escape any existing quotes.
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// dirOf returns the parent directory of path, recognizing both "/" and "\"
// as separators regardless of host OS: config paths in this package may
// come from user-supplied YAML written with either convention, not just
// os.PathSeparator (which alone missed forward-slash paths on Windows).
func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
