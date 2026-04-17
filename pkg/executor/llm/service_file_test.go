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

	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

// --- ModelService.DownloadModel("file", ...) ---------------------------------

func TestModelService_DownloadModel_File_CachedFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	// Pre-create the model file in the cache.
	path := filepath.Join(dir, "cached.llamafile")
	require.NoError(t, os.WriteFile(path, []byte("fake"), 0600))

	svc := llm.NewModelService(nil)
	err := svc.DownloadModel("file", "cached.llamafile")
	require.NoError(t, err)

	// Should now be executable.
	info, statErr := os.Stat(path)
	require.NoError(t, statErr)
	assert.NotZero(t, info.Mode()&0111, "file should be executable after DownloadModel")
}

func TestModelService_DownloadModel_File_RemoteURL(t *testing.T) {
	content := []byte("fake llamafile binary")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	svc := llm.NewModelService(nil)
	url := srv.URL + "/remote.llamafile"
	err := svc.DownloadModel("file", url)
	require.NoError(t, err)

	// File should be downloaded and executable.
	dest := filepath.Join(dir, "remote.llamafile")
	info, statErr := os.Stat(dest)
	require.NoError(t, statErr)
	assert.NotZero(t, info.Mode()&0111)
}

func TestModelService_DownloadModel_File_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	svc := llm.NewModelService(nil)
	err := svc.DownloadModel("file", "missing.llamafile")
	assert.Error(t, err)
}

// --- ModelService.ServeModel("file", ...) ------------------------------------

func TestModelService_ServeModel_File_AlreadyRunning(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var port int
	fmt.Sscanf(srv.URL, "http://127.0.0.1:%d", &port)

	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	bin := filepath.Join(dir, "model.llamafile")
	require.NoError(t, os.WriteFile(bin, []byte("#!/bin/sh\nsleep 9999"), 0750))

	svc := llm.NewModelService(nil)
	err := svc.ServeModel("file", "model.llamafile", "127.0.0.1", port)
	require.NoError(t, err)
}

func TestModelService_ServeModel_File_NotFound(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", dir)

	svc := llm.NewModelService(nil)
	err := svc.ServeModel("file", "missing.llamafile", "127.0.0.1", 0)
	assert.Error(t, err)
}
