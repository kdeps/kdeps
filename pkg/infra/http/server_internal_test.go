// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"mime/multipart"
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
	"github.com/kdeps/kdeps/v2/pkg/parser/yaml"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

func TestServer_SetupHotReload_AbsPathWarning(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(path string) (string, error) {
		if strings.HasSuffix(path, "workflow.yaml") {
			return "", errors.New("abs failed")
		}
		return path, nil
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{Routes: []domain.Route{}},
		},
	}
	server, err := NewServer(workflow, nil, slog.Default())
	require.NoError(t, err)
	server.SetWorkflowPath("workflow.yaml")
	server.SetWatcher(&callbackFileWatcher{})

	err = server.SetupHotReload()
	require.NoError(t, err)
}

func TestServer_SetupHotReload_ParserFactoryError(t *testing.T) {
	orig := workflowParserFactory
	t.Cleanup(func() { workflowParserFactory = orig })
	workflowParserFactory = func() (*yaml.Parser, error) {
		return nil, errors.New("parser factory failed")
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
	}
	server, err := NewServer(workflow, nil, slog.Default())
	require.NoError(t, err)
	server.SetWatcher(&callbackFileWatcher{})

	err = server.SetupHotReload()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parser factory failed")
}

func TestServer_ReloadWorkflow_EnsureReloadReadyError(t *testing.T) {
	orig := workflowParserFactory
	t.Cleanup(func() { workflowParserFactory = orig })
	workflowParserFactory = func() (*yaml.Parser, error) {
		return nil, errors.New("parser init failed")
	}

	server, err := NewServer(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}}, nil, slog.Default())
	require.NoError(t, err)
	server.parser = nil

	err = server.reloadWorkflow()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "parser init failed")
}

func TestServer_EnsureReloadReady_AbsError(t *testing.T) {
	orig := filepathAbs
	t.Cleanup(func() { filepathAbs = orig })
	filepathAbs = func(string) (string, error) {
		return "", errors.New("abs failed")
	}

	server, err := NewServer(&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}}, nil, slog.Default())
	require.NoError(t, err)
	server.workflowPath = ""

	err = server.ensureReloadReady()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve workflow path")
}

// TestServer_SetCorsMethods_EmptyDefaults verifies that setCorsMethods
// uses the default methods string when cors.AllowMethods is empty.
func TestServer_SetCorsMethods_EmptyDefaults(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	cors := &domain.CORS{
		AllowOrigins: []string{"*"},
		// AllowMethods intentionally empty to exercise the else branch
	}
	s.setCorsMethods(w, cors)

	assert.Equal(
		t,
		"GET, POST, PUT, DELETE, PATCH, OPTIONS",
		w.Header().Get("Access-Control-Allow-Methods"),
	)
}

// TestServer_SetCorsHeaders_EmptyDefaults verifies that setCorsHeaders
// uses the default headers string when cors.AllowHeaders is empty.
func TestServer_SetCorsHeaders_EmptyDefaults(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	cors := &domain.CORS{
		AllowOrigins: []string{"*"},
		// AllowHeaders intentionally empty to exercise the else branch
	}
	s.setCorsHeaders(w, cors)

	assert.Equal(
		t,
		"Content-Type, Authorization",
		w.Header().Get("Access-Control-Allow-Headers"),
	)
}

// TestServer_SetCorsMethods_WithValues verifies that setCorsMethods
// uses the configured methods when cors.AllowMethods is non-empty.
func TestServer_SetCorsMethods_WithValues(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	cors := &domain.CORS{
		AllowMethods: []string{"GET", "POST"},
	}
	s.setCorsMethods(w, cors)

	assert.Equal(t, "GET, POST", w.Header().Get("Access-Control-Allow-Methods"))
}

// TestServer_SetCorsHeaders_WithValues verifies that setCorsHeaders
// uses the configured headers when cors.AllowHeaders is non-empty.
func TestServer_SetCorsHeaders_WithValues(t *testing.T) {
	s := &Server{}
	w := httptest.NewRecorder()
	cors := &domain.CORS{
		AllowHeaders: []string{"X-Custom"},
	}
	s.setCorsHeaders(w, cors)

	assert.Equal(t, "X-Custom", w.Header().Get("Access-Control-Allow-Headers"))

	// Also test multiple headers
	w2 := httptest.NewRecorder()
	cors2 := &domain.CORS{
		AllowHeaders: []string{"X-One", "X-Two"},
	}
	s.setCorsHeaders(w2, cors2)
	assert.Equal(t, "X-One, X-Two", w2.Header().Get("Access-Control-Allow-Headers"))
}

// TestServer_HandleRequest_FileCleanupDeleteError exercises the
// s.fileStore.Delete error branch at lines 327-330 of server.go by
// replacing the server's file store with one whose Delete method fails.
func TestServer_HandleRequest_FileCleanupDeleteError(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{
				Routes: []domain.Route{
					{Path: "/api/test", Methods: []string{"POST"}},
				},
			},
		},
	}

	server, err := NewServer(workflow, &whiteboxMockExecutor{}, slog.Default())
	require.NoError(t, err)

	// Replace the file store with one whose Delete returns an error.
	// The upload handler still holds the original store, so storage
	// during HandleUpload succeeds, but the defer cleanup hits our mock.
	server.fileStore = &errFileStore{FileStore: server.fileStore}
	t.Cleanup(func() {
		_ = server.fileStore.Close()
	})

	// Build a multipart request with one file.
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "test.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("file content"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodPost, "/api/test", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	server.HandleRequest(w, req)

	// The response should still be 200 OK — the delete error is logged
	// but does not affect the HTTP response.
	assert.Equal(t, stdhttp.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp["success"].(bool))
}

// TestServer_HandleRequest_FileCleanupNoFiles exercises the defer cleanup loop
// when uploadedFiles is nil (no multipart upload), covering the loop body
// guard. The defer runs with an empty uploadedFiles slice so the body is
// never entered.
func TestServer_HandleRequest_FileCleanupNoFiles(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{
				Routes: []domain.Route{
					{Path: "/api/test", Methods: []string{"POST"}},
				},
			},
		},
	}

	server, err := NewServer(workflow, &whiteboxMockExecutor{}, slog.Default())
	require.NoError(t, err)

	// Replace file store with one that fails — even so, when no multipart
	// upload is present, uploadedFiles stays nil and the defer loop has
	// nothing to iterate over.
	server.fileStore = &errFileStore{FileStore: server.fileStore}
	t.Cleanup(func() {
		_ = server.fileStore.Close()
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(stdhttp.MethodPost, "/api/test", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")

	server.HandleRequest(w, req)

	assert.Equal(t, stdhttp.StatusOK, w.Code)
}

// TestServer_SetupHotReload_ResourcesCallbackReloadError exercises the
// resources callback error path at line 818-820 of server.go by making
// the workflow file invalid before triggering the resources callback.
func TestServer_SetupHotReload_ResourcesCallbackReloadError(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{
				Routes: []domain.Route{},
			},
		},
	}

	server, err := NewServer(workflow, nil, slog.Default())
	require.NoError(t, err)

	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")
	server.SetWorkflowPath(workflowPath)

	// Create a valid workflow file initially so SetupHotReload succeeds.
	validContent := `
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test
  version: 1.0.0
  targetActionId: main
settings:
  apiServer:
    routes: []
  agentSettings:
    timezone: UTC
`
	err = os.WriteFile(workflowPath, []byte(validContent), 0644)
	require.NoError(t, err)

	// Set parser
	schemaValidator, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	exprParser := expression.NewParser()
	parser := yaml.NewParser(schemaValidator, exprParser)
	server.SetParser(parser)

	// Use callback-capturing watcher
	watcher := &callbackFileWatcher{}
	server.SetWatcher(watcher)

	err = server.SetupHotReload()
	require.NoError(t, err)
	require.Len(t, watcher.callbacks, 2) // workflow file + resources dir

	// Make the workflow file invalid so reload fails
	invalidContent := `invalid: yaml: [`
	err = os.WriteFile(workflowPath, []byte(invalidContent), 0644)
	require.NoError(t, err)

	// Trigger resources directory callback (second callback) —
	// this should hit the reloadErr != nil branch.
	watcher.callbacks[1]()

	// Workflow should remain unchanged (reload failed, error was logged)
	assert.Equal(t, "test", server.Workflow.Metadata.Name)
}
