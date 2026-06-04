package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

const testHelperEnv = "GO_TEST_MCP_HELPER"

// runAsMCPHelper is the entry point for subprocess helper mode.
// Each test function calls this when the sentinel env var is set.
func runAsMCPHelper(toolHandler bool) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}

		var resp jsonRPCResponse
		switch req.Method {
		case "initialize":
			resp = jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  rawMsg(`{"protocolVersion":"2024-11-05","capabilities":{}}`),
			}
		case "notifications/initialized":
			continue
		case "tools/call":
			if !toolHandler {
				continue
			}
			params, ok := req.Params.(map[string]interface{})
			if !ok {
				continue
			}
			args, _ := params["arguments"].(map[string]interface{})
			text, _ := args["text"].(string)
			result := mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: text}},
			}
			data, _ := json.Marshal(result)
			resp = jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  rawMsg(string(data)),
			}
		default:
			continue
		}

		respData, _ := json.Marshal(resp)
		_, _ = os.Stdout.Write(append(respData, '\n'))
	}
}

// runAsMCPInitErrorHelper responds to initialize with an error.
func runAsMCPInitErrorHelper() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		if req.Method != "initialize" {
			continue
		}
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonRPCError{Code: -32603, Message: "init failed"},
		}
		respData, _ := json.Marshal(resp)
		_, _ = os.Stdout.Write(append(respData, '\n'))
		return
	}
}

func newStdioTestConfig(t *testing.T, helperFunc string) *domain.MCPConfig {
	t.Helper()
	return &domain.MCPConfig{
		Server: os.Args[0],
		Args:   []string{"-test.run=" + t.Name()},
		Env:    map[string]string{testHelperEnv: "1", "GO_TEST_MCP_ROLE": helperFunc},
	}
}

func TestNewStdioClient_Success(t *testing.T) {
	if os.Getenv(testHelperEnv) == "1" {
		runAsMCPHelper(false)
		return
	}
	cfg := newStdioTestConfig(t, "stdio")
	client, err := NewStdioClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewStdioClient failed: %v", err)
	}
	defer func() { _ = client.Close() }()

	if client.cmd == nil {
		t.Error("expected non-nil cmd")
	}
}

func TestNewStdioClient_EmptyServer(t *testing.T) {
	_, err := NewStdioClient(context.Background(), &domain.MCPConfig{})
	if err == nil {
		t.Fatal("expected error for empty server")
	}
	if err.Error() != "MCP server command is required" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewStdioClient_StartFailure(t *testing.T) {
	_, err := NewStdioClient(context.Background(), &domain.MCPConfig{Server: "/nonexistent/binary"})
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
}

func TestNewStdioClient_InitError(t *testing.T) {
	if os.Getenv(testHelperEnv) == "1" {
		runAsMCPInitErrorHelper()
		return
	}
	cfg := newStdioTestConfig(t, "stdio")
	_, err := NewStdioClient(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error from init failure")
	}
	if !strings.Contains(err.Error(), "MCP initialize") {
		t.Errorf("expected 'MCP initialize' in error, got: %v", err)
	}
}

func TestNewStdioClient_WithEnv(t *testing.T) {
	if os.Getenv(testHelperEnv) == "1" {
		runAsMCPHelper(false)
		return
	}
	cfg := &domain.MCPConfig{
		Server: os.Args[0],
		Args:   []string{"-test.run=" + t.Name()},
		Env: map[string]string{
			testHelperEnv: "1",
			"MCP_VAR":     "hello",
		},
	}
	client, err := NewStdioClient(context.Background(), cfg)
	if err != nil {
		t.Fatalf("NewStdioClient failed: %v", err)
	}
	defer func() { _ = client.Close() }()
}

func TestExecuteTool_Success(t *testing.T) {
	if os.Getenv(testHelperEnv) == "1" {
		runAsMCPHelper(true)
		return
	}
	cfg := &domain.MCPConfig{
		Server: os.Args[0],
		Args:   []string{"-test.run=" + t.Name()},
		Env:    map[string]string{testHelperEnv: "1"},
	}
	result, err := ExecuteTool(cfg, "echo", map[string]interface{}{"text": "hello world"})
	if err != nil {
		t.Fatalf("ExecuteTool failed: %v", err)
	}
	if result != "hello world" {
		t.Errorf("expected 'hello world', got %q", result)
	}
}

func TestExecuteTool_ClientError(t *testing.T) {
	cfg := &domain.MCPConfig{Server: "/nonexistent/binary"}
	_, err := ExecuteTool(cfg, "echo", nil)
	if err == nil {
		t.Fatal("expected error from client creation failure")
	}
}

// TestHelperMCPProcess is not a real test; it exists so the test binary
// can invoke it via -test.run to validate the helper runs correctly.
// This function is intentionally empty.
func TestHelperMCPProcess(_ *testing.T) {
	if os.Getenv(testHelperEnv) == "1" {
		switch os.Getenv("GO_TEST_MCP_ROLE") {
		case "stdio":
			runAsMCPHelper(false)
		case "stdio_tool":
			runAsMCPHelper(true)
		case "init_error":
			runAsMCPInitErrorHelper()
		default:
			runAsMCPHelper(false)
		}
	}
}

func TestNewStdioClient_StdinPipeError(t *testing.T) {
	saveExec := execCommandContext
	t.Cleanup(func() { execCommandContext = saveExec })

	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, name, arg...)
		r, _ := io.Pipe()
		cmd.Stdin = r
		return cmd
	}

	cfg := &domain.MCPConfig{Server: "/bin/echo", Args: []string{"hi"}}
	_, err := NewStdioClient(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error from stdin pipe failure")
	}
	if !strings.Contains(err.Error(), "stdin pipe") {
		t.Errorf("expected 'stdin pipe' in error, got: %v", err)
	}
}

func TestNewStdioClient_StdoutPipeError(t *testing.T) {
	saveExec := execCommandContext
	t.Cleanup(func() { execCommandContext = saveExec })

	execCommandContext = func(ctx context.Context, name string, arg ...string) *exec.Cmd {
		cmd := exec.CommandContext(ctx, name, arg...)
		cmd.Stdout = os.Stderr
		return cmd
	}

	cfg := &domain.MCPConfig{Server: "/bin/echo", Args: []string{"hi"}}
	_, err := NewStdioClient(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error from stdout pipe failure")
	}
	if !strings.Contains(err.Error(), "stdout pipe") {
		t.Errorf("expected 'stdout pipe' in error, got: %v", err)
	}
}
