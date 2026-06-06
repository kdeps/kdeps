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

// Package agent implements the kdeps agent loop: a stateless LLM-driven execution
// mode where every workflow resource, component, and agency is a callable tool.
package agent

import (
	"context"
	"fmt"
	"os"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

// Config holds agent loop configuration.
type Config struct {
	// Model is the LLM model name (default: KDEPS_AGENT_MODEL env or "llama3.2").
	Model string
	// Backend is the LLM backend (default: KDEPS_AGENT_BACKEND env or "ollama").
	Backend string
	// BaseURL is the LLM API base URL (default: KDEPS_AGENT_BASE_URL env or "").
	BaseURL string
	// SystemPrompt is injected as the first system message in every conversation.
	SystemPrompt string
	// Role is the chat role field (default: "user").
	Role string
}

// Loop drives a single agent conversation using the kdeps engine as the executor.
// All registered tools are wired into a synthetic chat resource so the engine's
// existing handleToolCalls path dispatches them without any additional plumbing.
type Loop struct {
	engine   *executor.Engine
	registry *tools.Registry
	workflow *domain.Workflow
	config   Config
}

// New creates a new Loop. cfg fields with zero values fall back to env vars and
// then to sensible defaults (model: "llama3.2", backend: "ollama", role: "user").
func New(eng *executor.Engine, workflow *domain.Workflow, reg *tools.Registry, cfg Config) *Loop {
	return &Loop{
		engine:   eng,
		registry: reg,
		workflow: workflow,
		config:   applyConfigDefaults(cfg),
	}
}

func applyConfigDefaults(cfg Config) Config {
	if cfg.Model == "" {
		cfg.Model = envOrDefault("KDEPS_AGENT_MODEL", "llama3.2")
	}
	if cfg.Backend == "" {
		cfg.Backend = envOrDefault("KDEPS_AGENT_BACKEND", "ollama")
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = os.Getenv("KDEPS_AGENT_BASE_URL")
	}
	if cfg.Role == "" {
		cfg.Role = "user"
	}
	return cfg
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Run executes one agent turn: the input is sent as the user prompt to a synthetic
// single-chat-resource workflow. All registry tools are attached so the engine's
// existing tool-call loop can dispatch them. Returns the final LLM text response.
func (l *Loop) Run(_ context.Context, input string) (string, error) {
	const actionID = "agent_loop_chat"
	single := l.buildSyntheticWorkflow(actionID, l.buildChatConfig(input))

	result, err := l.engine.Execute(single, nil)
	if err != nil {
		return "", fmt.Errorf("agent loop: %w", err)
	}
	return formatLoopResult(result), nil
}

func (l *Loop) buildChatConfig(input string) *domain.ChatConfig {
	chatCfg := &domain.ChatConfig{
		Model:   l.config.Model,
		Backend: l.config.Backend,
		BaseURL: l.config.BaseURL,
		Role:    l.config.Role,
		Prompt:  input,
		Tools:   l.registry.ToLLMTools(),
	}
	if l.config.SystemPrompt == "" {
		return chatCfg
	}
	chatCfg.Scenario = []domain.ScenarioItem{
		{Role: "system", Prompt: l.config.SystemPrompt},
	}
	return chatCfg
}

func (l *Loop) buildSyntheticWorkflow(actionID string, chatCfg *domain.ChatConfig) *domain.Workflow {
	return &domain.Workflow{
		APIVersion: l.workflow.APIVersion,
		Kind:       l.workflow.Kind,
		Metadata: domain.WorkflowMetadata{
			Name:           l.workflow.Metadata.Name,
			Version:        l.workflow.Metadata.Version,
			TargetActionID: actionID,
		},
		Settings:   l.workflow.Settings,
		Components: l.workflow.Components,
		Resources: []*domain.Resource{{
			ActionID: actionID,
			Name:     "agent_loop",
			Chat:     chatCfg,
		}},
	}
}

func formatLoopResult(result interface{}) string {
	if result == nil {
		return ""
	}
	if s, ok := result.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", result)
}
