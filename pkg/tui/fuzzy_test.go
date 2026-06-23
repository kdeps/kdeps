//go:build !js

package tui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for coverage gaps in fuzzy.go not covered by modelpicker_test.go

func TestFuzzyMatchEntries_AllDelimiters(t *testing.T) {
	// "/" is a FieldsFunc delimiter, so after TrimSpace "/" produces no tokens.
	entries := [][]ModelEntry{
		{{Name: "llama3"}},
		nil,
	}
	for _, e := range entries {
		got := fuzzyMatchEntries(e, "/")
		assert.Equal(t, e, got)
	}
}

func TestFuzzyScore_EmptyQuery(t *testing.T) {
	// Empty query always matches with score 0.
	ok, score := fuzzyScore("", "any text here")
	assert.True(t, ok)
	assert.Equal(t, float64(0), score)
}

func TestFuzzyMatchEntries_SortsByScore(t *testing.T) {
	// "llama" at start wins over "llama" buried mid-name (lower score = better).
	entries := []ModelEntry{
		{Name: "xxxllama"}, // 'l' found at position 3 -> higher score (worse)
		{Name: "llama"},    // 'l' found at position 0, word boundary -> lower score (better)
	}
	got := fuzzyMatchEntries(entries, "llama")
	require.Len(t, got, 2)
	assert.Equal(t, "llama", got[0].Name, "leading match should rank first")
}

func TestScoreFuzzy_ExactMatch(t *testing.T) {
	// When query equals text exactly, the exact bonus (fuzzyExactBonus) is applied.
	score, ok := scoreFuzzy("llama", "llama")
	assert.True(t, ok)
	assert.Less(t, score, float64(0), "exact match should have negative score from bonus")
}

func TestFuzzyScore_SwapDigitAlpha(t *testing.T) {
	// "3b" direct match in "llama b3 model" fails (no '3' before 'b' in order),
	// but the swap "b3" matches. The swap path should return ok=true.
	ok, _ := fuzzyScore("3b", "llama b3 model")
	assert.True(t, ok, "swap path (3b->b3) should match 'llama b3 model'")
}

func TestSwapAlphaDigit_DigitFirst(t *testing.T) {
	// digit-first: "3b" -> "b3"
	assert.Equal(t, "b3", swapAlphaDigit("3b"))
}

func TestSwapAlphaDigit_AlphaFirst(t *testing.T) {
	// alpha-first: "b3" -> "3b"
	assert.Equal(t, "3b", swapAlphaDigit("b3"))
}

func TestSwapAlphaDigit_NoSwap(t *testing.T) {
	assert.Equal(t, "", swapAlphaDigit(""))
	assert.Equal(t, "", swapAlphaDigit("abc"))
	assert.Equal(t, "", swapAlphaDigit("123"))
}

func TestSwapPrefix_DigitThenAlpha(t *testing.T) {
	// isDigit prefix + isAlpha rest: "3b" -> "b3"
	assert.Equal(t, "b3", swapPrefix("3b", isDigit, isAlpha))
}

func TestSwapPrefix_AlphaThenDigit(t *testing.T) {
	// isAlpha prefix + isDigit rest: "b3" -> "3b"
	assert.Equal(t, "3b", swapPrefix("b3", isAlpha, isDigit))
}

func TestSwapPrefix_NoMatch(t *testing.T) {
	// First char doesn't match first class
	assert.Equal(t, "", swapPrefix("123", isAlpha, isDigit))
}

func TestSwapPrefix_AllPrefix(t *testing.T) {
	// All chars match first class (no rest for second class)
	assert.Equal(t, "", swapPrefix("abc", isAlpha, isDigit))
}

func TestSwapPrefix_MixedRest(t *testing.T) {
	// "a1c": prefix "a" is alpha, rest has '1' (digit ok) then 'c' (not digit) -> "".
	assert.Equal(t, "", swapPrefix("a1c", isAlpha, isDigit))
}

func TestApplyFuzzyMatchScore_Gap(t *testing.T) {
	// Match at i=3 with lastMatch=0 (gap, not consecutive, lastMatch >= 0).
	tRunes := []rune("abcde")
	score, cons := applyFuzzyMatchScore(0, 3, 0, 0, tRunes)
	assert.Equal(t, 0, cons)
	assert.Greater(t, score, float64(0))
}

func TestApplyFuzzyMatchScore_WordBoundary(t *testing.T) {
	// Match at position 1 preceded by '-' (boundary char) -> bonus (score decreases).
	tRunes := []rune("-abc")
	score, _ := applyFuzzyMatchScore(0, 1, -1, 0, tRunes)
	assert.Less(t, score, float64(0), "word boundary should give bonus (negative score contribution)")
}

func TestApplyFuzzyMatchScore_FirstChar(t *testing.T) {
	// Match at position 0, lastMatch=-1. lastMatch == i-1 is -1 == -1 = true, so consecutive is incremented.
	tRunes := []rune("abc")
	score, cons := applyFuzzyMatchScore(0, 0, -1, 0, tRunes)
	// Position 0 is also a word boundary (i==0), so word boundary bonus applies (score < 0).
	assert.Equal(t, 1, cons) // consecutive incremented (lastMatch==i-1 when both are -1 and 0-1=-1)
	assert.Less(t, score, float64(0)) // word boundary bonus at pos 0
}

func TestFuzzyScore_NoMatchSwapFails(t *testing.T) {
	// Both direct and swap fail -> false.
	ok, _ := fuzzyScore("zzz", "abc")
	assert.False(t, ok)
}

func TestSwapPrefix_Empty(t *testing.T) {
	// Empty string has no prefix match.
	assert.Equal(t, "", swapPrefix("", isAlpha, isDigit))
}
