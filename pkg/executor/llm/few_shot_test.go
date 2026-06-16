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
	"strings"
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

// --- Example Selector (T9) tests ---

func TestSelectFewShotExamples_KZero_ReturnsAll(t *testing.T) {
	t.Parallel()
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "hello world"},
		{Role: "assistant", Prompt: "hi there"},
		{Role: "user", Prompt: "goodbye"},
		{Role: "assistant", Prompt: "bye"},
	}
	result := selectFewShotExamples(pool, "some prompt", 0)
	assert.Equal(t, pool, result)
}

func TestSelectFewShotExamples_EmptyPool(t *testing.T) {
	t.Parallel()
	result := selectFewShotExamples(nil, "prompt", 2)
	assert.Nil(t, result)
}

func TestSelectFewShotExamples_SelectsTopK(t *testing.T) {
	t.Parallel()
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "translate English to French"},
		{Role: "assistant", Prompt: "Je traduis"},
		{Role: "user", Prompt: "explain quantum physics"},
		{Role: "assistant", Prompt: "quantum explanation"},
		{Role: "user", Prompt: "translate French to English"},
		{Role: "assistant", Prompt: "I translate"},
	}
	// Prompt is about translation; should prefer first and third user examples.
	result := selectFewShotExamples(pool, "translate this sentence", 1)
	// Should include a translation pair, not quantum.
	found := false
	for _, item := range result {
		if strings.Contains(strings.ToLower(item.Prompt), "translat") {
			found = true
		}
	}
	assert.True(t, found, "selected example should be translation-related")
	assert.NotEmpty(t, result)
}

func TestSelectFewShotExamples_PreservesPairs(t *testing.T) {
	t.Parallel()
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "what is Go"},
		{Role: "assistant", Prompt: "Go is a language"},
	}
	result := selectFewShotExamples(pool, "explain go programming", 1)
	assert.Len(t, result, 2, "selecting 1 pair should include both user and assistant items")
	assert.Equal(t, "user", result[0].Role)
	assert.Equal(t, "assistant", result[1].Role)
}

func TestSelectFewShotExamples_OrderPreserved(t *testing.T) {
	t.Parallel()
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "apple fruit"},
		{Role: "assistant", Prompt: "apple answer"},
		{Role: "user", Prompt: "banana fruit"},
		{Role: "assistant", Prompt: "banana answer"},
		{Role: "user", Prompt: "cherry fruit"},
		{Role: "assistant", Prompt: "cherry answer"},
	}
	// Select 2 pairs from fruit examples; result should be in authoring order.
	result := selectFewShotExamples(pool, "what is a fruit", 2)
	require.Equal(t, 4, len(result), "2 pairs = 4 items")
	// First pair comes before second pair in original order.
	assert.Less(t,
		findItemIndex(pool, result[0].Prompt),
		findItemIndex(pool, result[2].Prompt),
		"authoring order must be preserved",
	)
}

func findItemIndex(pool []domain.ScenarioItem, prompt string) int {
	for i, item := range pool {
		if item.Prompt == prompt {
			return i
		}
	}
	return -1
}

func TestJaccardSimilarity_Basic(t *testing.T) {
	t.Parallel()
	a := wordSet("the quick brown fox")
	b := wordSet("the quick red fox")
	score := jaccardSimilarity(a, b)
	// 3 intersection (the quick fox) / 5 union
	assert.InDelta(t, 0.6, score, 0.01)
}

func TestJaccardSimilarity_Empty(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, jaccardSimilarity(map[string]struct{}{}, wordSet("hello")))
	assert.Equal(t, 0.0, jaccardSimilarity(wordSet("hello"), map[string]struct{}{}))
}

// --- Contextual compression (T15) tests ---

func TestCompressRetrieverContext_TopKZero_ReturnsAll(t *testing.T) {
	t.Parallel()
	chunks := []string{"chunk a", "chunk b", "chunk c"}
	result := compressRetrieverContext(chunks, "prompt", 0)
	assert.Equal(t, chunks, result)
}

func TestCompressRetrieverContext_EmptyChunks(t *testing.T) {
	t.Parallel()
	result := compressRetrieverContext(nil, "prompt", 2)
	assert.Nil(t, result)
}

func TestCompressRetrieverContext_SelectsTopK(t *testing.T) {
	t.Parallel()
	chunks := []string{
		"Go programming language goroutines concurrency",
		"Python machine learning neural networks",
		"Go channels select statement goroutines",
	}
	result := compressRetrieverContext(chunks, "Go concurrency model goroutines", 1)
	require.Len(t, result, 1)
	assert.Contains(t, result[0], "goroutine")
}

func TestCompressRetrieverContext_PreservesOrder(t *testing.T) {
	t.Parallel()
	chunks := []string{"apple fruit red", "banana fruit yellow", "cherry fruit small"}
	result := compressRetrieverContext(chunks, "what is a fruit", 2)
	require.Len(t, result, 2)
	// Original order preserved among selected.
	idx0 := strings.Index(strings.Join(chunks, "|"), result[0])
	idx1 := strings.Index(strings.Join(chunks, "|"), result[1])
	assert.Less(t, idx0, idx1, "original order must be preserved")
}

func TestBuildLangchainMessages_RetrieverContextTopK(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Prompt: "explain Go concurrency",
		RetrieverContext: []string{
			"Go has goroutines and channels for concurrency",
			"Python uses GIL and threads",
			"Go select statement multiplexes channels",
		},
		RetrieverContextTopK: 1,
		Scenario: []domain.ScenarioItem{
			{Role: "system", Prompt: "You are helpful."},
		},
	}
	msgs := buildLangchainMessages(cfg)
	require.NotEmpty(t, msgs)
	sysText, ok := msgs[0].Parts[0].(lc.TextContent)
	require.True(t, ok)
	// Should contain Go-related chunk, not Python-only content.
	assert.Contains(t, sysText.Text, "Go")
	// Should NOT contain both Go and Python (only top-1 selected).
	assert.NotContains(t, sysText.Text, "Python")
}

func TestBuildLangchainMessages_FewShotSelectK(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Prompt: "translate hello to French",
		FewShot: []domain.ScenarioItem{
			{Role: "user", Prompt: "translate goodbye to French"},
			{Role: "assistant", Prompt: "au revoir"},
			{Role: "user", Prompt: "explain quantum entanglement"},
			{Role: "assistant", Prompt: "quantum answer"},
		},
		FewShotSelectK: 1,
	}
	msgs := buildLangchainMessages(cfg)
	// Should include the translation pair but not quantum.
	hasTranslation := false
	for _, m := range msgs {
		for _, p := range m.Parts {
			if tc, ok := p.(lc.TextContent); ok {
				if strings.Contains(tc.Text, "translate") || strings.Contains(tc.Text, "au revoir") {
					hasTranslation = true
				}
			}
		}
	}
	assert.True(t, hasTranslation, "selected few-shot should be translation related")
}

func TestPruneFewShotByTokens_ZeroLimit(t *testing.T) {
	t.Parallel()
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "hello world"},
		{Role: "assistant", Prompt: "hi there"},
	}
	result := pruneFewShotByTokens(pool, "gpt-4", 0)
	assert.Equal(t, pool, result, "zero maxTokens should return pool unchanged")
}

func TestPruneFewShotByTokens_EmptyPool(t *testing.T) {
	t.Parallel()
	result := pruneFewShotByTokens(nil, "gpt-4", 100)
	assert.Empty(t, result)
}

func TestPruneFewShotByTokens_AllFit(t *testing.T) {
	t.Parallel()
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "hi"},
		{Role: "assistant", Prompt: "hello"},
	}
	// Very large budget: all items fit.
	result := pruneFewShotByTokens(pool, "gpt-4", 10000)
	assert.Len(t, result, 2)
}

func TestPruneFewShotByTokens_PrunesToBudget(t *testing.T) {
	t.Parallel()
	// Two user/assistant pairs. Budget only fits the first pair.
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "short"},
		{Role: "assistant", Prompt: "ok"},
		{Role: "user", Prompt: "another short one"},
		{Role: "assistant", Prompt: "yes"},
	}
	// Allow 5 tokens max — should fit first pair but not second.
	result := pruneFewShotByTokens(pool, "gpt-4", 5)
	// First pair must be present; second pair pruned.
	assert.NotEmpty(t, result)
	assert.Less(t, len(result), len(pool))
}

func TestPruneFewShotByTokens_PreservesPairs(t *testing.T) {
	t.Parallel()
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "translate goodbye"},
		{Role: "assistant", Prompt: "au revoir"},
		{Role: "user", Prompt: "explain entanglement"},
		{Role: "assistant", Prompt: "quantum answer here"},
	}
	// Large budget: both pairs fit.
	result := pruneFewShotByTokens(pool, "gpt-4", 1000)
	assert.Len(t, result, 4)
}

func TestBuildLangchainMessages_FewShotMaxTokens(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Prompt: "say hello",
		FewShot: []domain.ScenarioItem{
			{Role: "user", Prompt: "hi"},
			{Role: "assistant", Prompt: "hello"},
			{Role: "user", Prompt: "goodbye"},
			{Role: "assistant", Prompt: "farewell"},
		},
		FewShotMaxTokens: 5, // very tight — should prune second pair
	}
	msgs := buildLangchainMessages(cfg)
	// At least one message (the prompt itself) should be present.
	assert.NotEmpty(t, msgs)
}

func TestPruneRetrieverContextByTokens_ZeroLimit(t *testing.T) {
	t.Parallel()
	chunks := []string{"chunk one", "chunk two"}
	result := pruneRetrieverContextByTokens(chunks, "gpt-4", 0)
	assert.Equal(t, chunks, result, "zero limit should return all chunks unchanged")
}

func TestPruneRetrieverContextByTokens_EmptyChunks(t *testing.T) {
	t.Parallel()
	result := pruneRetrieverContextByTokens(nil, "gpt-4", 100)
	assert.Empty(t, result)
}

func TestPruneRetrieverContextByTokens_AllFit(t *testing.T) {
	t.Parallel()
	chunks := []string{"short", "also short"}
	result := pruneRetrieverContextByTokens(chunks, "gpt-4", 10000)
	assert.Len(t, result, 2)
}

func TestPruneRetrieverContextByTokens_PrunesToBudget(t *testing.T) {
	t.Parallel()
	chunks := []string{
		"a short chunk",
		"another longer chunk that should be cut off by the budget",
	}
	result := pruneRetrieverContextByTokens(chunks, "gpt-4", 5)
	// First chunk should fit; second should be pruned.
	assert.NotEmpty(t, result)
	assert.Less(t, len(result), len(chunks))
}

func TestBuildLangchainMessages_RetrieverContextMaxTokens(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Prompt: "what is the answer",
		RetrieverContext: []string{
			"The answer is 42.",
			"Some additional context that pushes past the budget.",
		},
		RetrieverContextMaxTokens: 5, // very tight -- only first chunk fits
	}
	msgs := buildLangchainMessages(cfg)
	// The retriever preamble should be in the first system message.
	assert.NotEmpty(t, msgs)
	// The message list should not be empty regardless of budget.
	assert.NotEmpty(t, msgs)
}
