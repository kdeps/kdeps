package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"
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
