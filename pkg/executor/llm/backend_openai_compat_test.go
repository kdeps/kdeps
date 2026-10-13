package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestOpenAICompatChatEndpoint_NoDoubling(t *testing.T) {
	t.Parallel()
	cases := []struct{ fmtStr, base, want string }{
		{"%s/v1/chat/completions", "https://api.openai.com", "https://api.openai.com/v1/chat/completions"},
		{"%s/v1/chat/completions", "http://localhost:11435/v1", "http://localhost:11435/v1/chat/completions"},
		{"%s/v1/chat/completions", "http://localhost:11435/v1/", "http://localhost:11435/v1/chat/completions"},
		{"%s/v1/chat/completions", "http://x/v1/chat/completions", "http://x/v1/chat/completions"},
		{"%s/openai/v1/chat/completions", "https://api.groq.com", "https://api.groq.com/openai/v1/chat/completions"},
		{
			"%s/openai/v1/chat/completions",
			"https://api.groq.com/openai/v1",
			"https://api.groq.com/openai/v1/chat/completions",
		},
		{"%s/chat/completions", "https://api.perplexity.ai", "https://api.perplexity.ai/chat/completions"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, openAICompatChatEndpoint(c.fmtStr, c.base), c.base)
	}
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

// Every provider in the providers asset set registers an OpenAI-compatible
// chat backend with the endpoint and key variable its file declares.
func TestProviderBackends_Registered(t *testing.T) {
	reg := NewBackendRegistry()
	want := map[string]struct{ url, endpoint, env string }{
		"openai":     {"https://api.openai.com", "https://api.openai.com/v1/chat/completions", "OPENAI_API_KEY"},
		"groq":       {"https://api.groq.com", "https://api.groq.com/openai/v1/chat/completions", "GROQ_API_KEY"},
		"perplexity": {"https://api.perplexity.ai", "https://api.perplexity.ai/chat/completions", "PERPLEXITY_API_KEY"},
		"openrouter": {"https://openrouter.ai", "https://openrouter.ai/api/v1/chat/completions", "OPENROUTER_API_KEY"},
		"huggingface": {"https://api-inference.huggingface.co",
			"https://api-inference.huggingface.co/v1/chat/completions", "HF_TOKEN"},
		"maritaca": {"https://chat.maritaca.ai", "https://chat.maritaca.ai/api/chat/inference", "MARITACA_API_KEY"},
	}
	for name, w := range want {
		b := reg.Get(name)
		require.NotNil(t, b, name)
		assert.Equal(t, w.url, b.DefaultURL(), name)
		assert.Equal(t, w.endpoint, b.ChatEndpoint(b.DefaultURL()), name)
		assert.Equal(t, w.env, b.APIKeyEnvVar(), name)
	}
	for _, name := range []string{"mistral", "together", "deepseek", "xai", "cloudflare", "ernie"} {
		assert.NotNil(t, reg.Get(name), name)
		assert.Contains(t, DefaultRegistryBackendNames(), name)
	}
}

func TestProviderBackends_CloudflareURLFromEnv(t *testing.T) {
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "acct-1")
	b := NewBackendRegistry().Get("cloudflare")
	require.NotNil(t, b)
	assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/acct-1/ai", b.DefaultURL())

	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	assert.Contains(t, NewBackendRegistry().Get("cloudflare").DefaultURL(), "/accounts/unknown/ai")
}

func TestLangchainBaseURL(t *testing.T) {
	for backend, want := range map[string]string{
		"openai":     "https://api.openai.com/v1",
		"groq":       "https://api.groq.com/openai/v1",
		"perplexity": "https://api.perplexity.ai",
		"cohere":     "https://api.cohere.com/compatibility/v1",
		"ollama":     "http://localhost:11434/v1",
	} {
		got, ok := langchainBaseURL(backend)
		assert.True(t, ok, backend)
		assert.Equal(t, want, got, backend)
	}
	_, ok := langchainBaseURL("ernie")
	assert.False(t, ok, "a provider without an OpenAI-shaped chat path has no SDK base")
	_, ok = langchainBaseURL("nope")
	assert.False(t, ok)
}
