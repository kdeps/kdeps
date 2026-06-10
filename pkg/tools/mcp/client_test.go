package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestNewClientForTesting(t *testing.T) {
	r, w := io.Pipe()
	client := NewClientForTesting(w, bufio.NewScanner(r))
	if client == nil {
		t.Fatal("NewClientForTesting returned nil")
	}
	_ = client.Close()
}

func TestSend(t *testing.T) {
	var buf bytes.Buffer
	client := &Client{stdin: nopWriteCloser{&buf}}

	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "test",
		Params:  map[string]string{"key": "value"},
	}

	if err := client.send(req); err != nil {
		t.Fatalf("send failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test") {
		t.Errorf("expected 'test' in output, got %q", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("expected 'key' in output, got %q", output)
	}
}

func TestReadResponse(t *testing.T) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(1),
		Result:  rawMsg(`{"content":[{"type":"text","text":"hello"}]}`),
	}
	data, _ := json.Marshal(resp)

	client := &Client{stdout: bufio.NewScanner(strings.NewReader(string(data) + "\n"))}
	got, err := client.readResponse()
	if err != nil {
		t.Fatalf("readResponse failed: %v", err)
	}
	if got.ID != float64(1) {
		t.Errorf("expected ID 1, got %v", got.ID)
	}
}

func TestReadResponse_SkipsNotifications(t *testing.T) {
	notif := jsonRPCResponse{JSONRPC: "2.0"}
	notifData, _ := json.Marshal(notif)

	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(2),
		Result:  rawMsg(`"ok"`),
	}
	respData, _ := json.Marshal(resp)

	input := string(notifData) + "\n" + string(respData) + "\n"
	client := &Client{stdout: bufio.NewScanner(strings.NewReader(input))}
	got, err := client.readResponse()
	if err != nil {
		t.Fatalf("readResponse failed: %v", err)
	}
	if got.ID != float64(2) {
		t.Errorf("expected ID 2 (skipping notification), got %v", got.ID)
	}
}

func TestCallTool(t *testing.T) {
	r, w := io.Pipe()
	client := NewClientForTesting(w, bufio.NewScanner(r))

	go callToolResponder(r, w)

	result, err := client.CallTool("echo", map[string]interface{}{"text": "hello"})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if result != "hello" {
		t.Errorf("expected 'hello', got %q", result)
	}

	_ = client.Close()
}

func callToolResponder(r io.Reader, w io.Writer) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(scanner.Text()), &req); err != nil {
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
			params := req.Params.(map[string]interface{})
			args := params["arguments"].(map[string]interface{})
			text := args["text"].(string)
			result := mcpToolResult{
				Content: []mcpContent{{Type: "text", Text: text}},
			}
			data, _ := json.Marshal(result)
			resp = jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  rawMsg(string(data)),
			}
		}

		respData, _ := json.Marshal(resp)
		_, _ = w.Write(append(respData, '\n'))
	}
}

func rawMsg(s string) *json.RawMessage {
	m := json.RawMessage(s)
	return &m
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

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

func TestSend_MarshalError(t *testing.T) {
	client := &Client{stdin: nopWriteCloser{&bytes.Buffer{}}}
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		Params:  make(chan int),
	}
	err := client.send(req)
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "marshal") {
		t.Errorf("expected 'marshal' in error, got: %v", err)
	}
}

func TestClose_KillsProcess(t *testing.T) {
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "sleep", "999")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	})

	client := NewClientForTesting(nopWriteCloser{&bytes.Buffer{}}, bufio.NewScanner(strings.NewReader("")))
	client.cmd = cmd

	if err := client.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if err := cmd.Wait(); err == nil {
		t.Fatal("expected error from killed process Wait")
	}
}

func TestInitialize_NotificationSendError(t *testing.T) {
	initResp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      float64(1),
		Result:  rawMsg(`{"protocolVersion":"2024-11-05","capabilities":{}}`),
	}
	respData, _ := json.Marshal(initResp)
	input := string(respData) + "\n"

	client := &Client{
		stdin:  &failAfterFirstWrite{buf: &bytes.Buffer{}},
		stdout: bufio.NewScanner(strings.NewReader(input)),
	}
	err := client.initialize()
	if err == nil {
		t.Fatal("expected error from notification send failure")
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
