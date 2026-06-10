package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func callToolResponderWithError(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(scanner.Text()), &req); err != nil {
			continue
		}
		if req.Method != "tools/call" {
			continue
		}
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonRPCError{Code: -32603, Message: "Internal error"},
		}
		respData, _ := json.Marshal(resp)
		_, _ = w.Write(append(respData, '\n'))
		return
	}
}

func callToolResponderNullResult(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(scanner.Text()), &req); err != nil {
			continue
		}
		if req.Method != "tools/call" {
			continue
		}
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
		}
		respData, _ := json.Marshal(resp)
		_, _ = w.Write(append(respData, '\n'))
		return
	}
}

func callToolResponderRawJSON(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(scanner.Text()), &req); err != nil {
			continue
		}
		if req.Method != "tools/call" {
			continue
		}
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  rawMsg(`"raw string result"`),
		}
		respData, _ := json.Marshal(resp)
		_, _ = w.Write(append(respData, '\n'))
		return
	}
}

func callToolResponderIsError(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(scanner.Text()), &req); err != nil {
			continue
		}
		if req.Method != "tools/call" {
			continue
		}
		result := mcpToolResult{
			IsError: true,
			Content: []mcpContent{{Type: "text", Text: "something went wrong"}},
		}
		data, _ := json.Marshal(result)
		resp := jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  rawMsg(string(data)),
		}
		respData, _ := json.Marshal(resp)
		_, _ = w.Write(append(respData, '\n'))
		return
	}
}

func initializeErrorResponder(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(scanner.Text()), &req); err != nil {
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
		_, _ = w.Write(append(respData, '\n'))
		return
	}
}

// failAfterFirstWrite succeeds on the first Write and fails on subsequent writes.
type failAfterFirstWrite struct {
	buf   *bytes.Buffer
	count int
}

func (w *failAfterFirstWrite) Write(p []byte) (int, error) {
	if w.count > 0 {
		return 0, errors.New("write failed")
	}
	w.count++
	return w.buf.Write(p)
}

func (w *failAfterFirstWrite) Close() error { return nil }

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
