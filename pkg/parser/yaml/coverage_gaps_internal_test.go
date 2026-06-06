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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

func TestLoadResources_FilepathAbsFallback(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(_ string) (string, error) {
		return "", errors.New("abs failed")
	}

	p := newWhiteboxParser()
	wf := &domain.Workflow{}
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(workflowPath, []byte("apiVersion: kdeps.io/v1\nkind: Workflow\n"), 0o600))

	err := p.loadResources(wf, workflowPath)
	require.NoError(t, err)
}

func TestLoadComponents_FilepathAbsFallback(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(_ string) (string, error) {
		return "", errors.New("abs failed")
	}

	p := newWhiteboxParser()
	wf := &domain.Workflow{}
	err := p.LoadComponents(wf, "/tmp/workflow.yaml")
	require.NoError(t, err)
}

func TestLoadComponentResources_FilepathAbsFallback(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(_ string) (string, error) {
		return "", errors.New("abs failed")
	}

	p := newWhiteboxParser()
	_, err := p.loadComponentResources(&domain.Component{}, "/tmp/component.yaml")
	require.NoError(t, err)
}

func TestAutoDiscoverAgents_ReadDirError(t *testing.T) {
	origFS := AppFS
	t.Cleanup(func() { AppFS = origFS })
	AppFS = afero.NewMemMapFs()

	dir := t.TempDir()
	agentsDir := filepath.Join(dir, "agents")
	require.NoError(t, os.MkdirAll(agentsDir, 0o755))

	p := newWhiteboxParser()
	_, err := p.autoDiscoverAgents(dir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read agents directory")
}
