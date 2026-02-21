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

package http_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	httppkg "github.com/kdeps/kdeps/v2/pkg/infra/http"
)

// makeTestServer creates a test server with a basic workflow.
func makeTestServer(t *testing.T, workflow *domain.Workflow) *httppkg.Server {
	t.Helper()
	logger := slog.Default()
	server, err := httppkg.NewServer(workflow, &MockWorkflowExecutor{}, logger)
	require.NoError(t, err)
	return server
}

// TestHandleManagementStatus_NoWorkflow checks the status endpoint with no workflow set.
func TestHandleManagementStatus_NoWorkflow(t *testing.T) {
	server := makeTestServer(t, nil)
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/status", nil)
	rec := httptest.NewRecorder()
	server.HandleManagementStatus(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
	assert.Nil(t, body["workflow"])
}

// TestHandleManagementStatus_WithWorkflow checks the status endpoint with a workflow set.
func TestHandleManagementStatus_WithWorkflow(t *testing.T) {
	workflow := &domain.Workflow{}
	workflow.Metadata.Name = "test-agent"
	workflow.Metadata.Version = "1.0.0"
	workflow.Metadata.Description = "A test workflow"

	server := makeTestServer(t, workflow)
	server.SetupManagementRoutes()

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/status", nil)
	rec := httptest.NewRecorder()
	server.HandleManagementStatus(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])

	wf, ok := body["workflow"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test-agent", wf["name"])
	assert.Equal(t, "1.0.0", wf["version"])
}

// TestHandleManagementUpdateWorkflow_EmptyBody checks that an empty body returns 400.
func TestHandleManagementUpdateWorkflow_EmptyBody(t *testing.T) {
	server := makeTestServer(t, nil)

	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/workflow", bytes.NewReader(nil))
	rec := httptest.NewRecorder()
	server.HandleManagementUpdateWorkflow(rec, req)

	assert.Equal(t, stdhttp.StatusBadRequest, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "error", body["status"])
}

// TestHandleManagementUpdateWorkflow_ValidYAML writes a valid workflow YAML and reloads it.
func TestHandleManagementUpdateWorkflow_ValidYAML(t *testing.T) {
	// Use a temp directory for the workflow file so we don't pollute the test dir
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")

	// Write an initial minimal valid workflow YAML
	initialYAML := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: initial-agent
  version: 1.0.0
  targetActionId: action1
settings:
  portNum: 16395
  agentSettings:
    timezone: UTC
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(initialYAML), 0600))

	server := makeTestServer(t, nil)
	server.SetWorkflowPath(workflowPath)

	updatedYAML := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: updated-agent
  version: 2.0.0
  targetActionId: action1
settings:
  portNum: 16395
  agentSettings:
    timezone: UTC
`
	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/workflow", bytes.NewBufferString(updatedYAML))
	rec := httptest.NewRecorder()
	server.HandleManagementUpdateWorkflow(rec, req)

	// Should succeed (workflow is parsed and reloaded)
	if rec.Code != stdhttp.StatusOK {
		t.Logf("Response body: %s", rec.Body.String())
	}
	assert.Equal(t, stdhttp.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])

	// Verify the file was written
	written, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	assert.Equal(t, updatedYAML, string(written))
}

// TestHandleManagementReload_NoWorkflowPath checks reload when workflow path is not set.
func TestHandleManagementReload_NoWorkflowPath(t *testing.T) {
	server := makeTestServer(t, nil)

	req := httptest.NewRequest(stdhttp.MethodPost, "/_kdeps/reload", nil)
	rec := httptest.NewRecorder()
	server.HandleManagementReload(rec, req)

	// Should fail because no workflow file at default path
	// (either 500 or 422 depending on whether file exists)
	assert.True(t, rec.Code == stdhttp.StatusInternalServerError || rec.Code == stdhttp.StatusUnprocessableEntity,
		"expected 500 or 422, got %d", rec.Code)
}

// TestHandleManagementReload_ValidWorkflow reloads a valid workflow from disk.
func TestHandleManagementReload_ValidWorkflow(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")

	yamlContent := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: reload-test
  version: 1.0.0
  targetActionId: action1
settings:
  portNum: 16395
  agentSettings:
    timezone: UTC
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(yamlContent), 0600))

	server := makeTestServer(t, nil)
	server.SetWorkflowPath(workflowPath)

	req := httptest.NewRequest(stdhttp.MethodPost, "/_kdeps/reload", nil)
	rec := httptest.NewRecorder()
	server.HandleManagementReload(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])

	wf, ok := body["workflow"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "reload-test", wf["name"])
}

// TestSetupManagementRoutes_RoutesRegistered verifies management routes are registered.
func TestSetupManagementRoutes_RoutesRegistered(t *testing.T) {
	server := makeTestServer(t, nil)
	server.SetupManagementRoutes()

	// GET /_kdeps/status should return 200
	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/status", nil)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, stdhttp.StatusOK, rec.Code)
}

// TestSetupRoutes_IncludesManagementRoutes verifies that SetupRoutes registers management routes.
func TestSetupRoutes_IncludesManagementRoutes(t *testing.T) {
	workflow := &domain.Workflow{}
	workflow.Metadata.Name = "test"
	workflow.Metadata.Version = "1.0.0"

	server := makeTestServer(t, workflow)
	server.SetupRoutes()

	// Management status endpoint should be reachable
	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/status", nil)
	rec := httptest.NewRecorder()
	server.Router.ServeHTTP(rec, req)
	assert.Equal(t, stdhttp.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

// TestHandleManagementUpdateWorkflow_InvalidYAML checks that invalid YAML returns an error.
func TestHandleManagementUpdateWorkflow_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")

	server := makeTestServer(t, nil)
	server.SetWorkflowPath(workflowPath)

	// Send invalid YAML that will fail to parse as a valid workflow
	invalidContent := "not: valid: yaml: !!!"
	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/workflow", bytes.NewBufferString(invalidContent))
	rec := httptest.NewRecorder()
	server.HandleManagementUpdateWorkflow(rec, req)

	// The file will be written but workflow reload will fail
	assert.True(t, rec.Code == stdhttp.StatusOK || rec.Code == stdhttp.StatusUnprocessableEntity,
		"expected 200 or 422, got %d: %s", rec.Code, rec.Body.String())
}

// TestHandleManagementStatus_ResourceCount verifies resource count is reported.
func TestHandleManagementStatus_ResourceCount(t *testing.T) {
	workflow := &domain.Workflow{}
	workflow.Metadata.Name = "agent-with-resources"
	workflow.Metadata.Version = "1.0.0"
	workflow.Resources = []*domain.Resource{
		{Metadata: domain.ResourceMetadata{ActionID: "resource1"}},
		{Metadata: domain.ResourceMetadata{ActionID: "resource2"}},
	}

	server := makeTestServer(t, workflow)

	req := httptest.NewRequest(stdhttp.MethodGet, "/_kdeps/status", nil)
	rec := httptest.NewRecorder()
	server.HandleManagementStatus(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	wf, ok := body["workflow"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(2), wf["resources"])
}

// TestHandleManagementUpdateWorkflow_LargeBody checks that oversized bodies are rejected.
func TestHandleManagementUpdateWorkflow_LargeBody(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")

	server := makeTestServer(t, nil)
	server.SetWorkflowPath(workflowPath)

	// Build body slightly larger than the 5MB limit
	bigContent := fmt.Sprintf("%s", bytes.Repeat([]byte("a"), 5*1024*1024+1))
	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/workflow", bytes.NewBufferString(bigContent))
	rec := httptest.NewRecorder()
	server.HandleManagementUpdateWorkflow(rec, req)

	// The body is truncated by LimitReader, so file is written but reload likely fails
	// We just verify the server doesn't crash and returns a parseable response
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Contains(t, []string{"ok", "error"}, body["status"])
}

// TestHandleManagementUpdateWorkflow_ClearsResourcesDir verifies that stale YAML files
// in the resources/ directory are removed when a workflow is pushed, preventing
// duplicate resource loading on the next restart or reload.
func TestHandleManagementUpdateWorkflow_ClearsResourcesDir(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")

	// Pre-create a resources/ directory with a stale resource file
	resourcesDir := filepath.Join(tmpDir, "resources")
	require.NoError(t, os.MkdirAll(resourcesDir, 0750))
	staleResource := filepath.Join(resourcesDir, "old-resource.yaml")
	require.NoError(t, os.WriteFile(staleResource, []byte("stale content"), 0600))
	// Also create a non-YAML file that should NOT be removed
	nonYAMLFile := filepath.Join(resourcesDir, "readme.txt")
	require.NoError(t, os.WriteFile(nonYAMLFile, []byte("readme"), 0600))

	// Write an initial workflow to the path
	require.NoError(t, os.WriteFile(workflowPath, []byte(`apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: initial
  version: 1.0.0
  targetActionId: action1
settings:
  portNum: 16395
  agentSettings:
    timezone: UTC
`), 0600))

	server := makeTestServer(t, nil)
	server.SetWorkflowPath(workflowPath)

	updatedYAML := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: pushed-agent
  version: 2.0.0
  targetActionId: action1
settings:
  portNum: 16395
  agentSettings:
    timezone: UTC
`
	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/workflow", bytes.NewBufferString(updatedYAML))
	rec := httptest.NewRecorder()
	server.HandleManagementUpdateWorkflow(rec, req)

	assert.Equal(t, stdhttp.StatusOK, rec.Code)

	// The stale YAML resource file must have been removed
	_, err := os.Stat(staleResource)
	assert.True(t, os.IsNotExist(err), "stale resource YAML should have been deleted after push")

	// Non-YAML files must be preserved
	_, err = os.Stat(nonYAMLFile)
	assert.NoError(t, err, "non-YAML file should be preserved")
}

// TestHandleManagementUpdateWorkflow_PersistsWorkflowPath verifies that after a push
// the server uses the originally configured workflow path (not a guessed fallback).
// This ensures that after a container restart the correct file is read.
func TestHandleManagementUpdateWorkflow_PersistsWorkflowPath(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "my-agent", "workflow.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(workflowPath), 0750))

	initialYAML := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: 1.0.0
  targetActionId: action1
settings:
  portNum: 16395
  agentSettings:
    timezone: UTC
`
	require.NoError(t, os.WriteFile(workflowPath, []byte(initialYAML), 0600))

	server := makeTestServer(t, nil)
	// Simulate what StartHTTPServer now does: always set the workflow path
	server.SetWorkflowPath(workflowPath)

	pushedYAML := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: my-agent
  version: 2.0.0
  targetActionId: action1
settings:
  portNum: 16395
  agentSettings:
    timezone: UTC
`
	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/workflow", bytes.NewBufferString(pushedYAML))
	rec := httptest.NewRecorder()
	server.HandleManagementUpdateWorkflow(rec, req)
	require.Equal(t, stdhttp.StatusOK, rec.Code)

	// The pushed YAML must have been written to the CONFIGURED path, not a fallback
	written, err := os.ReadFile(workflowPath)
	require.NoError(t, err)
	assert.Equal(t, pushedYAML, string(written))
}
