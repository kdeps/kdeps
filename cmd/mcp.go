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
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	kdeps_mcp "github.com/kdeps/kdeps/v2/pkg/mcp"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp [workflow.yaml | agency.yaml | directory]",
		Short: "Start kdeps as an MCP server (stdio transport)",
		Long: `Start kdeps as a Model Context Protocol (MCP) server over stdio.

When a workflow or agency path is provided, every resource defined in that
workflow (or across all agents in an agency) is registered as a callable MCP
tool. A directory is accepted and auto-discovers the workflow or agency file
inside it.

Without a path argument, the built-in fformat utilities and native resource
executors are exposed as tools.

Intended to be used as an MCP server entry in Claude Desktop or any
MCP-compatible client:

  {
    "mcpServers": {
      "my-agent": {
        "command": "kdeps",
        "args": ["mcp", "/path/to/my-agent"]
      }
    }
  }
`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			var path string
			if len(args) > 0 {
				path = args[0]
			}
			return runMCPServer(path)
		},
	}
	return cmd
}

func runMCPServer(path string) error {
	registry := tools.NewRegistry()
	tools.RegisterFFormatTools(registry)

	if path == "" {
		// Built-ins only — no workflow provided.
		server := kdeps_mcp.NewServer(registry)
		return server.Serve()
	}

	resolvedPath, cleanup, err := resolveMCPPath(path)
	if err != nil {
		return err
	}
	if cleanup != nil {
		defer cleanup()
	}

	if isAgencyFile(resolvedPath) {
		err = registerAgencyTools(registry, resolvedPath)
	} else {
		err = registerWorkflowTools(registry, resolvedPath, false)
	}
	if err != nil {
		return err
	}

	server := kdeps_mcp.NewServer(registry)
	return server.Serve()
}

// resolveMCPPath resolves a user-supplied path to an absolute workflow or agency file.
// Accepts a file path or a directory (uses ResolveDirectoryPath for directories).
func resolveMCPPath(path string) (string, func(), error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", nil, fmt.Errorf("mcp: invalid path %q: %w", path, err)
	}
	info, err := os.Stat(absPath)
	if err != nil {
		return "", nil, fmt.Errorf("mcp: path not found %q: %w", path, err)
	}
	if info.IsDir() {
		return ResolveDirectoryPath(absPath)
	}
	return absPath, nil, nil
}

// registerWorkflowTools parses a workflow file and registers its resources as MCP tools.
func registerWorkflowTools(r *tools.Registry, workflowPath string, debug bool) error {
	workflow, err := ParseWorkflowFile(workflowPath)
	if err != nil {
		return fmt.Errorf("mcp: failed to load workflow: %w", err)
	}

	eng := setupEngine(workflow, debug)

	for _, t := range tools.ResourceToolDefs(workflow, eng) {
		r.Register(t)
	}

	if len(workflow.Components) > 0 {
		comps := make([]*domain.Component, 0, len(workflow.Components))
		for _, c := range workflow.Components {
			comps = append(comps, c)
		}
		for _, t := range tools.ComponentToolDefs(comps, workflow, eng) {
			r.Register(t)
		}
	}
	return nil
}

// registerAgencyTools parses an agency and registers resources from all agent workflows.
func registerAgencyTools(r *tools.Registry, agencyPath string) error {
	_, agentPaths, yamlParser, err := ParseAgencyFileWithParser(agencyPath)
	if err != nil {
		return fmt.Errorf("mcp: failed to load agency: %w", err)
	}
	defer yamlParser.Cleanup()

	for _, p := range agentPaths {
		if regErr := registerWorkflowTools(r, p, false); regErr != nil {
			return regErr
		}
	}
	return nil
}
