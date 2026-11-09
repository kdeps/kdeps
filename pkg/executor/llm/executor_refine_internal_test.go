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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package llm

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/refine"
)

func TestRefineChat(t *testing.T) {
	refine.Configure(refine.Settings{
		SystemPrompt: func() string { return "REWRITE" },
		Trivial:      func(s string) bool { return s == "hi" },
	})
	t.Cleanup(func() { refine.Configure(refine.Settings{}) })
	var asked []map[string]any
	answer := func(reply string, err error) chatCall {
		return func(messages []map[string]any, _ ChatRequestConfig) (map[string]any, error) {
			asked = messages
			return map[string]any{jsonFieldMessage: map[string]any{jsonFieldContent: reply}}, err
		}
	}
	conversation := func(prompt any) []map[string]any {
		return []map[string]any{
			{jsonFieldRole: roleSystem, jsonFieldContent: "chat system"},
			{jsonFieldRole: roleUser, jsonFieldContent: "earlier"},
			{jsonFieldRole: roleUser, jsonFieldContent: prompt},
		}
	}

	msgs := conversation("fix it")
	got := refineChat(answer(`"fix the bug in a.go"`, nil), msgs, "fix it", 0)
	assert.Equal(t, "fix the bug in a.go", got)
	assert.Equal(t, "fix the bug in a.go", msgs[2][jsonFieldContent], "the prompt message is replaced")
	require.Len(t, asked, 3, "refiner system, history, request")
	assert.Equal(t, "REWRITE", asked[0][jsonFieldContent])
	assert.Equal(t, "earlier", asked[1][jsonFieldContent], "the chat's own system prompt is left out")
	assert.Equal(t, "Request:\nfix it", asked[2][jsonFieldContent])

	asked = nil
	assert.Equal(t, "hi", refineChat(answer("x", nil), conversation("hi"), "hi", 0), "trivial")
	assert.Nil(t, asked)
	assert.Equal(t, "p", refineChat(answer("x", nil), nil, "p", 0), "no messages")
	assert.Equal(t, "fix it", refineChat(answer("x", nil), conversation([]any{"image"}), "fix it", 0), "not plain text")
	failed := refineChat(answer("", errors.New("down")), conversation("fix it"), "fix it", 0)
	assert.Equal(t, "fix it", failed, "failed call")
}
