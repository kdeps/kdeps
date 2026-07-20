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

// Package tokenizer provides text tokenization and normalization for the searchlocal index.
package tokenizer

import (
	"regexp"
	"strings"
	"unicode"
)

// Token represents a processed token with its position.
type Token struct {
	Text     string
	Position int
}

// Tokenizer handles text tokenization and normalization.
type Tokenizer struct {
	minWordLength int
	stopWords     map[string]bool
	wordPattern   *regexp.Regexp
}

// NewTokenizer creates a new tokenizer instance.
func NewTokenizer() *Tokenizer {
	stopWords := map[string]bool{
		"a": true, "an": true, "and": true, "are": true, "as": true, "at": true,
		"be": true, "by": true, "for": true, "from": true, "has": true, "he": true,
		"in": true, "is": true, "it": true, "its": true, "of": true, "on": true,
		"that": true, "the": true, "to": true, "was": true, "will": true, "with": true,
	}

	return &Tokenizer{
		minWordLength: 2,
		stopWords:     stopWords,
		wordPattern:   regexp.MustCompile(`\w+`),
	}
}

// Tokenize splits text into tokens with positions.
func (t *Tokenizer) Tokenize(text string) []Token {
	var tokens []Token
	position := 0

	matches := t.wordPattern.FindAllStringIndex(text, -1)
	for _, match := range matches {
		word := text[match[0]:match[1]]
		normalized := t.normalize(word)

		if normalized != "" && !t.stopWords[normalized] {
			tokens = append(tokens, Token{
				Text:     normalized,
				Position: position,
			})
			position++
		}
	}

	return tokens
}

// TokenizeToStrings returns just the token strings without positions.
func (t *Tokenizer) TokenizeToStrings(text string) []string {
	tokens := t.Tokenize(text)
	result := make([]string, len(tokens))
	for i, token := range tokens {
		result[i] = token.Text
	}
	return result
}

// normalize converts a word to lowercase and applies basic stemming.
func (t *Tokenizer) normalize(word string) string {
	word = strings.ToLower(word)

	var builder strings.Builder
	for _, r := range word {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' {
			builder.WriteRune(r)
		}
	}
	word = builder.String()

	if len(word) < t.minWordLength {
		return ""
	}

	word = t.simpleStem(word)
	return word
}

// simpleStem applies basic stemming rules.
func (t *Tokenizer) simpleStem(word string) string {
	suffixes := []string{"ing", "ed", "ly", "es", "s"}

	for _, suffix := range suffixes {
		if len(word) > len(suffix)+2 && strings.HasSuffix(word, suffix) {
			return word[:len(word)-len(suffix)]
		}
	}

	return word
}
