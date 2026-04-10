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
func providerNames() []string {
	return []string{
		"openai", "anthropic", "google", "cohere",
		"mistral", "together", "perplexity", "groq", "deepseek", "openrouter",
	}
}

// providerMetaMap returns the metadata for each provider.
func providerMetaMap() map[string]providerKey {
	return map[string]providerKey{
		"openai":     {"OPENAI_API_KEY", func(c *Config, v string) { c.LLM.OpenAI = v }},
		"anthropic":  {"ANTHROPIC_API_KEY", func(c *Config, v string) { c.LLM.Anthropic = v }},
		"google":     {"GOOGLE_API_KEY", func(c *Config, v string) { c.LLM.Google = v }},
		"cohere":     {"COHERE_API_KEY", func(c *Config, v string) { c.LLM.Cohere = v }},
		"mistral":    {"MISTRAL_API_KEY", func(c *Config, v string) { c.LLM.Mistral = v }},
		"together":   {"TOGETHER_API_KEY", func(c *Config, v string) { c.LLM.Together = v }},
		"perplexity": {"PERPLEXITY_API_KEY", func(c *Config, v string) { c.LLM.Perplexity = v }},
		"groq":       {"GROQ_API_KEY", func(c *Config, v string) { c.LLM.Groq = v }},
		"deepseek":   {"DEEPSEEK_API_KEY", func(c *Config, v string) { c.LLM.DeepSeek = v }},
		"openrouter": {"OPENROUTER_API_KEY", func(c *Config, v string) { c.LLM.OpenRouter = v }},
	}
}

// Bootstrap writes an initial ~/.kdeps/config.yaml by interactively asking the
// user for their LLM provider and API key. It is called automatically on first
// run when the config file does not yet exist and stdin is a terminal.
//
// In non-interactive environments (CI, pipes) it falls back to Scaffold(),
// writing the commented template and returning without prompting.
func Bootstrap(out *os.File) error {
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

	fmt.Fprintln(out, "")
	fmt.Fprintln(out, "  Welcome to kdeps!")
	fmt.Fprintln(out, "  No configuration found. Let's set up ~/.kdeps/config.yaml.")
	fmt.Fprintln(out, "")

	// --- LLM provider selection ---
	fmt.Fprintln(out, "  Which LLM provider do you want to use?")
	for i, p := range providerNames() {
		fmt.Fprintf(out, "    [%d] %s\n", i+1, p)
	}
	fmt.Fprintln(out, "    [0] Skip (configure later)")
	fmt.Fprintln(out, "")

	reader := bufio.NewReader(os.Stdin)
	choice := promptLine(out, reader, "  Enter number [1]: ", "1")

	var cfg Config
	var chosenProvider string

	if choice != "0" {
		idx := 0
		if _, scanErr := fmt.Sscanf(choice, "%d", &idx); scanErr == nil && idx >= 1 && idx <= len(providerNames()) {
			chosenProvider = providerNames()[idx-1]
		}
	}

	if chosenProvider != "" {
		meta := providerMetaMap()[chosenProvider]
		fmt.Fprintf(out, "\n  Enter your %s API key (input hidden): ", chosenProvider)
		apiKey, readErr := readSecret(reader)
		fmt.Fprintln(out, "")
		if readErr == nil && strings.TrimSpace(apiKey) != "" {
			meta.setter(&cfg, strings.TrimSpace(apiKey))
		}
	}

	// --- Registry token (optional) ---
	fmt.Fprintln(out, "")
	tokenRaw := promptLine(out, reader, "  kdeps registry token (optional, press Enter to skip): ", "")
	if strings.TrimSpace(tokenRaw) != "" {
		cfg.Registry.Token = strings.TrimSpace(tokenRaw)
	}

	if writeErr := writeConfig(path, cfg); writeErr != nil {
		return writeErr
	}

	fmt.Fprintln(out, "")
	fmt.Fprintf(out, "  ✓ Configuration written to %s\n", path)
	fmt.Fprintln(out, "  You can edit it at any time to add more providers or change settings.")
	fmt.Fprintln(out, "")

	applyEnv(cfg)
	return nil
}

// promptLine prints prompt, reads a line, returns def if the line is blank.
func promptLine(out *os.File, r *bufio.Reader, prompt, def string) string {
	fmt.Fprint(out, prompt)
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// readSecret reads a line from stdin with echo disabled when possible.
func readSecret(fallback *bufio.Reader) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		return string(b), err
	}
	line, err := fallback.ReadString('\n')
	return strings.TrimSpace(line), err
}

// writeConfig marshals cfg into YAML and writes it to path, merging with the
// default template so all comment annotations are preserved.
func writeConfig(path string, cfg Config) error {
	if mkdirErr := os.MkdirAll(dirOf(path), configDirPerm); mkdirErr != nil {
		return fmt.Errorf("create config dir: %w", mkdirErr)
	}

	// Build a minimal YAML representation of filled-in fields.
	var lines []string
	lines = append(lines, "# kdeps global configuration — ~/.kdeps/config.yaml")
	lines = append(lines, "# Edit at any time. Explicit env vars always take precedence.")
	lines = append(lines, "")

	lines = append(lines, "llm:")
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

	lines = append(lines, "")
	lines = append(lines, "registry:")
	appendField(&lines, "  url", cfg.Registry.URL)
	appendField(&lines, "  token", cfg.Registry.Token)

	lines = append(lines, "")
	lines = append(lines, "storage:")
	appendField(&lines, "  agents_dir", cfg.Storage.AgentsDir)
	appendField(&lines, "  components_dir", cfg.Storage.ComponentsDir)
	lines = append(lines, "")

	content := strings.Join(lines, "\n")
	return os.WriteFile(path, []byte(content), configFilePerm)
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
