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

func TestMaritacaBackend_Name(t *testing.T) {
	b := &llm.MaritacaBackend{}
	assert.Equal(t, "maritaca", b.Name())
}

func TestMaritacaBackend_DefaultURL(t *testing.T) {
	b := &llm.MaritacaBackend{}
	assert.Contains(t, b.DefaultURL(), "maritaca.ai")
}

func TestMaritacaBackend_ChatEndpoint(t *testing.T) {
	b := &llm.MaritacaBackend{}
	got := b.ChatEndpoint("https://chat.maritaca.ai")
	assert.Contains(t, got, "maritaca.ai")
	assert.Contains(t, got, "inference")
}

func TestMaritacaBackend_BuildRequest(t *testing.T) {
	b := &llm.MaritacaBackend{}
	msgs := []map[string]interface{}{{"role": "user", "content": "ola"}}
	req, err := b.BuildRequest("sabia-3", msgs, llm.ChatRequestConfig{})
	require.NoError(t, err)
	assert.Equal(t, "sabia-3", req["model"])
}

func TestMaritacaBackend_ParseResponse_Error(t *testing.T) {
	b := &llm.MaritacaBackend{}
	resp := makeResp(stdhttp.StatusUnauthorized, `{"error":"unauthorized"}`)
	_, err := b.ParseResponse(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestMaritacaBackend_APIKeyEnvVar(t *testing.T) {
	b := &llm.MaritacaBackend{}
	assert.Equal(t, "MARITACA_API_KEY", b.APIKeyEnvVar())
}

func TestMaritacaBackend_GetAPIKeyHeader_Empty(t *testing.T) {
	t.Setenv("MARITACA_API_KEY", "")
	b := &llm.MaritacaBackend{}
	name, val := b.GetAPIKeyHeader("")
	assert.Empty(t, name)
	assert.Empty(t, val)
}

func TestMaritacaBackend_GetAPIKeyHeader_Set(t *testing.T) {
	t.Parallel()
	b := &llm.MaritacaBackend{}
	name, val := b.GetAPIKeyHeader("tok-abc")
	assert.Equal(t, "Authorization", name)
	assert.Equal(t, "Bearer tok-abc", val)
}

func TestMaritacaBackend_ChatEndpoint_CustomBaseURL(t *testing.T) {
	t.Parallel()
	b := &llm.MaritacaBackend{}
	got := b.ChatEndpoint("https://custom.example.com")
	assert.Equal(t, "https://custom.example.com/api/chat/inference", got)
}

func TestMaritacaBackend_ParseResponse_Success(t *testing.T) {
	t.Parallel()
	b := &llm.MaritacaBackend{}
	body := `{"choices":[{"message":{"role":"assistant","content":"Ola! Como posso ajudar?"}}]}`
	resp := makeResp(stdhttp.StatusOK, body)
	result, err := b.ParseResponse(resp)
	require.NoError(t, err)
	require.NotNil(t, result)
	msg, ok := result["message"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Ola! Como posso ajudar?", msg["content"])
}
