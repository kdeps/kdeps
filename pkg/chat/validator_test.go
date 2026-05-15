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
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validWorkflow() *GeneratedWorkflow {
	return &GeneratedWorkflow{
		Files: map[string]string{
			"workflow.yaml": `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test-agent
  version: 1.0.0
  targetActionId: main
`,
			"resources/main.yaml": `apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: main
apiResponse:
  data: "hello"
`,
		},
	}
}

func TestValidate_Valid(t *testing.T) {
	errs := Validate(validWorkflow())
	assert.Empty(t, errs)
}

func TestValidate_MissingWorkflowYAML(t *testing.T) {
	wf := &GeneratedWorkflow{Files: map[string]string{}}
	errs := Validate(wf)
	assert.Len(t, errs, 1)
	assert.Contains(t, errs[0], "missing workflow.yaml")
}

func TestValidate_InvalidWorkflowYAML(t *testing.T) {
	wf := &GeneratedWorkflow{Files: map[string]string{
		"workflow.yaml": ":\tbad: yaml: [",
	}}
	errs := Validate(wf)
	assert.NotEmpty(t, errs)
	assert.Contains(t, errs[0], "invalid YAML")
}

func TestValidate_MissingAPIVersion(t *testing.T) {
	wf := validWorkflow()
	wf.Files["workflow.yaml"] = `kind: Workflow
metadata:
  name: test
  targetActionId: main
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "apiVersion"))
}

func TestValidate_MissingKind(t *testing.T) {
	wf := validWorkflow()
	wf.Files["workflow.yaml"] = `apiVersion: kdeps.io/v1
metadata:
  name: test
  targetActionId: main
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "kind"))
}

func TestValidate_MissingMetadataName(t *testing.T) {
	wf := validWorkflow()
	wf.Files["workflow.yaml"] = `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  targetActionId: main
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "metadata.name"))
}

func TestValidate_MissingTargetActionId(t *testing.T) {
	wf := validWorkflow()
	wf.Files["workflow.yaml"] = `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "targetActionId"))
}

func TestValidate_TargetActionIdMismatch(t *testing.T) {
	wf := validWorkflow()
	wf.Files["workflow.yaml"] = `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test
  targetActionId: nonexistent
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "nonexistent"))
}

func TestValidate_ResourceInvalidYAML(t *testing.T) {
	wf := validWorkflow()
	wf.Files["resources/bad.yaml"] = ":\tbad yaml ["
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "invalid YAML"))
}

func TestValidate_ResourceMissingAPIVersion(t *testing.T) {
	wf := validWorkflow()
	wf.Files["resources/main.yaml"] = `kind: Resource
metadata:
  actionId: main
apiResponse:
  data: "ok"
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "apiVersion"))
}

func TestValidate_ResourceMissingKind(t *testing.T) {
	wf := validWorkflow()
	wf.Files["resources/main.yaml"] = `apiVersion: kdeps.io/v1
metadata:
  actionId: main
apiResponse:
  data: "ok"
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "kind"))
}

func TestValidate_ResourceMissingActionId(t *testing.T) {
	wf := validWorkflow()
	wf.Files["resources/main.yaml"] = `apiVersion: kdeps.io/v1
kind: Resource
metadata:
  name: something
chat: {}
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "actionId"))
}

func TestValidate_ResourceMissingRunSection(t *testing.T) {
	wf := validWorkflow()
	wf.Files["resources/main.yaml"] = `apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: main
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "no recognized action"))
}

func TestValidate_ResourceUnrecognizedAction(t *testing.T) {
	wf := validWorkflow()
	wf.Files["resources/main.yaml"] = `apiVersion: kdeps.io/v1
kind: Resource
metadata:
  actionId: main
http:
  url: "https://example.com"
`
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "no recognized action"))
}

func TestValidate_NoResourceFiles(t *testing.T) {
	wf := &GeneratedWorkflow{
		Files: map[string]string{
			"workflow.yaml": `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test
  targetActionId: main
`,
		},
	}
	errs := Validate(wf)
	assert.True(t, containsErr(errs, "no resource files"))
}

func TestValidate_AllValidRunActions(t *testing.T) {
	actions := []string{
		"chat", "httpClient", "exec", "python", "sql",
		"apiResponse", "component", "agent", "scraper",
		"embedding", "searchLocal", "searchWeb", "telephony",
	}
	for _, action := range actions {
		wf := validWorkflow()
		wf.Files["resources/main.yaml"] = "apiVersion: kdeps.io/v1\nkind: Resource\nmetadata:\n  actionId: main\n" + action + ": {}\n"
		errs := Validate(wf)
		assert.Empty(t, errs, "action %q should be valid", action)
	}
}

func containsErr(errs []string, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}
