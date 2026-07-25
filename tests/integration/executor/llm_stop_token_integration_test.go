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

package executor_test

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

// TestStreamChat_StripsLocalStopToken verifies that the chat template's stop
// token a llamafile/gguf server streams as ordinary text reaches neither the
// live output writer nor the returned content.
func TestStreamChat_StripsLocalStopToken(t *testing.T) {
	t.Parallel()

	// Split the stop token across two SSE chunks: local servers stream one
	// token per event, but the filter must also survive a mid-token split.
	chunks := []string{"Hello", "!", "<|eot", "_id|>"}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Error("test server writer must support flushing")
			return
		}
		for _, c := range chunks {
			fmt.Fprintf(w,
				"data: {\"choices\":[{\"index\":0,\"delta\":{\"content\":%q}}]}\n\n", c)
			flusher.Flush()
		}
		fmt.Fprint(w,
			"data: {\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	cfg := &domain.ChatConfig{
		Model:   "tinyllama1.1",
		Backend: llm.BackendFile,
		BaseURL: srv.URL,
		Prompt:  "say hello",
	}

	var out bytes.Buffer
	content, toolCalls, err := llm.NewAdapter(srv.URL).StreamChat(t.Context(), cfg, &out)
	if err != nil {
		t.Fatalf("StreamChat: %v", err)
	}
	if len(toolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(toolCalls))
	}

	for _, token := range []string{"<|eot_id|>", "<|eot", "_id|>"} {
		if strings.Contains(out.String(), token) {
			t.Errorf("live output leaked %q: %q", token, out.String())
		}
		if strings.Contains(content, token) {
			t.Errorf("returned content leaked %q: %q", token, content)
		}
	}
	if got := strings.TrimSpace(out.String()); got != "Hello!" {
		t.Errorf("live output = %q, want %q", got, "Hello!")
	}
	if got := strings.TrimSpace(content); got != "Hello!" {
		t.Errorf("content = %q, want %q", got, "Hello!")
	}
}
