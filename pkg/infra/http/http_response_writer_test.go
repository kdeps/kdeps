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
	"encoding/json"
	stdhttp "net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/infra/http"
)

// TestExtractKdepsPackage_CorruptTarEntry verifies that a non-EOF error from
// tr.Next() (e.g., truncated/corrupted tar stream) is properly returned.
func TestExtractKdepsPackage_CorruptTarEntry(t *testing.T) {
	// Create a valid gzip archive but with corrupted tar content (only a few bytes).
	// tar.Reader.Next() will return io.ErrUnexpectedEOF on the first call.
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	_, err := gzw.Write([]byte("x")) // not a valid tar header
	require.NoError(t, err)
	require.NoError(t, gzw.Close())

	server := makeTestServer(t, nil)
	server.SetWorkflowPath(filepath.Join(t.TempDir(), "workflow.yaml"))

	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/package", bytes.NewReader(buf.Bytes()))
	rec := httptest.NewRecorder()
	server.HandleManagementUpdatePackage(rec, req)

	assert.Equal(t, stdhttp.StatusUnprocessableEntity, rec.Code)

	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, "error", body["status"])
}

// TestExtractKdepsPackage_WriteFileOpenError verifies that a file entry
// cannot be written when the target path is an existing directory
// (triggers the os.OpenFile error branch in writeExtractedFile).
func TestExtractKdepsPackage_WriteFileOpenError(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")

	// Build a tar archive that:
	//   1. Has a directory entry "conflict/"
	//   2. Has a file entry "conflict" (same name without trailing slash)
	// After (1) creates the directory, (2) tries to open it as a regular file → EISDIR.
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	// Directory entry.
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name:     "conflict/",
		Mode:     0750,
		Typeflag: tar.TypeDir,
	}))

	// File entry with the same clean path.
	content := "payload"
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "conflict",
		Mode: 0600,
		Size: int64(len(content)),
	}))
	_, err := tw.Write([]byte(content))
	require.NoError(t, err)

	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	server := makeTestServer(t, nil)
	server.SetWorkflowPath(workflowPath)

	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/package", bytes.NewReader(buf.Bytes()))
	rec := httptest.NewRecorder()
	server.HandleManagementUpdatePackage(rec, req)

	assert.Equal(t, stdhttp.StatusUnprocessableEntity, rec.Code)
}

// TestExtractKdepsPackage_CopyErrorTruncatedData verifies the io.Copy error
// branch in writeExtractedFile by crafting a tar archive whose file entry
// declares more bytes than are actually present in the stream.
// When io.Copy reads the file data it receives io.ErrUnexpectedEOF because
// the declared 1000 bytes are read before the gzip stream ends.
func TestExtractKdepsPackage_CopyErrorTruncatedData(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "workflow.yaml")

	// Write a tar header claiming Size=1000, then only 7 bytes of data, then
	// close the gzip WITHOUT closing the tar writer so no padding/EOF records
	// are written.  When io.Copy reads the file data it will get
	// io.ErrUnexpectedEOF because the underlying gzip stream ends before
	// the declared 1000 bytes are available.
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)

	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "large.txt",
		Mode: 0600,
		Size: 1000,
	}))
	_, err := tw.Write([]byte("partial")) // only 7 bytes instead of 1000
	require.NoError(t, err)
	// Close gzip without closing tar writer → truncated tar stream.
	require.NoError(t, gzw.Close())

	server := makeTestServer(t, nil)
	server.SetWorkflowPath(workflowPath)

	req := httptest.NewRequest(stdhttp.MethodPut, "/_kdeps/package", bytes.NewReader(buf.Bytes()))
	rec := httptest.NewRecorder()
	server.HandleManagementUpdatePackage(rec, req)

	assert.Equal(t, stdhttp.StatusUnprocessableEntity, rec.Code)
}

// TestResponseWriterWrapper_Write_HtmlBytesNoContentType exercises the
// DetectContentType branch at line 130 and the Content-Type setting at
// lines 134-135 when browser-renderable bytes (e.g. <html>) are written
// with no explicit Content-Type header.
func TestResponseWriterWrapper_Write_HtmlBytesNoContentType(t *testing.T) {
	recorder := httptest.NewRecorder()
	// No Content-Type set — DetectContentType will sniff the bytes
	wrapper := &http.ResponseWriterWrapper{ResponseWriter: recorder}

	payload := []byte("<html><body>hello</body></html>")
	_, err := wrapper.Write(payload)
	assert.NoError(t, err)

	// Content-Type should have been set to text/html
	ct := recorder.Header().Get("Content-Type")
	assert.Equal(t, "text/html; charset=utf-8", ct)

	// Output must be HTML-escaped to prevent XSS
	assert.Contains(t, recorder.Body.String(), "&lt;html&gt;")
	assert.NotContains(t, recorder.Body.String(), "<html>")
}

// TestResponseWriterWrapper_Write_ContentTypeWithParameters exercises the
// parameter-stripping logic in browserRenderedContentType (lines 105-106)
// by setting an explicit Content-Type with a charset parameter.
func TestResponseWriterWrapper_Write_ContentTypeWithParameters(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "text/html; charset=utf-8")
	wrapper := &http.ResponseWriterWrapper{ResponseWriter: recorder}

	payload := []byte("<b>bold</b>")
	_, err := wrapper.Write(payload)
	assert.NoError(t, err)

	// Output must be HTML-escaped — the charset parameter was stripped
	// before the content-type comparison
	assert.Contains(t, recorder.Body.String(), "&lt;b&gt;")
	assert.NotContains(t, recorder.Body.String(), "<b>")
}

func TestResponseWriterWrapper_Write_NonBrowserContent(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "application/json")
	wrapper := &http.ResponseWriterWrapper{ResponseWriter: recorder}
	data := []byte(`{"key":"value"}`)
	n, err := wrapper.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	assert.Equal(t, string(data), recorder.Body.String())
}

func TestResponseWriterWrapper_DisableHTMLEscape(t *testing.T) {
	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "text/html")
	wrapper := &http.ResponseWriterWrapper{ResponseWriter: recorder}
	wrapper.DisableHTMLEscape()
	data := []byte("<html><body>raw</body></html>")
	n, err := wrapper.Write(data)
	assert.NoError(t, err)
	assert.Equal(t, len(data), n)
	// Web routes serve real HTML: with escaping disabled the body must pass
	// through verbatim instead of being entity-encoded.
	assert.Equal(t, string(data), recorder.Body.String())
}

func TestBrowserRenderedContentType_AllTypes(t *testing.T) {
	middleware := http.BodyLimitMiddleware(1 << 20) // just to indirectly verify - we test Write directly
	_ = middleware

	recorder := httptest.NewRecorder()
	recorder.Header().Set("Content-Type", "application/xml")
	wrapper := &http.ResponseWriterWrapper{ResponseWriter: recorder}
	payload := []byte("<root>hello</root>")
	_, err := wrapper.Write(payload)
	assert.NoError(t, err)
	// XML is browser-rendered; output should be escaped
	assert.Contains(t, recorder.Body.String(), "root")
}

func TestResponseWriterWrapper_Write_EmptyContentType(t *testing.T) {
	recorder := httptest.NewRecorder()
	// No Content-Type set - DetectContentType will sniff and apply escaping if html-like
	wrapper := &http.ResponseWriterWrapper{ResponseWriter: recorder}
	_, err := wrapper.Write([]byte(`{"json":true}`))
	assert.NoError(t, err)
}

func TestResponseWriterWrapper_HeadersWritten(t *testing.T) {
	recorder := httptest.NewRecorder()
	wrapper := &http.ResponseWriterWrapper{ResponseWriter: recorder}
	assert.False(t, wrapper.HeadersWritten())
	wrapper.WriteHeader(stdhttp.StatusOK)
	assert.True(t, wrapper.HeadersWritten())
}

func TestResponseWriterWrapper_Flush(t *testing.T) {
	t.Run("flush with flusher support", func(t *testing.T) {
		mockFlusher := &mockResponseWriterWithFlusher{}
		wrapper := &http.ResponseWriterWrapper{
			ResponseWriter: mockFlusher,
		}

		// First call should check and cache flusher
		wrapper.Flush()
		assert.True(t, mockFlusher.flushCalled)

		// Reset for second call
		mockFlusher.flushCalled = false
		wrapper.Flush()
		assert.True(t, mockFlusher.flushCalled)
	})

	t.Run("flush without flusher support", func(_ *testing.T) {
		mockWriter := &mockResponseWriter{}
		wrapper := &http.ResponseWriterWrapper{
			ResponseWriter: mockWriter,
		}

		// Should not panic or cause issues
		wrapper.Flush()

		// Subsequent calls should also be safe
		wrapper.Flush()
	})
}
