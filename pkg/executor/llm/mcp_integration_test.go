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

// Package llm — MCP integration tests using the self-binary trick.
//
// When FAKE_MCP_SERVER=1 is set, TestMain runs a minimal MCP stdio server
// that handles initialize + tools/call and then exits.  The actual tests
// start the compiled test binary as a subprocess with that env var, giving
// us a real MCP server without any external dependencies.
package llm

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// fakeMCPServerEnvKey is the environment variable that switches the test
// binary into fake-MCP-server mode.
const fakeMCPServerEnvKey = "FAKE_MCP_SERVER"

// TestMain is the entry point for the llm package test binary.
// When FAKE_MCP_SERVER=1 it acts as a minimal MCP server over stdio;
// otherwise it runs the test suite normally.
func TestMain(m *testing.M) {
	if os.Getenv(fakeMCPServerEnvKey) == "1" {
		runFakeMCPServerStdio()
		os.Exit(0)
	}
	os.Exit(m.Run())
}

// runFakeMCPServerStdio reads JSON-RPC requests from stdin and writes
// responses to stdout, behaving like a real MCP server.
//
// Supported methods:
//   - initialize  → returns protocol version + capabilities
//   - tools/call  → returns a single text content item: "fake mcp result"
//   - (other)     → returns null result
//
// Notifications (no "id" field) are silently ignored.
//
// When FAKE_MCP_EXIT_EARLY=1, the function returns immediately without reading
// anything, causing the client's readResponse to see EOF (closed unexpectedly).
func runFakeMCPServerStdio() {
	if os.Getenv("FAKE_MCP_EXIT_EARLY") == "1" {
		return // exit without writing anything
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req map[string]interface{}
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		// Notifications have no id — skip them
		id, hasID := req["id"]
		if !hasID || id == nil {
			continue
		}

		method, _ := req["method"].(string)

		var result interface{}
		switch method {
		case "initialize":
			result = map[string]interface{}{
				"protocolVersion": mcpProtocolVersion,
				"capabilities":    map[string]interface{}{},
				"serverInfo": map[string]interface{}{
					"name":    "fake-mcp",
					"version": "1.0",
				},
			}
		case "tools/call":
			result = map[string]interface{}{
				"content": []map[string]interface{}{
					{"type": "text", "text": "fake mcp result"},
				},
				"isError": false,
			}
		default:
			result = nil
		}

		resp := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      id,
			"result":  result,
		}
		data, _ := json.Marshal(resp)
		fmt.Fprintf(os.Stdout, "%s\n", data) //nolint:errcheck
	}
}

// TestExecuteMCPTool_Success verifies the full happy-path for executeMCPTool:
// start subprocess → initialize handshake → tools/call → return text result.
// The test binary itself acts as the MCP server via the FAKE_MCP_SERVER env var.
func TestExecuteMCPTool_Success(t *testing.T) {
	execPath, err := os.Executable()
	require.NoError(t, err)

	cfg := &domain.MCPConfig{
		Server: execPath,
		// -test.run=DOESNOTMATCH prevents any test from running in the subprocess;
		// TestMain will still be invoked and will enter fake-server mode.
		Args: []string{"-test.run=DOESNOTMATCH_INTEGRATION"},
		Env:  map[string]string{fakeMCPServerEnvKey: "1"},
	}

	result, execErr := executeMCPTool(cfg, "search", map[string]interface{}{"q": "hello"})
	require.NoError(t, execErr)
	assert.Equal(t, "fake mcp result", result)
}
