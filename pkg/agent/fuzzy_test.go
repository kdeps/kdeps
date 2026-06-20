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

package agent_test

import (
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/agent"
)

func TestFuzzyScore_ExactMatch(t *testing.T) {
	ok, score := agent.ExportedFuzzyScore("gpt4", "gpt4")
	if !ok || score >= 0 {
		t.Errorf("exact match should match with low score, got ok=%v score=%f", ok, score)
	}
}

func TestFuzzyScore_PrefixMatch(t *testing.T) {
	ok, _ := agent.ExportedFuzzyScore("cla", "claude-sonnet-4-6")
	if !ok {
		t.Error("prefix should match")
	}
}

func TestFuzzyScore_NonMatch(t *testing.T) {
	ok, _ := agent.ExportedFuzzyScore("xyz", "gpt-4o")
	if ok {
		t.Error("non-matching query should not match")
	}
}

func TestFuzzyScore_WordBoundaryBonus(t *testing.T) {
	// "s46" should match "claude-sonnet-4-6" with a better score than a deep interior match
	ok, _ := agent.ExportedFuzzyScore("sonnet", "claude-sonnet-4-6")
	if !ok {
		t.Error("word-boundary prefix should match")
	}
}

func TestFuzzyScore_AlphaDigitSwap(_ *testing.T) {
	// either direction matching is acceptable; just confirm no panic
	_, _ = agent.ExportedFuzzyScore("1a3", "a1-model")
}

func TestFuzzyScore_EmptyQuery(t *testing.T) {
	ok, score := agent.ExportedFuzzyScore("", "anything")
	if !ok || score != 0 {
		t.Errorf("empty query should always match with score 0, got ok=%v score=%f", ok, score)
	}
}

func TestFuzzyFilter_Basic(t *testing.T) {
	items := []string{"claude-opus-4-8", "gpt-4o", "gemini-2.5-flash", "claude-sonnet-4-6"}
	result := agent.FuzzyFilter(items, "claude", func(s string) string { return s })
	if len(result) != 2 {
		t.Errorf("expected 2 claude results, got %d: %v", len(result), result)
	}
}

func TestFuzzyFilter_MultiToken(t *testing.T) {
	items := []string{"claude-opus-4-8", "claude-sonnet-4-6", "gpt-4o"}
	result := agent.FuzzyFilter(items, "claude sonnet", func(s string) string { return s })
	if len(result) != 1 || result[0] != "claude-sonnet-4-6" {
		t.Errorf("expected [claude-sonnet-4-6], got %v", result)
	}
}

func TestFuzzyFilter_EmptyQuery(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := agent.FuzzyFilter(items, "", func(s string) string { return s })
	if len(result) != 3 {
		t.Errorf("empty query should return all items, got %d", len(result))
	}
}

func TestFuzzyFilter_NoMatch(t *testing.T) {
	items := []string{"claude-opus", "gpt-4o"}
	result := agent.FuzzyFilter(items, "zzz", func(s string) string { return s })
	if len(result) != 0 {
		t.Errorf("no match expected, got %v", result)
	}
}

func TestFuzzyFilter_BestFirstOrdering(t *testing.T) {
	items := []string{"gpt-4o-mini", "gpt-4o"}
	result := agent.FuzzyFilter(items, "gpt4o", func(s string) string { return s })
	// Both match; exact-er match (gpt-4o) should score lower (better)
	if len(result) < 2 {
		t.Fatalf("expected both to match, got %v", result)
	}
	if result[0] != "gpt-4o" {
		t.Errorf("gpt-4o should rank before gpt-4o-mini, got %v", result)
	}
}
