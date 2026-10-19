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
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/afero"

	"github.com/kdeps/kdeps/v2/pkg/api"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	execEmbedding "github.com/kdeps/kdeps/v2/pkg/executor/embedding"
	execHTTP "github.com/kdeps/kdeps/v2/pkg/executor/http"
	execLoader "github.com/kdeps/kdeps/v2/pkg/executor/loader"
	execSearch "github.com/kdeps/kdeps/v2/pkg/executor/searchlocal"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

// registerResourceTools registers all resource-based tools (HTTP, SearchLocal, etc.)
// that an LLM agent can use without needing a workflow YAML file.
func registerResourceTools(ctx context.Context, reg *kdepstools.Registry) {
	registerMemoryTools(reg) // memory works in both agent and workflow modes
	registerHTTPTool(ctx, reg)
	registerSearchLocalTool(ctx, reg)
	registerLoaderTool(ctx, reg)
	registerEmbeddingTools(ctx, reg)
}

// registerHTTPTool registers an HTTP request tool (http_request).
func registerHTTPTool(_ context.Context, reg *kdepstools.Registry) {
	exec := execHTTP.NewExecutor()

	reg.Register(defined(&kdepstools.Tool{
		Name: "http_request",
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
	}))
}

// registerSearchLocalTool registers a local file search tool (search_local).
// Indexing is deferred to first use. Run '/search index' to build the
// inverted index for faster ranked searches.
func registerSearchLocalTool(_ context.Context, reg *kdepstools.Registry) {
	exec := execSearch.NewExecutor()

	fmt.Fprintln(os.Stderr, "searchLocal: files auto-indexed on read/search — /search index for full pre-index")

	reg.Register(defined(&kdepstools.Tool{
		Name: toolNameSearchLocal,
		Execute: func(args map[string]any) (string, error) {
			query, _ := args[toolParamQuery].(string)
			return trackCodeCall(query, func() (string, error) {
				return executeSearchLocal(exec, args)
			})
		},
	}))
}

// registerLoaderTool registers a document loader tool (load_document).
func registerLoaderTool(_ context.Context, reg *kdepstools.Registry) {
	exec := execLoader.NewExecutor()

	reg.Register(defined(&kdepstools.Tool{
		Name: "load_document",
		Execute: func(args map[string]any) (string, error) {
			config := &domain.LoaderConfig{}
			if v, ok := args["source"].(string); ok {
				config.Source = v
				if err := ValidateRootPath(v); err != nil {
					return "", fmt.Errorf("load_document: %w", err)
				}
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
	}))
}

// registerEmbeddingTools registers embedding tools (embedding_vectorize, embedding_search).
func registerEmbeddingTools(_ context.Context, reg *kdepstools.Registry) {
	exec := execEmbedding.NewExecutor()

	embedTools := []struct {
		name, op string
	}{
		{
			name: "embedding_search",
			op:   "search",
		},
		{
			name: "embedding_vectorize",
			op:   "vectorize",
		},
	}

	for _, et := range embedTools {
		reg.Register(defined(&kdepstools.Tool{
			Name:    et.name,
			Execute: makeEmbeddingExecute(exec, et.op),
		}))
	}
}

// extractResultPaths extracts file paths from search results.
func extractResultPaths(results []map[string]interface{}) []string {
	paths := make([]string, 0, len(results))
	for _, r := range results {
		if p, ok := r["path"].(string); ok {
			paths = append(paths, p)
		}
	}
	return paths
}

// projectSearchRoots returns the folders the working project's searches
// cover (its places, then its search folders), or nil with no project. The
// knowledge vault sets it.
//
//nolint:gochecknoglobals // set once by the knowledge vault, read by search_local
var projectSearchRoots = func() []string { return nil }

// executeSearchLocal runs a search_local tool execution. With no path it
// searches every folder of the working project, else the working directory.
//
//nolint:gocognit // flat sequence of optional-arg extractions, not nested logic
func executeSearchLocal(exec *execSearch.Executor, args map[string]any) (string, error) {
	q, _ := args[toolParamQuery].(string)
	g, _ := args["glob"].(string)
	if strings.TrimSpace(q) == "" && strings.TrimSpace(g) == "" {
		return "", errors.New("search_local: give a query (or a glob) to search for")
	}
	if p, _ := args[toolParamPath].(string); strings.TrimSpace(p) == "" {
		roots := projectSearchRoots()
		if len(roots) > 1 {
			return searchProjectRoots(exec, roots, args)
		}
		if len(roots) == 1 {
			args[toolParamPath] = roots[0]
		} else if wd, err := os.Getwd(); err == nil {
			args[toolParamPath] = wd
		}
	}
	config := &domain.SearchLocalConfig{}
	if v, ok := args[toolParamPath].(string); ok {
		config.Path = v
		if err := ValidateRootPath(v); err != nil {
			return "", fmt.Errorf("search_local: %w", err)
		}
	}
	if v, ok := args[toolParamQuery].(string); ok {
		config.Query = v
	}
	if v, ok := args["glob"].(string); ok {
		config.Glob = v
	}
	if v, ok := args["limit"].(float64); ok && v > 0 {
		config.Limit = int(v)
	} else {
		config.Limit = 3
	}

	result, err := exec.Execute(nil, config)
	if err != nil {
		return "", err
	}

	// Lazily index result files into the search index.
	rm, rmOK := result.(map[string]interface{})
	if rmOK {
		results, resultsOK := rm["results"].([]map[string]interface{})
		if resultsOK {
			paths := extractResultPaths(results)
			if len(paths) > 0 {
				indexInBackground(func() { exec.IndexFiles(config.Path, paths) })
			}
			annotateSearchMatches(results, config.Query)
		}
	}

	out, _ := json.MarshalIndent(result, "", "  ")

	// Also search memory so the LLM gets relevant context alongside
	// file search results without a separate tool call.
	if ms := GetOrCreateMemoryStore(); ms != nil {
		if memResults := ms.Search(config.Query); len(memResults) > 0 {
			var sb strings.Builder
			sb.Write(out)
			sb.WriteString("\n\n--- memory ---\n")
			for _, entry := range memResults {
				fmt.Fprintf(&sb, "%s: %s\n", entry.Key, entry.Value)
			}
			return sb.String(), nil
		}
	}

	return string(out), nil
}

// searchProjectRoots runs search_local over each root and takes the best
// hits of each in turn: scores of two folders' searches do not compare.
func searchProjectRoots(exec *execSearch.Executor, roots []string, args map[string]any) (string, error) {
	limit := 3
	if v, ok := args["limit"].(float64); ok && v > 0 {
		limit = int(v)
	}
	query, _ := args[toolParamQuery].(string)
	glob, _ := args["glob"].(string)
	var perRoot [][]map[string]interface{}
	var errs []string
	for _, root := range roots {
		res, err := exec.Execute(nil, &domain.SearchLocalConfig{Path: root, Query: query, Glob: glob, Limit: limit})
		if err != nil {
			errs = append(errs, root+": "+err.Error())
			continue
		}
		rm, _ := res.(map[string]interface{})
		results, _ := rm["results"].([]map[string]interface{})
		if paths := extractResultPaths(results); len(paths) > 0 {
			indexInBackground(func() { exec.IndexFiles(root, paths) })
		}
		perRoot = append(perRoot, results)
	}
	merged := interleaveResults(perRoot, limit)
	annotateSearchMatches(merged, query)
	out := map[string]interface{}{"results": merged, "count": len(merged), "folders": roots}
	if len(errs) > 0 {
		out["errors"] = errs
	}
	data, _ := json.MarshalIndent(out, "", "  ")
	return string(data), nil
}

// interleaveResults takes the first result of each list, then the second
// of each, and so on, up to limit, dropping a path seen already.
func interleaveResults(lists [][]map[string]interface{}, limit int) []map[string]interface{} {
	out := []map[string]interface{}{}
	seen := map[string]bool{}
	for i := 0; len(out) < limit; i++ {
		more := false
		for _, l := range lists {
			if i >= len(l) {
				continue
			}
			more = true
			p, _ := l[i]["path"].(string)
			if seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, l[i])
			if len(out) == limit {
				break
			}
		}
		if !more {
			break
		}
	}
	return out
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
