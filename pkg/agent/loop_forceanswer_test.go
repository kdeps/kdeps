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
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func toolHistory(t *testing.T, msgs []map[string]any) string {
	t.Helper()
	b, err := json.Marshal(msgs)
	if err != nil {
		t.Fatalf("marshal history: %v", err)
	}
	return string(b)
}

func TestGatheredToolDigest_InlinesResultsSkipsBlocks(t *testing.T) {
	history := toolHistory(t, []map[string]any{
		{"role": RoleUser, toolParamContent: "world news"},
		{"role": "tool", "name": "web_scraper", toolParamContent: "BBC: ceasefire signed in region X"},
		{"role": "tool", "name": "web_search", toolParamContent: "convergence (5 calls): ALL web/search calls blocked"},
		{"role": "tool", "name": "web_scraper", toolParamContent: "Reuters: markets rose 2%"},
	})
	digest := gatheredToolDigest(history, maxForceAnswerDigestBytes)

	if !strings.Contains(digest, "ceasefire signed") || !strings.Contains(digest, "markets rose 2%") {
		t.Fatalf("digest missing gathered research:\n%s", digest)
	}
	if strings.Contains(digest, "convergence (") {
		t.Fatalf("digest must skip convergence-block notices:\n%s", digest)
	}
	if !strings.Contains(digest, "[web_scraper]") {
		t.Fatalf("digest should label results by tool name:\n%s", digest)
	}
}

// The force-answer inlining is tool-type agnostic: every category's limiter
// emits the same "convergence (N calls): ..." error, so bash/file/code blocks
// are skipped while their gathered results are inlined, exactly like web.
func TestGatheredToolDigest_AllToolTypes(t *testing.T) {
	history := toolHistory(t, []map[string]any{
		{"role": "tool", "name": "bash_exec", toolParamContent: "go version go1.26"},
		{
			"role":           "tool",
			"name":           "bash_exec",
			toolParamContent: "convergence (25 calls): ALL shell commands blocked — consolidate your approach and continue without bash_exec",
		},
		{"role": "tool", "name": "read_file", toolParamContent: "package main func main"},
		{
			"role":           "tool",
			"name":           "read_file",
			toolParamContent: "convergence (40 calls): ALL file reads blocked — work with what you have already read",
		},
		{"role": "tool", "name": "code_search", toolParamContent: "loop.go:914 appendToolRoundTrip"},
		{
			"role":           "tool",
			"name":           "code_search",
			toolParamContent: "convergence (15 calls): ALL code searches blocked — narrow your approach and work with existing results",
		},
	})
	digest := gatheredToolDigest(history, maxForceAnswerDigestBytes)

	for _, want := range []string{"go version go1.26", "package main func main", "loop.go:914 appendToolRoundTrip"} {
		if !strings.Contains(digest, want) {
			t.Fatalf("digest missing gathered %q:\n%s", want, digest)
		}
	}
	if strings.Contains(digest, "convergence (") {
		t.Fatalf("digest must skip every tool type's convergence-block notice:\n%s", digest)
	}
}

func TestGatheredToolDigest_Truncates(t *testing.T) {
	const limit = 64
	big := strings.Repeat("x", limit*2)
	history := toolHistory(t, []map[string]any{
		{"role": "tool", "name": "web_scraper", toolParamContent: big},
	})
	digest := gatheredToolDigest(history, limit)
	if len(digest) > limit+len("\n...[truncated]") {
		t.Fatalf("digest not truncated: %d bytes", len(digest))
	}
	if !strings.HasSuffix(digest, "[truncated]") {
		t.Fatal("expected truncation marker")
	}
}

// Over budget, the newest results win: a late page scrape carries more of the
// answer than the first search snippets.
func TestGatheredToolDigest_KeepsMostRecent(t *testing.T) {
	filler := strings.Repeat("o", 400)
	history := toolHistory(t, []map[string]any{
		{"role": "tool", "name": "web_search", toolParamContent: "OLDEST " + filler},
		{"role": "tool", "name": "web_search", toolParamContent: "MIDDLE " + filler},
		{"role": "tool", "name": "web_scraper", toolParamContent: "NEWEST " + filler},
	})
	digest := gatheredToolDigest(history, 600) // fits roughly one result

	if !strings.Contains(digest, "NEWEST") {
		t.Fatalf("most recent result must be kept:\n%s", digest)
	}
	if strings.Contains(digest, "OLDEST") {
		t.Fatalf("oldest result should be dropped when over budget:\n%s", digest)
	}
}

// The last result is kept even when it alone blows the budget (then truncated).
func TestGatheredToolDigest_KeepsLastEvenIfOversized(t *testing.T) {
	history := toolHistory(t, []map[string]any{
		{"role": "tool", "name": "web_scraper", toolParamContent: strings.Repeat("z", 5000)},
	})
	if d := gatheredToolDigest(history, 100); d == "" {
		t.Fatal("an oversized sole result must still be inlined (truncated), not dropped")
	}
}

// Truncation must not split a UTF-8 rune — scraped CJK/accented text would
// otherwise become invalid UTF-8 in the prompt.
func TestGatheredToolDigest_TruncatesOnRuneBoundary(t *testing.T) {
	history := toolHistory(t, []map[string]any{
		{"role": "tool", "name": "web_scraper", toolParamContent: strings.Repeat("世界", 2000)},
	})
	digest := gatheredToolDigest(history, 501) // odd budget: lands mid-rune
	if !utf8.ValidString(digest) {
		t.Fatalf("digest must remain valid UTF-8 after truncation")
	}
}

func TestTruncateAtRune(t *testing.T) {
	if got := truncateAtRune("hello", 100); got != "hello" {
		t.Fatalf("under budget should pass through, got %q", got)
	}
	got := truncateAtRune(strings.Repeat("世", 10), 7) // 世 = 3 bytes; 7 lands mid-rune
	if !utf8.ValidString(got) {
		t.Fatalf("truncation split a rune: %q", got)
	}
	if !strings.HasSuffix(got, "[truncated]") {
		t.Fatalf("expected truncation marker, got %q", got)
	}
}

func TestGatheredToolDigest_EmptyWhenNoToolResults(t *testing.T) {
	history := toolHistory(t, []map[string]any{
		{"role": RoleUser, toolParamContent: "hi"},
		{"role": RoleAssistant, toolParamContent: "hello"},
	})
	if d := gatheredToolDigest(history, maxForceAnswerDigestBytes); d != "" {
		t.Fatalf("expected empty digest, got %q", d)
	}
}

func TestForceAnswerConfig_InlinesGatheredResults(t *testing.T) {
	history := toolHistory(t, []map[string]any{
		{"role": RoleUser, toolParamContent: "world news"},
		{"role": "tool", "name": "web_scraper", toolParamContent: "BBC: ceasefire signed in region X"},
	})
	cfg := &domain.ChatConfig{
		Tools:    []domain.Tool{{Name: "web_search"}},
		Messages: history,
	}
	got := forceAnswerConfig(cfg)

	if len(got.Tools) != 0 {
		t.Fatal("forceAnswerConfig must strip tools")
	}
	if !strings.Contains(got.Prompt, "ceasefire signed") {
		t.Fatalf("forced prompt must inline gathered research so the model sees it:\n%s", got.Prompt)
	}
	if !strings.Contains(got.Prompt, "Tool budget exhausted") {
		t.Fatalf("forced prompt missing the answer directive:\n%s", got.Prompt)
	}
}

func TestForceAnswerConfig_NoToolsNoOp(t *testing.T) {
	cfg := &domain.ChatConfig{Prompt: "keep"}
	if got := forceAnswerConfig(cfg); got != cfg {
		t.Fatal("forceAnswerConfig should be a no-op when there are no tools")
	}
}
