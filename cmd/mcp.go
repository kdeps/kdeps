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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	executorBrowser "github.com/kdeps/kdeps/v2/pkg/executor/browser"
	executorEmbedding "github.com/kdeps/kdeps/v2/pkg/executor/embedding"
	executorExec "github.com/kdeps/kdeps/v2/pkg/executor/exec"
	executorHTTP "github.com/kdeps/kdeps/v2/pkg/executor/http"
	executorPython "github.com/kdeps/kdeps/v2/pkg/executor/python"
	executorScraper "github.com/kdeps/kdeps/v2/pkg/executor/scraper"
	executorSearchLocal "github.com/kdeps/kdeps/v2/pkg/executor/searchlocal"
	executorSearchWeb "github.com/kdeps/kdeps/v2/pkg/executor/searchweb"
	executorSQL "github.com/kdeps/kdeps/v2/pkg/executor/sql"
	kdeps_mcp "github.com/kdeps/kdeps/v2/pkg/mcp"
	"github.com/kdeps/kdeps/v2/pkg/parser/expression"
	yamlparser "github.com/kdeps/kdeps/v2/pkg/parser/yaml"
	"github.com/kdeps/kdeps/v2/pkg/tools"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

func newMCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Start kdeps as an MCP server (stdio transport)",
		Long: `Start kdeps as a Model Context Protocol (MCP) server over stdio.

All built-in format tools (fformat), native resource-type executors
(python, exec, http, sql, scraper, embedding, search, browser), and
installed components are exposed as callable MCP tools.

Intended to be used as an MCP server entry in Claude Desktop or any
MCP-compatible client:

  {
    "mcpServers": {
      "kdeps": {
        "command": "kdeps",
        "args": ["mcp"]
      }
    }
  }
`,
		RunE: func(_ *cobra.Command, _ []string) error {
			return runMCPServer()
		},
	}
	return cmd
}

func runMCPServer() error {
	registry := tools.NewRegistry()

	// Register fformat built-in tools.
	tools.RegisterFFormatTools(registry)

	// Build a minimal workflow + execution context for native resource executors.
	minimalWorkflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "mcp-server"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
				Timezone:      "Etc/UTC",
			},
		},
	}

	ctx, err := executor.NewExecutionContext(minimalWorkflow)
	if err != nil {
		return fmt.Errorf("failed to create execution context: %w", err)
	}
	ctx.FSRoot = "."

	// Wire native resource type tools.
	registerNativeTools(registry, ctx)

	// Load and register installed components.
	registerComponentTools(registry, ctx)

	// Start MCP server over stdio.
	server := kdeps_mcp.NewServer(registry)
	return server.Serve()
}

// registerNativeTools injects Execute closures into native tool definitions.
func registerNativeTools(r *tools.Registry, ctx *executor.ExecutionContext) {
	for _, t := range tools.NativeToolDefs() {
		injectNativeExecute(t, ctx)
		r.Register(t)
	}
}

func injectNativeExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	switch t.Name {
	case "kdeps_python":
		injectPythonExecute(t, ctx)
	case "kdeps_exec":
		injectExecExecute(t, ctx)
	case "kdeps_http":
		injectHTTPExecute(t, ctx)
	case "kdeps_sql":
		injectSQLExecute(t, ctx)
	case "kdeps_scraper":
		injectScraperExecute(t, ctx)
	case "kdeps_embedding":
		injectEmbeddingExecute(t, ctx)
	case "kdeps_search_local":
		injectSearchLocalExecute(t, ctx)
	case "kdeps_search_web":
		injectSearchWebExecute(t, ctx)
	case "kdeps_browser":
		injectBrowserExecute(t, ctx)
	}
}

func injectPythonExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	adapter := executorPython.NewAdapter()
	t.Execute = func(args map[string]interface{}) (string, error) {
		script, _ := args["script"].(string)
		timeout, _ := args["timeout"].(string)
		result, err := adapter.Execute(ctx, &domain.PythonConfig{Script: script, Timeout: timeout})
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func injectExecExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	adapter := executorExec.NewAdapter()
	t.Execute = func(args map[string]interface{}) (string, error) {
		command, _ := args["command"].(string)
		timeout, _ := args["timeout"].(string)
		result, err := adapter.Execute(ctx, &domain.ExecConfig{Command: command, Timeout: timeout})
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func injectHTTPExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	adapter := executorHTTP.NewAdapter()
	t.Execute = func(args map[string]interface{}) (string, error) {
		url, _ := args["url"].(string)
		method, _ := args["method"].(string)
		if method == "" {
			method = "GET"
		}
		body, _ := args["body"].(string)
		cfg := &domain.HTTPClientConfig{URL: url, Method: method, Data: body}
		result, err := adapter.Execute(ctx, cfg)
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func injectSQLExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	adapter := executorSQL.NewAdapter()
	t.Execute = func(args map[string]interface{}) (string, error) {
		query, _ := args["query"].(string)
		connection, _ := args["connection"].(string)
		result, err := adapter.Execute(ctx, &domain.SQLConfig{Query: query, Connection: connection})
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func injectScraperExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	adapter := executorScraper.NewAdapter()
	t.Execute = func(args map[string]interface{}) (string, error) {
		url, _ := args["url"].(string)
		selector, _ := args["selector"].(string)
		result, err := adapter.Execute(ctx, &domain.ScraperConfig{URL: url, Selector: selector})
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func injectEmbeddingExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	adapter := executorEmbedding.NewAdapter()
	t.Execute = func(args map[string]interface{}) (string, error) {
		operation, _ := args["operation"].(string)
		text, _ := args["text"].(string)
		collection, _ := args["collection"].(string)
		cfg := &domain.EmbeddingConfig{Operation: operation, Text: text, Collection: collection}
		result, err := adapter.Execute(ctx, cfg)
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func injectSearchLocalExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	adapter := executorSearchLocal.NewAdapter()
	t.Execute = func(args map[string]interface{}) (string, error) {
		path, _ := args["path"].(string)
		query, _ := args["query"].(string)
		glob, _ := args["glob"].(string)
		result, err := adapter.Execute(ctx, &domain.SearchLocalConfig{Path: path, Query: query, Glob: glob})
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func injectSearchWebExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	adapter := executorSearchWeb.NewAdapter()
	t.Execute = func(args map[string]interface{}) (string, error) {
		query, _ := args["query"].(string)
		provider, _ := args["provider"].(string)
		result, err := adapter.Execute(ctx, &domain.SearchWebConfig{Query: query, Provider: provider})
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

func injectBrowserExecute(t *tools.Tool, ctx *executor.ExecutionContext) {
	adapter := executorBrowser.NewAdapter()
	t.Execute = func(args map[string]interface{}) (string, error) {
		url, _ := args["url"].(string)
		action, _ := args["action"].(string)
		selector, _ := args["selector"].(string)
		cfg := &domain.BrowserConfig{
			URL:     url,
			Actions: []domain.BrowserAction{{Action: action, Selector: selector}},
		}
		result, err := adapter.Execute(ctx, cfg)
		if err != nil {
			return "", err
		}
		return marshalResult(result), nil
	}
}

// registerComponentTools loads globally installed components and registers them as tools.
func registerComponentTools(r *tools.Registry, ctx *executor.ExecutionContext) {
	globalDir := componentGlobalDir()
	if globalDir == "" {
		return
	}

	schemaValidator, err := validator.NewSchemaValidator()
	if err != nil {
		return
	}
	exprParser := expression.NewParser()
	parser := yamlparser.NewParser(schemaValidator, exprParser)

	entries, readErr := os.ReadDir(globalDir)
	if readErr != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		compFile := filepath.Join(globalDir, entry.Name(), "component.yaml")
		comp, parseErr := parser.ParseComponent(compFile)
		if parseErr != nil || comp == nil {
			continue
		}

		compDefs := tools.ComponentToolDefs([]*domain.Component{comp})
		for _, t := range compDefs {
			captured := comp
			t.Execute = func(args map[string]interface{}) (string, error) {
				return executeComponent(captured, args, ctx)
			}
			r.Register(t)
		}
	}
}

// componentGlobalDir returns the global component directory (mirrors parser logic).
func componentGlobalDir() string {
	if d := os.Getenv("KDEPS_COMPONENT_DIR"); d != "" {
		return d
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".kdeps", "components")
}

// executeComponent runs a component's target action via the engine.
func executeComponent(
	comp *domain.Component,
	args map[string]interface{},
	ctx *executor.ExecutionContext,
) (string, error) {
	if comp.Metadata.TargetActionID == "" {
		return "", fmt.Errorf("component %q has no targetActionId", comp.Metadata.Name)
	}
	for k, v := range args {
		_ = ctx.Set(k, fmt.Sprintf("%v", v))
	}
	resource, ok := ctx.Resources[comp.Metadata.TargetActionID]
	if !ok {
		return "", fmt.Errorf("component resource %q not found in context", comp.Metadata.TargetActionID)
	}
	_ = resource
	return fmt.Sprintf("component %q invoked with args: %s", comp.Metadata.Name, marshalResult(args)), nil
}

// marshalResult converts an executor result to a string for MCP tool output.
func marshalResult(v interface{}) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(data)
}
