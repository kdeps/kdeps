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
	"os"
	"strings"

	"github.com/spf13/afero"

	"github.com/kdeps/kdeps/v2/pkg/api"
	execEmbedding "github.com/kdeps/kdeps/v2/pkg/executor/embedding"
	_ "github.com/kdeps/kdeps/v2/pkg/executor/http" // wires http_request; that package imports pkg/api
	execSearch "github.com/kdeps/kdeps/v2/pkg/executor/searchlocal"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

// registerResourceTools registers all resource-based tools (HTTP, SearchLocal, etc.)
// that an LLM agent can use without needing a workflow YAML file.
func registerResourceTools(ctx context.Context, reg *kdepstools.Registry) {
	registerMemoryTools(ctx, reg) // memory works in both agent and workflow modes
	registerHTTPTool(ctx, reg)
	registerSearchLocalTool(ctx, reg)
	registerLoaderTool(ctx, reg)
	registerEmbeddingTools(ctx, reg)
}

// registerHTTPTool registers an HTTP request tool (http_request).
// The handler is the kdeps API.
func registerHTTPTool(ctx context.Context, reg *kdepstools.Registry) {
	registerAPITool(ctx, reg, "http_request", api.HTTPRequest)
}

// registerSearchLocalTool registers a local file search tool (search_local).
// Indexing is deferred to first use. Run '/search index' to build the
// inverted index for faster ranked searches.
func registerSearchLocalTool(ctx context.Context, reg *kdepstools.Registry) {
	fmt.Fprintln(os.Stderr, "searchLocal: files auto-indexed on read/search — /search index for full pre-index")

	tool := defined(&kdepstools.Tool{Name: toolNameSearchLocal})
	if tool == nil {
		return
	}
	tool.Execute = func(args map[string]any) (string, error) {
		query, _ := args[toolParamQuery].(string)
		return trackCodeCall(query, func() (string, error) {
			callCtx := api.WithSearch(toolCallCtx(ctx, args), api.SearchOpts{
				Roots: projectSearchRoots(),
				Index: func(root string, paths []string) {
					indexInBackground(func() { api.IndexSearchFiles(root, paths) })
				},
			})
			out, err := api.CallTool(callCtx, toolNameSearchLocal, jsonArgs(args))
			if err != nil {
				return "", err
			}
			return finishSearchLocal(out, query)
		})
	}
	api.RegisterTool(api.ToolSpec{
		Name:        tool.Name,
		Description: tool.Description,
		Parameters:  tool.Parameters,
		Call:        api.SearchLocal,
	})
	reg.Register(tool)
}

// registerLoaderTool registers a document loader tool (load_document).
// The handler is the kdeps API.
func registerLoaderTool(ctx context.Context, reg *kdepstools.Registry) {
	registerAPITool(ctx, reg, "load_document", api.LoadDocument)
}

// registerEmbeddingTools registers embedding tools (embedding_vectorize, embedding_search).
// The handler is the kdeps API.
func registerEmbeddingTools(ctx context.Context, reg *kdepstools.Registry) {
	registerAPITool(ctx, reg, "embedding_search", api.EmbeddingSearch)
	registerAPITool(ctx, reg, "embedding_vectorize", api.EmbeddingVectorize)
}

// projectSearchRoots returns the folders the working project's searches
// cover (its places, then its search folders), or nil with no project. The
// knowledge vault sets it.
//
//nolint:gochecknoglobals // set once by the knowledge vault, read by search_local
var projectSearchRoots = func() []string { return nil }

// executeSearchLocal runs a search_local tool execution. With no path it
// searches every folder of the working project, else the working directory.
func executeSearchLocal(exec *execSearch.Executor, args map[string]any) (string, error) {
	query, _ := args[toolParamQuery].(string)
	out, err := api.SearchLocalWith(api.SearchOpts{
		Roots: projectSearchRoots(),
		Exec:  exec,
		Index: func(root string, paths []string) {
			indexInBackground(func() { exec.IndexFiles(root, paths) })
		},
	}, args)
	if err != nil {
		return "", err
	}
	return finishSearchLocal(out, query)
}

// interleaveResults takes the first result of each list, then the second
// of each, and so on, up to limit, dropping a path seen already.
func interleaveResults(lists [][]map[string]interface{}, limit int) []map[string]interface{} {
	return api.InterleaveSearchResults(lists, limit)
}

// makeEmbeddingExecute builds an Execute closure for embedding tools.
func makeEmbeddingExecute(
	exec *execEmbedding.Executor,
	op string,
) func(map[string]any) (string, error) {
	return func(args map[string]any) (string, error) {
		return api.RunEmbedding(exec, op, args)
	}
}

// finishSearchLocal adds match_id fields and, for a single folder, memory hits.
// A multi-folder result carries "folders" and skips memory.
func finishSearchLocal(out, query string) (string, error) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		return "", err
	}
	annotateSearchMatches(searchRows(doc), query)
	encoded, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return "", err
	}
	if _, multi := doc["folders"]; multi {
		return string(encoded), nil
	}
	return appendSearchMemory(string(encoded), query), nil
}

func searchRows(doc map[string]any) []map[string]any {
	raw, ok := doc["results"].([]any)
	if !ok {
		return nil
	}
	rows := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		row, rowOK := item.(map[string]any)
		if rowOK {
			rows = append(rows, row)
		}
	}
	return rows
}

func appendSearchMemory(out, query string) string {
	ms := GetOrCreateMemoryStore()
	if ms == nil {
		return out
	}
	memResults := ms.Search(query)
	if len(memResults) == 0 {
		return out
	}
	var sb strings.Builder
	sb.WriteString(out)
	sb.WriteString("\n\n--- memory ---\n")
	for _, entry := range memResults {
		fmt.Fprintf(&sb, "%s: %s\n", entry.Key, entry.Value)
	}
	return sb.String()
}

// annotateSearchMatches adds match_id/line/revision to each result so
// read_file can reference a hit directly (path + line) instead of the
// model retyping a path and guessing a line range from a truncated
// snippet. Best-effort: a result whose file can't be bound-read, or that
// has no findable query line, is left without these fields rather than
// failing the whole search.
func annotateSearchMatches(results []map[string]interface{}, query string) {
	if query == "" {
		return
	}
	for _, r := range results {
		path, _ := r["path"].(string)
		if path == "" {
			continue
		}
		info, statErr := AppFS.Stat(path)
		if statErr != nil || info.IsDir() || info.Size() > int64(maxFileReadBytes()) {
			continue
		}
		data, readErr := afero.ReadFile(AppFS, path)
		if readErr != nil {
			continue
		}
		line := firstMatchLine(string(data), query)
		if line == 0 {
			continue
		}
		revision := api.FileRevision(data)
		id := mintMatchID(path, line, revision)
		rememberMatch(id, matchRef{path: path, line: line, revision: revision})
		r["match_id"] = id
		r["line"] = line
		r["revision"] = revision
	}
}

// firstMatchLine returns the 1-based line query first (case-insensitively)
// occurs on in content, or 0 if it doesn't appear.
func firstMatchLine(content, query string) int {
	idx := strings.Index(strings.ToLower(content), strings.ToLower(query))
	if idx < 0 {
		return 0
	}
	return 1 + strings.Count(content[:idx], "\n")
}
