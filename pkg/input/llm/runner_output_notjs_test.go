//go:build !js

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
