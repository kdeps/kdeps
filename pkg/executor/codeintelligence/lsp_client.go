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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const lspShutdownTimeout = 5 * time.Second

// lspClient communicates with an LSP server via JSON-RPC 2.0 over stdio.
type lspClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	mu     sync.Mutex
	id     atomic.Int64
}

// lspRequest is a JSON-RPC 2.0 request.
type lspRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int64       `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// lspResponse is a JSON-RPC 2.0 response.
type lspResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *lspError       `json:"error,omitempty"`
}

// lspError is a JSON-RPC 2.0 error.
type lspError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *lspError) Error() string {
	return fmt.Sprintf("LSP error %d: %s", e.Code, e.Message)
}

// lspNotification is a JSON-RPC 2.0 notification (no id).
type lspNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// startLSPClient starts an LSP server binary and returns a connected client.
func startLSPClient(bin string, args []string) (*lspClient, error) {
	cmd := exec.CommandContext(context.Background(), bin, args...)
	cmd.Stderr = nil // LSP servers use stderr for logging only

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("lsp: stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, fmt.Errorf("lsp: stdout pipe: %w", err)
	}

	if err = cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, fmt.Errorf("lsp: start %s: %w", bin, err)
	}

	return &lspClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}, nil
}

// call sends a JSON-RPC 2.0 request and waits for the matching response.
func (c *lspClient) call(method string, params, result interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.id.Add(1)
	req := lspRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.write(&req); err != nil {
		return fmt.Errorf("lsp: write %s: %w", method, err)
	}

	resp, err := c.readResponse()
	if err != nil {
		return fmt.Errorf("lsp: read %s response: %w", method, err)
	}

	if resp.ID != id {
		return fmt.Errorf("lsp: response id mismatch: expected %d, got %d", id, resp.ID)
	}

	if resp.Error != nil {
		return resp.Error
	}

	if result != nil && resp.Result != nil {
		if err = json.Unmarshal(resp.Result, result); err != nil {
			return fmt.Errorf("lsp: unmarshal %s result: %w", method, err)
		}
	}

	return nil
}

// notify sends a JSON-RPC 2.0 notification (no response expected).
func (c *lspClient) notify(method string, params interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	n := lspNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	if err := c.write(&n); err != nil {
		return fmt.Errorf("lsp: notify %s: %w", method, err)
	}

	return nil
}

// write sends a JSON-RPC message over stdin.
// LSP uses Content-Length headers per the protocol spec.
func (c *lspClient) write(msg interface{}) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(body))
	if _, err = io.WriteString(c.stdin, header); err != nil {
		return err
	}
	if _, err = c.stdin.Write(body); err != nil {
		return err
	}
	return nil
}

// readResponse reads the next JSON-RPC response from stdout.
func (c *lspClient) readResponse() (*lspResponse, error) {
	for {
		body, err := c.readMessage()
		if err != nil {
			return nil, err
		}

		var resp lspResponse
		if err = json.Unmarshal(body, &resp); err != nil {
			return nil, fmt.Errorf("lsp: parse response: %w", err)
		}

		// Skip notifications (server-to-client messages without an id).
		// LSP servers may send window/logMessage, textDocument/publishDiagnostics, etc.
		if resp.ID == 0 {
			continue
		}

		return &resp, nil
	}
}

// readMessage reads one LSP message (Content-Length header + body).
func (c *lspClient) readMessage() ([]byte, error) {
	// Read Content-Length header line.
	header, err := c.stdout.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read Content-Length header: %w", err)
	}
	header = strings.TrimSpace(header)

	if !strings.HasPrefix(header, "Content-Length: ") {
		return nil, fmt.Errorf("expected Content-Length header, got %q", header)
	}

	length, err := strconv.Atoi(strings.TrimSpace(header[len("Content-Length: "):]))
	if err != nil {
		return nil, fmt.Errorf("parse Content-Length: %w", err)
	}

	// Read the blank line after header.
	if _, err = c.stdout.ReadString('\n'); err != nil {
		return nil, fmt.Errorf("read header separator: %w", err)
	}

	// Read the body.
	body := make([]byte, length)
	if _, err = io.ReadFull(c.stdout, body); err != nil {
		return nil, fmt.Errorf("read message body (%d bytes): %w", length, err)
	}

	return body, nil
}

// close shuts down the LSP server.
func (c *lspClient) close() {
	_ = c.notify("shutdown", nil)
	_ = c.notify("exit", nil)

	// Give the server a moment to exit gracefully.
	done := make(chan error, 1)
	go func() { done <- c.cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(lspShutdownTimeout):
		_ = c.cmd.Process.Kill()
	}

	_ = c.stdin.Close()
}
