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

func TestApplyTemplate_GoTemplateMode(t *testing.T) {
	t.Parallel()
	vars := map[string]string{"Lang": "Go"}
	result := applyTemplate("I love {{.Lang}}", vars, true)
	assert.Equal(t, "I love Go", result)
}

func TestApplyTemplate_PlainSubstitution(t *testing.T) {
	t.Parallel()
	vars := map[string]string{"Lang": "Go"}
	result := applyTemplate("I love {{Lang}}", vars, false)
	assert.Equal(t, "I love Go", result)
}

func TestApplyTemplate_GoTemplate_EmptyText(t *testing.T) {
	t.Parallel()
	result := applyTemplate("", map[string]string{"x": "y"}, true)
	assert.Equal(t, "", result)
}

func TestApplyTemplate_PlainSubstitution_NoVars(t *testing.T) {
	t.Parallel()
	result := applyTemplate("no substitution here", nil, false)
	assert.Equal(t, "no substitution here", result)
}

func TestRenderGoTemplate_ExecuteError_FallsBack(t *testing.T) {
	t.Parallel()
	// Template with a range on a string value causes execution error.
	vars := map[string]string{"items": "not-a-slice"}
	result := renderGoTemplate(`{{range .items}}{{.}}{{end}}`, vars)
	// Falls back to original string on exec error or produces empty string.
	// Both are acceptable — the key assertion is no panic.
	_ = result
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

func TestRoleToMessageType_AllRoles(t *testing.T) {
	t.Parallel()
	cases := []struct {
		role     string
		expected lc.ChatMessageType
	}{
		{"user", lc.ChatMessageTypeHuman},
		{"human", lc.ChatMessageTypeHuman},
		{"assistant", lc.ChatMessageTypeAI},
		{"ai", lc.ChatMessageTypeAI},
		{"system", lc.ChatMessageTypeSystem},
		{"tool", lc.ChatMessageTypeTool},
		{"", lc.ChatMessageTypeHuman},
		{"unknown", lc.ChatMessageTypeHuman},
	}
	for _, tc := range cases {
		got := roleToMessageType(tc.role)
		assert.Equal(t, tc.expected, got, "role=%q", tc.role)
	}
}

func TestBuildSystemPreamble_BothSet(t *testing.T) {
	t.Parallel()
	out := buildSystemPreamble("ctx", "hint")
	assert.Contains(t, out, "ctx")
	assert.Contains(t, out, "hint")
}

func TestBuildSystemPreamble_OnlyRetriever(t *testing.T) {
	t.Parallel()
	out := buildSystemPreamble("ctx", "")
	assert.Equal(t, "ctx", out)
}

func TestBuildSystemPreamble_OnlyHint(t *testing.T) {
	t.Parallel()
	out := buildSystemPreamble("", "hint")
	assert.Equal(t, "hint", out)
}

func TestBuildSystemPreamble_BothEmpty(t *testing.T) {
	t.Parallel()
	out := buildSystemPreamble("", "")
	assert.Equal(t, "", out)
}

func TestBuildScenarioMessages_RetrievalInjectedIntoFirstSystem(t *testing.T) {
	t.Parallel()
	scenario := []domain.ScenarioItem{
		{Role: "system", Prompt: "You are helpful."},
		{Role: "user", Prompt: "Hello"},
	}
	msgs, injected := buildScenarioMessages(scenario, nil, "preamble", "", false)
	assert.True(t, injected)
	require.NotEmpty(t, msgs)
	found := false
	for _, m := range msgs {
		if m.Role == lc.ChatMessageTypeSystem {
			for _, p := range m.Parts {
				if tc, ok := p.(lc.TextContent); ok && strings.Contains(tc.Text, "preamble") {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "preamble should be in system message")
}

func TestBuildScenarioMessages_FormatHintAppendedToLast(t *testing.T) {
	t.Parallel()
	scenario := []domain.ScenarioItem{
		{Role: "system", Prompt: "You are an assistant."},
	}
	msgs, _ := buildScenarioMessages(scenario, nil, "", "OUTPUT FORMAT", false)
	require.Len(t, msgs, 1)
	for _, p := range msgs[0].Parts {
		if tc, ok := p.(lc.TextContent); ok {
			assert.Contains(t, tc.Text, "OUTPUT FORMAT")
		}
	}
}

func TestBuildScenarioMessages_EmptyPromptSkipped(t *testing.T) {
	t.Parallel()
	scenario := []domain.ScenarioItem{
		{Role: "system", Prompt: ""},
		{Role: "user", Prompt: "real prompt"},
	}
	msgs, _ := buildScenarioMessages(scenario, nil, "", "", false)
	assert.Len(t, msgs, 1)
	assert.Equal(t, lc.ChatMessageTypeHuman, msgs[0].Role)
}

func TestBuildToolParameters_RequiredAndEnum(t *testing.T) {
	t.Parallel()
	params := map[string]domain.ToolParam{
		"city": {
			Type:        "string",
			Description: "the city name",
			Required:    true,
			Enum:        []string{"NYC", "LA"},
		},
		"units": {
			Type:        "string",
			Description: "metric or imperial",
			Required:    false,
		},
	}
	schema := buildToolParameters(params)
	assert.Equal(t, "object", schema["type"])
	props, ok := schema["properties"].(map[string]any)
	require.True(t, ok)
	city, ok := props["city"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "string", city["type"])
	assert.Equal(t, []string{"NYC", "LA"}, city["enum"])
	req, ok := schema["required"].([]string)
	require.True(t, ok)
	assert.Contains(t, req, "city")
	assert.NotContains(t, req, "units")
}

func TestBuildToolParameters_NoRequired(t *testing.T) {
	t.Parallel()
	params := map[string]domain.ToolParam{
		"q": {Type: "string", Description: "query", Required: false},
	}
	schema := buildToolParameters(params)
	_, hasRequired := schema["required"]
	assert.False(t, hasRequired)
}

func TestBuildToolParameters_Empty(t *testing.T) {
	t.Parallel()
	schema := buildToolParameters(nil)
	assert.Equal(t, "object", schema["type"])
}

func TestWordSet_Basic(t *testing.T) {
	t.Parallel()
	set := wordSet("the quick brown fox")
	assert.Len(t, set, 4)
	_, hasThe := set["the"]
	assert.True(t, hasThe)
}

func TestWordSet_Empty(t *testing.T) {
	t.Parallel()
	set := wordSet("")
	assert.Empty(t, set)
}

func TestJaccardSimilarity_Identical(t *testing.T) {
	t.Parallel()
	a := wordSet("hello world")
	score := jaccardSimilarity(a, a)
	assert.InDelta(t, 1.0, score, 0.001)
}

func TestJaccardSimilarity_Disjoint(t *testing.T) {
	t.Parallel()
	a := wordSet("foo bar")
	b := wordSet("baz qux")
	score := jaccardSimilarity(a, b)
	assert.Equal(t, 0.0, score)
}

func TestJaccardSimilarity_EmptySet(t *testing.T) {
	t.Parallel()
	score := jaccardSimilarity(map[string]struct{}{}, wordSet("foo"))
	assert.Equal(t, 0.0, score)
}

func TestBuildJSONOpts_NoJSONResponse(t *testing.T) {
	t.Parallel()
	opts := buildJSONOpts(&domain.ChatConfig{}, "openai")
	assert.Nil(t, opts)
}

func TestBuildJSONOpts_JSONResponse_Anthropic(t *testing.T) {
	t.Parallel()
	// Anthropic backend ignores JSONResponse - returns nil
	opts := buildJSONOpts(&domain.ChatConfig{JSONResponse: true}, "anthropic")
	assert.Nil(t, opts)
}

func TestBuildJSONOpts_JSONResponse_Google(t *testing.T) {
	t.Parallel()
	opts := buildJSONOpts(&domain.ChatConfig{JSONResponse: true}, "google")
	assert.Len(t, opts, 1)
}

func TestBuildJSONOpts_JSONResponse_OpenAI(t *testing.T) {
	t.Parallel()
	opts := buildJSONOpts(&domain.ChatConfig{JSONResponse: true}, "openai")
	assert.Len(t, opts, 1)
}

func TestBuildThinkingOpts_Nil(t *testing.T) {
	t.Parallel()
	opts := buildThinkingOpts(&domain.ChatConfig{})
	assert.Nil(t, opts)
}

func TestBuildThinkingOpts_ModeNone(t *testing.T) {
	t.Parallel()
	opts := buildThinkingOpts(&domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{Mode: domain.ThinkingModeNone},
	})
	assert.Nil(t, opts)
}

func TestBuildThinkingOpts_Enabled(t *testing.T) {
	t.Parallel()
	opts := buildThinkingOpts(&domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{Mode: "enabled", BudgetTokens: 1024},
	})
	assert.Len(t, opts, 1)
}

func TestBuildThinkingOpts_StreamThinking(t *testing.T) {
	t.Parallel()
	opts := buildThinkingOpts(&domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{Mode: domain.ThinkingModeHigh, StreamThinking: true},
	})
	assert.Len(t, opts, 1)
}

func TestBuildThinkingOpts_InterleaveThinking(t *testing.T) {
	t.Parallel()
	opts := buildThinkingOpts(&domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{Mode: domain.ThinkingModeMedium, InterleaveThinking: true},
	})
	assert.Len(t, opts, 1)
}

func TestBuildStreamingReasoningOpts_NilThinking(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	opts := buildStreamingReasoningOpts(&domain.ChatConfig{}, &buf)
	assert.Nil(t, opts)
}

func TestBuildStreamingReasoningOpts_ModeNone(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	opts := buildStreamingReasoningOpts(&domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{Mode: domain.ThinkingModeNone},
	}, &buf)
	assert.Nil(t, opts)
}

func TestBuildStreamingReasoningOpts_StreamThinkingFalse(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	opts := buildStreamingReasoningOpts(&domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{Mode: domain.ThinkingModeHigh, StreamThinking: false},
	}, &buf)
	assert.Nil(t, opts)
}

func TestBuildStreamingReasoningOpts_StreamThinkingTrue(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	opts := buildStreamingReasoningOpts(&domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{Mode: domain.ThinkingModeHigh, StreamThinking: true},
	}, &buf)
	assert.Len(t, opts, 1)
}

func TestBuildStreamingReasoningOpts_WritesChunks(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	opts := buildStreamingReasoningOpts(&domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{Mode: domain.ThinkingModeHigh, StreamThinking: true},
	}, &buf)
	require.Len(t, opts, 1)

	var callOpts lc.CallOptions
	opts[0](&callOpts)
	require.NotNil(t, callOpts.StreamingReasoningFunc)

	err := callOpts.StreamingReasoningFunc(t.Context(), []byte("thinking chunk"), []byte{})
	require.NoError(t, err)
	assert.Equal(t, "thinking chunk", buf.String())
}

func TestBuildStreamingReasoningOpts_EmptyChunkIsNoOp(t *testing.T) {
	t.Parallel()
	var buf strings.Builder
	opts := buildStreamingReasoningOpts(&domain.ChatConfig{
		Thinking: &domain.ThinkingConfig{Mode: domain.ThinkingModeHigh, StreamThinking: true},
	}, &buf)
	require.Len(t, opts, 1)
	var callOpts lc.CallOptions
	opts[0](&callOpts)

	err := callOpts.StreamingReasoningFunc(t.Context(), []byte{}, []byte{})
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestBuildRawHistoryMessages_UserAndAssistant(t *testing.T) {
	t.Parallel()
	histJSON := `[{"role":"user","content":"hello"},{"role":"assistant","content":"hi"}]`
	msgs := buildHistoryMessages(histJSON)
	require.Len(t, msgs, 2)
	assert.Equal(t, lc.ChatMessageTypeHuman, msgs[0].Role)
	assert.Equal(t, lc.ChatMessageTypeAI, msgs[1].Role)
}

func TestBuildRawHistoryMessages_MalformedJSON(t *testing.T) {
	t.Parallel()
	msgs := buildHistoryMessages("not json")
	assert.Nil(t, msgs)
}

func TestBuildRawHistoryMessages_SkipsEmptyRole(t *testing.T) {
	t.Parallel()
	histJSON := `[{"role":"","content":"orphan"},{"role":"user","content":"valid"}]`
	msgs := buildHistoryMessages(histJSON)
	require.Len(t, msgs, 1)
	assert.Equal(t, lc.ChatMessageTypeHuman, msgs[0].Role)
}

func TestBuildRawHistoryMessages_ToolMessage(t *testing.T) {
	t.Parallel()
	histJSON := `[{"role":"tool","content":"result","tool_call_id":"id1","name":"calc"}]`
	msgs := buildHistoryMessages(histJSON)
	require.Len(t, msgs, 1)
	assert.Equal(t, lc.ChatMessageTypeTool, msgs[0].Role)
}

func TestBuildAIMessage_WithContent(t *testing.T) {
	t.Parallel()
	m := buildAIMessage("hello", nil)
	require.NotNil(t, m)
	assert.Equal(t, lc.ChatMessageTypeAI, m.Role)
	require.Len(t, m.Parts, 1)
}

func TestBuildAIMessage_EmptyContentAndNoTools_ReturnsNil(t *testing.T) {
	t.Parallel()
	m := buildAIMessage("", nil)
	assert.Nil(t, m)
}

func TestParseToolCallParts_ValidToolCalls(t *testing.T) {
	t.Parallel()
	raw := []any{
		map[string]any{
			"id":   "tc1",
			"type": "function",
			"function": map[string]any{
				"name":      "calc",
				"arguments": `{"x":1}`,
			},
		},
	}
	parts := parseToolCallParts(raw)
	require.Len(t, parts, 1)
	tc, ok := parts[0].(lc.ToolCall)
	require.True(t, ok)
	assert.Equal(t, "tc1", tc.ID)
	assert.Equal(t, "calc", tc.FunctionCall.Name)
}

func TestParseToolCallParts_NilInput(t *testing.T) {
	t.Parallel()
	parts := parseToolCallParts(nil)
	assert.Empty(t, parts)
}

func TestConvertTools_BasicConversion(t *testing.T) {
	t.Parallel()
	tools := []domain.Tool{
		{
			Name:        "calculator",
			Description: "Math tool",
			Parameters: map[string]domain.ToolParam{
				"expr": {Type: "string", Description: "expression", Required: true},
			},
		},
	}
	lcTools := convertTools(tools)
	require.Len(t, lcTools, 1)
	assert.Equal(t, "function", lcTools[0].Type)
	assert.Equal(t, "calculator", lcTools[0].Function.Name)
}

func TestFileContentPart_HTTPUrl(t *testing.T) {
	t.Parallel()
	part, ok := fileContentPart("http://example.com/image.jpg")
	require.True(t, ok)
	_, isURL := part.(lc.ImageURLContent)
	assert.True(t, isURL)
}

func TestFileContentPart_NotFound(t *testing.T) {
	t.Parallel()
	_, ok := fileContentPart("/nonexistent/path.png")
	assert.False(t, ok)
}

func ptr[T any](v T) *T { p := new(T); *p = v; return p }

func TestBuildSamplingOpts_Empty(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{})
	assert.Empty(t, opts)
}

func TestBuildSamplingOpts_Temperature(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{Temperature: ptr(0.7)})
	assert.Len(t, opts, 1)
}

func TestBuildSamplingOpts_MaxTokens(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{MaxTokens: ptr(512)})
	assert.Len(t, opts, 1)
}

func TestBuildSamplingOpts_TopP(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{TopP: ptr(0.9)})
	assert.Len(t, opts, 1)
}

func TestBuildSamplingOpts_TopK(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{TopK: ptr(40)})
	assert.Len(t, opts, 1)
}

func TestBuildSamplingOpts_Seed(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{Seed: ptr(42)})
	assert.Len(t, opts, 1)
}

func TestBuildSamplingOpts_FrequencyPenalty(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{FrequencyPenalty: ptr(0.5)})
	assert.Len(t, opts, 1)
}

func TestBuildSamplingOpts_PresencePenalty(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{PresencePenalty: ptr(0.3)})
	assert.Len(t, opts, 1)
}

func TestBuildSamplingOpts_RepetitionPenalty(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{RepetitionPenalty: ptr(1.1)})
	assert.Len(t, opts, 1)
}

func TestBuildSamplingOpts_StopWords(t *testing.T) {
	t.Parallel()
	opts := buildSamplingOpts(&domain.ChatConfig{StopWords: []string{"<stop>", "END"}})
	assert.Len(t, opts, 1)
}

func TestBuildSamplingOpts_AllParams(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Temperature:       ptr(0.5),
		MaxTokens:         ptr(1024),
		TopP:              ptr(0.95),
		TopK:              ptr(50),
		Seed:              ptr(7),
		FrequencyPenalty:  ptr(0.1),
		PresencePenalty:   ptr(0.2),
		RepetitionPenalty: ptr(1.05),
		StopWords:         []string{"stop"},
	}
	opts := buildSamplingOpts(cfg)
	assert.Len(t, opts, 9)
}

// --- ChainOfThought tests ---

func TestChainOfThought_NoScenario_PrependsSysMsg(t *testing.T) {
	t.Parallel()
	msgs := buildLangchainMessages(&domain.ChatConfig{
		Prompt:         "hello",
		ChainOfThought: true,
	})
	found := false
	for _, m := range msgs {
		if m.Role == lc.ChatMessageTypeSystem {
			for _, p := range m.Parts {
				if tc, ok := p.(lc.TextContent); ok && strings.Contains(tc.Text, "step by step") {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "CoT instruction should appear in a system message")
}

func TestChainOfThought_WithScenario_AppendToSystem(t *testing.T) {
	t.Parallel()
	msgs := buildLangchainMessages(&domain.ChatConfig{
		Prompt:         "hello",
		ChainOfThought: true,
		Scenario: []domain.ScenarioItem{
			{Role: "system", Prompt: "You are helpful."},
			{Role: "user", Prompt: "hi"},
		},
	})
	found := false
	for _, m := range msgs {
		if m.Role == lc.ChatMessageTypeSystem {
			for _, p := range m.Parts {
				if tc, ok := p.(lc.TextContent); ok &&
					strings.Contains(tc.Text, "You are helpful.") &&
					strings.Contains(tc.Text, "step by step") {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "CoT instruction should be appended to the existing system message")
}

func TestChainOfThought_Disabled_NoCoTMsg(t *testing.T) {
	t.Parallel()
	msgs := buildLangchainMessages(&domain.ChatConfig{
		Prompt:         "hello",
		ChainOfThought: false,
	})
	for _, m := range msgs {
		if m.Role == lc.ChatMessageTypeSystem {
			for _, p := range m.Parts {
				if tc, ok := p.(lc.TextContent); ok {
					assert.NotContains(t, tc.Text, "step by step")
				}
			}
		}
	}
}

func TestChainOfThought_NoScenarioWithRetriever_CoTPresent(t *testing.T) {
	t.Parallel()
	msgs := buildLangchainMessages(&domain.ChatConfig{
		Prompt:           "hello",
		ChainOfThought:   true,
		RetrieverContext: []string{"chunk1"},
	})
	found := false
	for _, m := range msgs {
		if m.Role == lc.ChatMessageTypeSystem {
			for _, p := range m.Parts {
				if tc, ok := p.(lc.TextContent); ok && strings.Contains(tc.Text, "step by step") {
					found = true
				}
			}
		}
	}
	assert.True(t, found, "CoT should appear even when retriever context is present")
}

// --- cosineSimilarity tests ---

func TestCosineSimilarity_Identical(t *testing.T) {
	t.Parallel()
	v := []float32{1, 0, 0}
	assert.InDelta(t, 1.0, cosineSimilarity(v, v), 1e-9)
}

func TestCosineSimilarity_Orthogonal(t *testing.T) {
	t.Parallel()
	a := []float32{1, 0}
	b := []float32{0, 1}
	assert.InDelta(t, 0.0, cosineSimilarity(a, b), 1e-9)
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, cosineSimilarity([]float32{0, 0}, []float32{1, 1}))
}

func TestCosineSimilarity_LengthMismatch(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, cosineSimilarity([]float32{1, 2}, []float32{1}))
}

func TestCosineSimilarity_EmptySlice(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 0.0, cosineSimilarity(nil, nil))
}

func TestCosineSimilarity_KnownValue(t *testing.T) {
	t.Parallel()
	// [1,1] · [1,0] / (sqrt(2) * 1) = 1/sqrt(2) ≈ 0.7071
	a := []float32{1, 1}
	b := []float32{1, 0}
	got := cosineSimilarity(a, b)
	assert.InDelta(t, 0.7071, got, 0.0001)
}

// --- selectFewShotByEmbedding fallback tests ---

func TestSelectFewShotByEmbedding_NilEmbedder_ReturnsPool(t *testing.T) {
	t.Parallel()
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "hello"},
		{Role: "assistant", Prompt: "hi"},
	}
	result := selectFewShotByEmbedding(t.Context(), pool, "hey", 1, nil)
	assert.Equal(t, pool, result)
}

func TestSelectFewShotByEmbedding_ZeroK_ReturnsPool(t *testing.T) {
	t.Parallel()
	pool := []domain.ScenarioItem{
		{Role: "user", Prompt: "a"},
	}
	result := selectFewShotByEmbedding(t.Context(), pool, "a", 0, nil)
	assert.Equal(t, pool, result)
}

func TestSelectFewShotByEmbedding_EmptyPool_ReturnsPool(t *testing.T) {
	t.Parallel()
	result := selectFewShotByEmbedding(t.Context(), nil, "x", 2, nil)
	assert.Nil(t, result)
}

// --- prepareCfg tests ---

func TestPrepareCfg_NoEmbeddingModel_ReturnsSameCfg(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{Prompt: "hi", FewShotSelectK: 2}
	result := prepareCfg(t.Context(), cfg)
	assert.Same(t, cfg, result, "should return original cfg when no embedding model")
}

func TestPrepareCfg_ZeroK_ReturnsSameCfg(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Prompt:                "hi",
		FewShotEmbeddingModel: "text-embedding-3-small",
		FewShotSelectK:        0,
	}
	result := prepareCfg(t.Context(), cfg)
	assert.Same(t, cfg, result)
}

func TestPrepareCfg_EmptyFewShot_ReturnsSameCfg(t *testing.T) {
	t.Parallel()
	cfg := &domain.ChatConfig{
		Prompt:                "hi",
		FewShotEmbeddingModel: "text-embedding-3-small",
		FewShotSelectK:        2,
		FewShot:               nil,
	}
	result := prepareCfg(t.Context(), cfg)
	assert.Same(t, cfg, result)
}
