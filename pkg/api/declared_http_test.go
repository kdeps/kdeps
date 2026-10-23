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

package api

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeclaredHTTPTool_RequestError(t *testing.T) {
	orig := HTTPToolClient
	HTTPToolClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, errors.New("connection refused")
	})}
	t.Cleanup(func() { HTTPToolClient = orig })
	_, err := callDeclaredHTTPTool(context.Background(), "t", &HTTPToolSpec{Method: "GET", URL: "http://x"}, nil, nil)
	require.ErrorContains(t, err, "t: request:")
}

func TestHTTPToolSpec_Validate(t *testing.T) {
	ok := HTTPToolSpec{Method: "post", URL: "https://x/{{.args.q}}", Result: "{{.response.a}}"}
	require.NoError(t, ok.Validate())
	for name, s := range map[string]HTTPToolSpec{
		"method":   {Method: "TRACE", URL: "https://x"},
		"url":      {Method: "GET"},
		"template": {Method: "GET", URL: "https://x/{{.args.q"},
		"header":   {Method: "GET", URL: "https://x", Headers: map[string]string{"A": "{{end}}"}},
		"env":      {Method: "GET", URL: "https://x", Env: map[string][]string{"K": nil}},
		"timeout":  {Method: "GET", URL: "https://x", Timeout: "soon"},
	} {
		assert.Error(t, s.Validate(), name)
	}
}

func TestGetPath(t *testing.T) {
	v := map[string]any{"a": []any{map[string]any{"b": "x"}}}
	assert.Equal(t, "x", getPath(v, "a", 0, "b"))
	assert.Nil(t, getPath(v, "a", 5))
	assert.Nil(t, getPath(v, "a", 0, "b", "c"))
	assert.Nil(t, getPath(nil, "a"))
}

func TestHTTPToolSpec_Timeout(t *testing.T) {
	assert.Equal(t, defaultHTTPToolTimeout, (&HTTPToolSpec{}).timeout())
	assert.Equal(t, 5*time.Second, (&HTTPToolSpec{Timeout: "5s"}).timeout())

	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	t.Cleanup(func() { close(release); srv.Close() })
	_, err := callDeclaredHTTPTool(context.Background(), "slow",
		&HTTPToolSpec{Method: "GET", URL: srv.URL, Timeout: "50ms"}, nil, nil)
	require.ErrorContains(t, err, "slow: request:")
}

func TestDeclaredHTTPTool_MultipartFilesAndForm(t *testing.T) {
	var got struct{ model, lang, file, name, ctype, auth string }
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got.ctype, got.auth = r.Header.Get("Content-Type"), r.Header.Get("Authorization")
		require.NoError(t, r.ParseMultipartForm(1<<20))
		got.model, got.lang = r.FormValue("model"), r.FormValue("language")
		f, hdr, err := r.FormFile("file")
		require.NoError(t, err)
		defer f.Close()
		b, _ := io.ReadAll(f)
		got.file, got.name = string(b), hdr.Filename
		_, _ = w.Write([]byte("the transcript"))
	}))
	t.Cleanup(srv.Close)
	audio := filepath.Join(t.TempDir(), "a.wav")
	require.NoError(t, os.WriteFile(audio, []byte("RIFF"), 0o600))

	s := &HTTPToolSpec{
		Method:  "POST",
		URL:     srv.URL + "/audio/transcriptions",
		Headers: map[string]string{"Authorization": "{{with .env.KEY}}Bearer {{.}}{{end}}"},
		Files:   map[string]string{"file": "{{.args.file}}"},
		Form:    map[string]string{"model": "{{.args.model}}", "language": "{{.args.language}}"},
		Paths:   []string{"file"},
	}
	out, err := callDeclaredHTTPTool(context.Background(), "tr", s,
		map[string]any{"file": audio, "model": "whisper-1"}, map[string]string{"KEY": ""})
	require.NoError(t, err)
	assert.Equal(t, "the transcript", out)
	assert.Contains(t, got.ctype, "multipart/form-data")
	assert.Equal(t, "whisper-1", got.model)
	assert.Empty(t, got.lang, "an empty form field is left out")
	assert.Equal(t, "RIFF", got.file)
	assert.Equal(t, "a.wav", got.name)
	assert.Empty(t, got.auth, "a header that renders empty is not sent")

	t.Setenv("KDEPS_ALLOW_ROOT", "")
	_, err = callDeclaredHTTPTool(context.Background(), "tr", s, map[string]any{"file": "/"}, nil)
	require.ErrorContains(t, err, "tr: operation on root directory")
	_, err = callDeclaredHTTPTool(context.Background(), "tr", s,
		map[string]any{"file": filepath.Join(t.TempDir(), "missing.wav")}, nil)
	require.ErrorContains(t, err, "tr:")
}

func TestHTTPToolSpec_OptionalEnv(t *testing.T) {
	t.Setenv("OPT_A", "")
	t.Setenv("OPT_B", "")
	s := &HTTPToolSpec{Env: map[string][]string{"A": {"OPT_A", ""}, "B": {"OPT_B"}}}
	_, ok := s.ResolveEnv()
	assert.False(t, ok, "a required entry still gates registration")

	t.Setenv("OPT_B", "b")
	env, ok := s.ResolveEnv()
	require.True(t, ok, "an optional entry may stay empty")
	assert.Empty(t, env["A"])
	assert.Equal(t, "b", env["B"])

	t.Setenv("OPT_A", "a")
	env, _ = s.ResolveEnv()
	assert.Equal(t, "a", env["A"])
	require.Error(
		t,
		(&HTTPToolSpec{Method: "POST", URL: "https://x", Files: map[string]string{"f": "{{.args.f"}}).Validate(),
	)
}
