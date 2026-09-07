package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloudLLMProviders_MatchesRegistry(t *testing.T) {
	providers := CloudLLMProviders()
	require.Len(t, providers, len(cloudProvidersList))
	for i, p := range providers {
		assert.Equal(t, cloudProvidersList[i].name, p.Name)
		assert.Equal(t, cloudProvidersList[i].yamlKey, p.YAMLKey)
		assert.Equal(t, cloudProvidersList[i].envVar, p.EnvVar)
		assert.Equal(t, cloudProvidersList[i].defaultModel, p.DefaultModel)
	}
}

func TestCloudLLMProviders_DefaultModel(t *testing.T) {
	byName := make(map[string]LLMProvider)
	for _, p := range CloudLLMProviders() {
		byName[p.Name] = p
	}

	// Populated for providers pkg/agent's KnownCloudModels also represents.
	populated := map[string]string{
		"openai":     "gpt-4o",
		"anthropic":  "claude-sonnet-4-6",
		"google":     "gemini-2.5-flash",
		"xai":        "grok-3",
		"deepseek":   "deepseek-v4-flash",
		"groq":       "llama-3.3-70b-versatile",
		"mistral":    "mistral-large-latest",
		"cohere":     "command-r-plus",
		"together":   "meta-llama/Llama-3-70b-chat-hf",
		"perplexity": "llama-3.1-sonar-large-128k-online",
	}
	for name, want := range populated {
		p, ok := byName[name]
		require.True(t, ok, "provider %q missing from registry", name)
		assert.Equal(t, want, p.DefaultModel, "provider %q", name)
	}

	// Left empty: platforms hosting many models with no single canonical
	// default, none of which pkg/agent's own catalog picks either.
	empty := []string{"openrouter", "huggingface", "cloudflare", "maritaca", "ernie", "bedrock", "watsonx"}
	for _, name := range empty {
		p, ok := byName[name]
		require.True(t, ok, "provider %q missing from registry", name)
		assert.Empty(t, p.DefaultModel, "provider %q", name)
	}
}
