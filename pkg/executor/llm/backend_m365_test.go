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

package llm

import (
	"io"
	stdhttp "net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestM365Backend_Name(t *testing.T) {
	b := &M365Backend{}
	assert.Equal(t, "m365", b.Name())
}

// The local M365 server streams upstream failures as assistant text because an
// OpenAI-compatible client has no other channel for them. The executor must
// turn that marker back into an error, or a dropped Copilot stream is saved as
// the model's answer (instrument session kdeps-api-11 event 167).
func TestM365ContentError(t *testing.T) {
	assert.NoError(t, m365ContentError(backendM365, "ordinary answer"))
	assert.NoError(t, m365ContentError(backendM365, "a note: [m365 error: x] inside prose"))
	assert.NoError(t, m365ContentError(backendM365, "[m365 error: ]"))
	assert.NoError(t, m365ContentError(backendAnthropic, "[m365 error: boom]"))

	err := m365ContentError(
		backendM365, "[m365 error: m365: read frame: read: connection reset by peer]")
	require.Error(t, err)
	assert.Equal(t, "m365: read frame: read: connection reset by peer", err.Error())

	err = m365ContentError(
		backendM365, "  [m365 error: M365 Copilot returned an empty response]  ")
	require.Error(t, err)
	assert.Equal(t, "m365: M365 Copilot returned an empty response", err.Error())
}

// Workflow mode reads the local server's non-streaming response through
// ParseResponse; the same marker must not become the chat output there.
func TestM365Backend_ParseResponse_UpstreamError(t *testing.T) {
	respond := func(content string) *stdhttp.Response {
		body := `{"id":"x","object":"chat.completion","model":"m365","choices":[` +
			`{"index":0,"message":{"role":"assistant","content":"` + content + `"},` +
			`"finish_reason":"stop"}]}`
		return &stdhttp.Response{
			StatusCode: stdhttp.StatusOK,
			Header:     make(stdhttp.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
		}
	}

	_, err := (&M365Backend{}).ParseResponse(respond("[m365 error: m365: read frame: reset]"))
	require.Error(t, err)
	assert.Equal(t, "m365: read frame: reset", err.Error())

	out, err := (&M365Backend{}).ParseResponse(respond("ordinary answer"))
	require.NoError(t, err)
	assert.Equal(t, "ordinary answer", out["message"].(map[string]any)["content"])
}

func TestM365Backend_DefaultURL_StartsLocalServer(t *testing.T) {
	b := &M365Backend{}
	url := b.DefaultURL()
	require.NotEmpty(t, url, "the local server must start and report a base URL")
	assert.True(t, strings.HasPrefix(url, "http://127.0.0.1:"))

	// Calling again reuses the same server (sync.Once) and returns the same URL.
	assert.Equal(t, url, b.DefaultURL())
}

func TestM365Backend_ChatEndpoint(t *testing.T) {
	b := &M365Backend{}
	assert.Equal(t, "http://x/v1/chat/completions", b.ChatEndpoint("http://x"))

	// Empty baseURL falls back to DefaultURL (starting the local server).
	got := b.ChatEndpoint("")
	assert.True(t, strings.HasSuffix(got, "/v1/chat/completions"))
	assert.True(t, strings.HasPrefix(got, "http://127.0.0.1:"))
}

func TestM365Backend_BuildRequest(t *testing.T) {
	b := &M365Backend{}
	req, err := b.BuildRequest("gpt-4", []map[string]any{{"role": "user", "content": "hi"}},
		ChatRequestConfig{ContextLength: 4096})
	require.NoError(t, err)
	assert.Equal(t, "gpt-4", req["model"])
}

func TestM365Backend_ParseResponse(t *testing.T) {
	b := &M365Backend{}
	body := `{"choices":[{"message":{"role":"assistant","content":"hello"}}]}`
	resp := &stdhttp.Response{
		StatusCode: 200,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	parsed, err := b.ParseResponse(resp)
	require.NoError(t, err)
	assert.NotNil(t, parsed)
}

func TestM365Backend_GetAPIKeyHeader(t *testing.T) {
	b := &M365Backend{}
	name, val := b.GetAPIKeyHeader("anything")
	assert.Empty(t, name)
	assert.Empty(t, val)
}

func TestM365Backend_APIKeyEnvVar(t *testing.T) {
	b := &M365Backend{}
	assert.Empty(t, b.APIKeyEnvVar())
}
