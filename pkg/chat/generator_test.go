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
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*stdhttp.Request) (*stdhttp.Response, error)

func (f roundTripFunc) RoundTrip(req *stdhttp.Request) (*stdhttp.Response, error) {
	return f(req)
}

// mockLLMClient returns a fixed reply for testing.
type mockLLMClient struct {
	reply string
	err   error
}

func (m *mockLLMClient) Chat(
	_ context.Context,
	_, _, _ string,
	_ []map[string]interface{},
) (string, error) {
	return m.reply, m.err
}

// mockLLMSequence returns successive replies from a pre-set list.
type mockLLMSequence struct {
	replies []string
	call    int
}

func (m *mockLLMSequence) Chat(
	_ context.Context,
	_, _, _ string,
	_ []map[string]interface{},
) (string, error) {
	if m.call >= len(m.replies) {
		return "", errors.New("sequence exhausted")
	}
	reply := m.replies[m.call]
	m.call++
	return reply, nil
}

const validReply = `<kdeps-workflow>
<file name="workflow.yaml">
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-agent
  version: 1.0.0
  targetActionId: main
settings:
  agentSettings: {}
</file>
<file name="resources/main.yaml">
actionId: main
exec:
  command: "echo hello"
</file>
</kdeps-workflow>`

const invalidTargetWorkflow = `<kdeps-workflow>
<file name="workflow.yaml">
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-agent
  version: 1.0.0
  targetActionId: nonexistent
settings:
  agentSettings: {}
</file>
<file name="resources/main.yaml">
actionId: main
exec:
  command: "echo hello"
</file>
</kdeps-workflow>`

func TestGenerator_Generate_OK(t *testing.T) {
	client := &mockLLMClient{reply: validReply}
	gen := NewGenerator(client, "llama3", "", "", nil)

	wf, err := gen.Generate(context.Background(), []Turn{
		{Role: "user", Content: "say hello"},
	})

	require.NoError(t, err)
	assert.Contains(t, wf.Files, "workflow.yaml")
	assert.Contains(t, wf.Files, "resources/main.yaml")
	assert.Contains(t, wf.Files["workflow.yaml"], "test-agent")
}

func TestGenerator_Generate_LLMError(t *testing.T) {
	client := &mockLLMClient{err: errors.New("connection refused")}
	gen := NewGenerator(client, "llama3", "", "", nil)

	_, err := gen.Generate(context.Background(), []Turn{{Role: "user", Content: "test"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM call failed")
}

func TestGenerator_Generate_NoFileBlocks(t *testing.T) {
	client := &mockLLMClient{reply: "Sure! Here is the workflow: (no blocks)"}
	gen := NewGenerator(client, "llama3", "", "", nil)

	_, err := gen.Generate(context.Background(), []Turn{{Role: "user", Content: "test"}})
	assert.Error(t, err)
}

func TestGenerator_Generate_MissingWorkflowYAML(t *testing.T) {
	reply := `<kdeps-workflow>
<file name="resources/main.yaml">
id: main
</file>
</kdeps-workflow>`
	client := &mockLLMClient{reply: reply}
	gen := NewGenerator(client, "llama3", "", "", nil)

	_, err := gen.Generate(context.Background(), []Turn{{Role: "user", Content: "test"}})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing workflow.yaml")
}

func TestParseWorkflowBlocks_BareBlocks(t *testing.T) {
	// No outer <kdeps-workflow> wrapper - still works
	raw := `<file name="workflow.yaml">
apiVersion: kdeps.io/v1
</file>`

	wf, err := parseWorkflowBlocks(raw)
	require.NoError(t, err)
	assert.Contains(t, wf.Files["workflow.yaml"], "apiVersion")
}

func TestParseWorkflowBlocks_TrimsWhitespace(t *testing.T) {
	raw := `<kdeps-workflow>
<file name="workflow.yaml">
  apiVersion: kdeps.io/v1
</file>
</kdeps-workflow>`

	wf, err := parseWorkflowBlocks(raw)
	require.NoError(t, err)
	assert.Equal(t, "apiVersion: kdeps.io/v1", wf.Files["workflow.yaml"])
}

func TestParseWorkflowBlocks_XMLAttributes(t *testing.T) {
	// Small models sometimes emit <kdeps-workflow xmlns:xsi="..."> — must still parse.
	raw := `<kdeps-workflow xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
<file name="workflow.yaml">
apiVersion: kdeps.io/v1
</file>
</kdeps-workflow>`

	wf, err := parseWorkflowBlocks(raw)
	require.NoError(t, err)
	assert.Contains(t, wf.Files["workflow.yaml"], "apiVersion")
}

func TestParseFailureCorrection_ContainsExample(t *testing.T) {
	msg := parseFailureCorrection("no <file> blocks found")
	assert.Contains(t, msg, "PARSE ERROR")
	assert.Contains(t, msg, "<kdeps-workflow>")
	assert.Contains(t, msg, `<file name="workflow.yaml">`)
	assert.Contains(t, msg, "no XML namespaces")
}

func TestExtractContent_Ollama(t *testing.T) {
	data := []byte(`{"message":{"role":"assistant","content":"hello world"}}`)
	content, err := extractContent(data)
	require.NoError(t, err)
	assert.Equal(t, "hello world", content)
}

func TestExtractContent_OpenAI(t *testing.T) {
	data := []byte(`{"choices":[{"message":{"role":"assistant","content":"hello openai"}}]}`)
	content, err := extractContent(data)
	require.NoError(t, err)
	assert.Equal(t, "hello openai", content)
}

func TestExtractContent_Unknown(t *testing.T) {
	data := []byte(`{"result":"ok"}`)
	_, err := extractContent(data)
	assert.Error(t, err)
}

func TestNewGenerator_WithCatalog(t *testing.T) {
	catalog := []ComponentEntry{
		{Name: "search", Version: "1.0.0", Description: "web search"},
	}
	client := &mockLLMClient{reply: validReply}
	gen := NewGenerator(client, "model", "http://localhost:11434", "", catalog)
	assert.NotNil(t, gen)
	assert.Contains(t, gen.catalog, "search@1.0.0")
}

func TestBackendName(t *testing.T) {
	tests := []struct {
		baseURL string
		want    string
	}{
		{"http://localhost:11434", "ollama"},
		{"http://127.0.0.1:11434", "ollama"},
		{"http://ollama:11434", "ollama"},
		{"https://api.openai.com/v1", "openai"},
		{"https://api.openrouter.ai/v1", "openrouter"},
		{"https://api.anthropic.com/v1", "anthropic"},
		{"https://generativelanguage.googleapis.com", "google"},
		{"https://api.groq.com/openai", "groq"},
		{"https://api.deepseek.com", "deepseek"},
		{"", "llamafile (local, served on first message)"},
	}
	for _, tc := range tests {
		gen := NewGenerator(&mockLLMClient{}, "model", tc.baseURL, "", nil)
		label := gen.BackendLabel()
		assert.Contains(t, label, tc.want, "baseURL=%q", tc.baseURL)
	}
}

func TestNewHTTPLLMClient(t *testing.T) {
	client := NewHTTPLLMClient()
	require.NotNil(t, client)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 120*time.Second, client.httpClient.Timeout)
}

func TestHTTPLLMClient_chatOllama(t *testing.T) {
	server := httptest.NewServer(
		stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			assert.Equal(t, "/api/chat", r.URL.Path)
			w.WriteHeader(stdhttp.StatusOK)
			fmt.Fprint(w, `{"message":{"role":"assistant","content":"ollama response"}}`)
		}),
	)
	defer server.Close()

	client := NewHTTPLLMClient()
	client.httpClient = server.Client()

	reply, err := client.chatOllama(context.Background(), "llama3", server.URL, nil)
	require.NoError(t, err)
	assert.Equal(t, "ollama response", reply)
}

func TestHTTPLLMClient_chatOpenAI(t *testing.T) {
	server := httptest.NewServer(
		stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			assert.Equal(t, "/chat/completions", r.URL.Path)
			assert.Equal(t, "Bearer sk-test", r.Header.Get("Authorization"))
			w.WriteHeader(stdhttp.StatusOK)
			fmt.Fprint(
				w,
				`{"choices":[{"message":{"role":"assistant","content":"openai response"}}]}`,
			)
		}),
	)
	defer server.Close()

	client := NewHTTPLLMClient()
	client.httpClient = server.Client()

	reply, err := client.chatOpenAI(context.Background(), "gpt-4", server.URL, "sk-test", nil)
	require.NoError(t, err)
	assert.Equal(t, "openai response", reply)
}

func TestHTTPLLMClient_doRequest_Error(t *testing.T) {
	server := httptest.NewServer(
		stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
			w.WriteHeader(stdhttp.StatusBadRequest)
			fmt.Fprint(w, `{"error":"bad request"}`)
		}),
	)
	defer server.Close()

	client := NewHTTPLLMClient()
	client.httpClient = server.Client()

	_, err := client.doRequest(context.Background(), server.URL, "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "backend returned 400")
}

func TestHTTPLLMClient_doRequest_NetworkError(t *testing.T) {
	client := NewHTTPLLMClient()
	// Use a dial timeout so the test doesn't hang
	client.httpClient = &stdhttp.Client{Timeout: time.Millisecond}

	_, err := client.doRequest(context.Background(), "http://127.0.0.1:1", "", nil)
	require.Error(t, err)
}

func TestHTTPLLMClient_Chat_EmptyBaseURL(t *testing.T) {
	server := httptest.NewServer(
		stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			assert.Equal(t, "/api/chat", r.URL.Path)
			w.WriteHeader(stdhttp.StatusOK)
			fmt.Fprint(w, `{"message":{"role":"assistant","content":"local response"}}`)
		}),
	)
	defer server.Close()

	client := NewHTTPLLMClientWithBackend("ollama")
	client.httpClient = server.Client()

	baseURL := server.URL
	reply, err := client.Chat(context.Background(), "llama3", baseURL, "", nil)
	require.NoError(t, err)
	assert.Equal(t, "local response", reply)
}

func TestHTTPLLMClient_doRequest_WithAuth(t *testing.T) {
	server := httptest.NewServer(
		stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
			assert.Equal(t, "Bearer my-token", r.Header.Get("Authorization"))
			w.WriteHeader(stdhttp.StatusOK)
			fmt.Fprint(w, `{"choices":[{"message":{"role":"assistant","content":"auth ok"}}]}`)
		}),
	)
	defer server.Close()

	client := NewHTTPLLMClient()
	client.httpClient = server.Client()

	reply, err := client.doRequest(context.Background(), server.URL, "my-token", nil)
	require.NoError(t, err)
	assert.Equal(t, "auth ok", reply)
}

func TestParseWorkflowBlocks_MissingClosingTag(t *testing.T) {
	raw := `<kdeps-workflow>
<file name="workflow.yaml">
apiVersion: kdeps.io/v1
</file>
<!-- no closing tag -->`

	wf, err := parseWorkflowBlocks(raw)
	require.NoError(t, err)
	assert.Contains(t, wf.Files["workflow.yaml"], "apiVersion")
}

func TestHTTPLLMClient_Chat_OpenAIRoute(t *testing.T) {
	client := NewHTTPLLMClientWithBackend("")
	client.httpClient = &stdhttp.Client{
		Transport: roundTripFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
			assert.Equal(t, "/chat/completions", req.URL.Path)
			assert.Equal(t, "Bearer sk-test", req.Header.Get("Authorization"))
			return &stdhttp.Response{
				StatusCode: stdhttp.StatusOK,
				Body: io.NopCloser(
					strings.NewReader(
						`{"choices":[{"message":{"role":"assistant","content":"openai response"}}]}`,
					),
				),
				Header: make(stdhttp.Header),
			}, nil
		}),
		Timeout: 5 * time.Second,
	}

	reply, err := client.Chat(
		context.Background(),
		"gpt-4",
		"https://api.example.com",
		"sk-test",
		nil,
	)
	require.NoError(t, err)
	assert.Equal(t, "openai response", reply)
}

func TestHTTPLLMClient_Chat_EmptyBaseURLRoutesToOllama(t *testing.T) {
	client := NewHTTPLLMClientWithBackend("ollama")
	client.httpClient = &stdhttp.Client{
		Transport: roundTripFunc(func(req *stdhttp.Request) (*stdhttp.Response, error) {
			assert.Equal(t, "/api/chat", req.URL.Path)
			return &stdhttp.Response{
				StatusCode: stdhttp.StatusOK,
				Body: io.NopCloser(
					strings.NewReader(
						`{"message":{"role":"assistant","content":"ollama response"}}`,
					),
				),
				Header: make(stdhttp.Header),
			}, nil
		}),
		Timeout: 5 * time.Second,
	}

	reply, err := client.Chat(context.Background(), "llama3", "", "", nil)
	require.NoError(t, err)
	assert.Equal(t, "ollama response", reply)
}

func TestHTTPLLMClient_doRequest_NewRequestError(t *testing.T) {
	client := NewHTTPLLMClient()
	_, err := client.doRequest(context.Background(), "http://example.com/%gg", "", nil)
	require.Error(t, err)
}

func TestExtractContent_NonJSON(t *testing.T) {
	_, err := extractContent([]byte("not json"))
	require.Error(t, err)
}

func TestExtractOpenAIContent_BadChoicesType(t *testing.T) {
	data := []byte(`{"choices": ["string"]}`)
	_, err := extractContent(data)
	require.Error(t, err)
}

func TestExtractOpenAIContent_BadMessageType(t *testing.T) {
	data := []byte(`{"choices": [{"message": "direct string"}]}`)
	_, err := extractContent(data)
	require.Error(t, err)
}

func TestGenerate_ValidationErrorLastRetry(t *testing.T) {
	client := &mockLLMClient{reply: invalidTargetWorkflow}
	gen := NewGenerator(client, "llama3", "", "", nil)

	_, err := gen.Generate(context.Background(), []Turn{
		{Role: "user", Content: "test"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow failed validation")
}

func TestGenerate_ValidationErrorRecovery(t *testing.T) {
	client := &mockLLMSequence{
		replies: []string{invalidTargetWorkflow, validReply},
	}
	gen := NewGenerator(client, "llama3", "", "", nil)

	wf, err := gen.Generate(context.Background(), []Turn{
		{Role: "user", Content: "test"},
	})
	require.NoError(t, err)
	assert.Contains(t, wf.Files, "workflow.yaml")
	assert.Contains(t, wf.Files, "resources/main.yaml")
}

func TestGenerator_Generate_RetryExhausted(t *testing.T) {
	client := &mockLLMClient{reply: "no file blocks here"}
	origRetries := maxValidationRetries
	t.Cleanup(func() { maxValidationRetries = origRetries })
	maxValidationRetries = 0

	gen := NewGenerator(client, "llama3", "", "", nil)

	_, err := gen.Generate(context.Background(), []Turn{{Role: "user", Content: "test"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "retry loop exhausted")
}
