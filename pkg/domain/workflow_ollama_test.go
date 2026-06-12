package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveInstallOllama(t *testing.T) {
	installOllama := true
	skipOllama := false

	tests := []struct {
		name       string
		workflow   *Workflow
		envBackend string
		envRouter  string
		envModels  string
		expected   bool
	}{
		{
			name: "explicit install",
			workflow: &Workflow{
				Settings: WorkflowSettings{
					AgentSettings: AgentSettings{InstallOllama: &installOllama},
				},
			},
			expected: true,
		},
		{
			name: "explicit skip overrides chat resources",
			workflow: &Workflow{
				Settings: WorkflowSettings{
					AgentSettings: AgentSettings{InstallOllama: &skipOllama},
				},
				Resources: []*Resource{{Chat: &ChatConfig{}}},
			},
			expected: false,
		},
		{
			name: "chat resources default backend is file, not ollama",
			workflow: &Workflow{
				Resources: []*Resource{{Chat: &ChatConfig{}}},
			},
			expected: false,
		},
		{
			name: "chat resources ollama backend",
			workflow: &Workflow{
				Resources: []*Resource{{Chat: &ChatConfig{}}},
			},
			envBackend: "ollama",
			expected:   true,
		},
		{
			name: "chat resources online backend",
			workflow: &Workflow{
				Resources: []*Resource{{Chat: &ChatConfig{}}},
			},
			envBackend: "openai",
			expected:   false,
		},
		{
			name:      "router config ollama",
			workflow:  &Workflow{},
			envRouter: `{"backend":"ollama","models":["llama2:7b"]}`,
			expected:  true,
		},
		{
			name:      "models env alone does not trigger ollama install",
			workflow:  &Workflow{},
			envModels: "llama3.2:1b",
			expected:  false,
		},
		{
			name: "no signals",
			workflow: &Workflow{
				Resources: []*Resource{{HTTPClient: &HTTPClientConfig{URL: "https://example.com"}}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KDEPS_DEFAULT_BACKEND", tt.envBackend)
			t.Setenv("KDEPS_LLM_ROUTER", tt.envRouter)
			t.Setenv("KDEPS_LLM_MODELS", tt.envModels)

			assert.Equal(t, tt.expected, ResolveInstallOllama(tt.workflow))
		})
	}
}

func TestNeedsOllamaAtRuntime(t *testing.T) {
	tests := []struct {
		name       string
		envBackend string
		workflow   *Workflow
		expected   bool
	}{
		{
			name:       "no resources",
			envBackend: "",
			workflow:   &Workflow{Resources: []*Resource{}},
			expected:   false,
		},
		{
			name:       "ollama backend via env",
			envBackend: "ollama",
			workflow: &Workflow{
				Resources: []*Resource{{Chat: &ChatConfig{}}},
			},
			expected: true,
		},
		{
			name:       "empty backend defaults to file, not ollama",
			envBackend: "",
			workflow: &Workflow{
				Resources: []*Resource{{Chat: &ChatConfig{}}},
			},
			expected: false,
		},
		{
			name:       "non-ollama backend via env",
			envBackend: "openai",
			workflow: &Workflow{
				Resources: []*Resource{{Chat: &ChatConfig{}}},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("KDEPS_DEFAULT_BACKEND", tt.envBackend)
			assert.Equal(t, tt.expected, NeedsOllamaAtRuntime(tt.workflow))
		})
	}
}

func TestHasChatResources(t *testing.T) {
	tests := []struct {
		name     string
		workflow *Workflow
		expected bool
	}{
		{
			name:     "nil chat",
			workflow: &Workflow{Resources: []*Resource{{Chat: nil}}},
			expected: false,
		},
		{
			name:     "with chat",
			workflow: &Workflow{Resources: []*Resource{{Chat: &ChatConfig{}}}},
			expected: true,
		},
		{
			name:     "no resources",
			workflow: &Workflow{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, HasChatResources(tt.workflow))
		})
	}
}

func TestChatModels(t *testing.T) {
	tests := []struct {
		name     string
		workflow *Workflow
		expected []string
	}{
		{
			name:     "no chat",
			workflow: &Workflow{},
			expected: nil,
		},
		{
			name: "literal models",
			workflow: &Workflow{
				Resources: []*Resource{
					{Chat: &ChatConfig{Model: "llama3.2:1b"}},
					{Chat: &ChatConfig{Model: "gpt-4o"}},
				},
			},
			expected: []string{"llama3.2:1b", "gpt-4o"},
		},
		{
			name: "skips expression and router models",
			workflow: &Workflow{
				Resources: []*Resource{
					{Chat: &ChatConfig{Model: "{{ input('x') }}"}},
					{Chat: &ChatConfig{Model: "router"}},
					{Chat: &ChatConfig{Model: "llama3.2:3b"}},
				},
			},
			expected: []string{"llama3.2:3b"},
		},
		{
			name: "non-chat resources ignored",
			workflow: &Workflow{
				Resources: []*Resource{
					{HTTPClient: &HTTPClientConfig{URL: "https://example.com"}},
					{Chat: &ChatConfig{Model: "ministral:3b"}},
				},
			},
			expected: []string{"ministral:3b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, ChatModels(tt.workflow))
		})
	}
}
