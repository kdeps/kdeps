package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os/exec"
	"strings"
	"testing"
)

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
