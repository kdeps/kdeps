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

func TestTranscribeExecutor_MissingFile(t *testing.T) {
	e := NewExecutor()
	_, err := e.Execute(nil, &domain.TranscribeConfig{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file is required")
}

func TestTranscribeExecutor_FileNotFound(t *testing.T) {
	e := NewExecutor()
	_, err := e.Execute(nil, &domain.TranscribeConfig{
		File: "/nonexistent/audio.mp3",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "open")
}

func TestResolveTranscribeEndpoint_OpenAI(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "test-key")
	key, url := resolveTranscribeEndpoint(&domain.TranscribeConfig{})
	assert.Equal(t, "test-key", key)
	assert.Equal(t, openAIBaseURL, url)
}

func TestResolveTranscribeEndpoint_Groq(t *testing.T) {
	t.Setenv("GROQ_API_KEY", "groq-key")
	key, url := resolveTranscribeEndpoint(&domain.TranscribeConfig{Backend: "groq"})
	assert.Equal(t, "groq-key", key)
	assert.Equal(t, groqBaseURL, url)
}

func TestResolveTranscribeEndpoint_Local(t *testing.T) {
	key, url := resolveTranscribeEndpoint(&domain.TranscribeConfig{Backend: "local"})
	assert.Equal(t, "", key)
	assert.Equal(t, localBaseURL, url)
}

func TestResolveTranscribeEndpoint_CustomBaseURL(t *testing.T) {
	_, url := resolveTranscribeEndpoint(&domain.TranscribeConfig{
		Backend: "openai",
		BaseURL: "http://custom:8080/v1",
	})
	assert.Equal(t, "http://custom:8080/v1", url)
}

func TestTranscribeExecutor_APISuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/audio/transcriptions", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("Hello world"))
	}))
	defer ts.Close()

	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.mp3")
	require.NoError(t, os.WriteFile(audioFile, []byte("fake audio data"), 0o600))

	e := NewExecutor()
	result, err := e.Execute(nil, &domain.TranscribeConfig{
		File:    audioFile,
		Backend: "local",
		BaseURL: ts.URL + "/v1",
		Model:   "whisper-1",
	})
	require.NoError(t, err)
	assert.Equal(t, "Hello world", result)
}

func TestTranscribeExecutor_APIError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid_api_key"}`))
	}))
	defer ts.Close()

	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.mp3")
	require.NoError(t, os.WriteFile(audioFile, []byte("fake audio data"), 0o600))

	e := NewExecutor()
	_, err := e.Execute(nil, &domain.TranscribeConfig{
		File:    audioFile,
		Backend: "local",
		BaseURL: ts.URL + "/v1",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "API error 401")
}

func TestTranscribeExecutor_JSONResponseFormat(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"text":"transcribed text","duration":5.2}`))
	}))
	defer ts.Close()

	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.wav")
	require.NoError(t, os.WriteFile(audioFile, []byte("fake wav data"), 0o600))

	e := NewExecutor()
	result, err := e.Execute(nil, &domain.TranscribeConfig{
		File:           audioFile,
		Backend:        "local",
		BaseURL:        ts.URL + "/v1",
		ResponseFormat: "json",
	})
	require.NoError(t, err)
	assert.Equal(t, "transcribed text", result)
}

func TestTranscribeExecutor_PlainTextResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("  plain text output  "))
	}))
	defer ts.Close()

	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.wav")
	require.NoError(t, os.WriteFile(audioFile, []byte("fake wav"), 0o600))

	e := NewExecutor()
	result, err := e.Execute(nil, &domain.TranscribeConfig{
		File:    audioFile,
		Backend: "local",
		BaseURL: ts.URL + "/v1",
	})
	require.NoError(t, err)
	assert.Equal(t, "plain text output", result)
}

func TestNewAdapter_ReturnsNonNil(t *testing.T) {
	a := NewAdapter()
	assert.NotNil(t, a)
}

func TestTranscribeConfig_Defaults(t *testing.T) {
	e := NewExecutor()
	dir := t.TempDir()
	audioFile := filepath.Join(dir, "test.mp3")
	require.NoError(t, os.WriteFile(audioFile, []byte("data"), 0o600))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		parseErr := req.ParseMultipartForm(10 << 20)
		require.NoError(t, parseErr)
		assert.Equal(t, defaultModel, req.FormValue("model"))
		assert.Equal(t, defaultResponseFormat, req.FormValue("response_format"))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer ts.Close()

	_, _ = e.Execute(nil, &domain.TranscribeConfig{
		File:    audioFile,
		Backend: "local",
		BaseURL: ts.URL + "/v1",
	})
}
