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
	t.Parallel()
	if got := renderThinkingBlock(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
	if got := renderThinkingBlock("  "); got != "" {
		t.Errorf("expected empty output for whitespace input, got %q", got)
	}
}

func TestRenderThinkingBlock_NonEmpty(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	if got := renderThinkingMarkdown(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

func TestRenderThinkingMarkdown_NonEmpty(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	if got := renderMarkdown(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
	if got := renderMarkdown("  "); got != "" {
		t.Errorf("expected empty output for whitespace input, got %q", got)
	}
}

func TestRenderMarkdown_NonEmpty(t *testing.T) {
	t.Parallel()
	got := renderMarkdown("hello world")
	if got == "" {
		t.Error("renderMarkdown should return non-empty output")
	}
}

func TestRenderREPLOutput_Empty(t *testing.T) {
	t.Parallel()
	if got := renderREPLOutput(""); got != "" {
		t.Errorf("expected empty output for empty input, got %q", got)
	}
}

func TestRenderREPLOutput_Plain(t *testing.T) {
	t.Parallel()
	got := renderREPLOutput("hello world")
	if got == "" {
		t.Error("renderREPLOutput should return non-empty output")
	}
}

func TestRenderREPLOutput_WithThinking(t *testing.T) {
	t.Parallel()
	got := renderREPLOutput("<thinking>\nplanning\n</thinking>\nresponse")
	if !strings.Contains(got, "response") {
		t.Error("renderREPLOutput should include main response")
	}
	if !strings.Contains(got, "\x1b") {
		t.Error("renderREPLOutput should include thinking block with ANSI styling")
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
