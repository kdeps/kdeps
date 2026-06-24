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
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// lspServerInfo describes a detected LSP server.
type lspServerInfo struct {
	bin  string
	args []string
}

// lspManager manages LSP server lifecycles across languages and workspaces.
type lspManager struct {
	mu     sync.Mutex
	cache  map[string]*lspClient // key: "<languageID>:<workspaceRoot>"
	lookup func(string) bool     // exec.LookPath
}

// newLSPManager creates a new LSP manager.
func newLSPManager() *lspManager {
	return &lspManager{
		cache:  make(map[string]*lspClient),
		lookup: func(s string) bool { _, err := exec.LookPath(s); return err == nil },
	}
}

// getServer returns a cached or newly started LSP server for the given language and workspace.
func (m *lspManager) getServer(languageID, workspaceRoot, filePath string) (*lspClient, error) {
	cacheKey := languageID + ":" + workspaceRoot

	m.mu.Lock()
	if c, ok := m.cache[cacheKey]; ok {
		m.mu.Unlock()
		return c, nil
	}
	m.mu.Unlock()

	info := m.detectServer(languageID)
	if info == nil {
		return nil, fmt.Errorf("lsp: no server found for language %q", languageID)
	}

	client, err := startLSPClient(info.bin, info.args)
	if err != nil {
		return nil, err
	}

	// Initialize the server with workspace capabilities.
	if err := m.initialize(client, languageID, workspaceRoot, filePath); err != nil {
		client.close()
		return nil, fmt.Errorf("lsp: initialize %s: %w", languageID, err)
	}

	m.mu.Lock()
	m.cache[cacheKey] = client
	m.mu.Unlock()

	return client, nil
}

// detectServer finds the LSP binary for a language.
func (m *lspManager) detectServer(languageID string) *lspServerInfo {
	switch languageID {
	case "go":
		if m.lookup("gopls") {
			return &lspServerInfo{bin: "gopls", args: []string{"serve", "-mode=stdio"}}
		}
	case "python": //nolint:goconst
		if m.lookup("pyright-langserver") {
			return &lspServerInfo{bin: "pyright-langserver", args: []string{"--stdio"}}
		}
		if m.lookup("pyright") {
			return &lspServerInfo{bin: "pyright", args: []string{"--stdio"}}
		}
	case "rust":
		if m.lookup("rust-analyzer") {
			return &lspServerInfo{bin: "rust-analyzer", args: nil}
		}
	case "typescript", "javascript":
		if m.lookup("typescript-language-server") {
			return &lspServerInfo{bin: "typescript-language-server", args: []string{"--stdio"}}
		}
	case "c", "cpp":
		if m.lookup("clangd") {
			return &lspServerInfo{bin: "clangd", args: nil}
		}
	case "ruby":
		if m.lookup("solargraph") {
			return &lspServerInfo{bin: "solargraph", args: []string{"stdio"}}
		}
	}
	return nil
}

// initialize sends the LSP initialize request, then the initialized notification.
func (m *lspManager) initialize(client *lspClient, languageID, workspaceRoot, filePath string) error {
	if workspaceRoot == "" {
		workspaceRoot = filepath.Dir(filePath)
	}
	if workspaceRoot == "" || workspaceRoot == "." {
		workspaceRoot = "/"
	}

	uri := "file://" + workspaceRoot

	capabilities := map[string]interface{}{}
	params := map[string]interface{}{
		"processId":     nil,
		"rootUri":       uri,
		"capabilities":  capabilities,
		"workspaceFolders": []map[string]interface{}{
			{"uri": uri, "name": filepath.Base(workspaceRoot)},
		},
	}

	// Add language-specific initialization options.
	if opts := lspInitOptions(languageID); opts != nil {
		params["initializationOptions"] = opts
	}

	var initResult map[string]interface{}
	if err := client.call("initialize", params, &initResult); err != nil {
		return err
	}

	// Send initialized notification.
	return client.notify("initialized", map[string]interface{}{})
}

// lspInitOptions returns language-specific initialization options.
func lspInitOptions(languageID string) map[string]interface{} {
	switch languageID {
	case "python": //nolint:goconst
		return map[string]interface{}{
			"typeCheckingMode": "basic",
		}
	case "go":
		return map[string]interface{}{
			"ui.semanticTokens": true,
		}
	default:
		return nil
	}
}

// languageIDFromPath infers an LSP language ID from a file extension.
func languageIDFromPath(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp", ".hxx":
		return "cpp"
	case ".rb":
		return "ruby"
	case ".java":
		return "java"
	default:
		return ""
	}
}
