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

package executor_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

type inlineBotReplyMock struct{}

func (inlineBotReplyMock) Execute(_ *executor.ExecutionContext, _ interface{}) (interface{}, error) {
	return "bot reply sent", nil
}

func TestInlineBotReply_Integration(t *testing.T) {
	engine := executor.NewEngine(nil)
	reg := executor.NewRegistry()
	reg.SetBotReplyExecutor(inlineBotReplyMock{})
	engine.SetRegistry(reg)

	workflow := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata: domain.WorkflowMetadata{
			Name:           "inline-botreply",
			Version:        "1.0.0",
			TargetActionID: "main",
		},
		Resources: []*domain.Resource{
			{
				ActionID: "main",
				Name:     "Main",
				Before: []domain.InlineResource{
					{BotReply: &domain.BotReplyConfig{Text: "preamble"}},
				},
				After: []domain.ActionConfig{{Expr: "1+1"}},
			},
		},
	}

	_, err := engine.Execute(workflow, nil)
	require.NoError(t, err)
}

func TestInlineAPIServer_Integration(t *testing.T) {
	engine := executor.NewEngine(nil)

	workflow := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata: domain.WorkflowMetadata{
			Name:           "inline-apiserver",
			Version:        "1.0.0",
			TargetActionID: "main",
		},
		Resources: []*domain.Resource{
			{
				ActionID: "main",
				Name:     "Main",
				After: []domain.InlineResource{
					{
						APIServer: &domain.APIResponseConfig{
							Success:  true,
							Response: map[string]interface{}{"ok": true},
						},
					},
				},
			},
		},
	}

	_, err := engine.Execute(workflow, nil)
	require.NoError(t, err)
}

func TestInlineAPIResponse_Integration(t *testing.T) {
	engine := executor.NewEngine(nil)

	workflow := &domain.Workflow{
		APIVersion: "kdeps.io/v1",
		Kind:       "Workflow",
		Metadata: domain.WorkflowMetadata{
			Name:           "inline-apiresponse",
			Version:        "1.0.0",
			TargetActionID: "main",
		},
		Resources: []*domain.Resource{
			{
				ActionID: "main",
				Name:     "Main",
				After: []domain.InlineResource{
					{
						APIResponse: &domain.APIResponseConfig{
							Success:  true,
							Response: map[string]interface{}{"ok": true},
						},
					},
				},
			},
		},
	}

	_, err := engine.Execute(workflow, nil)
	require.NoError(t, err)
}
