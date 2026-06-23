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

package codeintelligence

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// LSP operation methods on Executor.

func (e *Executor) lspSymbolSearch(client *lspClient, config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Query == "" {
		return nil, errors.New("codeIntelligence: query is required for symbolSearch")
	}

	params := map[string]interface{}{
		"query": config.Query,
	}

	var result []map[string]interface{}
	if err := client.call("workspace/symbol", params, &result); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	var symbols []map[string]interface{}
	for _, s := range result {
		symbols = append(symbols, map[string]interface{}{
			"name": s["name"],
			"kind": s["kind"],
			"file": filepathFromURI(fmt.Sprint(s["location"].(map[string]interface{})["uri"])),
		})
	}

	return resultLSP(true, map[string]interface{}{
		"symbols": symbols,
		"count":   len(symbols),
	}), nil
}

func (e *Executor) lspDefinition(client *lspClient, config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Symbol == "" {
		return nil, errors.New("codeIntelligence: symbol is required for definition")
	}

	if err := e.lspEnsureDocument(client, config); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	pos := e.findSymbolPosition(config)

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": fileURI(config.Path),
		},
		"position": pos,
	}

	var locations []map[string]interface{}
	if err := client.call("textDocument/definition", params, &locations); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	var defs []map[string]interface{}
	for _, loc := range locations {
		defs = append(defs, map[string]interface{}{
			"file": filepathFromURI(fmt.Sprint(loc["uri"])),
			"line": lineFromPosition(loc["range"]),
		})
	}

	return resultLSP(true, map[string]interface{}{
		"definitions": defs,
		"count":       len(defs),
	}), nil
}

func (e *Executor) lspReferences(client *lspClient, config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Symbol == "" {
		return nil, errors.New("codeIntelligence: symbol is required for references")
	}

	if err := e.lspEnsureDocument(client, config); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	pos := e.findSymbolPosition(config)

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": fileURI(config.Path),
		},
		"position": pos,
		"context": map[string]interface{}{
			"includeDeclaration": true,
		},
	}

	var locations []map[string]interface{}
	if err := client.call("textDocument/references", params, &locations); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	var refs []map[string]interface{}
	for _, loc := range locations {
		refs = append(refs, map[string]interface{}{
			"file": filepathFromURI(fmt.Sprint(loc["uri"])),
			"line": lineFromPosition(loc["range"]),
		})
	}

	return resultLSP(true, map[string]interface{}{
		"references": refs,
		"count":      len(refs),
	}), nil
}

func (e *Executor) lspDocumentSymbols(client *lspClient, config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for documentSymbols")
	}

	if err := e.lspEnsureDocument(client, config); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": fileURI(config.Path),
		},
	}

	var result interface{}
	if err := client.call("textDocument/documentSymbol", params, &result); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	symbols := flattenLSPDocumentSymbols(result)
	return resultLSP(true, map[string]interface{}{
		"symbols": symbols,
		"count":   len(symbols),
	}), nil
}

func (e *Executor) lspHover(client *lspClient, config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for hover")
	}

	if err := e.lspEnsureDocument(client, config); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	pos := e.findSymbolPosition(config)

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": fileURI(config.Path),
		},
		"position": pos,
	}

	var hover map[string]interface{}
	if err := client.call("textDocument/hover", params, &hover); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	contents := ""
	if c, ok := hover["contents"]; ok {
		switch v := c.(type) {
		case string:
			contents = v
		case map[string]interface{}:
			contents = fmt.Sprint(v["value"])
		}
	}

	return resultLSP(true, map[string]interface{}{
		"hover": map[string]interface{}{
			"contents": contents,
		},
	}), nil
}

func (e *Executor) lspDiagnostics(client *lspClient, config *domain.CodeIntelligenceConfig) (interface{}, error) {
	if config.Path == "" {
		return nil, errors.New("codeIntelligence: path is required for diagnostics")
	}

	if err := e.lspEnsureDocument(client, config); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	params := map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri": fileURI(config.Path),
		},
	}

	var result map[string]interface{}
	if err := client.call("textDocument/diagnostic", params, &result); err != nil {
		return resultLSP(false, map[string]interface{}{"error": err.Error()}), err
	}

	items, _ := result["items"].([]interface{})
	var diagnostics []map[string]interface{}
	for _, item := range items {
		if diag, ok := item.(map[string]interface{}); ok {
			diagnostics = append(diagnostics, map[string]interface{}{
				"message": fmt.Sprint(diag["message"]),
				"severity": fmt.Sprint(diag["severity"]),
				"source":   fmt.Sprint(diag["source"]),
			})
		}
	}

	return resultLSP(true, map[string]interface{}{
		"diagnostics": diagnostics,
		"count":       len(diagnostics),
	}), nil
}

// lspEnsureDocument opens a file in the LSP server if not already open.
func (e *Executor) lspEnsureDocument(client *lspClient, config *domain.CodeIntelligenceConfig) error {
	if config.Path == "" {
		return nil
	}
	content, err := os.ReadFile(config.Path)
	if err != nil {
		return fmt.Errorf("lsp: read file %s: %w", config.Path, err)
	}

	return client.notify("textDocument/didOpen", map[string]interface{}{
		"textDocument": map[string]interface{}{
			"uri":        fileURI(config.Path),
			"languageId": languageIDFromPath(config.Path),
			"version":    1,
			"text":       string(content),
		},
	})
}

// --- LSP helpers ---

func resultLSP(success bool, data map[string]interface{}) map[string]interface{} {
	if data == nil {
		data = map[string]interface{}{}
	}
	data["success"] = success
	return data
}

func fileURI(path string) string {
	abs, _ := filepath.Abs(path)
	return "file://" + strings.TrimPrefix(abs, "/")
}

func filepathFromURI(uri string) string {
	return strings.TrimPrefix(uri, "file://")
}

func lineFromPosition(r interface{}) int {
	if rng, ok := r.(map[string]interface{}); ok {
		if start, ok := rng["start"].(map[string]interface{}); ok {
			if line, ok := start["line"].(float64); ok {
				return int(line) + 1 // LSP uses 0-based lines
			}
		}
	}
	return 0
}

func flattenLSPDocumentSymbols(result interface{}) []map[string]interface{} {
	var symbols []map[string]interface{}

	switch v := result.(type) {
	case []interface{}:
		for _, item := range v {
			if sym, ok := item.(map[string]interface{}); ok {
				addSymbol := map[string]interface{}{
					"name": sym["name"],
					"kind": sym["kind"],
				}
				symbols = append(symbols, addSymbol)

				// Recurse into children.
				if children, ok := sym["children"]; ok {
					childSymbols := flattenLSPDocumentSymbols(children)
					symbols = append(symbols, childSymbols...)
				}
			}
		}
	}
	return symbols
}

// findSymbolPosition uses rg to find the first occurrence of config.Symbol in config.Path
// and returns an LSP position. Falls back to {0,0} if not found.
func (e *Executor) findSymbolPosition(config *domain.CodeIntelligenceConfig) map[string]interface{} {
	pos := map[string]interface{}{
		"line":      0,
		"character": 0,
	}
	if config.Path == "" || config.Symbol == "" {
		return pos
	}

	args := []string{"--json", "--line-number", "--max-count", "1", config.Symbol, config.Path}
	matches, err := e.runRG(args)
	if err != nil || len(matches) == 0 {
		return pos
	}

	m := matches[0]
	line := m.Data.LineNumber - 1 // LSP uses 0-based lines
	if line < 0 {
		line = 0
	}

	ch := strings.Index(m.Data.Lines.Text, config.Symbol)
	if ch < 0 {
		ch = 0
	}

	pos["line"] = line
	pos["character"] = ch
	return pos
}
