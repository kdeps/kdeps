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

package chat

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHTTPLLMClient_DoRequest_MarshalError(t *testing.T) {
	orig := jsonMarshal
	t.Cleanup(func() { jsonMarshal = orig })
	jsonMarshal = func(_ any) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	c := &HTTPLLMClient{httpClient: http.DefaultClient}
	_, err := c.doRequest(context.Background(), "http://x", "", map[string]interface{}{"a": 1})
	require.Error(t, err)
}

func TestHTTPLLMClient_DoRequest_ReadBodyError(t *testing.T) {
	t.Parallel()
	c := &HTTPLLMClient{httpClient: &http.Client{
		Transport: roundTripperFunc(func(_ *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(&failReader{}),
			}, nil
		}),
	}}

	_, err := c.doRequest(context.Background(), "http://x", "", map[string]interface{}{"a": 1})
	require.Error(t, err)
}

func TestHTTPLLMClient_Chat_LazyLlamafileServe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/chat/completions", r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`))
	}))
	defer srv.Close()

	orig := serveLlamafileForChat
	t.Cleanup(func() { serveLlamafileForChat = orig })
	serves := 0
	serveLlamafileForChat = func(_ string) (string, error) {
		serves++
		return srv.URL + "/v1", nil
	}

	client := NewHTTPLLMClientWithBackend("file")
	messages := []map[string]interface{}{{"role": "user", "content": "hello"}}

	out, err := client.Chat(context.Background(), "llama3.2:1b", "", "", messages)
	require.NoError(t, err)
	assert.Equal(t, "hi", out)

	// Second call must reuse the memoized URL, not serve again.
	_, err = client.Chat(context.Background(), "llama3.2:1b", "", "", messages)
	require.NoError(t, err)
	assert.Equal(t, 1, serves, "llamafile must be served once per model")
}

func TestHTTPLLMClient_Chat_ExplicitBaseURLSkipsServe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"hi"}}]}`))
	}))
	defer srv.Close()

	orig := serveLlamafileForChat
	t.Cleanup(func() { serveLlamafileForChat = orig })
	serveLlamafileForChat = func(_ string) (string, error) {
		t.Fatal("must not serve a llamafile when a base URL is provided")
		return "", nil
	}

	client := NewHTTPLLMClientWithBackend("file")
	_, err := client.Chat(context.Background(), "m", srv.URL+"/v1", "", []map[string]interface{}{
		{"role": "user", "content": "hello"},
	})
	require.NoError(t, err)
}

func TestHTTPLLMClient_Chat_OllamaDefaultURL(t *testing.T) {
	client := NewHTTPLLMClientWithBackend("ollama")
	// No server on the default port path is fine: the request error must
	// reference the ollama endpoint, proving no llamafile serve was attempted.
	orig := serveLlamafileForChat
	t.Cleanup(func() { serveLlamafileForChat = orig })
	serveLlamafileForChat = func(_ string) (string, error) {
		t.Fatal("ollama backend must not serve a llamafile")
		return "", nil
	}
	_, err := client.Chat(context.Background(), "m", "http://127.0.0.1:1", "", nil)
	require.Error(t, err)
}

func TestHTTPLLMClient_Chat_OllamaDefaultBaseURL(_ *testing.T) {
	client := NewHTTPLLMClientWithBackend("ollama")
	// Empty base URL falls back to the local ollama default; the request
	// itself may succeed or fail depending on the host - only the branch
	// matters here.
	_, _ = client.Chat(context.Background(), "m", "", "", nil)
}

func TestServeLlamafileForChat_ManagerError(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", "/dev/null/impossible")
	_, err := serveLlamafileForChat("anything.llamafile")
	require.Error(t, err)
}

func TestServeLlamafileForChat_ResolveError(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", t.TempDir())
	_, err := serveLlamafileForChat("missing.llamafile")
	require.Error(t, err)
}

func TestChat_FileBackendCallsEnsureLlamafile(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", "/dev/null/impossible")

	client := NewHTTPLLMClientWithBackend("file")
	_, err := client.Chat(context.Background(), "x.llamafile", "", "", nil)
	require.Error(t, err)
}

func TestBackendLabel_UnknownHostIsOpenAICompatible(t *testing.T) {
	gen := NewGenerator(&mockLLMClient{}, "m", "https://unknown.example.com/v1", "", nil)
	require.Contains(t, gen.BackendLabel(), "openai-compatible")
}
