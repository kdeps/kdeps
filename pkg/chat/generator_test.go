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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockLLMClient returns a fixed reply for testing.
type mockLLMClient struct {
	reply string
	err   error
}

func (m *mockLLMClient) Chat(_ context.Context, _, _, _ string, _ []map[string]interface{}) (string, error) {
	return m.reply, m.err
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
