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

package cmd

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
)

func TestAgentLoopChat(t *testing.T) {
	chat := &domain.ChatConfig{Loop: &domain.ChatLoopConfig{}}
	host := &domain.Workflow{
		Metadata:  domain.WorkflowMetadata{TargetActionID: "a"},
		Resources: []*domain.Resource{{ActionID: "a", Chat: chat}},
	}
	got, err := agentLoopChat("./agents", host, "")
	require.NoError(t, err)
	assert.Same(t, chat, got, "with a path, the host workflow defines the session")

	t.Chdir(t.TempDir())
	got, err = agentLoopChat("", host, "")
	require.NoError(t, err)
	assert.Nil(t, got, "no path and no ./workflow.yaml: nothing applies")

	require.NoError(t, os.WriteFile("workflow.yaml", []byte("kind: [\n"), 0o644))
	_, err = agentLoopChat("", host, "")
	assert.Error(t, err, "a broken ./workflow.yaml is reported")
}
