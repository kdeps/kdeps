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
	"strings"

	tiktoken "github.com/pkoukk/tiktoken-go"
)

//nolint:gochecknoglobals // test-replaceable
var tikTokenGetEncoding = tiktoken.GetEncoding

const approxCharsPerToken = 4 // rough fallback when tiktoken encoding fails

// CountTokens returns the exact token count for text using tiktoken BPE encoding.
// Falls back to len(text)/4 if the encoding cannot be loaded.
func CountTokens(model, text string) int {
	enc, err := tikTokenGetEncoding(modelEncoding(model))
	if err != nil {
		return len(text) / approxCharsPerToken
	}
	return len(enc.Encode(text, nil, nil))
}

// modelEncoding maps a model name to its tiktoken encoding.
func modelEncoding(model string) string {
	m := strings.ToLower(model)
	switch {
	case strings.HasPrefix(m, "gpt-4o"):
		return "o200k_base"
	case strings.HasPrefix(m, "gpt-4"),
		strings.HasPrefix(m, "gpt-3.5"),
		strings.HasPrefix(m, "claude"),
		strings.HasPrefix(m, "gemini"),
		strings.HasPrefix(m, "text-embedding-ada"):
		return "cl100k_base"
	default:
		return "p50k_base"
	}
}
