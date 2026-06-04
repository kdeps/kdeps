package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
)

// readResponse tests for uncovered branches

func TestReadResponse_EmptyLine(t *testing.T) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(1),
		Result:  rawMsg(`"ok"`),
	}
	data, _ := json.Marshal(resp)
	input := "\n" + string(data) + "\n"
	client := &Client{stdout: bufio.NewScanner(strings.NewReader(input))}
	got, err := client.readResponse()
	if err != nil {
		t.Fatalf("readResponse failed: %v", err)
	}
	if got.ID != float64(1) {
		t.Errorf("expected ID 1, got %v", got.ID)
	}
}

func TestReadResponse_UnmarshalFail(t *testing.T) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(1),
		Result:  rawMsg(`"ok"`),
	}
	data, _ := json.Marshal(resp)
	input := "not json\n" + string(data) + "\n"
	client := &Client{stdout: bufio.NewScanner(strings.NewReader(input))}
	got, err := client.readResponse()
	if err != nil {
		t.Fatalf("readResponse failed: %v", err)
	}
	if got.ID != float64(1) {
		t.Errorf("expected ID 1, got %v", got.ID)
	}
}

func TestReadResponse_ScannerEOF(t *testing.T) {
	client := &Client{stdout: bufio.NewScanner(strings.NewReader(""))}
	_, err := client.readResponse()
	if err == nil {
		t.Fatal("expected error for closed reader")
	}
	if err.Error() != "MCP server stdout closed unexpectedly" {
		t.Errorf("unexpected error: %v", err)
	}
}

// CallTool tests for uncovered branches

func TestCallTool_ErrorResponse(t *testing.T) {
	r, w := io.Pipe()
	client := NewClientForTesting(w, bufio.NewScanner(r))
	go callToolResponderWithError(r, w)
	_, err := client.CallTool("echo", nil)
	if err == nil {
		t.Fatal("expected error from tool error response")
	}
	_ = client.Close()
}

func TestCallTool_NullResult(t *testing.T) {
	r, w := io.Pipe()
	client := NewClientForTesting(w, bufio.NewScanner(r))
	go callToolResponderNullResult(r, w)
	result, err := client.CallTool("echo", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
	_ = client.Close()
}

func TestCallTool_RawJSONFallback(t *testing.T) {
	r, w := io.Pipe()
	client := NewClientForTesting(w, bufio.NewScanner(r))
	go callToolResponderRawJSON(r, w)
	result, err := client.CallTool("echo", nil)
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result != `"raw string result"` {
		t.Errorf(`expected "raw string result", got %q`, result)
	}
	_ = client.Close()
}

func TestCallTool_IsError(t *testing.T) {
	r, w := io.Pipe()
	client := NewClientForTesting(w, bufio.NewScanner(r))
	go callToolResponderIsError(r, w)
	_, err := client.CallTool("echo", nil)
	if err == nil {
		t.Fatal("expected error from tool IsError")
	}
	_ = client.Close()
}

func TestCallTool_SendError(t *testing.T) {
	r, w := io.Pipe()
	w.Close()
	client := NewClientForTesting(w, bufio.NewScanner(r))
	_, err := client.CallTool("echo", nil)
	if err == nil {
		t.Fatal("expected error from send failure")
	}
	_ = client.Close()
}

func TestCallTool_ReadError(t *testing.T) {
	r, _ := io.Pipe()
	r.Close()
	client := NewClientForTesting(nopWriteCloser{&bytes.Buffer{}}, bufio.NewScanner(r))
	_, err := client.CallTool("echo", nil)
	if err == nil {
		t.Fatal("expected error from read failure")
	}
	_ = client.Close()
}

// initialize tests for uncovered branches

func TestInitialize_SendError(t *testing.T) {
	r, w := io.Pipe()
	w.Close()
	client := NewClientForTesting(w, bufio.NewScanner(r))
	err := client.initialize()
	if err == nil {
		t.Fatal("expected error from send failure")
	}
	_ = client.Close()
}

func TestInitialize_ReadError(t *testing.T) {
	r, _ := io.Pipe()
	r.Close()
	client := NewClientForTesting(nopWriteCloser{&bytes.Buffer{}}, bufio.NewScanner(r))
	err := client.initialize()
	if err == nil {
		t.Fatal("expected error from read failure")
	}
	_ = client.Close()
}

func TestInitialize_ServerError(t *testing.T) {
	r, w := io.Pipe()
	client := NewClientForTesting(w, bufio.NewScanner(r))
	go initializeErrorResponder(r, w)
	err := client.initialize()
	if err == nil {
		t.Fatal("expected error from server init error")
	}
	_ = client.Close()
}

// responder helpers

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
