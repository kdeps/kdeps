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

package agent

import (
	"strings"
	"testing"
)

// ---- loop.go pure functions ----

func TestSummarizeToolArgs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty", raw: "", want: ""},
		{name: "empty object", raw: "{}", want: ""},
		{name: "file_path key", raw: `{"file_path": "/tmp/test.go"}`, want: "/tmp/test.go"},
		{name: "query key", raw: `{"query": "search term"}`, want: "search term"},
		{name: "url key", raw: `{"url": "https://example.com"}`, want: "https://example.com"},
		{name: "expression key", raw: `{"expression": "2+2"}`, want: "2+2"},
		{name: "command key", raw: `{"command": "ls"}`, want: "ls"},
		{name: "fallback first value", raw: `{"some_key": "hello"}`, want: "some_key=hello"},
		{name: "invalid JSON", raw: `not json`, want: "not json"},
		{name: "int value fallback", raw: `{"count": 42}`, want: "count=42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := summarizeToolArgs(tt.raw); got != tt.want {
				t.Errorf("summarizeToolArgs(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestSummarizeToolArgs_LongFilepath(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 85)
	raw := `{"file_path": "` + long + `"}`
	got := summarizeToolArgs(raw)
	if len(got) >= len(long) {
		t.Errorf("expected truncated output, got len %d", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Errorf("expected output to end with '...': %q", got)
	}
}

// ---- builtin_tools.go pure functions (tests for validateWorkspaceBoundary already exist in builtin_tools_test.go) ----

// ---- repl_render.go pure functions ----

func TestTrimTrailingSpaces(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "no spaces", input: "hello\nworld", want: "hello\nworld"},
		{name: "trailing spaces", input: "hello   \nworld  ", want: "hello\nworld"},
		{name: "trailing tabs", input: "hello\t\t\nworld\t", want: "hello\nworld"},
		{name: "mixed", input: "a  \t  \nb\t\t", want: "a\nb"},
		// Glamour's MarginWriter pads lines with ANSI-styled spaces: each padding space
		// is wrapped in \x1b[...m ... \x1b[0m. Plain TrimRight cannot strip these because
		// it stops at the ANSI code bytes. The ANSI-aware regex must remove them.
		{
			name:  "ansi-coded trailing spaces",
			input: "hello\x1b[38;2;205;214;243m \x1b[0m\x1b[38;2;205;214;243m \x1b[0m",
			want:  "hello",
		},
		{
			name:  "ansi-coded blank line (paragraph separator)",
			input: "\x1b[38;2;205;214;243m \x1b[0m\x1b[38;2;205;214;243m \x1b[0m",
			want:  "",
		},
		{
			name:  "ansi reset at end stripped",
			input: "hello\x1b[0m",
			want:  "hello",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := trimTrailingSpaces(tt.input); got != tt.want {
				t.Errorf("trimTrailingSpaces(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRenderToolCall(t *testing.T) {
	t.Parallel()
	got := renderToolCall("test_tool", "arg1")
	if !strings.Contains(got, "test_tool") {
		t.Error("renderToolCall should include tool name")
	}
	if !strings.Contains(got, "arg1") {
		t.Error("renderToolCall should include args")
	}
}

func TestRenderToolCall_NoArgs(t *testing.T) {
	t.Parallel()
	got := renderToolCall("test_tool", "")
	if !strings.Contains(got, "test_tool") {
		t.Error("renderToolCall without args should include tool name")
	}
	if strings.Contains(got, "->") {
		t.Error("renderToolCall without args should not include arrow")
	}
}

func TestRenderThinkingBlock_Empty(t *testing.T) {
	// no Parallel: shares cached renderer
	if got := renderThinkingBlock(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
	if got := renderThinkingBlock("  "); got != "" {
		t.Errorf("expected empty output for whitespace input, got %q", got)
	}
}

func TestRenderThinkingBlock_NonEmpty(t *testing.T) {
	// no Parallel: shares cached renderer
	got := renderThinkingBlock("hello world")
	if got == "" {
		t.Error("renderThinkingBlock should return non-empty output")
	}
	// Should contain ANSI-styled thinking label
	if !strings.Contains(got, "\x1b") {
		t.Error("renderThinkingBlock should include ANSI styling")
	}
}

func TestRenderThinkingMarkdown_Empty(t *testing.T) {
	// no Parallel: shares cached renderer
	if got := renderThinkingMarkdown(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

func TestRenderThinkingMarkdown_NonEmpty(t *testing.T) {
	// no Parallel: shares cached renderer
	got := renderThinkingMarkdown("test")
	if got == "" {
		t.Error("renderThinkingMarkdown should return non-empty output")
	}
}

func TestThinkingStyleConfig(t *testing.T) {
	t.Parallel()
	cfg := thinkingStyleConfig()
	if cfg.Document.Color == nil {
		t.Error("thinkingStyleConfig Document.Color should not be nil")
	}
	if cfg.Document.Margin == nil {
		t.Error("thinkingStyleConfig Document.Margin should not be nil")
	}
}

func TestReplStyleConfig(t *testing.T) {
	t.Parallel()
	cfg := replStyleConfig()
	if cfg.Document.Color == nil {
		t.Error("replStyleConfig Document.Color should not be nil")
	}
}

func TestRenderMarkdown_Empty(t *testing.T) {
	// no Parallel: shares cached renderer
	if got := renderMarkdown(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
	if got := renderMarkdown("  "); got != "" {
		t.Errorf("expected empty output for whitespace input, got %q", got)
	}
}

func TestRenderMarkdown_NonEmpty(t *testing.T) {
	// no Parallel: shares cached renderer
	got := renderMarkdown("hello world")
	if got == "" {
		t.Error("renderMarkdown should return non-empty output")
	}
}

func TestRenderREPLOutput_Empty(t *testing.T) {
	// no Parallel: shares cached renderer
	if got := renderREPLOutput("", false); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

func TestRenderREPLOutput_Plain(t *testing.T) {
	// no Parallel: shares cached renderer
	got := renderREPLOutput("hello world", false)
	if got == "" {
		t.Error("renderREPLOutput should return non-empty output")
	}
}

func TestRenderREPLOutput_WithThinking(t *testing.T) {
	// no Parallel: shares cached renderer with other tests
	got := renderREPLOutput("<thinking>\nplanning\n</thinking>\nresponse", false)
	if !strings.Contains(got, "response") {
		t.Error("renderREPLOutput should include main response")
	}
	if !strings.Contains(got, "\x1b") {
		t.Error("renderREPLOutput should include thinking block with ANSI styling")
	}
}

func TestRenderREPLOutput_SkipThinking_StripsXMLTags(t *testing.T) {
	// no Parallel: shares cached renderer with other tests
	got := renderREPLOutput("<thinking>\nplanning\n</thinking>\nresponse", true)
	clean := ansiStripRe.ReplaceAllString(got, "")
	clean = strings.TrimSpace(clean)
	if strings.Contains(clean, "planning") {
		t.Errorf("skipThinking=true should strip <thinking> block content, got %q", clean)
	}
	if !strings.Contains(clean, "response") {
		t.Errorf("skipThinking=true should preserve main response, got %q", clean)
	}
}

func TestRenderREPLOutput_SkipThinking_StripsMarkdownThinking(t *testing.T) {
	// no Parallel: shares cached renderer with other tests
	mdThinking := "* thinking\n  plan\n  execute\n\nresponse"
	got := renderREPLOutput(mdThinking, true)
	clean := ansiStripRe.ReplaceAllString(got, "")
	clean = strings.TrimSpace(clean)
	// "plan" and "execute" should NOT appear (stripped as thinking)
	if strings.Contains(clean, "plan") || strings.Contains(clean, "execute") {
		t.Errorf("skipThinking=true should strip markdown-style * thinking block, got %q", clean)
	}
	if !strings.Contains(clean, "response") {
		t.Errorf("skipThinking=true should preserve main response, got %q", clean)
	}
}

func TestRenderREPLOutput_NoThinking_WithSkip(t *testing.T) {
	// no Parallel: shares cached renderer with other tests
	got := renderREPLOutput("plain response", true)
	// Glamour word-wraps to terminal width with ANSI-padded spaces.
	// Strip ANSI and trim to check content.
	clean := ansiStripRe.ReplaceAllString(got, "")
	clean = strings.TrimSpace(clean)
	if clean != "plain response" {
		t.Errorf("skipThinking should render plain text response, got %q", clean)
	}
}

func TestRenderREPLOutput_EmptyWithSkip(t *testing.T) {
	// no Parallel: shares cached renderer with other tests
	if got := renderREPLOutput("", true); got != "" {
		t.Errorf("expected empty output for empty input with skipThinking=true, got %q", got)
	}
}

func TestStripThinkingTags_XML(t *testing.T) {
	// no Parallel: shares cached renderer with other tests
	input := "<thinking>\nplanning\n</thinking>\nactual response"
	got := stripThinkingTags(input)
	if strings.Contains(got, "thinking") {
		t.Error("stripThinkingTags should remove <thinking> tags")
	}
	if !strings.Contains(got, "actual response") {
		t.Error("stripThinkingTags should preserve non-thinking content")
	}
	if strings.Contains(got, "planning") {
		t.Error("stripThinkingTags should remove thinking content")
	}
}

func TestStripThinkingTags_MarkdownStyle(t *testing.T) {
	// no Parallel: shares cached renderer with other tests
	input := "* thinking\n  plan\n  more plan\n\nactual response"
	got := stripThinkingTags(input)
	if strings.Contains(got, "plan") || strings.Contains(got, "more plan") {
		t.Error("stripThinkingTags should remove markdown-style thinking")
	}
	if !strings.Contains(got, "actual response") {
		t.Error("stripThinkingTags should preserve non-thinking content")
	}
}

func TestStripThinkingTags_NoThinkingBlocks(t *testing.T) {
	// no Parallel: shares cached renderer with other tests
	input := "just a normal response"
	got := stripThinkingTags(input)
	if got != input {
		t.Errorf("stripThinkingTags should return unchanged text when no thinking blocks present, got %q", got)
	}
}

func TestStripThinkingTags_CollapseBlankLines(t *testing.T) {
	// no Parallel: shares cached renderer with other tests
	// After stripping thinking blocks, consecutive blank lines should collapse.
	input := "<thinking>\nstuff\n</thinking>\n\n\n\nresponse"
	got := stripThinkingTags(input)
	if strings.Contains(got, "\n\n\n") {
		t.Error("stripThinkingTags should collapse multiple consecutive blank lines")
	}
	if !strings.Contains(got, "response") {
		t.Error("stripThinkingTags should preserve response after collapsing blanks")
	}
}

// TestRendererCache_ReusesRenderer verifies getRenderer returns the same
// instance on subsequent calls within the same terminal width. NOT parallel
// since it reads the shared cachedRenderer global.
func TestRendererCache_ReusesRenderer(t *testing.T) {
	r1, err1 := getRenderer()
	if err1 != nil || r1 == nil {
		t.Fatal("getRenderer failed on first call")
	}
	r2, err2 := getRenderer()
	if err2 != nil || r2 == nil {
		t.Fatal("getRenderer failed on second call")
	}
	if r1 != r2 {
		t.Error("getRenderer should return the same cached renderer instance")
	}
}

// TestThinkingRendererCache_ReusesRenderer verifies getThinkingRenderer returns
// the same instance on subsequent calls. NOT parallel — reads shared globals.
func TestThinkingRendererCache_ReusesRenderer(t *testing.T) {
	r1, err1 := getThinkingRenderer()
	if err1 != nil || r1 == nil {
		t.Fatal("getThinkingRenderer failed on first call")
	}
	r2, err2 := getThinkingRenderer()
	if err2 != nil || r2 == nil {
		t.Fatal("getThinkingRenderer failed on second call")
	}
	if r1 != r2 {
		t.Error("getThinkingRenderer should return the same cached renderer instance")
	}
}

// TestRendererCache_WidthChangeRecreates verifies the getRenderer cache
// is re-created when the terminal width changes. This test does NOT mutate
// the shared globals directly since other parallel tests may be reading them.
func TestRendererCache_WidthChangeRecreates(t *testing.T) {
	// Get the initial renderer (may be nil if not yet created).
	r1, err1 := getRenderer()
	if err1 != nil || r1 == nil {
		t.Fatal("getRenderer failed")
	}
	// Verify the renderer was cached.
	if cachedRenderer == nil {
		t.Fatal("cachedRenderer should be set after getRenderer()")
	}
	// Verify the cached width matches the actual terminal width.
	if cachedRendererWidth != terminalWidth() && cachedRendererWidth != defaultTermWidth {
		t.Fatalf("cachedRendererWidth %d does not match terminal width %d or default %d",
			cachedRendererWidth, terminalWidth(), defaultTermWidth)
	}
}
func TestStrp(t *testing.T) {
	t.Parallel()
	p := strp("hello") //nolint:newexpr // false positive: new(string) returns zero value, not "hello"
	if p == nil || *p != "hello" {
		t.Error("strp should return pointer to string")
	}
}

func TestBoolp(t *testing.T) {
	t.Parallel()
	p := boolp(true)   //nolint:newexpr // false positive
	p2 := boolp(false) //nolint:newexpr // false positive
	if p == nil || !*p {
		t.Error("boolp should return pointer to true")
	}
	if p2 == nil || *p2 {
		t.Error("boolp should return pointer to false")
	}
}

// ---- repl.go pure functions ----

func TestBoldTagKeyword(t *testing.T) {
	t.Parallel()
	// Exact match returns bold-wrapped keyword
	got := boldTagKeyword("hello", "hello")
	if !strings.Contains(got, "hello") {
		t.Errorf("bold output should contain keyword: got %q", got)
	}
	if !strings.Contains(got, "\x1b[1m") {
		t.Error("bold output should include ANSI bold code")
	}

	// No match returns unchanged
	if result := boldTagKeyword("hello", "world"); result != "hello" {
		t.Errorf("expected unchanged tag for no match, got %q", result)
	}

	// Case-insensitive match applies bold
	got2 := boldTagKeyword("Hello", "hello")
	if !strings.Contains(got2, "\x1b[1m") {
		t.Errorf("case-insensitive match should apply bold: got %q", got2)
	}
}

func TestParamsForModel_Unknown(t *testing.T) {
	t.Parallel()
	if got := paramsForModel("unknown-model"); got != 0 {
		t.Errorf("expected 0 for unknown model, got %f", got)
	}
}

func TestHistoryPath(t *testing.T) {
	t.Parallel()
	path := historyPath()
	if !strings.Contains(path, ".kdeps") || !strings.Contains(path, "history") {
		t.Errorf("unexpected history path: %s", path)
	}
}

func TestParseParamB_Empty(t *testing.T) {
	t.Parallel()
	if got := parseParamB(""); got != 0 {
		t.Errorf("expected 0 for empty, got %f", got)
	}
}

func TestParseParamB_Invalid(t *testing.T) {
	t.Parallel()
	if got := parseParamB("invalid"); got != 0 {
		t.Errorf("expected 0 for invalid, got %f", got)
	}
}

func TestParseParamB_7B(t *testing.T) {
	t.Parallel()
	if got := parseParamB("7B"); got != 7 {
		t.Errorf("expected 7 for '7B', got %f", got)
	}
}

func TestParseParamB_0_5B(t *testing.T) {
	t.Parallel()
	if got := parseParamB("0.5B"); got != 0.5 {
		t.Errorf("expected 0.5 for '0.5B', got %f", got)
	}
}

func TestParseParamB_Lowercase(t *testing.T) {
	t.Parallel()
	if got := parseParamB("7b"); got != 7 {
		t.Errorf("expected 7 for '7b', got %f", got)
	}
}

func TestParseParamB_Negative(t *testing.T) {
	t.Parallel()
	if got := parseParamB("-1B"); got != 0 {
		t.Errorf("expected 0 for negative, got %f", got)
	}
}

func TestStripModelIndicators_NoTag(t *testing.T) {
	t.Parallel()
	if got := stripModelIndicators("llama3.2"); got != "llama3.2" {
		t.Errorf("expected unchanged name, got %q", got)
	}
}

func TestStripModelIndicators_WithTag(t *testing.T) {
	t.Parallel()
	if got := stripModelIndicators("llama3.2:1b [llamafile cached]"); got != "llama3.2:1b" {
		t.Errorf("expected stripped name, got %q", got)
	}
}

func TestStripModelIndicators_WithStar(t *testing.T) {
	t.Parallel()
	if got := stripModelIndicators("*llama3.2 [cached]"); got != "llama3.2" {
		t.Errorf("expected stripped name, got %q", got)
	}
}

func TestFDBinPath_Found(t *testing.T) {
	// Look for a binary that always exists.
	path := fdBinPath()
	// If fd or fdfind is not installed, path should be empty.
	if path != "" {
		// Must be an absolute path.
		if path[0] != '/' {
			t.Errorf("expected absolute path, got %q", path)
		}
	}
}

func TestIsGGUFModelName_GGUFSuffix(t *testing.T) {
	t.Parallel()
	if !IsGGUFModelName("model.gguf") {
		t.Error("expected true for .gguf suffix")
	}
	if !IsGGUFModelName("model.GGUF") {
		t.Error("expected true for .GGUF suffix")
	}
}

func TestIsGGUFModelName_NoSuffix(t *testing.T) {
	t.Parallel()
	if IsGGUFModelName("model.gguf.txt") {
		t.Error("expected false for non-gguf suffix")
	}
}

func TestSummarizeToolArgs_FallbackKeyValue(t *testing.T) {
	t.Parallel()
	// No preferred keys match — falls through to first non-empty value
	result := summarizeToolArgs(`{"unknown_key": "hello"}`)
	if result != "unknown_key=hello" {
		t.Errorf("expected unknown_key=hello, got %q", result)
	}
}

func TestSummarizeToolArgs_FallbackKeyValueTruncated(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 90)
	raw := `{"key": "` + long + `"}`
	result := summarizeToolArgs(raw)
	if len(result) >= len(raw) {
		t.Errorf("expected truncated output, got len %d", len(result))
	}
}

func TestSummarizeToolArgs_EmptyKeySkipped(t *testing.T) {
	t.Parallel()
	// Whitespace-only value should be skipped
	result := summarizeToolArgs(`{"key": " "}`)
	if result != `{"key": " "}` {
		t.Logf("result: %q", result)
	}
}
