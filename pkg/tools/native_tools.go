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
				"script": {
					Type:        "string",
					Description: "The Python script source to execute.",
					Required:    true,
				},
				"timeout": {
					Type:        "string",
					Description: "Execution timeout (e.g. '30s'). Optional.",
					Required:    false,
				},
			},
		},
		{
			Name:        "kdeps_exec",
			Description: "Execute a shell command and return its output.",
			Parameters: map[string]domain.ToolParam{
				"command": {
					Type:        "string",
					Description: "The shell command to run.",
					Required:    true,
				},
				"timeout": {
					Type:        "string",
					Description: "Execution timeout (e.g. '30s'). Optional.",
					Required:    false,
				},
			},
		},
		{
			Name:        "kdeps_sql",
			Description: "Execute a SQL query against a database connection and return results as JSON.",
			Parameters: map[string]domain.ToolParam{
				"query": {
					Type:        "string",
					Description: "The SQL query to execute.",
					Required:    true,
				},
				"connection": {
					Type:        "string",
					Description: "Database connection string (DSN). Optional if a default connection is configured.",
					Required:    false,
				},
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
				"operation": {
					Type:        "string",
					Description: "The operation to perform.",
					Required:    true,
					Enum:        []string{"index", "search", "upsert", "delete"},
				},
				"text": {
					Type:        "string",
					Description: "The text to embed or search for.",
					Required:    false,
				},
				"collection": {
					Type:        "string",
					Description: "Vector store collection name.",
					Required:    false,
				},
			},
		},
		{
			Name:        "kdeps_search_local",
			Description: "Search the local filesystem for files matching a query or glob pattern.",
			Parameters: map[string]domain.ToolParam{
				"path": {
					Type:        "string",
					Description: "Root directory to search from.",
					Required:    true,
				},
				"query": {
					Type:        "string",
					Description: "Text query to match against file contents.",
					Required:    false,
				},
				"glob": {
					Type:        "string",
					Description: "Glob pattern to match filenames (e.g. '**/*.go').",
					Required:    false,
				},
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
				"url": {
					Type:        "string",
					Description: "The request URL.",
					Required:    true,
				},
				"method": {
					Type:        "string",
					Description: "HTTP method (GET, POST, PUT, DELETE, PATCH). Defaults to GET.",
					Required:    false,
					Enum:        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD"},
				},
				"body": {
					Type:        "string",
					Description: "Request body (for POST/PUT/PATCH). Optional.",
					Required:    false,
				},
			},
		},
		{
			Name:        "kdeps_scraper",
			Description: "Scrape a web page and return its text content.",
			Parameters: map[string]domain.ToolParam{
				"url": {
					Type:        "string",
					Description: "The URL to scrape.",
					Required:    true,
				},
				"selector": {
					Type:        "string",
					Description: "Optional CSS selector to extract a specific element.",
					Required:    false,
				},
			},
		},
		{
			Name:        "kdeps_search_web",
			Description: "Search the web using a configurable search provider and return results.",
			Parameters: map[string]domain.ToolParam{
				"query": {
					Type:        "string",
					Description: "The search query.",
					Required:    true,
				},
				"provider": {
					Type:        "string",
					Description: "Search provider: ddg (default), brave, bing, tavily.",
					Required:    false,
					Enum:        []string{"ddg", "brave", "bing", "tavily"},
				},
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
				"url": {
					Type:        "string",
					Description: "The URL to navigate to.",
					Required:    true,
				},
				"action": {
					Type:        "string",
					Description: "Browser action to perform (navigate, click, screenshot, extract). Default: navigate.",
					Required:    false,
					Enum:        []string{"navigate", "click", "screenshot", "extract"},
				},
				"selector": {
					Type:        "string",
					Description: "CSS selector for click/extract actions.",
					Required:    false,
				},
			},
		},
	}
}
