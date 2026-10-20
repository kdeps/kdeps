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
	"sync"

	tiktoken "github.com/pkoukk/tiktoken-go"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

//nolint:gochecknoglobals // test-replaceable
var tikTokenGetEncoding = tiktoken.GetEncoding

// tiktoken's GetEncoding rebuilds the BPE merge table on every call (~90ms),
// and CountTokens runs once per message in the REPL's token display and per
// prompt in the streamer. One encoding per encoding name is loaded and reused.
//
//nolint:gochecknoglobals // process-wide encoding cache
var (
	tokenEncMu    sync.Mutex
	tokenEncCache = map[string]*tiktoken.Tiktoken{}
)

// resetTokenEncodingCache clears the cache so tests can swap
// tikTokenGetEncoding without a previously cached encoding masking the swap.
func resetTokenEncodingCache() {
	tokenEncMu.Lock()
	tokenEncCache = map[string]*tiktoken.Tiktoken{}
	tokenEncMu.Unlock()
}

const approxCharsPerToken = 4 // rough fallback when tiktoken encoding fails

// CountTokens returns the exact token count for text using tiktoken BPE encoding.
// Falls back to len(text)/4 if the encoding cannot be loaded.
func CountTokens(model, text string) int {
	kdeps_debug.Log("enter: CountTokens")
	enc, err := resolveTokenEncoding(model)
	if err != nil {
		return approximateTokenCount(text)
	}
	return countEncodedTokens(enc, text)
}

func resolveTokenEncoding(model string) (*tiktoken.Tiktoken, error) {
	kdeps_debug.Log("enter: resolveTokenEncoding")
	name := modelEncoding(model)
	tokenEncMu.Lock()
	defer tokenEncMu.Unlock()
	if enc := tokenEncCache[name]; enc != nil {
		return enc, nil
	}
	enc, err := tikTokenGetEncoding(name)
	if err != nil {
		return nil, err
	}
	tokenEncCache[name] = enc
	return enc, nil
}

func approximateTokenCount(text string) int {
	kdeps_debug.Log("enter: approximateTokenCount")
	return len(text) / approxCharsPerToken
}

func countEncodedTokens(enc *tiktoken.Tiktoken, text string) int {
	kdeps_debug.Log("enter: countEncodedTokens")
	return len(enc.Encode(text, nil, nil))
}

// modelEncoding maps a model name to its tiktoken encoding.
func modelEncoding(model string) string {
	kdeps_debug.Log("enter: modelEncoding")
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
