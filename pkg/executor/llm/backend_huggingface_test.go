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

func TestHuggingFaceBackend_Name(t *testing.T) {
	b := &llm.HuggingFaceBackend{}
	assert.Equal(t, "huggingface", b.Name())
}

func TestHuggingFaceBackend_DefaultURL(t *testing.T) {
	b := &llm.HuggingFaceBackend{}
	assert.Equal(t, "https://api-inference.huggingface.co", b.DefaultURL())
}

func TestHuggingFaceBackend_ChatEndpoint(t *testing.T) {
	b := &llm.HuggingFaceBackend{}
	got := b.ChatEndpoint("https://api-inference.huggingface.co")
	assert.Equal(t, "https://api-inference.huggingface.co/v1/chat/completions", got)
}

func TestHuggingFaceBackend_BuildRequest(t *testing.T) {
	b := &llm.HuggingFaceBackend{}
	msgs := []map[string]interface{}{{"role": "user", "content": "hello"}}
	req, err := b.BuildRequest("mistralai/Mistral-7B-v0.1", msgs, llm.ChatRequestConfig{})
	require.NoError(t, err)
	assert.Equal(t, "mistralai/Mistral-7B-v0.1", req["model"])
}

func TestHuggingFaceBackend_ParseResponse_Error(t *testing.T) {
	b := &llm.HuggingFaceBackend{}
	resp := makeResp(stdhttp.StatusUnauthorized, `{"error":"unauthorized"}`)
	_, err := b.ParseResponse(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestHuggingFaceBackend_GetAPIKeyHeader_Empty(t *testing.T) {
	t.Setenv("HF_TOKEN", "")
	b := &llm.HuggingFaceBackend{}
	name, val := b.GetAPIKeyHeader("")
	assert.Empty(t, name)
	assert.Empty(t, val)
}

func TestHuggingFaceBackend_APIKeyEnvVar(t *testing.T) {
	b := &llm.HuggingFaceBackend{}
	assert.Equal(t, "HF_TOKEN", b.APIKeyEnvVar())
}

func TestHuggingFaceBackend_GetAPIKeyHeader_Set(t *testing.T) {
	t.Parallel()
	b := &llm.HuggingFaceBackend{}
	name, val := b.GetAPIKeyHeader("hf-abc123")
	assert.Equal(t, "Authorization", name)
	assert.Equal(t, "Bearer hf-abc123", val)
}

func TestHuggingFaceBackend_ChatEndpoint_CustomURL(t *testing.T) {
	t.Parallel()
	b := &llm.HuggingFaceBackend{}
	got := b.ChatEndpoint("https://custom.hf.co/proxy")
	assert.Equal(t, "https://custom.hf.co/proxy/v1/chat/completions", got)
}

func TestHuggingFaceBackend_ParseResponse_Success(t *testing.T) {
	t.Parallel()
	b := &llm.HuggingFaceBackend{}
	body := `{"choices":[{"message":{"role":"assistant","content":"Hello from HuggingFace!"}}]}`
	resp := makeResp(stdhttp.StatusOK, body)
	result, err := b.ParseResponse(resp)
	require.NoError(t, err)
	require.NotNil(t, result)
	msg, ok := result["message"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Hello from HuggingFace!", msg["content"])
}
