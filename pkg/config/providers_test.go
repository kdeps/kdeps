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
	}
}

func TestAllLLMProviderNames_MatchesRegistry(t *testing.T) {
	names := AllLLMProviderNames()
	assert.Equal(t, allProviderNames, names)
	assert.Equal(t, ollamaBackendStr, names[0])
}
