package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

//nolint:gochecknoglobals // read-only lookup tables for validation
var (
	knownTopLevelKeys = map[string]bool{
		"llm":               true,
		"defaults":          true,
		"resource_defaults": true,
		"agents":            true,
	}

	knownLLMKeys = map[string]bool{
		"ollama_host":        true,
		"backend":            true,
		"base_url":           true,
		"strategy":           true,
		"models":             true,
		"models_dir":         true,
		"openai_api_key":     true,
		"anthropic_api_key":  true,
		"google_api_key":     true,
		"cohere_api_key":     true,
		"mistral_api_key":    true,
		"together_api_key":   true,
		"perplexity_api_key": true,
		"groq_api_key":       true,
		"deepseek_api_key":   true,
		"openrouter_api_key": true,
	}

	knownDefaultsKeys = map[string]bool{
		"timezone":       true,
		"python_version": true,
		"offline_mode":   true,
	}

	knownResourceDefaultsKeys = map[string]bool{
		"chat":    true,
		"http":    true,
		"python":  true,
		"exec":    true,
		"sql":     true,
		"onError": true,
	}

	validStrategies = map[string]bool{
		"":                true,
		"token_threshold": true,
		"fallback":        true,
		"cost_optimized":  true,
		"round_robin":     true,
	}

	backendToKey = map[string]string{
		"openai":     "openai_api_key",
		"anthropic":  "anthropic_api_key",
		"google":     "google_api_key",
		"cohere":     "cohere_api_key",
		"mistral":    "mistral_api_key",
		"together":   "together_api_key",
		"perplexity": "perplexity_api_key",
		"groq":       "groq_api_key",
		"deepseek":   "deepseek_api_key",
		"openrouter": "openrouter_api_key",
	}
)

// getLLMAPIKey returns the value of the API key field for a given backend.
func getLLMAPIKey(llm LLMKeys, backend string) string {
	switch backend {
	case "openai":
		return llm.OpenAI
	case "anthropic":
		return llm.Anthropic
	case "google":
		return llm.Google
	case "cohere":
		return llm.Cohere
	case "mistral":
		return llm.Mistral
	case "together":
		return llm.Together
	case "perplexity":
		return llm.Perplexity
	case "groq":
		return llm.Groq
	case "deepseek":
		return llm.DeepSeek
	case "openrouter":
		return llm.OpenRouter
	}
	return ""
}

// Validate checks the config for common mistakes and returns human-readable
// warnings. Validation is non-fatal: the config is still usable even when
// warnings are returned.
func (c *Config) Validate(agentsDir string) []string {
	var warnings []string

	path, _ := Path()
	if path == "" {
		return warnings
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return warnings // file doesn't exist or unreadable, skip validation
	}

	warnings = append(warnings, validateUnknownKeys(data)...)
	warnings = append(warnings, c.validateValues()...)
	warnings = append(warnings, c.validateAgentProfiles(agentsDir)...)

	return warnings
}

// validateUnknownKeys re-parses the raw YAML to detect keys that don't
// correspond to any known config field (likely typos).
func validateUnknownKeys(data []byte) []string {
	var warnings []string

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return warnings
	}
	if len(doc.Content) == 0 {
		return warnings
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return warnings
	}

	// Check top-level keys.
	topUnknown := collectUnknownKeys(root, knownTopLevelKeys)
	for _, k := range topUnknown {
		warnings = append(warnings, fmt.Sprintf(
			"unknown top-level key %q — check for typos "+
				"(valid keys: llm, defaults, resource_defaults, agents)", k))
	}

	// Check llm: sub-keys.
	if llmNode := findMappingValue(root, "llm"); llmNode != nil {
		llmUnknown := collectUnknownKeys(llmNode, knownLLMKeys)
		for _, k := range llmUnknown {
			warnings = append(warnings, fmt.Sprintf(
				"unknown llm key %q — check for typos in API key or field name", k))
		}
	}

	// Check defaults: sub-keys.
	if defNode := findMappingValue(root, "defaults"); defNode != nil {
		defUnknown := collectUnknownKeys(defNode, knownDefaultsKeys)
		for _, k := range defUnknown {
			warnings = append(warnings, fmt.Sprintf(
				"unknown defaults key %q — "+
					"valid keys: timezone, python_version, offline_mode", k))
		}
	}

	// Check resource_defaults: sub-keys.
	if rdNode := findMappingValue(root, "resource_defaults"); rdNode != nil {
		rdUnknown := collectUnknownKeys(rdNode, knownResourceDefaultsKeys)
		for _, k := range rdUnknown {
			warnings = append(warnings, fmt.Sprintf(
				"unknown resource_defaults key %q — "+
					"valid keys: chat, http, python, exec, sql, onError", k))
		}
	}

	return warnings
}

// findMappingValue returns the value node for a given key in a mapping node.
func findMappingValue(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			v := mapping.Content[i+1]
			if v.Kind == yaml.MappingNode {
				return v
			}
		}
	}
	return nil
}

// collectUnknownKeys returns keys in mapping that are not in known.
func collectUnknownKeys(mapping *yaml.Node, known map[string]bool) []string {
	var unknown []string
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		k := mapping.Content[i].Value
		if !known[k] {
			unknown = append(unknown, k)
		}
	}
	return unknown
}

// validateValues checks field values for correctness.
func (c *Config) validateValues() []string {
	var warnings []string

	// Strategy must be a known value.
	if !validStrategies[c.LLM.Strategy] {
		valid := []string{
			"token_threshold", "fallback", "cost_optimized", "round_robin",
		}
		warnings = append(warnings, fmt.Sprintf(
			"llm.strategy %q is not a valid strategy — valid values: %s",
			c.LLM.Strategy, strings.Join(valid, ", ")))
	}

	// Backend set but no corresponding API key.
	if c.LLM.Backend != "" && c.LLM.Backend != "ollama" {
		if keyField := backendToKey[c.LLM.Backend]; keyField != "" {
			if getLLMAPIKey(c.LLM, c.LLM.Backend) == "" {
				warnings = append(warnings, fmt.Sprintf(
					"llm.backend is %q but llm.%s is not set",
					c.LLM.Backend, keyField))
			}
		}
	}

	// Duration fields.
	warnings = append(warnings,
		validateDuration("resource_defaults.chat.timeout", c.ResourceDefaults.Chat.Timeout)...)
	warnings = append(warnings,
		validateDuration("resource_defaults.http.timeout", c.ResourceDefaults.HTTP.Timeout)...)
	warnings = append(warnings,
		validateDuration("resource_defaults.http.retry_backoff", c.ResourceDefaults.HTTP.RetryBackoff)...)
	warnings = append(warnings,
		validateDuration("resource_defaults.http.retry_max_backoff", c.ResourceDefaults.HTTP.RetryMaxBackoff)...)
	warnings = append(warnings,
		validateDuration("resource_defaults.python.timeout", c.ResourceDefaults.Python.Timeout)...)
	warnings = append(warnings,
		validateDuration("resource_defaults.exec.timeout", c.ResourceDefaults.Exec.Timeout)...)
	warnings = append(warnings,
		validateDuration("resource_defaults.sql.timeout", c.ResourceDefaults.SQL.Timeout)...)
	warnings = append(warnings,
		validateDuration("resource_defaults.onError.retry_delay", c.ResourceDefaults.OnError.RetryDelay)...)

	return warnings
}

// validateDuration checks if a string parses as a valid Go duration.
func validateDuration(field, value string) []string {
	if value == "" {
		return nil
	}
	if _, err := time.ParseDuration(value); err != nil {
		return []string{fmt.Sprintf(
			"%s %q is not a valid duration (e.g. \"30s\", \"5m\")", field, value)}
	}
	return nil
}

// validateAgentProfiles checks that agent entries in the config reference
// installed workflows and are not empty.
func (c *Config) validateAgentProfiles(agentsDir string) []string {
	if len(c.Agents) == 0 {
		return nil
	}

	var warnings []string
	workflowNames := collectWorkflowNames(agentsDir)

	for name, profile := range c.Agents {
		if len(workflowNames) > 0 && !workflowNames[name] {
			warnings = append(warnings, fmt.Sprintf(
				"agents.%q does not match any installed workflow metadata.name", name))
		}

		if isEmptyAgentProfile(profile) {
			warnings = append(warnings, fmt.Sprintf(
				"agents.%q has no non-empty fields set — profile has no effect", name))
		}
	}

	return warnings
}

// isLLMKeysEmpty reports whether all LLM key fields are unset.
func isLLMKeysEmpty(llm LLMKeys) bool {
	return llm.OllamaHost == "" &&
		llm.Backend == "" &&
		llm.BaseURL == "" &&
		llm.Strategy == "" &&
		len(llm.Models) == 0 &&
		llm.ModelsDir == "" &&
		llm.OpenAI == "" &&
		llm.Anthropic == "" &&
		llm.Google == "" &&
		llm.Cohere == "" &&
		llm.Mistral == "" &&
		llm.Together == "" &&
		llm.Perplexity == "" &&
		llm.Groq == "" &&
		llm.DeepSeek == "" &&
		llm.OpenRouter == ""
}

// isDefaultsEmpty reports whether all global defaults are unset.
func isDefaultsEmpty(d Defaults) bool {
	return d.Timezone == "" && d.PythonVersion == "" && !d.OfflineMode
}

// isChatDefaultsEmpty reports whether all chat resource defaults are unset.
func isChatDefaultsEmpty(c ChatDefaults) bool {
	return c.Timeout == "" &&
		c.ContextLength == 0 &&
		!c.Streaming &&
		c.Temperature == nil &&
		c.MaxTokens == nil &&
		c.TopP == nil &&
		c.FrequencyPenalty == nil &&
		c.PresencePenalty == nil
}

// isHTTPDefaultsEmpty reports whether all HTTP resource defaults are unset.
func isHTTPDefaultsEmpty(h HTTPDefaults) bool {
	return h.Timeout == "" &&
		!h.FollowRedirects &&
		h.Proxy == "" &&
		h.RetryMaxAttempts == 0 &&
		h.RetryBackoff == "" &&
		h.RetryMaxBackoff == "" &&
		h.RetryOn == ""
}

// isResourceDefaultsEmpty reports whether all per-resource defaults are unset.
func isResourceDefaultsEmpty(rd ResourceDefaults) bool {
	return isChatDefaultsEmpty(rd.Chat) &&
		isHTTPDefaultsEmpty(rd.HTTP) &&
		rd.Python.Timeout == "" &&
		rd.Exec.Timeout == "" &&
		rd.SQL.Timeout == "" &&
		rd.SQL.MaxRows == 0 &&
		rd.OnError.Action == "" &&
		rd.OnError.MaxRetries == 0 &&
		rd.OnError.RetryDelay == ""
}

// isEmptyAgentProfile returns true when all fields in the profile are zero.
func isEmptyAgentProfile(cfg Config) bool {
	return isLLMKeysEmpty(cfg.LLM) &&
		isDefaultsEmpty(cfg.Defaults) &&
		isResourceDefaultsEmpty(cfg.ResourceDefaults)
}

// collectWorkflowNames scans agentsDir for workflow.yaml files and returns
// a set of metadata.name values. Returns nil if agentsDir is empty or doesn't exist.
func collectWorkflowNames(agentsDir string) map[string]bool {
	if agentsDir == "" {
		return nil
	}
	names := make(map[string]bool)
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		workflowPath := filepath.Join(agentsDir, entry.Name(), "workflow.yaml")
		data, readErr := os.ReadFile(workflowPath)
		if readErr != nil {
			continue
		}
		var wf struct {
			Metadata struct {
				Name string `yaml:"name"`
			} `yaml:"metadata"`
		}
		if yaml.Unmarshal(data, &wf) == nil && wf.Metadata.Name != "" {
			names[wf.Metadata.Name] = true
		}
	}
	if len(names) == 0 {
		return nil
	}
	return names
}
