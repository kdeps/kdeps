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

package llm

import (
	"errors"
	"strings"
	"testing"
	"time"

	tiktoken "github.com/pkoukk/tiktoken-go"
	"github.com/stretchr/testify/assert"
)

func TestModelEncoding(t *testing.T) {
	cases := []struct {
		model    string
		expected string
	}{
		{"gpt-4o", "o200k_base"},
		{"gpt-4o-mini", "o200k_base"},
		{"gpt-4", "cl100k_base"},
		{"gpt-4-turbo", "cl100k_base"},
		{"gpt-3.5-turbo", "cl100k_base"},
		{"claude-3-5-sonnet-20241022", "cl100k_base"},
		{"claude-opus-4-7", "cl100k_base"},
		{"gemini-pro", "cl100k_base"},
		{"text-embedding-ada-002", "cl100k_base"},
		{"llama3.2", "p50k_base"},
		{"mistral-7b", "p50k_base"},
		{"unknown-model", "p50k_base"},
	}

	for _, c := range cases {
		t.Run(c.model, func(t *testing.T) {
			assert.Equal(t, c.expected, modelEncoding(c.model))
		})
	}
}

func TestCountTokens(t *testing.T) {
	// Exact counts depend on tiktoken data files; just verify reasonable ranges.
	text := "Hello, world!"
	count := CountTokens("gpt-4o", text)
	assert.Greater(t, count, 0, "token count should be positive")
	assert.Less(t, count, 20, "token count for short text should be small")

	// Longer text should have more tokens.
	long := "The quick brown fox jumps over the lazy dog. " +
		"This sentence is repeated to produce more tokens. " +
		"More text here to ensure the count grows proportionally."
	longCount := CountTokens("gpt-4", long)
	assert.Greater(t, longCount, count, "longer text should have more tokens")
}

func TestCountTokensFallback(t *testing.T) {
	// With an invalid model, still returns a non-negative number.
	count := CountTokens("", "some text to count")
	assert.GreaterOrEqual(t, count, 0)
}

func TestApproximateTokenCount(t *testing.T) {
	resetTokenEncodingCache()
	// Force fallback path by making tiktoken fail.
	orig := tikTokenGetEncoding
	tikTokenGetEncoding = func(string) (*tiktoken.Tiktoken, error) {
		return nil, errors.New("forced failure")
	}
	t.Cleanup(func() {
		tikTokenGetEncoding = orig
		resetTokenEncodingCache()
	})

	text := "Hello, world!" // 13 chars / 4 = 3 tokens
	count := CountTokens("any-model", text)
	assert.Equal(t, 3, count)
}

func TestApproximateTokenCount_Empty(t *testing.T) {
	resetTokenEncodingCache()
	orig := tikTokenGetEncoding
	tikTokenGetEncoding = func(string) (*tiktoken.Tiktoken, error) {
		return nil, errors.New("forced failure")
	}
	t.Cleanup(func() {
		tikTokenGetEncoding = orig
		resetTokenEncodingCache()
	})

	assert.Equal(t, 0, CountTokens("m", ""))
}

func TestCountTokens_CachesEncoding(t *testing.T) {
	resetTokenEncodingCache()
	calls := 0
	orig := tikTokenGetEncoding
	tikTokenGetEncoding = func(name string) (*tiktoken.Tiktoken, error) {
		calls++
		return orig(name)
	}
	t.Cleanup(func() {
		tikTokenGetEncoding = orig
		resetTokenEncodingCache()
	})

	CountTokens("gpt-4", "hello")
	// gpt-4 and gpt-4-turbo share cl100k_base: one load serves both. tiktoken
	// rebuilds the BPE table on every GetEncoding call (~90ms), so a second
	// load here would be the regression this test guards.
	CountTokens("gpt-4-turbo", "hello")
	assert.Equal(t, 1, calls)
}

func TestCountEncodedTokens_ChunksMatchOneEncode(t *testing.T) {
	enc, err := resolveTokenEncoding("gpt-4o")
	if err != nil {
		t.Skip(err)
	}
	for _, text := range []string{
		"",
		"short text",
		strings.Repeat("y", 3*tokenChunkBytes+17),
		strings.Repeat("the quick brown fox ", 1000),
		strings.Repeat("é", 3000), // multi-byte runes across a chunk edge
	} {
		assert.Equal(t, len(enc.Encode(text, nil, nil)), CountEncodedTokens(enc, text), "len %d", len(text))
	}
}

func TestCountEncodedTokens_LongRunIsFast(t *testing.T) {
	enc, err := resolveTokenEncoding("gpt-4o")
	if err != nil {
		t.Skip(err)
	}
	start := time.Now()
	assert.Positive(t, CountEncodedTokens(enc, strings.Repeat("y", 256*1024)))
	// One Encode of this run took minutes; chunked it is well under a second.
	assert.Less(t, time.Since(start), 10*time.Second)
}
