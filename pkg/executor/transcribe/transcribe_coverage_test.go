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

package transcribe

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TestResolveTranscribeEndpoint_UnknownBackend covers the else branch at line 101
// (baseURL = openAIBaseURL) and the envKey fallback at line 115-116.
func TestResolveTranscribeEndpoint_UnknownBackend(t *testing.T) {
	t.Setenv("MYBACKEND_API_KEY", "my-key")
	key, url := resolveTranscribeEndpoint(&domain.TranscribeConfig{Backend: "mybackend"})
	assert.Equal(t, "my-key", key)
	assert.Equal(t, openAIBaseURL, url)
}

// TestResolveTranscribeEndpoint_UnknownBackend_NoKey covers the envKey fallback
// with an empty env var.
func TestResolveTranscribeEndpoint_UnknownBackend_NoKey(t *testing.T) {
	t.Setenv("CUSTOMVENDOR_API_KEY", "")
	key, url := resolveTranscribeEndpoint(&domain.TranscribeConfig{Backend: "customvendor"})
	assert.Equal(t, "", key)
	assert.Equal(t, openAIBaseURL, url)
}

// TestCallTranscribeAPI_OptionalFields covers Language, Prompt, Temperature,
// and TimestampGranularities fields in callTranscribeAPI (lines 147-158).
func TestCallTranscribeAPI_OptionalFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		assert.Equal(t, "en", req.FormValue("language"))
		assert.Equal(t, "Transcribe carefully", req.FormValue("prompt"))
		assert.Equal(t, "0.50", req.FormValue("temperature"))
		assert.Equal(t, "word", req.FormValue("timestamp_granularities[]"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("result text"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.wav")
	require.NoError(t, os.WriteFile(audioFile, []byte("audio data"), 0o600))

	e := NewExecutor()
	result, err := e.Execute(nil, &domain.TranscribeConfig{
		File:                   audioFile,
		Backend:                "local",
		BaseURL:                ts.URL + "/v1",
		Model:                  "whisper-1",
		Language:               "en",
		Prompt:                 "Transcribe carefully",
		Temperature:            0.5,
		TimestampGranularities: []string{"word"},
	})
	require.NoError(t, err)
	assert.Equal(t, "result text", result)
}

// TestCallTranscribeAPI_MultipleGranularities covers the for loop over
// TimestampGranularities with more than one entry.
func TestCallTranscribeAPI_MultipleGranularities(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseMultipartForm(10 << 20); err != nil {
			http.Error(w, "bad form", http.StatusBadRequest)
			return
		}
		vals := req.Form["timestamp_granularities[]"]
		assert.Len(t, vals, 2)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.wav")
	require.NoError(t, os.WriteFile(audioFile, []byte("data"), 0o600))

	e := NewExecutor()
	_, err := e.Execute(nil, &domain.TranscribeConfig{
		File:                   audioFile,
		Backend:                "local",
		BaseURL:                ts.URL + "/v1",
		TimestampGranularities: []string{"word", "segment"},
	})
	require.NoError(t, err)
}

// TestCallTranscribeAPI_WithAuthHeader covers the Authorization header
// setting when apiKey is not empty (line 176).
func TestCallTranscribeAPI_WithAuthHeader(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "sk-test-key")

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer sk-test-key", r.Header.Get("Authorization"))
		require.NoError(t, r.ParseMultipartForm(10<<20))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("transcribed text"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.wav")
	require.NoError(t, os.WriteFile(audioFile, []byte("audio data"), 0o600))

	e := NewExecutor()
	result, err := e.Execute(nil, &domain.TranscribeConfig{
		File:    audioFile,
		Backend: "openai",
		BaseURL: ts.URL + "/v1",
	})
	require.NoError(t, err)
	assert.Equal(t, "transcribed text", result)
}

// TestCallTranscribeAPI_BuildRequestError covers the reqErr branch
// when http.NewRequestWithContext fails (line 172).
func TestCallTranscribeAPI_BuildRequestError(t *testing.T) {
	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.wav")
	require.NoError(t, os.WriteFile(audioFile, []byte("audio data"), 0o600))

	_, err := callTranscribeAPI(nil, "key", "http://invalid\x00url", "whisper-1", "text",
		&domain.TranscribeConfig{File: audioFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcribe: build request")
}

// TestCallTranscribeAPI_DoError covers the doErr branch when
// http.DefaultClient.Do fails (line 181).
func TestCallTranscribeAPI_DoError(t *testing.T) {
	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.wav")
	require.NoError(t, os.WriteFile(audioFile, []byte("audio data"), 0o600))

	_, err := callTranscribeAPI(nil, "key", "httpx://invalid-scheme", "whisper-1", "text",
		&domain.TranscribeConfig{File: audioFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcribe: request")
}

// TestCallTranscribeAPI_ReadResponseError covers the readErr branch
// when io.ReadAll on the response body fails (line 187).
func TestCallTranscribeAPI_ReadResponseError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			http.Error(w, "hijack not supported", http.StatusInternalServerError)
			return
		}
		conn, _, err := hj.Hijack()
		require.NoError(t, err, "hijack failed")
		defer conn.Close()
		// Send a partial response with a Content-Length larger than the
		// actual body to trigger an unexpected EOF on the client side.
		_, _ = conn.Write([]byte("HTTP/1.1 200 OK\r\nContent-Length: 99999\r\n\r\n"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.wav")
	require.NoError(t, os.WriteFile(audioFile, []byte("audio data"), 0o600))

	_, err := callTranscribeAPI(nil, "key", ts.URL, "whisper-1", "text",
		&domain.TranscribeConfig{File: audioFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "transcribe: read response")
}
