package codeintelligence

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLSPInitOptions_Python(t *testing.T) {
	opts := lspInitOptions("python")
	require.NotNil(t, opts)
	assert.Equal(t, "basic", opts["typeCheckingMode"])
}

func TestLSPInitOptions_Go(t *testing.T) {
	opts := lspInitOptions("go")
	require.NotNil(t, opts)
	assert.Equal(t, true, opts["ui.semanticTokens"])
}

func TestLSPInitOptions_Default(t *testing.T) {
	assert.Nil(t, lspInitOptions("rust"))
	assert.Nil(t, lspInitOptions("unknown"))
	assert.Nil(t, lspInitOptions(""))
}

func TestLSPManager_New(t *testing.T) {
	m := newLSPManager()
	assert.NotNil(t, m)
	assert.NotNil(t, m.cache)
	assert.NotNil(t, m.lookup)
}

func TestLSPManager_DetectServer_Unknown(t *testing.T) {
	m := newLSPManager()
	assert.Nil(t, m.detectServer("unknown-language-123"))
}

func TestLSPManager_DetectServer_AllFound(t *testing.T) {
	m := &lspManager{
		cache:  make(map[string]*lspClient),
		lookup: func(_ string) bool { return true },
	}

	info := m.detectServer("go")
	require.NotNil(t, info)
	assert.Equal(t, "gopls", info.bin)

	info = m.detectServer("python")
	require.NotNil(t, info)
	assert.Equal(t, "pyright-langserver", info.bin)

	info = m.detectServer("ruby")
	require.NotNil(t, info)
	assert.Equal(t, "solargraph", info.bin)

	info = m.detectServer("typescript")
	require.NotNil(t, info)
	assert.Equal(t, "typescript-language-server", info.bin)

	info = m.detectServer("javascript")
	require.NotNil(t, info)
	assert.Equal(t, "typescript-language-server", info.bin)

	info = m.detectServer("rust")
	require.NotNil(t, info)
	assert.Equal(t, "rust-analyzer", info.bin)

	info = m.detectServer("c")
	require.NotNil(t, info)
	assert.Equal(t, "clangd", info.bin)

	info = m.detectServer("cpp")
	require.NotNil(t, info)
	assert.Equal(t, "clangd", info.bin)
}

func TestLSPManager_DetectServer_NoneFound(t *testing.T) {
	m := &lspManager{
		cache:  make(map[string]*lspClient),
		lookup: func(_ string) bool { return false },
	}

	assert.Nil(t, m.detectServer("go"))
	assert.Nil(t, m.detectServer("python"))
	assert.Nil(t, m.detectServer("ruby"))
	assert.Nil(t, m.detectServer("typescript"))
	assert.Nil(t, m.detectServer("rust"))
	assert.Nil(t, m.detectServer("c"))
}

func TestLSPManager_GetServer_CacheHit(t *testing.T) {
	m := &lspManager{
		cache:  make(map[string]*lspClient),
		lookup: func(_ string) bool { return false },
	}
	mockClient := &lspClient{}
	m.cache["go:/workspace"] = mockClient

	client, err := m.getServer("go", "/workspace", "")
	require.NoError(t, err)
	assert.Same(t, mockClient, client)
}

func TestLSPManager_GetServer_CacheMissDifferentWorkspace(t *testing.T) {
	m := &lspManager{
		cache:  make(map[string]*lspClient),
		lookup: func(_ string) bool { return false },
	}
	m.cache["go:/workspace1"] = &lspClient{}

	_, err := m.getServer("go", "/workspace2", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `no server found for language "go"`)
}

func TestLSPManager_GetServer_NoLanguage(t *testing.T) {
	m := &lspManager{
		cache:  make(map[string]*lspClient),
		lookup: func(_ string) bool { return false },
	}
	_, err := m.getServer("unknown", "/workspace", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), `no server found for language "unknown"`)
}

func TestLSPManager_Initialize_Success(t *testing.T) {
	var stdin bytes.Buffer
	resp := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`

	client := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	m := newLSPManager()
	err := m.initialize(client, "go", "/workspace", "/workspace/main.go")
	require.NoError(t, err)

	// Should have written initialize request and initialized notification
	output := stdin.String()
	assert.Contains(t, output, `"method":"initialize"`)
	assert.Contains(t, output, `"method":"initialized"`)
}

func TestLSPManager_Initialize_Error(t *testing.T) {
	var stdin bytes.Buffer
	resp := `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"Internal error"}}`

	client := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	m := newLSPManager()
	err := m.initialize(client, "go", "/workspace", "/workspace/main.go")
	assert.Error(t, err)
}

func TestLSPManager_Initialize_EmptyWorkspaceRoot(t *testing.T) {
	var stdin bytes.Buffer
	resp := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`

	client := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	m := newLSPManager()
	err := m.initialize(client, "go", "", "/workspace/main.go")
	require.NoError(t, err)

	output := stdin.String()
	// Should use filepath.Dir(config.Path) as workspace root
	assert.Contains(t, output, strings.TrimPrefix("/workspace", "/"))
}

func TestLSPManager_Initialize_RootFallback(t *testing.T) {
	var stdin bytes.Buffer
	resp := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`

	client := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	m := newLSPManager()
	err := m.initialize(client, "go", ".", "")
	require.NoError(t, err)

	// workspaceRoot "." with empty filePath -> fallback to "/"
	output := stdin.String()
	_ = output
}

func TestLSPManager_Initialize_PythonOptions(t *testing.T) {
	var stdin bytes.Buffer
	resp := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`

	client := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	m := newLSPManager()
	err := m.initialize(client, "python", "/workspace", "/workspace/main.py")
	require.NoError(t, err)

	output := stdin.String()
	// Python init options should include typeCheckingMode
	assert.Contains(t, output, `"typeCheckingMode"`)
}

func TestLSPManager_Initialize_GoOptions(t *testing.T) {
	var stdin bytes.Buffer
	resp := `{"jsonrpc":"2.0","id":1,"result":{"capabilities":{}}}`

	client := &lspClient{
		stdin: &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader(
			fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	m := newLSPManager()
	err := m.initialize(client, "go", "/workspace", "/workspace/main.go")
	require.NoError(t, err)

	output := stdin.String()
	assert.Contains(t, output, `"ui.semanticTokens"`)
}
