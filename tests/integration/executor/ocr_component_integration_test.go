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

package executor_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
	"github.com/kdeps/kdeps/v2/pkg/parser/yaml"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

// The ocr component comes from the components asset set: the parser loads it
// for a workflow that calls it, and the engine runs its exec resource.
func TestOCRComponent_ParsesAndExtractsText(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("KDEPS_COMPONENT_DIR", t.TempDir())
	_, thisFile, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(thisFile), "..", "..", "e2e", "fixtures", "ocr-sample.png")

	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "resources"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(`apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: ocr-component
  version: "1.0.0"
  targetActionId: extract
settings:
  agentSettings: {}
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "resources", "extract.yaml"), []byte(`actionId: extract
name: Extract
component:
  name: ocr
  with:
    file: "`+filepath.ToSlash(fixture)+`"
`), 0o600))

	sv, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	wf, err := yaml.NewParser(sv, expression.NewParser()).ParseWorkflow(filepath.Join(dir, "workflow.yaml"))
	require.NoError(t, err)
	comp := wf.Components["ocr"]
	require.NotNil(t, comp)
	require.NotNil(t, comp.Interface)
	assert.Contains(t, wf.Settings.AgentSettings.OSPackages, "tesseract-ocr")

	if _, lookErr := exec.LookPath("tesseract"); lookErr != nil {
		t.Skip("tesseract not found in PATH")
	}
	result, err := newExecEngine().Execute(wf, nil)
	require.NoError(t, err)
	out, ok := result.(map[string]interface{})
	require.True(t, ok, "%T", result)
	assert.Contains(t, out["stdout"], "KDeps OCR Sample")
}
