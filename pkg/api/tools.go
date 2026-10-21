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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"slices"
	"sync"

	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/assets"
	"github.com/kdeps/kdeps/v2/pkg/domain"
)

const (
	toolsPrefix     = "/apis/" + groupTools + "/" + versionV1 + "/tools"
	contentTypeText = "text/plain; charset=utf-8"
	maxArgsBytes    = 1 << 20
	kindToolList    = "ToolList"
	msgBadJSON      = "invalid json"
	msgBodyTooLarge = "body too large"
	msgToolNotFound = "tool not found"
)

// ToolSpec is one callable operation. Call runs in this process.
type ToolSpec struct {
	Name        string
	Description string
	Parameters  map[string]domain.ToolParam
	Call        func(context.Context, map[string]any) (string, error)
}

var errBodyTooLarge = errors.New(msgBodyTooLarge)

var (
	toolMu  sync.RWMutex            //nolint:gochecknoglobals // guards the tool registry
	toolReg = map[string]ToolSpec{} //nolint:gochecknoglobals // call-route tools
)

// openToolYAML reads one tool asset. Tests replace it.
//
//nolint:gochecknoglobals // test-replaceable asset read
var openToolYAML = func(name string) ([]byte, error) {
	return assets.Open("tools").ReadFile("tools/" + name + ".yaml")
}

//nolint:gochecknoinits // these tools are served before RegisterBuiltinTools runs
func init() {
	RegisterTool(readFileSpec())
	initWebClients()
	if webSearchCall != nil {
		RegisterTool(loadToolSpec(toolWebSearch, "Search the web using DuckDuckGo.", WebSearch))
	}
	RegisterTool(loadToolSpec(toolWikipedia, "Look up a topic on Wikipedia.", Wikipedia))
	if scrapeCall != nil {
		RegisterTool(loadToolSpec(toolWebScraper, "Fetch readable text from a URL.", WebScrape))
	}
	RegisterTool(loadToolSpec(toolCalculator, "Evaluate a mathematical expression.", Calculator))
	RegisterTool(loadToolSpec(toolSQLList, "List tables in a SQLite database.", SQLListTables))
	RegisterTool(loadToolSpec(toolSQLDescribe, "Describe one SQLite table.", SQLDescribeTable))
	RegisterTool(loadToolSpec(toolSQLQuery, "Run a read-only SQL query.", SQLQuery))
	RegisterTool(loadToolSpec(toolSQLOps, "Run SQL against a named connection.", SQLOps))
	RegisterTool(loadToolSpec(toolBashExec, "Execute a bash shell command.", BashExec))
	RegisterTool(loadToolSpec(toolExec, "Run a shell command with structured output.", ExecOps))
	RegisterTool(loadToolSpec(toolPython, "Run a Python script with structured output.", PythonOps))
	RegisterTool(loadToolSpec(toolBashJobList, "List background bash jobs.", BashJobList))
	RegisterTool(loadToolSpec(toolBashJobWait, "Wait for a background bash job.", BashJobWait))
	RegisterTool(loadToolSpec(toolListFiles, "List files and directories.", ListFiles))
	RegisterTool(loadToolSpec(toolMD5File, "Compute the MD5 checksum of a file.", MD5File))
	RegisterTool(loadToolSpec(toolTailFile, "Read the last lines of a file.", TailFile))
	RegisterTool(loadToolSpec(toolWriteFile, "Write or overwrite a text file.", WriteFile))
	RegisterTool(loadToolSpec(toolEditFile, "Edit a file.", EditFile))
	RegisterTool(loadToolSpec(toolSerpAPI, "Search Google via SerpAPI.", SerpAPISearch))
	RegisterTool(loadToolSpec(toolPerplexity, "Search the web using Perplexity AI.", PerplexitySearch))
	RegisterTool(loadToolSpec(toolGoogleCacheCreate, "Create a Google AI cached content entry.", GoogleCacheCreate))
	RegisterTool(loadToolSpec(toolGoogleCacheDelete, "Delete a Google AI cached content entry.", GoogleCacheDelete))
	RegisterTool(loadToolSpec(toolGoogleCacheList, "List Google AI cached content entries.", GoogleCacheList))
	RegisterTool(loadToolSpec(toolMemorySave, "Save a fact to persistent memory.", MemorySave))
	RegisterTool(loadToolSpec(toolMemorySearch, "Search persistent memory.", MemorySearch))
	RegisterTool(loadToolSpec(toolMemoryDelete, "Delete a memory entry by key.", MemoryDelete))
	RegisterTool(loadToolSpec(toolMemoryQuery, "Run a relational query over agent state.", MemoryQuery))
	registerCodeTools()
	RegisterTool(loadToolSpec(toolEmbeddingSearch, "Search the local embedding database.", EmbeddingSearch))
	RegisterTool(loadToolSpec(toolEmbeddingVectorize, "Convert text into embeddings.", EmbeddingVectorize))
	RegisterTool(loadToolSpec(toolEmbedding, "Run one embedding operation.", EmbeddingOps))
	RegisterTool(loadToolSpec(toolHTTPRequest, "Make an HTTP request.", HTTPRequest))
	RegisterTool(loadToolSpec(toolTwiml, "Build a TwiML voice response.", TwiML))
	RegisterTool(loadToolSpec(toolEmailOps, "Send email and work with IMAP mailboxes.", EmailOps))
	RegisterTool(loadToolSpec(toolBrowserOps, "Drive a browser with Playwright.", BrowserOps))
	RegisterTool(loadToolSpec(toolFileOps, "Run one structured filesystem operation.", FileOps))
	RegisterTool(loadToolSpec(toolGitOps, "Run one structured git operation.", GitOps))
	RegisterTool(loadToolSpec(toolLoadDocument, "Load a document as text.", LoadDocument))
	RegisterTool(loadToolSpec(toolSearchLocal, "Search local files.", SearchLocal))
	RegisterTool(loadToolSpec(toolVectorStore, "Run one vector store operation.", VectorStore))
	registerRegistryTools()
}

// registerCodeTools registers the code intelligence tools.
func registerCodeTools() {
	RegisterTool(loadToolSpec(toolCodeSearch, "Search for a symbol or pattern.", CodeSearch))
	RegisterTool(loadToolSpec(toolCodeDefinition, "Find the definition of a symbol.", CodeDefinition))
	RegisterTool(loadToolSpec(toolCodeReferences, "Find references to a symbol.", CodeReferences))
	RegisterTool(loadToolSpec(toolCodeSymbols, "List symbols in a file.", CodeSymbols))
	RegisterTool(loadToolSpec(toolCodeHover, "Get documentation for a symbol.", CodeHover))
	RegisterTool(loadToolSpec(toolCodeDiagnostics, "Get diagnostics for a file.", CodeDiagnostics))
	RegisterTool(loadToolSpec(toolCodeIndexFolder, "Index the working directory into a graph.", CodeIndexFolder))
	RegisterTool(loadToolSpec(toolCodeGraphFile, "Graph one indexed file.", CodeGraphFile))
	RegisterTool(loadToolSpec(toolCodeGraphTopic, "Graph indexed files for a topic.", CodeGraphTopic))
	RegisterTool(loadToolSpec(toolCodeGraphAll, "Graph the whole index.", CodeGraphAll))
}

// loadToolSpec builds a call spec from the tool asset. A missing or empty
// asset keeps the fallback description.
func loadToolSpec(name, fallback string, call func(context.Context, map[string]any) (string, error)) ToolSpec {
	spec := ToolSpec{Name: name, Description: fallback, Call: call}
	data, err := openToolYAML(name)
	if err != nil {
		return spec
	}
	var doc struct {
		Description string                      `yaml:"description"`
		Parameters  map[string]domain.ToolParam `yaml:"parameters"`
	}
	if yaml.Unmarshal(data, &doc) != nil || doc.Description == "" {
		return spec
	}
	spec.Description = doc.Description
	spec.Parameters = doc.Parameters
	return spec
}

// RegisterTool adds or replaces a tool served by the call route.
// A spec with an empty name or a nil Call is ignored.
func RegisterTool(spec ToolSpec) {
	if spec.Name == "" || spec.Call == nil {
		return
	}
	toolMu.Lock()
	toolReg[spec.Name] = spec
	toolMu.Unlock()
}

func lookupTool(name string) (ToolSpec, bool) {
	toolMu.RLock()
	spec, ok := toolReg[name]
	toolMu.RUnlock()
	return spec, ok
}

func listToolItems() []toolItem {
	toolMu.RLock()
	defer toolMu.RUnlock()
	names := make([]string, 0, len(toolReg))
	for name := range toolReg {
		names = append(names, name)
	}
	slices.Sort(names)
	items := make([]toolItem, 0, len(names))
	for _, name := range names {
		spec := toolReg[name]
		items = append(items, toolItem{
			Name:        spec.Name,
			Description: spec.Description,
			Parameters:  spec.Parameters,
		})
	}
	return items
}

func handleToolList(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, toolListDoc{
		Kind:       kindToolList,
		APIVersion: groupTools + "/" + versionV1,
		Items:      listToolItems(),
	})
}

func handleToolCall(w http.ResponseWriter, r *http.Request) {
	spec, ok := lookupTool(r.PathValue("name"))
	if !ok || spec.Call == nil {
		writeStatus(w, http.StatusNotFound, msgToolNotFound)
		return
	}
	args, err := decodeArgs(r.Body)
	if err != nil {
		msg := msgBadJSON
		if errors.Is(err, errBodyTooLarge) {
			msg = msgBodyTooLarge
		}
		writeStatus(w, http.StatusBadRequest, msg)
		return
	}
	out, err := spec.Call(r.Context(), args)
	if err != nil {
		writeStatus(w, http.StatusBadRequest, err.Error())
		return
	}
	w.Header().Set("Content-Type", contentTypeText)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(out))
}

func decodeArgs(r io.Reader) (map[string]any, error) {
	if r == nil {
		return map[string]any{}, nil
	}
	data, err := io.ReadAll(io.LimitReader(r, maxArgsBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxArgsBytes {
		return nil, errBodyTooLarge
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return map[string]any{}, nil
	}
	var args map[string]any
	if err = json.Unmarshal(data, &args); err != nil {
		return nil, err
	}
	if args == nil {
		return map[string]any{}, nil
	}
	return args, nil
}

func writeStatus(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, statusDoc{
		Kind:    kindStatus,
		Status:  "Failure",
		Message: message,
		Code:    code,
	})
}

type toolListDoc struct {
	Kind       string     `json:"kind"`
	APIVersion string     `json:"apiVersion"`
	Items      []toolItem `json:"items"`
}

type toolItem struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	Parameters  map[string]domain.ToolParam `json:"parameters,omitempty"`
}
