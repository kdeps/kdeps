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

package mcp_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	mcpserver "github.com/kdeps/kdeps/v2/pkg/mcp"
	"github.com/kdeps/kdeps/v2/pkg/tools"
	"github.com/kdeps/kdeps/v2/pkg/tools/fformat"
)

func TestServer_Initialize(t *testing.T) {
	registry := tools.NewRegistry()
	var out bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}` + "\n"
	server := mcpserver.NewServerWithIO(registry, strings.NewReader(input), &out)
	_ = server.Serve()

	lines := splitLines(out.String())
	if len(lines) == 0 {
		t.Fatal("expected at least one response line")
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp["result"] == nil {
		t.Fatalf("expected result, got: %v", resp)
	}
	result := resp["result"].(map[string]interface{})
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("protocolVersion = %v, want 2024-11-05", result["protocolVersion"])
	}
}

func TestServer_ToolsList(t *testing.T) {
	registry := tools.NewRegistry()
	tools.RegisterFFormatTools(registry)
	var out bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/list"}` + "\n"
	server := mcpserver.NewServerWithIO(registry, strings.NewReader(input), &out)
	_ = server.Serve()

	lines := splitLines(out.String())
	if len(lines) == 0 {
		t.Fatal("expected response")
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	result := resp["result"].(map[string]interface{})
	toolList := result["tools"].([]interface{})
	if len(toolList) < 4 {
		t.Errorf("expected at least 4 fformat tools, got %d", len(toolList))
	}
}

func TestServer_ToolsCall_FFormatValidate(t *testing.T) {
	registry := tools.NewRegistry()
	tools.RegisterFFormatTools(registry)
	var out bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"fformat_validate","arguments":{"input":"{\"key\":\"value\"}","format":"json"}}}` + "\n"
	server := mcpserver.NewServerWithIO(registry, strings.NewReader(input), &out)
	_ = server.Serve()

	lines := splitLines(out.String())
	if len(lines) == 0 {
		t.Fatal("expected response")
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["error"] != nil {
		t.Fatalf("unexpected error: %v", resp["error"])
	}
	result := resp["result"].(map[string]interface{})
	content := result["content"].([]interface{})
	text := content[0].(map[string]interface{})["text"].(string)

	var fmtResult fformat.Result
	if err := json.Unmarshal([]byte(text), &fmtResult); err != nil {
		t.Fatalf("unmarshal fformat result: %v", err)
	}
	if !fmtResult.Valid {
		t.Errorf("expected valid=true, got false: %s", fmtResult.Error)
	}
}

func TestServer_ToolsCall_UnknownTool(t *testing.T) {
	registry := tools.NewRegistry()
	var out bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"nonexistent","arguments":{}}}` + "\n"
	server := mcpserver.NewServerWithIO(registry, strings.NewReader(input), &out)
	_ = server.Serve()

	lines := splitLines(out.String())
	if len(lines) == 0 {
		t.Fatal("expected response")
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["error"] == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestServer_MethodNotFound(t *testing.T) {
	registry := tools.NewRegistry()
	var out bytes.Buffer
	input := `{"jsonrpc":"2.0","id":1,"method":"unknown/method"}` + "\n"
	server := mcpserver.NewServerWithIO(registry, strings.NewReader(input), &out)
	_ = server.Serve()

	lines := splitLines(out.String())
	if len(lines) == 0 {
		t.Fatal("expected response")
	}
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["error"] == nil {
		t.Fatal("expected error for unknown method")
	}
}

func splitLines(s string) []string {
	var result []string
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}
	return result
}
