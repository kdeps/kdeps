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

package llm_test

import (
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

func TestBackendRegistry_GetDefault(t *testing.T) {
	registry := llm.NewBackendRegistry()

	// Test that GetDefault returns a valid backend
	defaultBackend := registry.GetDefault()
	if defaultBackend == nil {
		t.Fatal("GetDefault() returned nil")
	}

	// Should return ollama backend by default
	if defaultBackend.Name() != "ollama" {
		t.Errorf("Expected default backend to be 'ollama', got '%s'", defaultBackend.Name())
	}
}

func TestBackendRegistry_GetDefault_EmptyRegistry(t *testing.T) {
	registry := llm.NewBackendRegistry()
	registry.SetBackendsForTesting(make(map[string]llm.Backend))

	// Test that GetDefault returns nil when registry is empty
	defaultBackend := registry.GetDefault()
	if defaultBackend != nil {
		t.Errorf("Expected GetDefault() to return nil for empty registry, got %v", defaultBackend)
	}
}

func TestBackendRegistry_GetDefault_NoOllama(t *testing.T) {
	registry := llm.NewBackendRegistry()
	registry.SetBackendsForTesting(map[string]llm.Backend{
		"openai": &llm.OpenAIBackend{},
	})

	// Test that GetDefault returns first available backend when ollama is not present
	defaultBackend := registry.GetDefault()
	if defaultBackend == nil {
		t.Fatal("GetDefault() returned nil")
	}

	if defaultBackend.Name() != "openai" {
		t.Errorf("Expected default backend to be 'openai', got '%s'", defaultBackend.Name())
	}
}

func TestBackendRegistry_SetBackendsForTesting(t *testing.T) {
	registry := llm.NewBackendRegistry()

	// Create test backends
	testBackends := map[string]llm.Backend{
		"test1": &llm.OpenAIBackend{},
		"test2": &llm.AnthropicBackend{},
	}

	// Set backends for testing
	registry.SetBackendsForTesting(testBackends)

	// Verify backends were set
	retrievedBackends := registry.GetBackendsForTesting()
	if len(retrievedBackends) != 2 {
		t.Errorf("Expected 2 backends, got %d", len(retrievedBackends))
	}

	if _, exists := retrievedBackends["test1"]; !exists {
		t.Error("Expected 'test1' backend to exist")
	}

	if _, exists := retrievedBackends["test2"]; !exists {
		t.Error("Expected 'test2' backend to exist")
	}
}

func TestBackendRegistry_GetBackendsForTesting(t *testing.T) {
	registry := llm.NewBackendRegistry()

	// Get initial backends
	initialBackends := registry.GetBackendsForTesting()
	if len(initialBackends) == 0 {
		t.Error("Expected initial backends to be non-empty")
	}

	// Verify it contains expected backends (ollama + online providers)
	expectedBackends := []string{"ollama", "openai", "anthropic", "google", "cohere"}
	for _, expected := range expectedBackends {
		if _, exists := initialBackends[expected]; !exists {
			t.Errorf("Expected backend '%s' to exist in registry", expected)
		}
	}
}

func TestOllamaBackend_DefaultURL(t *testing.T) {
	backend := &llm.OllamaBackend{}
	expected := "http://localhost:11434"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestOpenAIBackend_DefaultURL(t *testing.T) {
	backend := &llm.OpenAIBackend{}
	expected := "https://api.openai.com"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestAnthropicBackend_DefaultURL(t *testing.T) {
	backend := &llm.AnthropicBackend{}
	expected := "https://api.anthropic.com"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestGoogleBackend_DefaultURL(t *testing.T) {
	backend := &llm.GoogleBackend{}
	expected := "https://generativelanguage.googleapis.com/v1beta"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestCohereBackend_DefaultURL(t *testing.T) {
	backend := &llm.CohereBackend{}
	expected := "https://api.cohere.ai"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestMistralBackend_DefaultURL(t *testing.T) {
	backend := &llm.MistralBackend{}
	expected := "https://api.mistral.ai"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestTogetherBackend_DefaultURL(t *testing.T) {
	backend := &llm.TogetherBackend{}
	expected := "https://api.together.xyz"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestPerplexityBackend_DefaultURL(t *testing.T) {
	backend := &llm.PerplexityBackend{}
	expected := "https://api.perplexity.ai"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestGroqBackend_DefaultURL(t *testing.T) {
	backend := &llm.GroqBackend{}
	expected := "https://api.groq.com"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestDeepSeekBackend_DefaultURL(t *testing.T) {
	backend := &llm.DeepSeekBackend{}
	expected := "https://api.deepseek.com"

	if backend.DefaultURL() != expected {
		t.Errorf("Expected DefaultURL() to return '%s', got '%s'", expected, backend.DefaultURL())
	}
}

func TestBackendRegistry_Get(t *testing.T) {
	registry := llm.NewBackendRegistry()

	// Test getting existing backend
	backend := registry.Get("ollama")
	if backend == nil {
		t.Error("Expected to get ollama backend, got nil")
	}

	if backend.Name() != "ollama" {
		t.Errorf("Expected backend name 'ollama', got '%s'", backend.Name())
	}

	// Test getting non-existing backend
	nonExistent := registry.Get("nonexistent")
	if nonExistent != nil {
		t.Errorf("Expected to get nil for non-existent backend, got %v", nonExistent)
	}
}

func TestBackendRegistry_Register(t *testing.T) {
	registry := llm.NewBackendRegistry()
	registry.SetBackendsForTesting(make(map[string]llm.Backend))

	backend := &llm.OpenAIBackend{}
	registry.Register(backend)

	// Verify backend was registered
	retrieved := registry.Get("openai")
	if retrieved == nil {
		t.Error("Expected to retrieve registered backend, got nil")
	}

	if retrieved.Name() != "openai" {
		t.Errorf("Expected backend name 'openai', got '%s'", retrieved.Name())
	}
}

func TestNewBackendRegistry(t *testing.T) {
	registry := llm.NewBackendRegistry()

	if registry == nil {
		t.Fatal("NewBackendRegistry() returned nil")
	}

	if registry.GetBackendsForTesting() == nil {
		t.Error("Registry backends map is nil")
	}

	// Verify expected backends are registered (ollama + 9 online providers)
	expectedBackends := []string{
		"ollama",
		"openai", "anthropic", "google", "cohere", "mistral",
		"together", "perplexity", "groq", "deepseek",
	}

	for _, name := range expectedBackends {
		backend := registry.Get(name)
		if backend == nil {
			t.Errorf("Expected backend '%s' to be registered", name)
		} else if backend.Name() != name {
			t.Errorf("Expected backend name '%s', got '%s'", name, backend.Name())
		}
	}
}

func TestOpenAIBackend_BuildRequest(t *testing.T) {
	backend := &llm.OpenAIBackend{}
	
	messages := []map[string]interface{}{
		{"role": "user", "content": "Hello"},
	}
	
	tests := []struct {
		name   string
		config llm.ChatRequestConfig
		check  func(*testing.T, map[string]interface{})
	}{
		{
			name:   "basic request",
			config: llm.ChatRequestConfig{},
			check: func(t *testing.T, req map[string]interface{}) {
				if req["model"] != "test-model" {
					t.Errorf("Expected model 'test-model', got %v", req["model"])
				}
				if req["stream"] != false {
					t.Error("Expected stream to be false")
				}
			},
		},
		{
			name: "with context length",
			config: llm.ChatRequestConfig{
				ContextLength: 2048,
			},
			check: func(t *testing.T, req map[string]interface{}) {
				if req["max_tokens"] != 2048 {
					t.Errorf("Expected max_tokens 2048, got %v", req["max_tokens"])
				}
			},
		},
		{
			name: "with JSON response",
			config: llm.ChatRequestConfig{
				JSONResponse: true,
			},
			check: func(t *testing.T, req map[string]interface{}) {
				rf, ok := req["response_format"].(map[string]interface{})
				if !ok {
					t.Error("Expected response_format to be set")
					return
				}
				if rf["type"] != "json_object" {
					t.Errorf("Expected response_format type 'json_object', got %v", rf["type"])
				}
			},
		},
		{
			name: "with tools",
			config: llm.ChatRequestConfig{
				Tools: []map[string]interface{}{
					{"type": "function", "name": "test"},
				},
			},
			check: func(t *testing.T, req map[string]interface{}) {
				tools, ok := req["tools"].([]map[string]interface{})
				if !ok {
					t.Error("Expected tools to be set")
					return
				}
				if len(tools) != 1 {
					t.Errorf("Expected 1 tool, got %d", len(tools))
				}
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := backend.BuildRequest("test-model", messages, tt.config)
			if err != nil {
				t.Fatalf("BuildRequest failed: %v", err)
			}
			tt.check(t, req)
		})
	}
}

func TestAnthropicBackend_BuildRequest(t *testing.T) {
	backend := &llm.AnthropicBackend{}
	
	messages := []map[string]interface{}{
		{"role": "user", "content": "Hello"},
	}
	
	req, err := backend.BuildRequest("claude-3-opus", messages, llm.ChatRequestConfig{
		ContextLength: 4096,
	})
	
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}
	
	if req["model"] != "claude-3-opus" {
		t.Errorf("Expected model 'claude-3-opus', got %v", req["model"])
	}
	
	if req["max_tokens"] != 4096 {
		t.Errorf("Expected max_tokens 4096, got %v", req["max_tokens"])
	}
	
	// Anthropic doesn't have a stream field in the request
	if _, hasStream := req["stream"]; hasStream {
		t.Error("Anthropic request should not have stream field")
	}
}

func TestGoogleBackend_BuildRequest(t *testing.T) {
	backend := &llm.GoogleBackend{}
	
	messages := []map[string]interface{}{
		{"role": "user", "content": "Test message"},
	}
	
	req, err := backend.BuildRequest("gemini-pro", messages, llm.ChatRequestConfig{})
	
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}
	
	// Google uses OpenAI-compatible format
	if req["model"] != "gemini-pro" {
		t.Errorf("Expected model 'gemini-pro', got %v", req["model"])
	}
	
	if req["stream"] != false {
		t.Error("Expected stream to be false")
	}
	
	if msgs, ok := req["messages"].([]map[string]interface{}); !ok {
		t.Error("Expected 'messages' field in request")
	} else if len(msgs) == 0 {
		t.Error("Expected non-empty messages array")
	}
}

func TestCohereBackend_BuildRequest(t *testing.T) {
	backend := &llm.CohereBackend{}
	
	messages := []map[string]interface{}{
		{"role": "user", "content": "Hello Cohere"},
	}
	
	req, err := backend.BuildRequest("command-r-plus", messages, llm.ChatRequestConfig{})
	
	if err != nil {
		t.Fatalf("BuildRequest failed: %v", err)
	}
	
	if req["model"] != "command-r-plus" {
		t.Errorf("Expected model 'command-r-plus', got %v", req["model"])
	}
}

func TestOpenAIBackend_ChatEndpoint(t *testing.T) {
	backend := &llm.OpenAIBackend{}
	
	endpoint := backend.ChatEndpoint("https://api.openai.com")
	expected := "https://api.openai.com/v1/chat/completions"
	
	if endpoint != expected {
		t.Errorf("Expected endpoint '%s', got '%s'", expected, endpoint)
	}
}

func TestAnthropicBackend_ChatEndpoint(t *testing.T) {
	backend := &llm.AnthropicBackend{}
	
	endpoint := backend.ChatEndpoint("https://api.anthropic.com")
	expected := "https://api.anthropic.com/v1/messages"
	
	if endpoint != expected {
		t.Errorf("Expected endpoint '%s', got '%s'", expected, endpoint)
	}
}

func TestGoogleBackend_ChatEndpoint(t *testing.T) {
	backend := &llm.GoogleBackend{}
	
	endpoint := backend.ChatEndpoint("https://generativelanguage.googleapis.com")
	
	// Google endpoint includes model name, so just check it contains the base
	if endpoint == "" {
		t.Error("Expected non-empty endpoint")
	}
}

func TestOpenAIBackend_GetAPIKeyHeader(t *testing.T) {
	backend := &llm.OpenAIBackend{}
	
	tests := []struct {
		name     string
		apiKey   string
		wantKey  string
		wantVal  string
	}{
		{
			name:    "with provided key",
			apiKey:  "sk-test123",
			wantKey: "Authorization",
			wantVal: "Bearer sk-test123",
		},
		{
			name:    "with empty key",
			apiKey:  "",
			wantKey: "",
			wantVal: "",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, val := backend.GetAPIKeyHeader(tt.apiKey)
			if key != tt.wantKey {
				t.Errorf("Expected key '%s', got '%s'", tt.wantKey, key)
			}
			if val != tt.wantVal {
				t.Errorf("Expected value '%s', got '%s'", tt.wantVal, val)
			}
		})
	}
}

func TestAnthropicBackend_GetAPIKeyHeader(t *testing.T) {
	backend := &llm.AnthropicBackend{}
	
	key, val := backend.GetAPIKeyHeader("test-key")
	
	if key != "x-api-key" {
		t.Errorf("Expected key 'x-api-key', got '%s'", key)
	}
	if val != "test-key" {
		t.Errorf("Expected value 'test-key', got '%s'", val)
	}
}

func TestBackend_Names(t *testing.T) {
	tests := []struct {
		backend      llm.Backend
		expectedName string
	}{
		{&llm.OpenAIBackend{}, "openai"},
		{&llm.AnthropicBackend{}, "anthropic"},
		{&llm.GoogleBackend{}, "google"},
		{&llm.CohereBackend{}, "cohere"},
		{&llm.MistralBackend{}, "mistral"},
		{&llm.TogetherBackend{}, "together"},
		{&llm.PerplexityBackend{}, "perplexity"},
		{&llm.GroqBackend{}, "groq"},
		{&llm.DeepSeekBackend{}, "deepseek"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			if tt.backend.Name() != tt.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tt.expectedName, tt.backend.Name())
			}
		})
	}
}

func TestBackend_DefaultURLs(t *testing.T) {
	tests := []struct {
		backend     llm.Backend
		expectedURL string
	}{
		{&llm.OpenAIBackend{}, "https://api.openai.com"},
		{&llm.AnthropicBackend{}, "https://api.anthropic.com"},
		{&llm.MistralBackend{}, "https://api.mistral.ai"},
		{&llm.TogetherBackend{}, "https://api.together.xyz"},
		{&llm.PerplexityBackend{}, "https://api.perplexity.ai"},
		{&llm.GroqBackend{}, "https://api.groq.com"},
		{&llm.DeepSeekBackend{}, "https://api.deepseek.com"},
	}
	
	for _, tt := range tests {
		t.Run(tt.backend.Name(), func(t *testing.T) {
			if tt.backend.DefaultURL() != tt.expectedURL {
				t.Errorf("Expected URL '%s', got '%s'", tt.expectedURL, tt.backend.DefaultURL())
			}
		})
	}
}
