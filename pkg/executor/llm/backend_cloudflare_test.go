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

package llm_test

import (
	stdhttp "net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

func TestCloudflareBackend_Name(t *testing.T) {
	b := &llm.CloudflareBackend{}
	assert.Equal(t, "cloudflare", b.Name())
}

func TestCloudflareBackend_DefaultURL_WithAccountID(t *testing.T) {
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "abc123")
	b := &llm.CloudflareBackend{}
	assert.Contains(t, b.DefaultURL(), "abc123")
	assert.Contains(t, b.DefaultURL(), "api.cloudflare.com")
}

func TestCloudflareBackend_DefaultURL_NoAccountID(t *testing.T) {
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "")
	b := &llm.CloudflareBackend{}
	assert.Contains(t, b.DefaultURL(), "unknown")
}

func TestCloudflareBackend_ChatEndpoint(t *testing.T) {
	b := &llm.CloudflareBackend{}
	got := b.ChatEndpoint("https://api.cloudflare.com/client/v4/accounts/abc/ai")
	assert.Equal(t, "https://api.cloudflare.com/client/v4/accounts/abc/ai/v1/chat/completions", got)
}

func TestCloudflareBackend_BuildRequest(t *testing.T) {
	b := &llm.CloudflareBackend{}
	msgs := []map[string]interface{}{{"role": "user", "content": "hello"}}
	req, err := b.BuildRequest("@cf/meta/llama-3.1-8b-instruct", msgs, llm.ChatRequestConfig{})
	require.NoError(t, err)
	assert.Equal(t, "@cf/meta/llama-3.1-8b-instruct", req["model"])
}

func TestCloudflareBackend_ParseResponse_Error(t *testing.T) {
	b := &llm.CloudflareBackend{}
	resp := makeResp(stdhttp.StatusForbidden, `{"error":"forbidden"}`)
	_, err := b.ParseResponse(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestCloudflareBackend_GetAPIKeyHeader_Empty(t *testing.T) {
	t.Setenv("CLOUDFLARE_API_TOKEN", "")
	b := &llm.CloudflareBackend{}
	name, val := b.GetAPIKeyHeader("")
	assert.Empty(t, name)
	assert.Empty(t, val)
}

func TestCloudflareBackend_APIKeyEnvVar(t *testing.T) {
	b := &llm.CloudflareBackend{}
	assert.Equal(t, "CLOUDFLARE_API_TOKEN", b.APIKeyEnvVar())
}

func TestCloudflareBackend_ChatEndpoint_CustomBaseURL(t *testing.T) {
	t.Parallel()
	b := &llm.CloudflareBackend{}
	// When user overrides base_url, ChatEndpoint appends /v1/chat/completions.
	customBase := "https://custom.endpoint.example.com/v4/accounts/myid/ai"
	got := b.ChatEndpoint(customBase)
	assert.Equal(t, customBase+"/v1/chat/completions", got)
}

func TestCloudflareBackend_DefaultURL_AccountIDInterpolated(t *testing.T) {
	t.Setenv("CLOUDFLARE_ACCOUNT_ID", "my-account-id-xyz")
	b := &llm.CloudflareBackend{}
	url := b.DefaultURL()
	assert.Contains(t, url, "my-account-id-xyz")
	assert.Contains(t, url, "accounts/my-account-id-xyz/ai")
}

func TestCloudflareBackend_ParseResponse_Success(t *testing.T) {
	t.Parallel()
	b := &llm.CloudflareBackend{}
	// Cloudflare /v1/chat/completions returns OpenAI-compat format.
	body := `{"choices":[{"message":{"role":"assistant","content":"Hello from Cloudflare!"}}]}`
	resp := makeResp(stdhttp.StatusOK, body)
	result, err := b.ParseResponse(resp)
	require.NoError(t, err)
	require.NotNil(t, result)
	msg, ok := result["message"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Hello from Cloudflare!", msg["content"])
}
