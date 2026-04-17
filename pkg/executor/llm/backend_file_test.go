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
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

func newFileBackend() *llm.FileBackend {
	return &llm.FileBackend{}
}

// --- FileBackend interface methods ------------------------------------------

func TestFileBackend_Name(t *testing.T) {
	b := newFileBackend()
	if b.Name() != "file" {
		t.Errorf("Name() = %q, want %q", b.Name(), "file")
	}
}

func TestFileBackend_DefaultURL(t *testing.T) {
	b := newFileBackend()
	u := b.DefaultURL()
	if !strings.HasPrefix(u, "http://127.0.0.1:") {
		t.Errorf("DefaultURL() = %q, expected local 127.0.0.1 URL", u)
	}
}

func TestFileBackend_ChatEndpoint(t *testing.T) {
	b := newFileBackend()
	ep := b.ChatEndpoint("http://127.0.0.1:8080")
	if ep != "http://127.0.0.1:8080/v1/chat/completions" {
		t.Errorf("ChatEndpoint() = %q", ep)
	}
}

func TestFileBackend_GetAPIKeyHeader(t *testing.T) {
	b := newFileBackend()
	header, val := b.GetAPIKeyHeader("anything")
	if header != "" || val != "" {
		t.Errorf("GetAPIKeyHeader() = (%q, %q), want empty strings", header, val)
	}
}

func TestFileBackend_BuildRequest_Basic(t *testing.T) {
	b := newFileBackend()
	msgs := []map[string]interface{}{{"role": "user", "content": "hi"}}
	req, err := b.BuildRequest("mymodel", msgs, llm.ChatRequestConfig{})
	if err != nil {
		t.Fatalf("BuildRequest error: %v", err)
	}
	if req["model"] != "mymodel" {
		t.Errorf("model = %v", req["model"])
	}
	if req["stream"] != false {
		t.Errorf("stream should be false")
	}
}

func TestFileBackend_BuildRequest_ContextLength(t *testing.T) {
	b := newFileBackend()
	req, err := b.BuildRequest("m", nil, llm.ChatRequestConfig{ContextLength: 2048})
	if err != nil {
		t.Fatalf("%v", err)
	}
	if req["max_tokens"] != 2048 {
		t.Errorf("max_tokens = %v, want 2048", req["max_tokens"])
	}
}

func TestFileBackend_BuildRequest_JSONResponse(t *testing.T) {
	b := newFileBackend()
	req, err := b.BuildRequest("m", nil, llm.ChatRequestConfig{JSONResponse: true})
	if err != nil {
		t.Fatalf("%v", err)
	}
	rf, ok := req["response_format"].(map[string]interface{})
	if !ok || rf["type"] != "json_object" {
		t.Errorf("response_format = %v", req["response_format"])
	}
}

func TestFileBackend_BuildRequest_Tools(t *testing.T) {
	b := newFileBackend()
	tools := []map[string]interface{}{{"type": "function", "function": map[string]interface{}{"name": "search"}}}
	req, err := b.BuildRequest("m", nil, llm.ChatRequestConfig{Tools: tools})
	if err != nil {
		t.Fatalf("%v", err)
	}
	if req["tools"] == nil {
		t.Error("tools should be set")
	}
}

func TestFileBackend_ParseResponse_OK(t *testing.T) {
	b := newFileBackend()
	body := `{"choices":[{"message":{"role":"assistant","content":"Hello!"}}]}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	result, err := b.ParseResponse(resp)
	if err != nil {
		t.Fatalf("ParseResponse error: %v", err)
	}
	msg, ok := result["message"].(map[string]interface{})
	if !ok {
		t.Fatalf("result[message] not a map: %v", result)
	}
	if msg["content"] != "Hello!" {
		t.Errorf("content = %v", msg["content"])
	}
}

func TestFileBackend_ParseResponse_HTTPError(t *testing.T) {
	b := newFileBackend()
	body := `{"error":"model not loaded"}`
	resp := &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	_, err := b.ParseResponse(resp)
	if err == nil {
		t.Error("expected error on non-200 status")
	}
}

func TestFileBackend_ParseResponse_InvalidJSON(t *testing.T) {
	b := newFileBackend()
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader("not json")),
	}
	_, err := b.ParseResponse(resp)
	if err == nil {
		t.Error("expected error on invalid JSON")
	}
}

func TestFileBackend_ParseResponse_EmptyChoices(t *testing.T) {
	b := newFileBackend()
	body := `{"choices":[]}`
	resp := &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	result, err := b.ParseResponse(resp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result["message"] != nil {
		t.Errorf("expected no message for empty choices, got %v", result["message"])
	}
}

// --- IsRemoteModel -----------------------------------------------------------

func TestIsRemoteModel_HTTP(t *testing.T) {
	if !llm.IsRemoteModel("http://example.com/model.llamafile") {
		t.Error("expected true for http://")
	}
}

func TestIsRemoteModel_HTTPS(t *testing.T) {
	if !llm.IsRemoteModel("https://huggingface.co/Mozilla/model.llamafile") {
		t.Error("expected true for https://")
	}
}

func TestIsRemoteModel_LocalAbsolute(t *testing.T) {
	if llm.IsRemoteModel("/home/user/models/model.llamafile") {
		t.Error("expected false for absolute path")
	}
}

func TestIsRemoteModel_LocalRelative(t *testing.T) {
	if llm.IsRemoteModel("./models/model.llamafile") {
		t.Error("expected false for relative path")
	}
}

func TestIsRemoteModel_BareName(t *testing.T) {
	if llm.IsRemoteModel("model.llamafile") {
		t.Error("expected false for bare filename")
	}
}

func TestIsRemoteModel_Short(t *testing.T) {
	if llm.IsRemoteModel("http") {
		t.Error("expected false for short string")
	}
}

// --- DefaultModelsDir --------------------------------------------------------

func TestDefaultModelsDir(t *testing.T) {
	t.Setenv("KDEPS_MODELS_DIR", "") // ensure env override is not active
	dir, err := llm.DefaultModelsDir()
	if err != nil {
		t.Fatalf("DefaultModelsDir: %v", err)
	}
	if dir == "" {
		t.Error("expected non-empty dir")
	}
}

func TestDefaultModelsDir_EnvOverride(t *testing.T) {
	custom := t.TempDir()
	t.Setenv("KDEPS_MODELS_DIR", custom)
	got, err := llm.DefaultModelsDir()
	if err != nil {
		t.Fatalf("DefaultModelsDir: %v", err)
	}
	if got != custom {
		t.Errorf("got %q, want %q", got, custom)
	}
}

func TestDefaultModelsDir_EnvOverride_CreatesSubDir(t *testing.T) {
	base := t.TempDir()
	sub := base + "/models/cache"
	t.Setenv("KDEPS_MODELS_DIR", sub)
	got, err := llm.DefaultModelsDir()
	if err != nil {
		t.Fatalf("DefaultModelsDir: %v", err)
	}
	if got != sub {
		t.Errorf("got %q, want %q", got, sub)
	}
	if _, statErr := os.Stat(sub); statErr != nil {
		t.Errorf("expected dir to be created: %v", statErr)
	}
}

func TestDefaultModelsDir_EnvOverride_MkdirFails(t *testing.T) {
	if os.Getuid() == 0 {
		t.Skip("root bypasses file permissions")
	}
	base := t.TempDir()
	// Make base read-only so subdirectory creation fails.
	if err := os.Chmod(base, 0500); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(base, 0750) })
	t.Setenv("KDEPS_MODELS_DIR", base+"/sub")
	_, err := llm.DefaultModelsDir()
	if err == nil {
		t.Error("expected error when mkdir fails")
	}
}

// --- FileBackend via HTTP test server ----------------------------------------

func TestFileBackend_RoundTrip(t *testing.T) {
	// Spin up a fake llamafile server.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": "42",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	b := newFileBackend()
	ep := b.ChatEndpoint(srv.URL)
	if ep != srv.URL+"/v1/chat/completions" {
		t.Errorf("endpoint mismatch: %s", ep)
	}
}
