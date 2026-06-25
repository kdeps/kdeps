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
	execEmbedding "github.com/kdeps/kdeps/v2/pkg/executor/embedding"
	execHTTP "github.com/kdeps/kdeps/v2/pkg/executor/http"
	execLoader "github.com/kdeps/kdeps/v2/pkg/executor/loader"
	execSearch "github.com/kdeps/kdeps/v2/pkg/executor/searchlocal"
	execTranscribe "github.com/kdeps/kdeps/v2/pkg/executor/transcribe"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

// registerResourceTools registers all resource-based tools (HTTP, SearchLocal, etc.)
// that an LLM agent can use without needing a workflow YAML file.
func registerResourceTools(ctx context.Context, reg *kdepstools.Registry) {
	registerHTTPTool(ctx, reg)
	registerSearchLocalTool(ctx, reg)
	registerTranscribeTool(ctx, reg)
	registerLoaderTool(ctx, reg)
	registerEmbeddingTools(ctx, reg)
}

// registerHTTPTool registers an HTTP request tool (http_request).
func registerHTTPTool(_ context.Context, reg *kdepstools.Registry) {
	exec := execHTTP.NewExecutor()

	reg.Register(&kdepstools.Tool{
		Name:        "http_request",
		Description: "Make an HTTP request to a URL. Returns response status, headers, and body. Use for calling APIs, fetching web content, or interacting with external services. Requires: url. Optional: method (default GET), headers, data (JSON body), timeout.",
		Parameters: map[string]domain.ToolParam{
			"url": {Type: toolParamString, Description: "The URL to request", Required: true},
			"method": {
				Type:        toolParamString,
				Description: "HTTP method: GET, POST, PUT, DELETE, PATCH. Default: GET",
			},
			"headers":     {Type: "object", Description: "HTTP headers as key-value pairs"},
			toolParamData: {Type: "object", Description: "Request body as JSON (for POST/PUT/PATCH)"},
			"timeout":     {Type: toolParamString, Description: "Request timeout, e.g. '30s'. Default: 30s"},
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
			if v, ok := args[toolParamData]; ok {
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
			toolParamPath: {
				Type:        toolParamString,
				Description: "Directory to search in (absolute path)",
				Required:    true,
			},
			toolParamQuery: {Type: toolParamString, Description: "Search term or regex pattern", Required: true},
			"glob":         {Type: toolParamString, Description: "File glob filter, e.g. '*.go', '*.py'"},
		},
		Execute: func(args map[string]any) (string, error) {
			config := &domain.SearchLocalConfig{}
			if v, ok := args[toolParamPath].(string); ok {
				config.Path = v
			}
			if v, ok := args[toolParamQuery].(string); ok {
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

// registerTranscribeTool registers an audio transcription tool (transcribe_audio).
func registerTranscribeTool(_ context.Context, reg *kdepstools.Registry) {
	exec := execTranscribe.NewExecutor()

	reg.Register(&kdepstools.Tool{
		Name:        "transcribe_audio",
		Description: "Transcribe an audio or video file to text using Whisper API. Supports mp3, mp4, mpeg, mpga, m4a, wav, webm. Returns the transcribed text. Requires: file (absolute path to audio file). Optional: model (default whisper-1), backend (openai, groq, local).",
		Parameters: map[string]domain.ToolParam{
			"file": {
				Type:        toolParamString,
				Description: "Absolute path to the audio/video file to transcribe",
				Required:    true,
			},
			toolParamModel: {
				Type:        toolParamString,
				Description: "Transcription model. Default: whisper-1. Groq: whisper-large-v3",
			},
			"backend": {
				Type:        toolParamString,
				Description: "API provider: openai (default), groq, or local",
			},
		},
		Execute: func(args map[string]any) (string, error) {
			config := &domain.TranscribeConfig{}
			if v, ok := args["file"].(string); ok {
				config.File = v
			}
			if v, ok := args[toolParamModel].(string); ok {
				config.Model = v
			}
			if v, ok := args["backend"].(string); ok {
				config.Backend = v
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

// registerLoaderTool registers a document loader tool (load_document).
func registerLoaderTool(_ context.Context, reg *kdepstools.Registry) {
	exec := execLoader.NewExecutor()

	reg.Register(&kdepstools.Tool{
		Name:        "load_document",
		Description: "Load a document file and return its content as text. Supports PDF (go-pdf, pdftotext, pdfcpu), DOCX/EPUB/RTF/ODT (pandoc), HTML (goquery, lynx), CSV, text, directory, notion, markdown (textutil). Use for reading documents into the conversation for analysis or RAG pipelines. Returns document content with optional splitting into chunks. Requires: source (absolute file path). Optional: type (pdf, csv, html, text — auto-detected from extension), chunkSize (split into chunks of N characters).",
		Parameters: map[string]domain.ToolParam{
			"source": {
				Type:        toolParamString,
				Description: "Absolute path to the document file",
				Required:    true,
			},
			"type": {
				Type:        toolParamString,
				Description: "Document type: pdf, pdf_pdftotext, pdf_cpu, pandoc, docx, epub, rtf, odt, html, html_lynx, csv, text, textutil, directory, notion. Auto-detected if omitted. Auto-detected if omitted. pdf=Go lib, pdf_pdftotext=poppler, pdf_cpu=pdfcpu, pandoc=universal, docx/epub/rtf/odt=pandoc, html_lynx=lynx, textutil=macOS.",
			},
			"chunkSize": {
				Type:        toolParamNumber,
				Description: "Split into chunks of this many characters (for RAG). 0 = no splitting.",
			},
		},
		Execute: func(args map[string]any) (string, error) {
			config := &domain.LoaderConfig{}
			if v, ok := args["source"].(string); ok {
				config.Source = v
			}
			if v, ok := args["type"].(string); ok {
				config.Type = v
			}
			if v, ok := args["chunkSize"].(float64); ok {
				config.ChunkSize = int(v)
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

// registerEmbeddingTools registers embedding tools (embedding_vectorize, embedding_search).
func registerEmbeddingTools(_ context.Context, reg *kdepstools.Registry) {
	exec := execEmbedding.NewExecutor()

	embedTools := []struct {
		name, desc, op string
		params         map[string]domain.ToolParam
	}{
		{
			name: "embedding_search",
			desc: "Search for documents semantically similar to a query in the local embedding database. Returns ranked results with similarity scores. Requires: query (natural language search query), collection (name of the document collection). Optional: limit (max results, default 5).",
			op:   "search",
			params: map[string]domain.ToolParam{
				toolParamQuery: {
					Type:        toolParamString,
					Description: "Natural language search query",
					Required:    true,
				},
				"collection": {
					Type:        toolParamString,
					Description: "Name of the document collection to search",
					Required:    true,
				},
				"limit": {
					Type:        toolParamNumber,
					Description: "Maximum number of results. Default: 5",
				},
			},
		},
		{
			name: "embedding_vectorize",
			desc: "Convert text into vector embeddings using an embedding model. Returns the embedding vectors. Use for indexing documents into the local embedding database or computing semantic similarity. Requires: texts (list of strings to embed). Optional: model, backend.",
			op:   "vectorize",
			params: map[string]domain.ToolParam{
				"texts": {
					Type:        "array",
					Description: "List of text strings to convert to embeddings",
					Required:    true,
				},
				toolParamModel: {
					Type:        toolParamString,
					Description: "Embedding model, e.g. text-embedding-3-small",
				},
				"backend": {Type: toolParamString, Description: "Backend: openai, ollama, google"},
			},
		},
	}

	for _, et := range embedTools {
		reg.Register(&kdepstools.Tool{
			Name:        et.name,
			Description: et.desc,
			Parameters:  et.params,
			Execute:     makeEmbeddingExecute(exec, et.op),
		})
	}
}

// makeEmbeddingExecute builds an Execute closure for embedding tools.
func makeEmbeddingExecute(
	exec *execEmbedding.Executor,
	op string,
) func(map[string]any) (string, error) {
	return func(args map[string]any) (string, error) {
		config := &domain.EmbeddingConfig{Operation: op}
		if v, ok := args[toolParamQuery].(string); ok {
			config.Text = v
		}
		if v, ok := args["collection"].(string); ok {
			config.Collection = v
		}
		if v, ok := args["limit"].(float64); ok {
			config.Limit = int(v)
		}
		if v, ok := args["texts"].([]any); ok {
			for _, t := range v {
				config.Inputs = append(config.Inputs, fmt.Sprint(t))
			}
		}
		if v, ok := args[toolParamModel].(string); ok {
			config.Model = v
		}
		if v, ok := args["backend"].(string); ok {
			config.Backend = v
		}
		result, err := exec.Execute(nil, config)
		if err != nil {
			return "", err
		}
		out, _ := json.MarshalIndent(result, "", "  ")
		return string(out), nil
	}
}
