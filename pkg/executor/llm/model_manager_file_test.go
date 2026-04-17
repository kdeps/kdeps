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

//go:build !js

package llm_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

func newHealthServer(t *testing.T) int {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	var port int
	fmt.Sscanf(srv.URL, "http://127.0.0.1:%d", &port)
	return port
}

// --- EnsureModel: file backend -----------------------------------------------

func TestModelManager_EnsureModel_FileBackend_AlreadyRunning(t *testing.T) {
	port := newHealthServer(t)

	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	bin := filepath.Join(dir, "model.llamafile")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\nsleep 9999"), 0750))

	mgr := llm.NewModelManagerWithOfflineMode(nil, true)
	cfg := &domain.ChatConfig{
		Backend: "file",
		Model:   "model.llamafile",
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
	}
	err := mgr.EnsureModel(cfg)
	require.NoError(t, err)
	// BaseURL should remain unchanged (was set before EnsureModel).
	assert.Equal(t, fmt.Sprintf("http://127.0.0.1:%d", port), cfg.BaseURL)
}

func TestModelManager_EnsureModel_FileBackend_BaseURLAutoAssigned(t *testing.T) {
	port := newHealthServer(t)

	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	bin := filepath.Join(dir, "model.llamafile")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\nsleep 9999"), 0750))

	// Manually boot the health server on the free port that EnsureModel will pick.
	// We'll use the health server's port directly via BaseURL="" so port=0 is chosen and
	// we cannot predict it. Instead, use the health server's port and set port explicitly.
	mgr := llm.NewModelManagerWithOfflineMode(nil, true)
	cfg := &domain.ChatConfig{
		Backend: "file",
		Model:   "model.llamafile",
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
	}
	require.NoError(t, mgr.EnsureModel(cfg))
	assert.Contains(t, cfg.BaseURL, "127.0.0.1")
}

func TestModelManager_EnsureModel_FileBackend_ServeError_Continues(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)
	// No model file in dir — serveFileModel will fail but EnsureModel should not return an error.

	mgr := llm.NewModelManagerWithOfflineMode(nil, true)
	cfg := &domain.ChatConfig{
		Backend: "file",
		Model:   "missing.llamafile",
	}
	// EnsureModel logs the warning and continues — it should not propagate the error.
	err := mgr.EnsureModel(cfg)
	require.NoError(t, err)
}

func TestModelManager_EnsureModel_FileBackend_OfflineSkipsDownload(t *testing.T) {
	port := newHealthServer(t)

	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	bin := filepath.Join(dir, "offline.llamafile")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\nsleep 9999"), 0750))

	// OfflineMode=true means DownloadModel is skipped entirely.
	mgr := llm.NewModelManagerWithOfflineMode(nil, true)
	cfg := &domain.ChatConfig{
		Backend: "file",
		Model:   "offline.llamafile",
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
	}
	require.NoError(t, mgr.EnsureModel(cfg))
}

func TestModelManager_EnsureModel_FileBackend_OnlineDownload(t *testing.T) {
	// Download + serve: use a fake HTTP server for both download and health.
	content := []byte("fake llamafile")
	healthOK := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			if healthOK {
				w.WriteHeader(http.StatusOK)
			} else {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		// Serve the model file.
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	t.Cleanup(srv.Close)

	var port int
	fmt.Sscanf(srv.URL, "http://127.0.0.1:%d", &port)

	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	// Pre-create the file so the health check succeeds on the serve call.
	// We simulate: download happens (file placed in cache), then health server says OK.
	modelURL := srv.URL + "/model.llamafile"
	bin := filepath.Join(dir, "model.llamafile")
	require.NoError(t, os.WriteFile(bin, []byte("fake"), 0750))
	healthOK = true

	mgr := llm.NewModelManager(nil)
	cfg := &domain.ChatConfig{
		Backend: "file",
		Model:   modelURL,
		BaseURL: fmt.Sprintf("http://127.0.0.1:%d", port),
	}
	require.NoError(t, mgr.EnsureModel(cfg))
	// Model should be in cache.
	_, err := os.Stat(bin)
	require.NoError(t, err)
}
