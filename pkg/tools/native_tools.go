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

package tools

import "github.com/kdeps/kdeps/v2/pkg/domain"

func requiredStringParam(description string) domain.ToolParam {
	return domain.ToolParam{
		Type:        "string",
		Description: description,
		Required:    true,
	}
}

func optionalStringParam(description string) domain.ToolParam {
	return domain.ToolParam{
		Type:        "string",
		Description: description,
		Required:    false,
	}
}

func requiredStringEnumParam(description string, enum []string) domain.ToolParam {
	return domain.ToolParam{
		Type:        "string",
		Description: description,
		Required:    true,
		Enum:        enum,
	}
}

func optionalStringEnumParam(description string, enum []string) domain.ToolParam {
	return domain.ToolParam{
		Type:        "string",
		Description: description,
		Required:    false,
		Enum:        enum,
	}
}

func optionalTimeoutParam() domain.ToolParam {
	return optionalStringParam("Execution timeout (e.g. '30s'). Optional.")
}

// NativeToolDefs returns the tool metadata (name, description, parameters) for
// all built-in kdeps resource types. Execute functions are not set here; callers
// (e.g. the MCP server command) inject them after creating executor adapters.
func NativeToolDefs() []*Tool {
	return append(
		append(nativeComputeToolDefs(), nativeDataToolDefs()...),
		nativeWebToolDefs()...,
	)
}

func nativeComputeToolDefs() []*Tool {
	return []*Tool{
		{
			Name:        "kdeps_python",
			Description: "Execute a Python script and return its stdout.",
			Parameters: map[string]domain.ToolParam{
				"script":  requiredStringParam("The Python script source to execute."),
				"timeout": optionalTimeoutParam(),
			},
		},
		{
			Name:        "kdeps_exec",
			Description: "Execute a shell command and return its output.",
			Parameters: map[string]domain.ToolParam{
				"command": requiredStringParam("The shell command to run."),
				"timeout": optionalTimeoutParam(),
			},
		},
		{
			Name:        "kdeps_sql",
			Description: "Execute a SQL query against a database connection and return results as JSON.",
			Parameters: map[string]domain.ToolParam{
				"query": requiredStringParam("The SQL query to execute."),
				"connection": optionalStringParam(
					"Database connection string (DSN). Optional if a default connection is configured.",
				),
			},
		},
	}
}

func nativeDataToolDefs() []*Tool {
	return []*Tool{
		{
			Name:        "kdeps_embedding",
			Description: "Perform an embedding operation (index, search, upsert, delete) on a vector store.",
			Parameters: map[string]domain.ToolParam{
				"operation": requiredStringEnumParam(
					"The operation to perform.",
					[]string{"index", "search", "upsert", "delete"},
				),
				"text":       optionalStringParam("The text to embed or search for."),
				"collection": optionalStringParam("Vector store collection name."),
			},
		},
		{
			Name:        "kdeps_search_local",
			Description: "Search the local filesystem for files matching a query or glob pattern.",
			Parameters: map[string]domain.ToolParam{
				"path":  requiredStringParam("Root directory to search from."),
				"query": optionalStringParam("Text query to match against file contents."),
				"glob":  optionalStringParam("Glob pattern to match filenames (e.g. '**/*.go')."),
			},
		},
	}
}

func nativeWebToolDefs() []*Tool {
	return []*Tool{
		{
			Name:        "kdeps_http",
			Description: "Make an HTTP request and return the response body.",
			Parameters: map[string]domain.ToolParam{
				"url": requiredStringParam("The request URL."),
				"method": optionalStringEnumParam(
					"HTTP method (GET, POST, PUT, DELETE, PATCH). Defaults to GET.",
					[]string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"},
				),
				"body": optionalStringParam("Request body (for POST/PUT/PATCH). Optional."),
			},
		},
		{
			Name:        "kdeps_scraper",
			Description: "Scrape a web page and return its text content.",
			Parameters: map[string]domain.ToolParam{
				"url":      requiredStringParam("The URL to scrape."),
				"selector": optionalStringParam("Optional CSS selector to extract a specific element."),
			},
		},
		{
			Name:        "kdeps_search_web",
			Description: "Search the web using a configurable search provider and return results.",
			Parameters: map[string]domain.ToolParam{
				"query": requiredStringParam("The search query."),
				"provider": optionalStringEnumParam(
					"Search provider: ddg (default), brave, bing, tavily.",
					[]string{"ddg", "brave", "bing", "tavily"},
				),
				"max_results": {
					Type:        "integer",
					Description: "Maximum number of results to return. Default 5.",
					Required:    false,
				},
			},
		},
		{
			Name:        "kdeps_browser",
			Description: "Control a headless browser to interact with web pages.",
			Parameters: map[string]domain.ToolParam{
				"url": requiredStringParam("The URL to navigate to."),
				"action": optionalStringEnumParam(
					"Browser action to perform (navigate, click, screenshot, extract). Default: navigate.",
					[]string{"navigate", "click", "screenshot", "extract"},
				),
				"selector": optionalStringParam("CSS selector for click/extract actions."),
			},
		},
	}
}
