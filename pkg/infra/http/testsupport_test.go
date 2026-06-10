// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package http_test

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"log/slog"
	stdhttp "net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	httppkg "github.com/kdeps/kdeps/v2/pkg/infra/http"
)

// chatbotTestWorkflow creates a test workflow with a POST /api/v1/chat route and
// a resource that validates a "message" field.
func chatbotTestWorkflow() *domain.Workflow {
	minLen := 1
	return &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:        "chatbot",
			Description: "A simple chatbot agent",
			Version:     "1.0.0",
		},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{
				Routes: []domain.Route{
					{Path: "/api/v1/chat", Methods: []string{"POST"}},
				},
			},
		},
		Resources: []*domain.Resource{
			{

				ActionID:    "llmResource",
				Name:        "LLM Chat Handler",
				Description: "Handles chat requests",

				Validations: &domain.ValidationsConfig{
					Methods:  []string{"POST"},
					Routes:   []string{"/api/v1/chat"},
					Required: []string{"message"},
					Rules: []domain.FieldRule{
						{
							Field:     "message",
							Type:      domain.FieldTypeString,
							MinLength: &minLen,
							Message:   "Message cannot be empty",
						},
					},
				},
			},
		},
	}
}

// buildKdepsArchive creates an in-memory .kdeps tar.gz archive containing the
// provided files map (path → content). Used by package endpoint tests.
func buildKdepsArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	return buf.Bytes()
}

// buildKdepsArchiveWithDirs creates an in-memory .kdeps tar.gz archive containing
// explicit directory entries plus file entries. Used to test directory-extraction paths.
func buildKdepsArchiveWithDirs(t *testing.T, dirs []string, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	for _, d := range dirs {
		hdr := &tar.Header{
			Name:     d,
			Mode:     0750,
			Typeflag: tar.TypeDir,
		}
		require.NoError(t, tw.WriteHeader(hdr))
	}

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err := tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	return buf.Bytes()
}

// makeTestServer creates a test server with a basic workflow.
func makeTestServer(t *testing.T, workflow *domain.Workflow) *httppkg.Server {
	t.Helper()
	logger := slog.Default()
	server, err := httppkg.NewServer(workflow, &MockWorkflowExecutor{}, logger)
	require.NoError(t, err)
	return server
}

// errReader is an io.Reader that always returns an error.
type errReader struct{ err error }

func (e *errReader) Read(_ []byte) (int, error) { return 0, e.err }

// mockResponseWriterWithFlusher implements http.ResponseWriter and http.Flusher.
type mockResponseWriterWithFlusher struct {
	flushCalled bool
}

func (m *mockResponseWriterWithFlusher) Header() stdhttp.Header {
	return stdhttp.Header{}
}

func (m *mockResponseWriterWithFlusher) Write([]byte) (int, error) {
	return 0, nil
}

func (m *mockResponseWriterWithFlusher) WriteHeader(int) {}

func (m *mockResponseWriterWithFlusher) Flush() {
	m.flushCalled = true
}

// mockResponseWriter implements http.ResponseWriter but not http.Flusher.
type mockResponseWriter struct{}

func (m *mockResponseWriter) Header() stdhttp.Header {
	return stdhttp.Header{}
}

func (m *mockResponseWriter) Write([]byte) (int, error) {
	return 0, nil
}

func (m *mockResponseWriter) WriteHeader(int) {}

// writeErrorRecorder is a ResponseRecorder that fails on every Write call,
// used to exercise write-error branches in HandleRequest.
type writeErrorRecorder struct {
	*httptest.ResponseRecorder
}

func (w *writeErrorRecorder) Write(_ []byte) (int, error) {
	return 0, errors.New("forced write error")
}

// failingResponseWriter is a response writer that fails on Write.
type failingResponseWriter struct {
	stdhttp.ResponseWriter
	writeCalled bool
}

func (w *failingResponseWriter) Write(p []byte) (int, error) {
	if !w.writeCalled {
		w.writeCalled = true
		return w.ResponseWriter.Write(p)
	}
	// Fail on second write
	return 0, errors.New("write failed")
}

// MockFileWatcherWithError implements FileWatcher that returns errors.
type MockFileWatcherWithError struct {
	watchError error
	closeError error
}

func (m *MockFileWatcherWithError) Watch(_ string, _ func()) error {
	return m.watchError
}

func (m *MockFileWatcherWithError) Close() error {
	return m.closeError
}

// MockFileWatcherWithCallback extends MockFileWatcher to capture callbacks.
type MockFileWatcherWithCallback struct {
	MockFileWatcher
	callbacks []func()
}

func NewMockFileWatcherWithCallback() *MockFileWatcherWithCallback {
	return &MockFileWatcherWithCallback{
		MockFileWatcher: MockFileWatcher{},
		callbacks:       []func(){},
	}
}

func (m *MockFileWatcherWithCallback) Watch(path string, callback func()) error {
	m.callbacks = append(m.callbacks, callback)
	return m.MockFileWatcher.Watch(path, callback)
}
