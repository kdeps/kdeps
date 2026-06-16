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

//go:build !js

package llm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lc "github.com/tmc/langchaingo/llms"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func findTextInMessages(msgs []lc.MessageContent, role lc.ChatMessageType, text string) bool {
	for _, m := range msgs {
		if m.Role != role {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(lc.TextContent); ok && tc.Text == text {
				return true
			}
		}
	}
	return false
}

func indexOfTextInMessages(msgs []lc.MessageContent, role lc.ChatMessageType, text string) int {
	for i, m := range msgs {
		if m.Role != role {
			continue
		}
		for _, p := range m.Parts {
			if tc, ok := p.(lc.TextContent); ok && tc.Text == text {
				return i
			}
		}
	}
	return -1
}

func TestBuildLangchainMessages_FewShot_Injected(t *testing.T) {
	cfg := &domain.ChatConfig{
		Prompt: "translate: hello",
		FewShot: []domain.ScenarioItem{
			{Role: "user", Prompt: "translate: bonjour"},
			{Role: "assistant", Prompt: "hello"},
		},
	}
	msgs := buildLangchainMessages(cfg)
	require.GreaterOrEqual(t, len(msgs), 3, "expected few-shot + prompt messages")
	assert.True(t, findTextInMessages(msgs, lc.ChatMessageTypeHuman, "translate: bonjour"), "few-shot user not found")
	assert.True(t, findTextInMessages(msgs, lc.ChatMessageTypeAI, "hello"), "few-shot assistant not found")
}

func TestBuildLangchainMessages_FewShot_BeforeHistory(t *testing.T) {
	cfg := &domain.ChatConfig{
		Prompt:   "current question",
		Messages: `[{"role":"user","content":"prev question"},{"role":"assistant","content":"prev answer"}]`,
		FewShot: []domain.ScenarioItem{
			{Role: "user", Prompt: "few-shot-q"},
			{Role: "assistant", Prompt: "few-shot-a"},
		},
	}
	msgs := buildLangchainMessages(cfg)
	fewShotIdx := indexOfTextInMessages(msgs, lc.ChatMessageTypeHuman, "few-shot-q")
	historyIdx := indexOfTextInMessages(msgs, lc.ChatMessageTypeHuman, "prev question")
	require.NotEqual(t, -1, fewShotIdx, "few-shot message not found")
	require.NotEqual(t, -1, historyIdx, "history message not found")
	assert.Less(t, fewShotIdx, historyIdx, "few-shot must appear before history")
}

func TestBuildLangchainMessages_FewShot_Empty(t *testing.T) {
	cfg := &domain.ChatConfig{
		Prompt:  "hello",
		FewShot: nil,
	}
	msgs := buildLangchainMessages(cfg)
	// Only the user prompt message expected (no system, no history)
	assert.Len(t, msgs, 1)
	assert.Equal(t, lc.ChatMessageTypeHuman, msgs[0].Role)
}

func TestPromptVars_SubstitutionInPrompt(t *testing.T) {
	cfg := &domain.ChatConfig{
		Prompt:     "Hello, my name is {{name}} and I am a {{role}}.",
		PromptVars: map[string]string{"name": "Alice", "role": "developer"},
	}
	msgs := buildLangchainMessages(cfg)
	require.Len(t, msgs, 1)
	require.Len(t, msgs[0].Parts, 1)
	text, ok := msgs[0].Parts[0].(lc.TextContent)
	require.True(t, ok, "expected TextContent")
	assert.Contains(t, text.Text, "Alice")
	assert.Contains(t, text.Text, "developer")
	assert.NotContains(t, text.Text, "{{name}}")
	assert.NotContains(t, text.Text, "{{role}}")
}

func TestPromptVars_SubstitutionInScenario(t *testing.T) {
	cfg := &domain.ChatConfig{
		Scenario: []domain.ScenarioItem{
			{Role: "system", Prompt: "You are a {{role}} assistant."},
		},
		PromptVars: map[string]string{"role": "helpful"},
	}
	msgs := buildLangchainMessages(cfg)
	require.Len(t, msgs, 1)
	require.Len(t, msgs[0].Parts, 1)
	text, ok := msgs[0].Parts[0].(lc.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "helpful")
	assert.NotContains(t, text.Text, "{{role}}")
}

func TestPromptVars_NoVars_Unchanged(t *testing.T) {
	cfg := &domain.ChatConfig{
		Prompt:     "Hello {{name}}",
		PromptVars: nil,
	}
	msgs := buildLangchainMessages(cfg)
	require.Len(t, msgs, 1)
	text, ok := msgs[0].Parts[0].(lc.TextContent)
	require.True(t, ok)
	assert.Equal(t, "Hello {{name}}", text.Text)
}

func TestApplyPromptVars_Empty(t *testing.T) {
	result := applyPromptVars("hello {{x}}", nil)
	assert.Equal(t, "hello {{x}}", result)
}

func TestApplyPromptVars_MultipleVars(t *testing.T) {
	vars := map[string]string{"a": "1", "b": "2"}
	result := applyPromptVars("{{a}} and {{b}}", vars)
	assert.Equal(t, "1 and 2", result)
}

func TestLangchainBaseURLs_LocalBackendPresent(t *testing.T) {
	t.Parallel()
	url, ok := langchainBaseURLs["local"]
	assert.True(t, ok, "local backend must have a base URL entry")
	assert.Contains(t, url, "localhost", "local backend URL should point to localhost")
}

func TestBuildRetrieverPreamble_Empty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "", buildRetrieverPreamble(nil))
	assert.Equal(t, "", buildRetrieverPreamble([]string{}))
}

func TestBuildRetrieverPreamble_SingleChunk(t *testing.T) {
	t.Parallel()
	out := buildRetrieverPreamble([]string{"chunk one"})
	assert.Contains(t, out, "Retrieved context:")
	assert.Contains(t, out, "chunk one")
}

func TestBuildRetrieverPreamble_MultipleChunks(t *testing.T) {
	t.Parallel()
	out := buildRetrieverPreamble([]string{"a", "b", "c"})
	assert.Contains(t, out, "a")
	assert.Contains(t, out, "b")
	assert.Contains(t, out, "c")
}

func TestBuildLangchainMessages_RetrieverContextInjectsIntoSystemMessage(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Scenario: []domain.ScenarioItem{
			{Role: "system", Prompt: "You are helpful."},
		},
		RetrieverContext: []string{"doc chunk 1", "doc chunk 2"},
	}
	msgs := buildLangchainMessages(cfg)
	require.Len(t, msgs, 1)
	text, ok := msgs[0].Parts[0].(lc.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "Retrieved context:")
	assert.Contains(t, text.Text, "doc chunk 1")
	assert.Contains(t, text.Text, "You are helpful.")
}

func TestBuildLangchainMessages_RetrieverContextNoScenario(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Prompt:           "What is kdeps?",
		RetrieverContext: []string{"kdeps is a Go framework"},
	}
	msgs := buildLangchainMessages(cfg)
	require.GreaterOrEqual(t, len(msgs), 2)
	text, ok := msgs[0].Parts[0].(lc.TextContent)
	require.True(t, ok)
	assert.Contains(t, text.Text, "Retrieved context:")
	assert.Contains(t, text.Text, "kdeps is a Go framework")
}

func TestRenderGoTemplate_Basic(t *testing.T) {
	t.Parallel()
	vars := map[string]string{"Name": "world"}
	result := renderGoTemplate("Hello {{.Name}}", vars)
	assert.Equal(t, "Hello world", result)
}

func TestRenderGoTemplate_Conditional(t *testing.T) {
	t.Parallel()
	vars := map[string]string{"Debug": "true"}
	result := renderGoTemplate(`{{if .Debug}}debug mode{{else}}prod mode{{end}}`, vars)
	assert.Equal(t, "debug mode", result)
}

func TestRenderGoTemplate_ParseError_FallsBack(t *testing.T) {
	t.Parallel()
	result := renderGoTemplate("{{invalid template !!!", map[string]string{"x": "y"})
	assert.Equal(t, "{{invalid template !!!", result, "should fall back to raw string on parse error")
}

func TestRenderGoTemplate_EmptyVars(t *testing.T) {
	t.Parallel()
	result := renderGoTemplate("hello world", nil)
	assert.Equal(t, "hello world", result)
}

func TestBuildLangchainMessages_GoTemplate(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		GoTemplate: true,
		PromptVars: map[string]string{"Lang": "Go"},
		Prompt:     "I love {{.Lang}} programming",
	}
	msgs := buildLangchainMessages(cfg)
	require.Len(t, msgs, 1)
	text, ok := msgs[0].Parts[0].(lc.TextContent)
	require.True(t, ok)
	assert.Equal(t, "I love Go programming", text.Text)
}
