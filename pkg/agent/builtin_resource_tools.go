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

package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	execHTTP "github.com/kdeps/kdeps/v2/pkg/executor/http"
	execSearch "github.com/kdeps/kdeps/v2/pkg/executor/searchlocal"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

// registerResourceTools registers all resource-based tools (HTTP, SearchLocal, etc.)
// that an LLM agent can use without needing a workflow YAML file.
func registerResourceTools(ctx context.Context, reg *kdepstools.Registry) {
	registerHTTPTool(ctx, reg)
	registerSearchLocalTool(ctx, reg)
}

// registerHTTPTool registers an HTTP request tool (http_request).
func registerHTTPTool(_ context.Context, reg *kdepstools.Registry) {
	exec := execHTTP.NewExecutor()

	reg.Register(&kdepstools.Tool{
		Name:        "http_request",
		Description: "Make an HTTP request to a URL. Returns response status, headers, and body. Use for calling APIs, fetching web content, or interacting with external services. Requires: url. Optional: method (default GET), headers, data (JSON body), timeout.",
		Parameters: map[string]domain.ToolParam{
			"url":     {Type: "string", Description: "The URL to request", Required: true},
			"method":  {Type: "string", Description: "HTTP method: GET, POST, PUT, DELETE, PATCH. Default: GET"},
			"headers": {Type: "object", Description: "HTTP headers as key-value pairs"},
			"data":    {Type: "object", Description: "Request body as JSON (for POST/PUT/PATCH)"},
			"timeout": {Type: "string", Description: "Request timeout, e.g. '30s'. Default: 30s"},
		},
		Execute: func(args map[string]any) (string, error) {
			config := &domain.HTTPClientConfig{}
			if v, ok := args["url"].(string); ok {
				config.URL = v
			}
			if v, ok := args["method"].(string); ok {
				config.Method = v
			}
			if v, ok := args["headers"].(map[string]any); ok {
				config.Headers = make(map[string]string)
				for k, val := range v {
					config.Headers[k] = fmt.Sprint(val)
				}
			}
			if v, ok := args["data"]; ok {
				config.Data = v
			}
			if v, ok := args["timeout"].(string); ok {
				config.Timeout = v
			}

			result, err := exec.Execute(nil, config)
			if err != nil {
				return "", err
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return string(out), nil
		},
	})
}

// registerSearchLocalTool registers a local file search tool (search_local).
func registerSearchLocalTool(_ context.Context, reg *kdepstools.Registry) {
	exec := execSearch.NewExecutor()

	reg.Register(&kdepstools.Tool{
		Name:        "search_local",
		Description: "Search for text patterns in local files using ripgrep. Returns matching files with line numbers and content. Use for finding usages, patterns, or strings across the codebase. Requires: path (directory to search), query (search term). Optional: glob (file pattern).",
		Parameters: map[string]domain.ToolParam{
			"path":  {Type: "string", Description: "Directory to search in (absolute path)", Required: true},
			"query": {Type: "string", Description: "Search term or regex pattern", Required: true},
			"glob":  {Type: "string", Description: "File glob filter, e.g. '*.go', '*.py'"},
		},
		Execute: func(args map[string]any) (string, error) {
			config := &domain.SearchLocalConfig{}
			if v, ok := args["path"].(string); ok {
				config.Path = v
			}
			if v, ok := args["query"].(string); ok {
				config.Query = v
			}
			if v, ok := args["glob"].(string); ok {
				config.Glob = v
			}

			result, err := exec.Execute(nil, config)
			if err != nil {
				return "", err
			}
			out, _ := json.MarshalIndent(result, "", "  ")
			return string(out), nil
		},
	})
}
