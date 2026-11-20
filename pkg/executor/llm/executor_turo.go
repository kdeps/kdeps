// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package llm

import (
	"context"
	"sync/atomic"
)

// PromptReducer rewrites text into fewer tokens. Agent mode installs turo.
type PromptReducer func(ctx context.Context, text string) string

//nolint:gochecknoglobals // installed once at startup, read by chats with harness.turo
var promptReducer atomic.Pointer[PromptReducer]

// SetPromptReducer installs the reducer a chat with harness.turo uses.
func SetPromptReducer(fn PromptReducer) { promptReducer.Store(&fn) }

// turoChat reduces the text of every system and user message, and returns the
// reduced prompt. Tool calls, tool results and images stay as they are. With
// no reducer installed, nothing changes.
func turoChat(ctx context.Context, messages []map[string]any, prompt string) string {
	f := promptReducer.Load()
	if f == nil || *f == nil {
		return prompt
	}
	reduce := *f
	for _, m := range messages {
		role := m[jsonFieldRole]
		if role != roleSystem && role != roleUser {
			continue
		}
		content, ok := m[jsonFieldContent].(string)
		if !ok || content == "" {
			continue
		}
		reduced := reduce(ctx, content)
		m[jsonFieldContent] = reduced
		if content == prompt {
			prompt = reduced
		}
	}
	return prompt
}
