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

package chat

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor_Run_NoWorkflow(t *testing.T) {
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	session := &Session{Dir: t.TempDir()}
	err := exec.Run(context.Background(), session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow")
}

func TestExecutor_ExportK8s_NoWorkflow(t *testing.T) {
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	session := &Session{Dir: t.TempDir()}
	err := exec.ExportK8s(context.Background(), session)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workflow")
}

func TestExecutor_Run_BinaryNotFound(t *testing.T) {
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	exec.KDepsBin = "/nonexistent/kdeps-binary"

	session := &Session{
		Dir: t.TempDir(),
		Workflow: &GeneratedWorkflow{
			Files: map[string]string{"workflow.yaml": "apiVersion: kdeps.io/v1\n"},
		},
	}

	err := exec.Run(context.Background(), session)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed")
}

func TestNewExecutor_DefaultBin(t *testing.T) {
	out := &strings.Builder{}
	exec := NewExecutor(out, out)
	// KDepsBin should be set to the current executable or "kdeps"
	assert.NotEmpty(t, exec.KDepsBin)
}
