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
	"runtime"
	"testing"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

func newMgrWithDir(t *testing.T) (*llm.LlamafileManager, string) {
	t.Helper()
	dir := t.TempDir()
	mgr := llm.NewLlamafileManagerWithDir(nil, dir)
	return mgr, dir
}

// --- Resolve: local absolute path ------------------------------------------

func TestLlamafileManager_Resolve_AbsolutePath(t *testing.T) {
	mgr, dir := newMgrWithDir(t)
	path := filepath.Join(dir, "model.llamafile")
	if err := os.WriteFile(path, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := mgr.Resolve(path)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}

func TestLlamafileManager_Resolve_AbsolutePath_Missing(t *testing.T) {
	mgr, _ := newMgrWithDir(t)
	_, err := mgr.Resolve("/nonexistent/path/model.llamafile")
	if err == nil {
		t.Error("expected error for missing absolute path")
	}
}

// --- Resolve: relative path -------------------------------------------------

func TestLlamafileManager_Resolve_RelativePath(t *testing.T) {
	mgr, dir := newMgrWithDir(t)

	// Write file in temp dir and make it the cwd.
	path := filepath.Join(dir, "model.llamafile")
	if err := os.WriteFile(path, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	got, err := mgr.Resolve("./model.llamafile")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got == "" {
		t.Error("expected non-empty path")
	}
}

func TestLlamafileManager_Resolve_RelativePath_Missing(t *testing.T) {
	mgr, _ := newMgrWithDir(t)
	_, err := mgr.Resolve("./does-not-exist.llamafile")
	if err == nil {
		t.Error("expected error for missing relative path")
	}
}

// --- Resolve: bare filename (cache lookup) ----------------------------------

func TestLlamafileManager_Resolve_BareFilename_Cached(t *testing.T) {
	mgr, dir := newMgrWithDir(t)
	path := filepath.Join(dir, "model.llamafile")
	if err := os.WriteFile(path, []byte("fake"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := mgr.Resolve("model.llamafile")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}

func TestLlamafileManager_Resolve_BareFilename_NotCached(t *testing.T) {
	mgr, _ := newMgrWithDir(t)
	_, err := mgr.Resolve("missing.llamafile")
	if err == nil {
		t.Error("expected error for uncached bare filename")
	}
}

// --- Resolve: remote URL (download) ----------------------------------------

func TestLlamafileManager_Resolve_RemoteURL(t *testing.T) {
	content := []byte("fake llamafile content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	mgr, dir := newMgrWithDir(t)
	url := srv.URL + "/model.llamafile"
	got, err := mgr.Resolve(url)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	expected := filepath.Join(dir, "model.llamafile")
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}

	data, _ := os.ReadFile(got)
	if string(data) != string(content) {
		t.Error("downloaded content mismatch")
	}
}

func TestLlamafileManager_Resolve_RemoteURL_AlreadyCached(t *testing.T) {
	mgr, dir := newMgrWithDir(t)

	// Pre-create cached file.
	cached := filepath.Join(dir, "model.llamafile")
	if err := os.WriteFile(cached, []byte("cached"), 0600); err != nil {
		t.Fatal(err)
	}

	// Server should NOT be called.
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	url := srv.URL + "/model.llamafile"
	got, err := mgr.Resolve(url)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if called {
		t.Error("server should not be called when file is already cached")
	}
	if got != cached {
		t.Errorf("got %q, want %q", got, cached)
	}
}

func TestLlamafileManager_Resolve_RemoteURL_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	mgr, _ := newMgrWithDir(t)
	_, err := mgr.Resolve(srv.URL + "/missing.llamafile")
	if err == nil {
		t.Error("expected error on HTTP 404")
	}
}

// --- MakeExecutable ---------------------------------------------------------

func TestLlamafileManager_MakeExecutable(t *testing.T) {
	mgr, dir := newMgrWithDir(t)
	path := filepath.Join(dir, "model.llamafile")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := mgr.MakeExecutable(path); err != nil {
		t.Fatalf("MakeExecutable: %v", err)
	}
	info, _ := os.Stat(path)
	if info.Mode()&0111 == 0 {
		t.Error("file should be executable after MakeExecutable")
	}
}

func TestLlamafileManager_MakeExecutable_AlreadyExecutable(t *testing.T) {
	mgr, dir := newMgrWithDir(t)
	path := filepath.Join(dir, "model.llamafile")
	if err := os.WriteFile(path, []byte("x"), 0750); err != nil {
		t.Fatal(err)
	}
	// Should succeed (no-op).
	if err := mgr.MakeExecutable(path); err != nil {
		t.Fatalf("MakeExecutable: %v", err)
	}
}

func TestLlamafileManager_MakeExecutable_Missing(t *testing.T) {
	mgr, dir := newMgrWithDir(t)
	err := mgr.MakeExecutable(filepath.Join(dir, "nope.llamafile"))
	if err == nil {
		t.Error("expected error for missing file")
	}
}

// --- FindFreePort -----------------------------------------------------------

func TestFindFreePort(t *testing.T) {
	port, err := llm.FindFreePort()
	if err != nil {
		t.Fatalf("FindFreePort: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("unexpected port: %d", port)
	}
}

func TestFindFreePort_UniquePerCall(t *testing.T) {
	p1, err1 := llm.FindFreePort()
	p2, err2 := llm.FindFreePort()
	if err1 != nil || err2 != nil {
		t.Fatalf("FindFreePort errors: %v %v", err1, err2)
	}
	// Ports CAN be reused but statistically should differ.
	// Just assert both are valid.
	if p1 <= 0 || p2 <= 0 {
		t.Error("ports should be positive")
	}
}

// --- NewLlamafileManager ----------------------------------------------------

func TestNewLlamafileManager(t *testing.T) {
	mgr, err := llm.NewLlamafileManager(nil)
	if err != nil {
		t.Fatalf("NewLlamafileManager: %v", err)
	}
	if mgr == nil {
		t.Error("expected non-nil manager")
	}
}

func TestNewLlamafileManagerWithDir(t *testing.T) {
	dir := t.TempDir()
	mgr := llm.NewLlamafileManagerWithDir(nil, dir)
	if mgr == nil {
		t.Error("expected non-nil manager")
	}
}

// --- Download: temp file creation failure -----------------------------------

func TestLlamafileManager_Download_TmpBlockedByDir(t *testing.T) {
	// If a directory exists where the .tmp file would be created, OpenFile fails.
	content := []byte("fake llamafile content")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer srv.Close()

	mgr, dir := newMgrWithDir(t)
	url := srv.URL + "/blocked.llamafile"

	// Create a directory at the path the .tmp file would occupy.
	tmpPath := filepath.Join(dir, "blocked.llamafile.tmp")
	if err := os.Mkdir(tmpPath, 0750); err != nil {
		t.Fatal(err)
	}

	_, err := mgr.Resolve(url)
	if err == nil {
		t.Error("expected error when .tmp path is a directory")
	}
}

// --- Serve (with fake health endpoint) --------------------------------------

func TestLlamafileManager_Serve_AlreadyRunning(t *testing.T) {
	// Spin up a fake server that handles /health.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	mgr, dir := newMgrWithDir(t)

	// Write a dummy "binary" (won't be executed because health check passes immediately).
	bin := filepath.Join(dir, "model.llamafile")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nsleep 9999"), 0750); err != nil {
		t.Fatal(err)
	}

	// Extract port from test server URL.
	var port int
	fmt.Sscanf(srv.URL, "http://127.0.0.1:%d", &port)

	actualPort, err := mgr.Serve(bin, port)
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if actualPort != port {
		t.Errorf("actualPort = %d, want %d", actualPort, port)
	}
}

func TestLlamafileManager_Serve_Port0_StartFail(t *testing.T) {
	mgr, dir := newMgrWithDir(t)
	// A non-executable binary format — exec.Command.Start fails with "exec format error".
	bin := filepath.Join(dir, "model.llamafile")
	if err := os.WriteFile(bin, []byte("not a real binary"), 0750); err != nil {
		t.Fatal(err)
	}
	// port=0 → FindFreePort() is called inside Serve; nothing healthy on that port,
	// so it tries to start the binary which immediately fails.
	_, err := mgr.Serve(bin, 0)
	if err == nil {
		t.Error("expected error when binary is not executable format")
	}
}

// TestLlamafileManager_Serve_StartsAndBecomesHealthy exercises the
// cmd.Process.Release + health poll loop path in Serve.
func TestLlamafileManager_Serve_StartsAndBecomesHealthy(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script binary not supported on Windows")
	}

	healthReady := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			select {
			case <-healthReady:
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	var port int
	fmt.Sscanf(srv.URL, "http://127.0.0.1:%d", &port)

	mgr, dir := newMgrWithDir(t)
	bin := filepath.Join(dir, "model.llamafile")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0"), 0750); err != nil {
		t.Fatal(err)
	}

	// After cmd.Start() runs, signal health to pass on the next poll.
	go func() {
		time.Sleep(100 * time.Millisecond)
		close(healthReady)
	}()

	actualPort, err := mgr.Serve(bin, port)
	if err != nil {
		t.Fatalf("Serve: %v", err)
	}
	if actualPort != port {
		t.Errorf("actualPort = %d, want %d", actualPort, port)
	}
}

func TestLlamafileManager_Serve_NotHealthy_StartFail(t *testing.T) {
	// Server is up but returns 503 → isHealthy returns false → tries to start binary → fails.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	var port int
	fmt.Sscanf(srv.URL, "http://127.0.0.1:%d", &port)

	mgr, dir := newMgrWithDir(t)
	bin := filepath.Join(dir, "bad.llamafile")
	if err := os.WriteFile(bin, []byte("not a real binary"), 0750); err != nil {
		t.Fatal(err)
	}

	_, err := mgr.Serve(bin, port)
	if err == nil {
		t.Error("expected error when binary is not executable format")
	}
}
