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
