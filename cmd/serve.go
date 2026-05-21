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

//go:build !js

package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/agent"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

// serveFlags holds command-line flags for the serve subcommand.
type serveFlags struct {
	Model        string
	Backend      string
	BaseURL      string
	SystemPrompt string
	Debug        bool
}

func newServeCmd() *cobra.Command {
	flags := &serveFlags{}

	cmd := &cobra.Command{
		Use:   "serve [workflow.yaml | agency.yaml]",
		Short: "Run a workflow or agency in agent mode (interactive LLM loop)",
		Long: `Run a KDeps workflow or agency in agent mode.

Every resource, component, and agency defined in the workflow is auto-registered
as a callable LLM tool. The agent uses the kdeps engine as its tool executor so
all existing resource types (http, python, exec, sql, chat, ...) work unchanged.

The session runs as an interactive REPL on stdin/stdout.

Examples:
  # Start agent mode with a workflow
  kdeps serve workflow.yaml

  # Override the model
  kdeps serve workflow.yaml --model llama3.2

  # Provide a system prompt
  kdeps serve workflow.yaml --system "You are a helpful assistant."

Environment variables (override defaults):
  KDEPS_AGENT_MODEL      LLM model name (default: llama3.2)
  KDEPS_AGENT_BACKEND    LLM backend (default: ollama)
  KDEPS_AGENT_BASE_URL   LLM API base URL`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			debugMode, _ := cmd.Flags().GetBool("debug")
			flags.Debug = debugMode
			return runServeCmd(args[0], flags)
		},
	}

	cmd.Flags().StringVar(&flags.Model, "model", "", "LLM model to use (default: KDEPS_AGENT_MODEL env or llama3.2)")
	cmd.Flags().StringVar(&flags.Backend, "backend", "", "LLM backend (default: KDEPS_AGENT_BACKEND env or ollama)")
	cmd.Flags().StringVar(&flags.BaseURL, "base-url", "", "LLM API base URL (default: KDEPS_AGENT_BASE_URL env)")
	cmd.Flags().StringVar(
		&flags.SystemPrompt, "system", "",
		"System prompt injected at the start of every conversation",
	)

	return cmd
}

func runServeCmd(workflowPath string, flags *serveFlags) error {
	workflow, err := ParseWorkflowFile(workflowPath)
	if err != nil {
		return fmt.Errorf("serve: failed to load workflow: %w", err)
	}

	eng := setupEngine(workflow, flags.Debug)

	registry := tools.NewRegistry()

	// Register fformat built-in tools.
	tools.RegisterFFormatTools(registry)

	// Register all workflow resources as tools.
	for _, t := range tools.ResourceToolDefs(workflow, eng) {
		registry.Register(t)
	}

	// Convert components map to slice and register with execution engine.
	if len(workflow.Components) > 0 {
		comps := make([]*domain.Component, 0, len(workflow.Components))
		for _, c := range workflow.Components {
			comps = append(comps, c)
		}
		for _, t := range tools.ComponentToolDefs(comps, workflow, eng) {
			registry.Register(t)
		}
	}

	cfg := agent.Config{
		Model:        flags.Model,
		Backend:      flags.Backend,
		BaseURL:      flags.BaseURL,
		SystemPrompt: flags.SystemPrompt,
	}
	loop := agent.New(eng, workflow, registry, cfg)

	return runREPL(loop)
}

// runREPL runs a simple stdin/stdout interactive loop.
func runREPL(loop *agent.Loop) error {
	ctx := context.Background()
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Fprintln(os.Stdout, "kdeps agent mode — type your message and press Enter. Ctrl+D to exit.")
	for {
		fmt.Fprint(os.Stdout, "> ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()
		if input == "" {
			continue
		}
		resp, err := loop.Run(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			continue
		}
		fmt.Fprintln(os.Stdout, resp)
	}
	return scanner.Err()
}
