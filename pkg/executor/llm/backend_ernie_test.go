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

func TestErnieBackend_Name(t *testing.T) {
	b := &llm.ErnieBackend{}
	assert.Equal(t, "ernie", b.Name())
}

func TestErnieBackend_DefaultURL(t *testing.T) {
	b := &llm.ErnieBackend{}
	assert.Contains(t, b.DefaultURL(), "baidubce.com")
}

func TestErnieBackend_ChatEndpoint(t *testing.T) {
	b := &llm.ErnieBackend{}
	got := b.ChatEndpoint("https://aip.baidubce.com/rpc/2.0/ai_custom/v1/wenxinworkshop/chat")
	assert.Contains(t, got, "baidubce.com")
}

func TestErnieBackend_BuildRequest(t *testing.T) {
	b := &llm.ErnieBackend{}
	msgs := []map[string]interface{}{{"role": "user", "content": "hello"}}
	req, err := b.BuildRequest("ERNIE-Bot", msgs, llm.ChatRequestConfig{})
	require.NoError(t, err)
	assert.Equal(t, "ERNIE-Bot", req["model"])
}

func TestErnieBackend_ParseResponse_Error(t *testing.T) {
	b := &llm.ErnieBackend{}
	resp := makeResp(stdhttp.StatusForbidden, `{"error":"forbidden"}`)
	_, err := b.ParseResponse(resp)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestErnieBackend_APIKeyEnvVar(t *testing.T) {
	b := &llm.ErnieBackend{}
	assert.Equal(t, "ERNIE_API_KEY", b.APIKeyEnvVar())
}
