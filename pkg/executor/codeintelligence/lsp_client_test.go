package codeintelligence

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nopWriteCloser wraps an io.Writer and adds a no-op Close for testing.
type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

// errWriteCloser returns a fixed error on every Write call.
type errWriteCloser struct{}

func (errWriteCloser) Write(_ []byte) (int, error) { return 0, errors.New("write error") }
func (errWriteCloser) Close() error                { return nil }

func TestLSPError_Error(t *testing.T) {
	tests := []struct {
		code    int
		message string
		want    string
	}{
		{-32601, "Method not found", "LSP error -32601: Method not found"},
		{-32603, "Internal error", "LSP error -32603: Internal error"},
		{0, "", "LSP error 0: "},
	}
	for _, tc := range tests {
		t.Run(tc.message, func(t *testing.T) {
			err := &lspError{Code: tc.code, Message: tc.message}
			assert.Equal(t, tc.want, err.Error())
		})
	}
}

func TestStartLSPClient_BadBinary(t *testing.T) {
	_, err := startLSPClient("nonexistent-binary-42-test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lsp: start")
}

func TestLSPClient_Write(t *testing.T) {
	var buf bytes.Buffer
	c := &lspClient{
		stdin: &nopWriteCloser{Writer: &buf},
	}
	err := c.write(lspRequest{JSONRPC: "2.0", ID: 1, Method: "test", Params: map[string]interface{}{"key": "val"}})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Content-Length: ")
	assert.Contains(t, output, `"method":"test"`)
	assert.Contains(t, output, `"key":"val"`)
}

func TestLSPClient_Write_JSONError(t *testing.T) {
	// JSON marshal will fail for a channel
	c := &lspClient{}
	err := c.write(make(chan int))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "json:")
}

func TestLSPClient_Write_StdinError(t *testing.T) {
	c := &lspClient{
		stdin: errWriteCloser{},
	}
	err := c.write(lspRequest{JSONRPC: "2.0", ID: 1, Method: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "write error")
}

func TestLSPClient_ReadMessage(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"result":"ok"}`
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)

	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader(input)),
	}
	data, err := c.readMessage()
	require.NoError(t, err)
	assert.JSONEq(t, body, string(data))
}

func TestLSPClient_ReadMessage_InvalidHeader(t *testing.T) {
	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader("InvalidHeader\r\n\r\n{}")),
	}
	_, err := c.readMessage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expected Content-Length header")
}

func TestLSPClient_ReadMessage_InvalidLength(t *testing.T) {
	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader("Content-Length: abc\r\n\r\n{}")),
	}
	_, err := c.readMessage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "parse Content-Length")
}

func TestLSPClient_ReadMessage_TruncatedBody(t *testing.T) {
	body := `{"jsonrpc":"2.0"}`
	input := fmt.Sprintf("Content-Length: 999\r\n\r\n%s", body)

	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader(input)),
	}
	_, err := c.readMessage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read message body")
}

func TestLSPClient_ReadMessage_EOF(t *testing.T) {
	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader("")),
	}
	_, err := c.readMessage()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "read Content-Length header")
}

func TestLSPClient_ReadResponse_Success(t *testing.T) {
	resp := `{"jsonrpc":"2.0","id":1,"result":{"value":"ok"}}`
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp)

	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader(input)),
	}
	result, err := c.readResponse()
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
	assert.Contains(t, string(result.Result), `"value":"ok"`)
	assert.Nil(t, result.Error)
}

func TestLSPClient_ReadResponse_SkipNotification(t *testing.T) {
	notif := `{"jsonrpc":"2.0","method":"textDocument/publishDiagnostics","params":{}}`
	resp := `{"jsonrpc":"2.0","id":1,"result":"ok"}`
	input := fmt.Sprintf("Content-Length: %d\r\n\r\n%sContent-Length: %d\r\n\r\n%s",
		len(notif), notif, len(resp), resp)

	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader(input)),
	}
	result, err := c.readResponse()
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
}

func TestLSPClient_ReadResponse_InvalidJSON(t *testing.T) {
	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader("Content-Length: 5\r\n\r\n{bad}")),
	}
	_, err := c.readResponse()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lsp: parse response")
}

func TestLSPClient_ReadResponse_EOF(t *testing.T) {
	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader("")),
	}
	_, err := c.readResponse()
	assert.Error(t, err)
}

func TestLSPClient_Call_Success(t *testing.T) {
	var stdin bytes.Buffer
	resp := `{"jsonrpc":"2.0","id":1,"result":{"value":"ok"}}`

	c := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	var result map[string]interface{}
	err := c.call("test", map[string]interface{}{"key": "val"}, &result)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["value"])
}

func TestLSPClient_Call_NilResult(t *testing.T) {
	var stdin bytes.Buffer
	resp := `{"jsonrpc":"2.0","id":1,"result":{"key":"val"}}`

	c := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	err := c.call("test", nil, nil)
	require.NoError(t, err)
}

func TestLSPClient_Call_WriteError(t *testing.T) {
	c := &lspClient{
		stdin: errWriteCloser{},
	}
	err := c.call("test", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lsp: write test:")
}

func TestLSPClient_Call_IDMismatch(t *testing.T) {
	var stdin bytes.Buffer
	// Response ID is 99, but request will have ID=1
	resp := `{"jsonrpc":"2.0","id":99,"result":"ok"}`

	c := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	err := c.call("test", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "response id mismatch")
}

func TestLSPClient_Call_ErrorResponse(t *testing.T) {
	var stdin bytes.Buffer
	resp := `{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`

	c := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	err := c.call("test", nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LSP error -32601")
}

func TestLSPClient_Call_UnmarshalError(t *testing.T) {
	var stdin bytes.Buffer
	// Result is a string, but target is map[string]interface{} -> will fail
	resp := `{"jsonrpc":"2.0","id":1,"result":"string_result"}`

	c := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	var result map[string]interface{}
	err := c.call("test", nil, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal test result")
}

func TestLSPClient_Notify(t *testing.T) {
	var buf bytes.Buffer
	c := &lspClient{
		stdin: &nopWriteCloser{Writer: &buf},
	}
	err := c.notify("test/notification", map[string]interface{}{"key": "val"})
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, `"method":"test/notification"`)
	assert.Contains(t, output, `"key":"val"`)
}

func TestLSPClient_Notify_WriteError(t *testing.T) {
	c := &lspClient{
		stdin: errWriteCloser{},
	}
	err := c.notify("test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notify test:")
}

func TestLSPClient_Close(t *testing.T) {
	cmd := exec.Command("echo")
	stdin, err := cmd.StdinPipe()
	require.NoError(t, err)
	stdout, err := cmd.StdoutPipe()
	require.NoError(t, err)
	require.NoError(t, cmd.Start())

	c := &lspClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}
	c.close()
}
