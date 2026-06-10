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

package yaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestLoadResources_InitializesNilResources(t *testing.T) {
	tmpDir := t.TempDir()
	resourcesDir := filepath.Join(tmpDir, "resources")
	require.NoError(t, os.MkdirAll(resourcesDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(resourcesDir, "res.yaml"),
		[]byte("actionId: test-res\n"), 0o600,
	))

	workflowPath := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(workflowPath, []byte(`apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test
  version: 1.0.0
  targetActionId: main
settings:
  agentSettings:
    timezone: UTC
`), 0o600))

	p := newWhiteboxParser()
	wf := &domain.Workflow{}
	err := p.loadResources(wf, workflowPath)
	require.NoError(t, err)
	require.NotNil(t, wf.Resources)
}
