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

//go:build !js

package llm

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/config"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

// simpleMockToolExecutor implements toolExecutorInterface for testing tool execution.
type simpleMockToolExecutor struct{}

func (m *simpleMockToolExecutor) ExecuteResource(
	_ *domain.Resource,
	_ *executor.ExecutionContext,
) (interface{}, error) {
	return "mock resource result", nil
}

// ─── applyRoute ──────────────────────────────────────────────────────────────

func TestApplyRoute_SetsAllFields(t *testing.T) {
	cfg := &domain.ChatConfig{}
	route := &config.ModelEntry{
		Model:   "gpt-4o",
		Backend: "openai",
		BaseURL: "https://api.openai.com",
	}
	applyRoute(cfg, route)
	assert.Equal(t, "gpt-4o", cfg.Model)
	assert.Equal(t, "openai", cfg.Backend)
	assert.Equal(t, "https://api.openai.com", cfg.BaseURL)
}

func TestApplyRoute_DoesNotOverwriteNonEmpty(t *testing.T) {
	cfg := &domain.ChatConfig{
		Model:   "existing-model",
		Backend: "existing-backend",
		BaseURL: "existing-url",
	}
	route := &config.ModelEntry{} // empty entry — should not overwrite
	applyRoute(cfg, route)
	assert.Equal(t, "existing-model", cfg.Model)
	assert.Equal(t, "existing-backend", cfg.Backend)
	assert.Equal(t, "existing-url", cfg.BaseURL)
}

// ─── applyLLMRouter ──────────────────────────────────────────────────────────

func TestApplyLLMRouter_FallbackStrategy(t *testing.T) {
	routerJSON, err := json.Marshal(config.UnifiedModelsConfig{
		Strategy: "fallback",
		Models: []config.ModelEntry{
			{Model: "llama3.2:1b", Backend: "ollama", BaseURL: "http://host-a:11434", Priority: 2},
			{Model: "llama3.1:8b", Backend: "ollama", BaseURL: "http://host-b:11434", Priority: 1},
		},
	})
	require.NoError(t, err)
	t.Setenv("KDEPS_LLM_ROUTER", string(routerJSON))

	e := NewExecutor("")
	cfg := &domain.ChatConfig{Model: "router"}
	entries := e.applyLLMRouter(cfg, "test prompt")
	require.NotNil(t, entries, "should return sorted entries for fallback strategy")
	// Priority 1 (lower) comes first -> llama3.1:8b applied
	assert.Equal(t, "llama3.1:8b", cfg.Model)
	assert.Equal(t, "ollama", cfg.Backend)
	assert.Equal(t, "http://host-b:11434", cfg.BaseURL)
	assert.Len(t, entries, 2)
	assert.Equal(t, "llama3.1:8b", entries[0].Model)
	assert.Equal(t, "llama3.2:1b", entries[1].Model)
}

func TestApplyLLMRouter_CostOptimizedStrategy(t *testing.T) {
	routerJSON, err := json.Marshal(config.UnifiedModelsConfig{
		Strategy: "cost_optimized",
		Models: []config.ModelEntry{
			{Model: "gpt-4o", Backend: "openai", CostPerInputToken: floatPtr(0.0025)},
			{Model: "gpt-4o-mini", Backend: "openai", CostPerInputToken: floatPtr(0.00015)},
		},
	})
	require.NoError(t, err)
	t.Setenv("KDEPS_LLM_ROUTER", string(routerJSON))

	e := NewExecutor("")
	cfg := &domain.ChatConfig{Model: "router"}
	entries := e.applyLLMRouter(cfg, "some prompt")
	assert.Nil(t, entries, "cost_optimized returns nil entries")
	assert.Equal(t, "gpt-4o-mini", cfg.Model, "should pick cheapest")
	assert.Equal(t, "openai", cfg.Backend)
}

func TestApplyLLMRouter_RoundRobinStrategy(t *testing.T) {
	routerJSON, err := json.Marshal(config.UnifiedModelsConfig{
		Strategy: "round_robin",
		Models: []config.ModelEntry{
			{Model: "model-a", Backend: "openai"},
			{Model: "model-b", Backend: "openai"},
		},
	})
	require.NoError(t, err)
	t.Setenv("KDEPS_LLM_ROUTER", string(routerJSON))

	e := NewExecutor("")
	cfg := &domain.ChatConfig{Model: "router"}
	entries := e.applyLLMRouter(cfg, "prompt")
	assert.Nil(t, entries, "round_robin returns nil entries")
	assert.NotEmpty(t, cfg.Model)
	assert.Equal(t, "openai", cfg.Backend)
}

func TestApplyLLMRouter_InvalidJSON(t *testing.T) {
	t.Setenv("KDEPS_LLM_ROUTER", "not-valid-json")
	e := NewExecutor("")
	cfg := &domain.ChatConfig{Model: "router"}
	entries := e.applyLLMRouter(cfg, "prompt")
	assert.Nil(t, entries)
	assert.Equal(t, "router", cfg.Model, "should not mutate cfg on invalid JSON")
}

func TestApplyLLMRouter_EmptyModels(t *testing.T) {
	routerJSON, err := json.Marshal(config.UnifiedModelsConfig{
		Strategy: "fallback",
		Models:   []config.ModelEntry{},
	})
	require.NoError(t, err)
	t.Setenv("KDEPS_LLM_ROUTER", string(routerJSON))
	e := NewExecutor("")
	cfg := &domain.ChatConfig{Model: "router"}
	entries := e.applyLLMRouter(cfg, "prompt")
	assert.Nil(t, entries)
}

func TestApplyLLMRouter_NoEnvVar(t *testing.T) {
	e := NewExecutor("")
	cfg := &domain.ChatConfig{Model: "router"}
	entries := e.applyLLMRouter(cfg, "prompt")
	assert.Nil(t, entries)
}

// ─── retryFallbackRoutes ─────────────────────────────────────────────────────

func TestRetryFallbackRoutes_LoopsThroughRoutes(t *testing.T) {
	// Server 1: returns error response
	server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{"error": "model temporarily unavailable"}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server1.Close()

	// Server 2: returns success
	server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		resp := map[string]interface{}{
			"model":   "llama3.2:1b",
			"message": map[string]interface{}{"role": "assistant", "content": "fallback success"},
			"done":    true,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server2.Close()

	e := NewExecutor("")
	cfg := &domain.ChatConfig{Model: "llama3.2:1b", Backend: "ollama", BaseURL: server1.URL}

	// Two fallback routes sorted by priority: first (applied) fails, second succeeds.
	fallbackRoutes := []config.ModelEntry{
		{Model: "llama3.2:1b", Backend: "ollama", BaseURL: server1.URL, Priority: 1},
		{Model: "llama3.2:1b", Backend: "ollama", BaseURL: server2.URL, Priority: 2},
	}

	messages := []map[string]interface{}{
		{"role": "user", "content": "hello"},
	}
	requestConfig := ChatRequestConfig{
		ContextLength: 4096,
	}
	response := map[string]interface{}{"error": "first backend failed"}

	result, lastErr := e.retryFallbackRoutes(fallbackRoutes, cfg, messages, requestConfig, response, 5*time.Second)
	assert.NoError(t, lastErr)
	// Should have switched to server2
	assert.Equal(t, "llama3.2:1b", cfg.Model)
	assert.Equal(t, "ollama", cfg.Backend)
	assert.Equal(t, server2.URL, cfg.BaseURL)
	// Response should be success from server2
	msg, ok := result["message"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "fallback success", msg["content"])
}

func TestRetryFallbackRoutes_SingleRouteReturnsEarly(t *testing.T) {
	e := NewExecutor("")
	cfg := &domain.ChatConfig{}
	fallbackRoutes := []config.ModelEntry{
		{Model: "llama3.2:1b", Priority: 0},
	}
	messages := []map[string]interface{}{}
	requestConfig := ChatRequestConfig{}
	response := map[string]interface{}{"error": "some error"}

	result, lastErr := e.retryFallbackRoutes(fallbackRoutes, cfg, messages, requestConfig, response, time.Second)
	assert.NoError(t, lastErr)
	assert.Equal(t, response, result, "single route should return original response unchanged")
}

// ─── executeTool ─────────────────────────────────────────────────────────────

func TestExecuteTool_ExecuteFunc(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(
		&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}},
	)
	require.NoError(t, err)

	tool := domain.Tool{
		Name: "my_tool",
		Execute: func(_ map[string]interface{}) (string, error) {
			return "direct execute result", nil
		},
	}

	result, execErr := e.executeTool(tool, `{"key":"val"}`, ctx)
	require.NoError(t, execErr)
	assert.Equal(t, "direct execute result", result)
}

func TestExecuteTool_KdepsResourcePath(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(
		&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}},
	)
	require.NoError(t, err)

	ctx.Resources = map[string]*domain.Resource{
		"my-resource": {},
	}

	e.SetToolExecutor(&simpleMockToolExecutor{})

	tool := domain.Tool{
		Name:   "resource_tool",
		Script: "my-resource",
	}

	result, execErr := e.executeTool(tool, `{"x":1}`, ctx)
	require.NoError(t, execErr)
	assert.Equal(t, "mock resource result", result)
}

func TestExecuteTool_NoToolExecutor(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(
		&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}},
	)
	require.NoError(t, err)

	ctx.Resources = map[string]*domain.Resource{
		"my-resource": {},
	}

	// toolExecutor is nil
	tool := domain.Tool{
		Name:   "resource_tool",
		Script: "my-resource",
	}

	_, execErr := e.executeTool(tool, `{}`, ctx)
	require.Error(t, execErr)
	assert.Contains(t, execErr.Error(), "tool executor not available")
}

// ─── handleToolCalls via Execute ─────────────────────────────────────────────

func TestExecutor_Execute_HandleToolCalls_Loop(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		if callCount == 1 {
			// First call: return tool_calls
			resp := map[string]interface{}{
				"model": "llama3.2:1b",
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "",
					"tool_calls": []interface{}{
						map[string]interface{}{
							"id": "call_1",
							"function": map[string]interface{}{
								"name":      "my_tool",
								"arguments": `{}`,
							},
						},
					},
				},
				"done": true,
			}
			_ = json.NewEncoder(w).Encode(resp)
		} else {
			// Second call: return final response
			resp := map[string]interface{}{
				"model": "llama3.2:1b",
				"message": map[string]interface{}{
					"role":    "assistant",
					"content": "Final answer after tool call",
				},
				"done": true,
			}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	executorLLM := NewExecutor(server.URL)
	executorLLM.SetToolExecutor(&simpleMockToolExecutor{})

	ctx, innerErr := executor.NewExecutionContext(
		&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}},
	)
	require.NoError(t, innerErr)

	config := &domain.ChatConfig{
		Model:   "llama3.2:1b",
		Prompt:  "Use the tool",
		BaseURL: server.URL,
		Tools: []domain.Tool{
			{
				Name: "my_tool",
				Execute: func(_ map[string]interface{}) (string, error) {
					return "tool result", nil
				},
			},
		},
	}

	result, execErr := executorLLM.Execute(ctx, config)
	require.NoError(t, execErr)
	assert.Equal(t, 2, callCount, "should have made 2 calls (initial + tool follow-up)")

	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	message, ok := resultMap["message"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "Final answer after tool call", message["content"])
}

func TestExecutor_Execute_HandleToolCalls_MaxIterations(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		callCount++
		// Always return tool_calls to trigger the loop repeatedly
		resp := map[string]interface{}{
			"model": "llama3.2:1b",
			"message": map[string]interface{}{
				"role":    "assistant",
				"content": "",
				"tool_calls": []interface{}{
					map[string]interface{}{
						"id": "call_x",
						"function": map[string]interface{}{
							"name":      "my_tool",
							"arguments": `{}`,
						},
					},
				},
			},
			"done": true,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	executorLLM := NewExecutor(server.URL)
	executorLLM.SetToolExecutor(&simpleMockToolExecutor{})

	ctx, innerErr := executor.NewExecutionContext(
		&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}},
	)
	require.NoError(t, innerErr)

	config := &domain.ChatConfig{
		Model:   "llama3.2:1b",
		Prompt:  "Use the tool repeatedly",
		BaseURL: server.URL,
		Tools: []domain.Tool{
			{
				Name: "my_tool",
				Execute: func(_ map[string]interface{}) (string, error) {
					return "tool result", nil
				},
			},
		},
	}

	// The loop has maxIterations=5, so we expect 6 calls (1 initial + 5 follow-ups),
	// after which the loop exits (returns the last response which still has tool_calls).
	// handleToolCalls returns currentResponse, so the final response still has tool_calls.
	result, execErr := executorLLM.Execute(ctx, config)
	require.NoError(t, execErr)
	assert.Equal(t, 6, callCount, "should hit max 5 iterations = 6 total calls")

	// Final response still contains tool_calls (loop exhausted)
	resultMap, ok := result.(map[string]interface{})
	require.True(t, ok)
	message, ok := resultMap["message"].(map[string]interface{})
	require.True(t, ok)
	_, hasToolCalls := message["tool_calls"]
	assert.True(t, hasToolCalls)
}

// ─── executeToolCalls with tool.Execute ──────────────────────────────────────

func TestExecuteToolCalls_WithExecuteFunc(t *testing.T) {
	e := NewExecutor("")
	ctx, err := executor.NewExecutionContext(
		&domain.Workflow{Metadata: domain.WorkflowMetadata{Name: "test"}},
	)
	require.NoError(t, err)

	toolCalls := []map[string]interface{}{
		{
			"id": "tc1",
			"function": map[string]interface{}{
				"name":      "my_tool",
				"arguments": `{"x":1}`,
			},
		},
	}
	tools := []domain.Tool{
		{
			Name: "my_tool",
			Execute: func(_ map[string]interface{}) (string, error) {
				return "executed", nil
			},
		},
	}

	results, execErr := e.executeToolCalls(toolCalls, tools, ctx)
	require.NoError(t, execErr)
	require.Len(t, results, 1)
	assert.Equal(t, "executed", results[0]["content"])
}

// ─── loadImageAsBase64 error path ────────────────────────────────────────────

func TestLoadImageAsBase64_FileNotFound(t *testing.T) {
	e := NewExecutor("")
	ctx := &executor.ExecutionContext{}

	_, _, err := e.loadImageAsBase64("/nonexistent/path/image.png", ctx)
	require.Error(t, err)
}

// ─── detectImageMimeType content-based ───────────────────────────────────────

func TestDetectImageMimeType_ContentBasedDetection(t *testing.T) {
	tmp := t.TempDir()
	// Create a file with PNG magic bytes but a .xyz extension (not in known list)
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}
	filePath := filepath.Join(tmp, "test.xyz")
	require.NoError(t, os.WriteFile(filePath, pngHeader, 0o600))

	e := NewExecutor("")
	mime, err := e.detectImageMimeType(filePath)
	require.NoError(t, err)
	assert.Equal(t, "image/png", mime)
}

func TestDetectImageMimeType_ContentBasedNonImage(t *testing.T) {
	tmp := t.TempDir()
	// Create a file with unknown bytes and unknown extension
	filePath := filepath.Join(tmp, "data.xyz")
	require.NoError(t, os.WriteFile(filePath, []byte("plain text data"), 0o600))

	e := NewExecutor("")
	_, err := e.detectImageMimeType(filePath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported image type")
}

// ─── encodeFileToBase64 empty MIME type ──────────────────────────────────────

func TestEncodeFileToBase64_EmptyMimeTypeDefaultsToJPEG(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "test.bin")
	require.NoError(t, os.WriteFile(filePath, []byte("image data"), 0o600))

	e := NewExecutor("")
	dataURI, mime, err := e.encodeFileToBase64(filePath, "")
	require.NoError(t, err)
	assert.Equal(t, "image/jpeg", mime)
	assert.Contains(t, dataURI, "data:image/jpeg;base64,")
}

// ─── parseJSONResponse empty key filtering ───────────────────────────────────

func TestParseJSONResponse_EmptyKeyFilter(t *testing.T) {
	e := NewExecutor("")
	response := map[string]interface{}{
		"message": map[string]interface{}{
			"content": `{"score": 9, "reason": "great match"}`,
		},
	}

	// Keys that don't exist in the response — should return full jsonData
	result, err := e.parseJSONResponse(response, []string{"nonexistent"})
	require.NoError(t, err)
	m, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(9), m["score"])
	assert.Equal(t, "great match", m["reason"])
}

// ─── extractHostPortFromParsedURL invalid port ───────────────────────────────

func TestExtractHostPortFromParsedURL_EmptyPort(t *testing.T) {
	// URL with empty port after colon triggers the manual extraction fallback.
	// url.Parse("http://host:/") creates Host="host:" and Port() returns "".
	parsedURL, err := url.Parse("http://host:/path")
	require.NoError(t, err)

	host, port := extractHostPortFromParsedURL(parsedURL, "default", 8080)
	assert.Equal(t, "host", host)
	assert.Equal(t, 8080, port, "should use default port when port part is empty")
}

// ─── evaluateStringOrLiteral nil evaluator ───────────────────────────────────

func TestEvaluateStringOrLiteral_NilEvaluator(t *testing.T) {
	e := NewExecutor("")
	// An expression string with {{ }} syntax and nil evaluator
	_, err := e.evaluateStringOrLiteral(nil, nil, "{{get('key')}}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expression evaluation not available")
}

// ─── parseHostPortFromURL ────────────────────────────────────────────────────

func TestParseHostPortFromURL_ManualExtractionOnParseError(t *testing.T) {
	// A URL that causes url.Parse to fail — triggers manual extraction path.
	// The manual extraction for "http://%" extracts "%" as the hostname.
	host, port := parseHostPortFromURL("http://%", "default", 9090)
	assert.Equal(t, "%", host)
	assert.Equal(t, 9090, port)
}
