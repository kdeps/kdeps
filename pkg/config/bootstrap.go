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
)

// providerKey maps a friendly provider name to its API key field and env var.
type providerKey struct {
	envVar string
	setter func(*Config, string)
}

// providerNames returns the ordered list of supported LLM provider names.
// "ollama" is the local option (no API key needed).
func providerNames() []string {
	return []string{
		"ollama",
		"openai", "anthropic", "google", "cohere",
		"mistral", "together", "perplexity", "groq", "deepseek", "openrouter",
	}
}

// providerMetaMap returns the metadata for each provider.
func providerMetaMap() map[string]providerKey {
	return map[string]providerKey{
		ollamaBackendStr: {"OLLAMA_HOST", func(c *Config, v string) { c.LLM.OllamaHost = v }},
		"openai":         {"OPENAI_API_KEY", func(c *Config, v string) { c.LLM.OpenAI = v }},
		"anthropic":      {"ANTHROPIC_API_KEY", func(c *Config, v string) { c.LLM.Anthropic = v }},
		"google":         {"GOOGLE_API_KEY", func(c *Config, v string) { c.LLM.Google = v }},
		"cohere":         {"COHERE_API_KEY", func(c *Config, v string) { c.LLM.Cohere = v }},
		"mistral":        {"MISTRAL_API_KEY", func(c *Config, v string) { c.LLM.Mistral = v }},
		"together":       {"TOGETHER_API_KEY", func(c *Config, v string) { c.LLM.Together = v }},
		"perplexity":     {"PERPLEXITY_API_KEY", func(c *Config, v string) { c.LLM.Perplexity = v }},
		"groq":           {"GROQ_API_KEY", func(c *Config, v string) { c.LLM.Groq = v }},
		"deepseek":       {"DEEPSEEK_API_KEY", func(c *Config, v string) { c.LLM.DeepSeek = v }},
		"openrouter":     {"OPENROUTER_API_KEY", func(c *Config, v string) { c.LLM.OpenRouter = v }},
	}
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
	if _, statErr := os.Stat(path); statErr == nil {
		return nil // config already exists
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Non-interactive: write template and continue silently.
		return Scaffold()
	}

	reader := bufio.NewReader(os.Stdin)
	return bootstrapInteractive(out, reader, path)
}

// bootstrapInteractive runs the interactive setup wizard.
// Separated from Bootstrap so it can be tested without a real TTY.
func bootstrapInteractive(out io.StringWriter, reader *bufio.Reader, path string) error {
	w := &fmtWriter{out}

	w.println("")
	w.println("  Welcome to kdeps!")
	w.println("  No configuration found. Let's set up ~/.kdeps/config.yaml.")
	w.println("")

	// --- LLM provider selection ---
	w.println("  Which LLM provider do you want to use?")
	for i, p := range providerNames() {
		w.printf("    [%d] %s\n", i+1, p)
	}
	w.println("    [0] Skip (configure later)")
	w.println("")

	choice := promptLine(out, reader, "  Enter number [1]: ", "1")

	var cfg Config
	chosenProvider := resolveProviderChoice(choice)

	if chosenProvider != "" {
		if err := configureProvider(out, reader, w, &cfg, chosenProvider); err != nil {
			return err
		}
	}

	if writeErr := writeConfig(path, cfg); writeErr != nil {
		return writeErr
	}

	w.println("")
	w.printf("  ✓ Configuration written to %s\n", path)
	w.println("  You can edit it at any time to add more providers or change settings.")
	w.println("")

	applyEnv(cfg)
	return nil
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
	if term.IsTerminal(int(os.Stdin.Fd())) {
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
	if mkdirErr := os.MkdirAll(dirOf(path), configDirPerm); mkdirErr != nil {
		return fmt.Errorf("create config dir: %w", mkdirErr)
	}

	userFields := buildUserFields(cfg)
	content := userFields + "\n" + configOptionsReference
	return os.WriteFile(path, []byte(content), configFilePerm)
}

// buildUserFields builds the YAML for fields the user set during bootstrap.
func buildUserFields(cfg Config) string {
	var lines []string
	lines = append(lines, "# kdeps global configuration — ~/.kdeps/config.yaml")
	lines = append(lines, "# Edit at any time. Explicit env vars always take precedence.")
	lines = append(lines, "")
	lines = append(lines, "llm:")
	appendField(&lines, "  ollama_host", cfg.LLM.OllamaHost)
	appendField(&lines, "  models_dir", cfg.LLM.ModelsDir)
	appendField(&lines, "  openai_api_key", cfg.LLM.OpenAI)
	appendField(&lines, "  anthropic_api_key", cfg.LLM.Anthropic)
	appendField(&lines, "  google_api_key", cfg.LLM.Google)
	appendField(&lines, "  cohere_api_key", cfg.LLM.Cohere)
	appendField(&lines, "  mistral_api_key", cfg.LLM.Mistral)
	appendField(&lines, "  together_api_key", cfg.LLM.Together)
	appendField(&lines, "  perplexity_api_key", cfg.LLM.Perplexity)
	appendField(&lines, "  groq_api_key", cfg.LLM.Groq)
	appendField(&lines, "  deepseek_api_key", cfg.LLM.DeepSeek)
	appendField(&lines, "  openrouter_api_key", cfg.LLM.OpenRouter)
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

func yamlQuote(s string) string {
	// Wrap in double quotes and escape any existing quotes.
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

func dirOf(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == os.PathSeparator {
			return path[:i]
		}
	}
	return "."
}

// resolveProviderChoice maps a numeric menu selection to a provider name.
// Returns "" when the user skips (choice "0") or enters an invalid number.
func resolveProviderChoice(choice string) string {
	if choice == "0" {
		return ""
	}
	idx := 0
	if _, scanErr := fmt.Sscanf(choice, "%d", &idx); scanErr != nil || idx < 1 || idx > len(providerNames()) {
		return ""
	}
	return providerNames()[idx-1]
}

// configureProvider handles interactive setup for one provider (ollama or online).
func configureProvider(
	out io.StringWriter, reader *bufio.Reader, w *fmtWriter, cfg *Config, chosenProvider string,
) error {
	meta := providerMetaMap()[chosenProvider]
	if chosenProvider == ollamaBackendStr {
		hostRaw := promptLine(out, reader, "  Ollama host URL [http://localhost:11434]: ", "http://localhost:11434")
		if strings.TrimSpace(hostRaw) != "" {
			meta.setter(cfg, strings.TrimSpace(hostRaw))
		}
		return nil
	}
	w.printf("\n  Enter your %s API key (input hidden): ", chosenProvider)
	apiKey, readErr := readSecretFunc(reader)
	w.println("")
	if readErr != nil {
		return readErr
	}
	if strings.TrimSpace(apiKey) != "" {
		meta.setter(cfg, strings.TrimSpace(apiKey))
	}
	return nil
}
