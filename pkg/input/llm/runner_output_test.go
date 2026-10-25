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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestLlmConfig_NonNull(t *testing.T) {
	wf := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			LLM: &domain.LLMInputConfig{
				Prompt:    "custom> ",
				SessionID: "custom-session",
			},
		},
	}
	cfg := llmConfig(wf)
	assert.Equal(t, "custom> ", cfg.Prompt)
	assert.Equal(t, "custom-session", cfg.SessionID)
}

// A chat resource's reply prints the model's words, not a Go map dump; other
// structured results print as JSON.
func TestFormatResult_ChatReplyAndStructured(t *testing.T) {
	reply := map[string]interface{}{
		"done": true, "model": "m",
		"message": map[string]interface{}{"role": "assistant", "content": "The answer is 93."},
	}
	assert.Equal(t, "The answer is 93.", formatResult(reply))
	assert.Equal(t, "{\n  \"a\": 1\n}", formatResult(map[string]interface{}{"a": 1}))
	assert.Equal(t, "[\n  1,\n  2\n]", formatResult([]int{1, 2}))
	assert.Equal(t, "plain", formatResult("plain"))
}
