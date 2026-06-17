package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOpenAICompatBackend_Accessors(t *testing.T) {
	t.Parallel()
	b := newOpenAICompatBackend(openAICompatBackendConfig{
		name:        "testprovider",
		defaultURL:  "https://api.test.com",
		endpointFmt: "%s/v1/chat/completions",
		envVar:      "TEST_API_KEY",
		apiName:     "TestAPI",
	})

	assert.Equal(t, "testprovider", b.Name())
	assert.Equal(t, "https://api.test.com", b.DefaultURL())
	assert.Equal(t, "https://api.test.com/v1/chat/completions", b.ChatEndpoint("https://api.test.com"))
	assert.Equal(t, "TEST_API_KEY", b.APIKeyEnvVar())
}

func TestOpenAICompatBackend_GetAPIKeyHeader(t *testing.T) {
	t.Parallel()
	b := newOpenAICompatBackend(openAICompatBackendConfig{
		name:   "mybackend",
		envVar: "MY_API_KEY",
	})
	k, v := b.GetAPIKeyHeader("secret123")
	assert.Equal(t, "Authorization", k)
	assert.Contains(t, v, "secret123")
}

func TestOpenAICompatBackend_BuildRequest(t *testing.T) {
	t.Parallel()
	b := newOpenAICompatBackend(openAICompatBackendConfig{name: "x"})
	req, err := b.BuildRequest("gpt-4o", nil, ChatRequestConfig{})
	assert.NoError(t, err)
	assert.Equal(t, "gpt-4o", req["model"])
}

func TestDefaultBackends_Registered(t *testing.T) {
	t.Parallel()
	for _, b := range []*openAICompatBackend{
		defaultOpenAIBackend,
		defaultMistralBackend,
		defaultTogetherBackend,
		defaultPerplexityBackend,
		defaultGroqBackend,
		defaultDeepSeekBackend,
		defaultOpenRouterBackend,
		defaultXAIBackend,
	} {
		assert.NotEmpty(t, b.Name())
		assert.NotEmpty(t, b.DefaultURL())
		assert.NotEmpty(t, b.APIKeyEnvVar())
	}
}
