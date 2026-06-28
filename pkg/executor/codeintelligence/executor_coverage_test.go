//go:build !js

package codeintelligence

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// TestIsGoDir_RegularFile covers the !info.IsDir() branch (returns false for a file).
func TestIsGoDir_RegularFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "notadir.txt")
	if err := os.WriteFile(f, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}
	if isGoDir(f) {
		t.Fatal("expected false for a regular file")
	}
}

// TestIsGoDir_UnreadableDir covers the ReadDir error path.
func TestIsGoDir_UnreadableDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "locked")
	if err := os.Mkdir(sub, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(sub, 0o000); err != nil {
		t.Skip("cannot set permissions (running as root?)")
	}
	t.Cleanup(func() { _ = os.Chmod(sub, 0o750) })
	if isGoDir(sub) {
		t.Fatal("expected false for unreadable dir")
	}
}

// TestInferKind_FuncWithoutParen covers inferKind returning "function"
// for a func line that has no parenthesis.
func TestInferKind_FuncWithoutParen(t *testing.T) {
	result := inferKind("func myFunc")
	if result != "function" {
		t.Fatalf("expected 'function', got %q", result)
	}
}

// TestExecuteLSP_UnsupportedOperation2 covers the default case in executeLSP.
func TestExecuteLSP_UnsupportedOperation2(t *testing.T) {
	e := &Executor{}
	_, err := e.executeLSP(nil, &domain.CodeIntelligenceConfig{
		Operation: "unknown_lsp_op_xyz",
		Path:      "file.go",
	}, "go")
	if err == nil {
		t.Fatal("expected error for unsupported LSP operation")
	}
}

// TestExecuteRG_UnsupportedOperation2 covers the default case in executeRG.
func TestExecuteRG_UnsupportedOperation2(t *testing.T) {
	runner := &mockRunner{}
	e := newTestExecutor(runner)
	_, err := e.executeRG(&domain.CodeIntelligenceConfig{
		Operation: "unknown_rg_op_xyz",
		Path:      "file.go",
	})
	if err == nil {
		t.Fatal("expected error for unsupported rg operation")
	}
}

// --- executeLSP dispatch tests (executor.go) ---

// TestExecuteLSP_DispatchSymbolSearch covers the symbolSearch case in executeLSP.
func TestExecuteLSP_DispatchSymbolSearch(t *testing.T) {
	resp := `{"jsonrpc":"2.0","id":1,"result":[{"name":"test","kind":"Function","location":{"uri":"file:///path/test.go"}}]}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	e := &Executor{}
	result, err := e.executeLSP(client, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpSymbolSearch,
		Query:     "test",
	}, "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

// TestExecuteLSP_DispatchDefinition covers the definition case in executeLSP.
func TestExecuteLSP_DispatchDefinition(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	resp := `{"jsonrpc":"2.0","id":1,"result":[{"uri":"file:///path/main.go","range":{"start":{"line":1,"character":5}}}]}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	result, err := e.executeLSP(client, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDefinition,
		Symbol:    "main",
		Path:      filePath,
	}, "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

// TestExecuteLSP_DispatchReferences covers the references case in executeLSP.
func TestExecuteLSP_DispatchReferences(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	resp := `{"jsonrpc":"2.0","id":1,"result":[{"uri":"file:///path/main.go","range":{"start":{"line":1,"character":5}}}]}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	result, err := e.executeLSP(client, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpReferences,
		Symbol:    "main",
		Path:      filePath,
	}, "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

// TestExecuteLSP_DispatchDiagnostics covers the diagnostics case in executeLSP.
func TestExecuteLSP_DispatchDiagnostics(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	e := &Executor{}

	resp := `{"jsonrpc":"2.0","id":1,"result":{"items":[{"message":"test","severity":1,"source":"go"}]}}`
	client := &lspClient{
		stdin:  &nopWriteCloser{Writer: &bytes.Buffer{}},
		stdout: bufio.NewReader(strings.NewReader(fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(resp), resp))),
	}
	result, err := e.executeLSP(client, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDiagnostics,
		Path:      filePath,
	}, "go")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := result.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

// --- runRG tests (executor.go) ---

// exitCodeOneRunner returns an exec.ExitError with exit code 1.
type exitCodeOneRunner struct{}

func (r *exitCodeOneRunner) Run(_ string, _ ...string) (string, string, error) {
	child := exec.Command("sh", "-c", "exit 1")
	var stdout, stderr bytes.Buffer
	child.Stdout = &stdout
	child.Stderr = &stderr
	if err := child.Run(); err != nil {
		return "", "", err
	}
	return "", "", nil
}

// TestRunRG_ExitCode1 covers the exit code 1 branch in runRG.
func TestRunRG_ExitCode1(t *testing.T) {
	e := newTestExecutor(&exitCodeOneRunner{})
	matches, err := e.runRG([]string{"test", "/path"})
	if err != nil {
		t.Fatalf("expected no error for exit code 1, got: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches for exit code 1, got %d", len(matches))
	}
}

// TestRunRG_InvalidJSON covers the JSON unmarshal continue branch in runRG.
func TestRunRG_InvalidJSON(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			"test /path": "not valid json\n",
		},
	}
	e := newTestExecutor(runner)
	matches, err := e.runRG([]string{"test", "/path"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(matches))
	}
}

// --- rgDefinition error path tests (executor.go) ---

// TestRgDefinition_RGError covers the runRG error return in rgDefinition.
func TestRgDefinition_RGError(t *testing.T) {
	runner := &mockRunner{rgErr: errors.New("rg failed")}
	e := newTestExecutor(runner)
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDefinition,
		Symbol:    "test",
		Path:      "/path",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRgReferences_RGError covers the runRG error return in rgReferences.
func TestRgReferences_RGError(t *testing.T) {
	runner := &mockRunner{rgErr: errors.New("rg failed")}
	e := newTestExecutor(runner)
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpReferences,
		Symbol:    "test",
		Path:      "/path",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRgDocumentSymbols_RGError covers the runRG error return in rgDocumentSymbols.
func TestRgDocumentSymbols_RGError(t *testing.T) {
	runner := &mockRunner{rgErr: errors.New("rg failed")}
	e := newTestExecutor(runner)
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
		Path:      "/path/file.py",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestGoDocumentSymbols_WithMatches covers the loop body in goDocumentSymbols.
func TestGoDocumentSymbols_WithMatches(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number --sort path ^(func |type |var |const |struct |interface ) main.go`: rgMatchJSON(
				"main.go",
				"func main() {}",
				1,
			),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
		Path:      "main.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

// TestRgHover_RGError covers the runRG error return in rgHover.
func TestRgHover_RGError(t *testing.T) {
	runner := &mockRunner{rgErr: errors.New("rg failed")}
	e := newTestExecutor(runner)
	_, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpHover,
		Symbol:    "test",
		Path:      "/path",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// TestRgHover_WithMatches covers the loop body in rgHover.
func TestRgHover_WithMatches(t *testing.T) {
	runner := &mockRunner{
		entries: map[string]string{
			`--json --line-number -B 5 -A 2 ^(func |type |var |const )?testFunc /path`: rgMatchJSON(
				"main.go",
				"func testFunc() {}",
				10,
			),
		},
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpHover,
		Symbol:    "testFunc",
		Path:      "/path",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

// --- goVetDiagnostics tests (executor.go) ---

// TestGoVetDiagnostics_WithPattern covers the config.Pattern case in goVetDiagnostics.
func TestGoVetDiagnostics_WithPattern(t *testing.T) {
	runner := &vetRunner{
		stderr: "",
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDiagnostics,
		Path:      "file.go",
		Pattern:   "somefile.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

// TestGoVetDiagnostics_RunError covers the err != nil branch in goVetDiagnostics.
func TestGoVetDiagnostics_RunError(t *testing.T) {
	runner := &vetRunner{
		stderr: "file.go:10:2: undefined: x\n",
		err:    errors.New("go vet failed"),
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDiagnostics,
		Path:      "file.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
}

// TestGoVetDiagnostics_NoColonLine covers the continue for lines without colon.
func TestGoVetDiagnostics_NoColonLine(t *testing.T) {
	runner := &vetRunner{
		stderr: "no colon here\nfile.go:10:6: actual error\n",
	}
	e := newTestExecutor(runner)
	res, err := e.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDiagnostics,
		Path:      "file.go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	m := res.(map[string]interface{})
	if m["success"] != true {
		t.Fatal("expected success true")
	}
	diags := m["diagnostics"].([]map[string]interface{})
	if len(diags) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(diags))
	}
}

// --- lsp_client.go tests ---

// TestLSPClient_Call_ReadResponseError covers the readResponse error path in call.
func TestLSPClient_Call_ReadResponseError(t *testing.T) {
	var stdin bytes.Buffer
	c := &lspClient{
		stdin:  &nopWriteCloser{Writer: &stdin},
		stdout: bufio.NewReader(strings.NewReader("Content-Length: 5\r\n\r\n{")), // truncated body
	}
	err := c.call("test", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "lsp: read test response") {
		t.Fatalf("expected read response error, got: %v", err)
	}
}

// secondWriteFailsWriter succeeds on the first Write and fails on the second.
type secondWriteFailsWriter struct {
	buf     bytes.Buffer
	written bool
}

func (w *secondWriteFailsWriter) Write(p []byte) (int, error) {
	if w.written {
		return 0, errors.New("write error")
	}
	w.written = true
	return w.buf.Write(p)
}

func (secondWriteFailsWriter) Close() error { return nil }

// TestLSPClient_Write_BodyWriteError covers the body write error path in write.
func TestLSPClient_Write_BodyWriteError(t *testing.T) {
	c := &lspClient{
		stdin: &secondWriteFailsWriter{},
	}
	// write() does: 1) json.Marshal, 2) io.WriteString(header), 3) stdin.Write(body)
	// secondWriteFailsWriter fails on the second call to Write.
	err := c.write(lspRequest{JSONRPC: "2.0", ID: 1, Method: "test"})
	if err == nil {
		t.Fatal("expected error on body write")
	}
}

// TestLSPClient_ReadMessage_NoBlankLine covers the header separator error in readMessage.
func TestLSPClient_ReadMessage_NoBlankLine(t *testing.T) {
	input := "Content-Length: 5\r\n" // no trailing \r\n after header
	c := &lspClient{
		stdout: bufio.NewReader(strings.NewReader(input)),
	}
	_, err := c.readMessage()
	if err == nil || !strings.Contains(err.Error(), "read header separator") {
		t.Fatalf("expected header separator error, got: %v", err)
	}
}

// countOpenFDs returns the number of currently open file descriptors (up to 2048).
func countOpenFDs() int {
	var count int
	for i := 0; i < 2048; i++ {
		var stat syscall.Stat_t
		if err := syscall.Fstat(i, &stat); err == nil {
			count++
		}
	}
	return count
}

// setRlimitNoFile sets RLIMIT_NOFILE to cur and registers cleanup to restore it.
func setRlimitNoFile(t *testing.T, cur uint64) {
	t.Helper()
	var oldLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &oldLimit); err != nil {
		t.Skipf("cannot get rlimit: %v", err)
	}
	newLimit := syscall.Rlimit{Cur: cur, Max: oldLimit.Max}
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &newLimit); err != nil {
		t.Skipf("cannot set rlimit to %d: %v", cur, err)
	}
	t.Cleanup(func() { _ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &oldLimit) })
}

// TestStartLSPClient_StdinPipeError covers the stdin pipe error path in startLSPClient.
// Uses RLIMIT_NOFILE to force os.Pipe to fail.
func TestStartLSPClient_StdinPipeError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource-intensive test in short mode")
	}
	cur := uint64(countOpenFDs() + 1) // only allow 1 more FD — os.Pipe needs 2
	setRlimitNoFile(t, cur)
	_, err := startLSPClient("echo", nil)
	if err == nil || !strings.Contains(err.Error(), "lsp: stdin pipe") {
		t.Fatalf("expected stdin pipe error, got: %v", err)
	}
}

// TestStartLSPClient_StdoutPipeError covers the stdout pipe error path in startLSPClient.
// Uses RLIMIT_NOFILE to allow stdin pipe to succeed but stdout pipe to fail.
func TestStartLSPClient_StdoutPipeError(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping resource-intensive test in short mode")
	}
	cur := uint64(countOpenFDs() + 2) // allow exactly 2 more FDs — stdin pipe consumes both
	setRlimitNoFile(t, cur)
	_, err := startLSPClient("echo", nil)
	if err == nil || !strings.Contains(err.Error(), "lsp: stdout pipe") {
		t.Fatalf("expected stdout pipe error, got: %v", err)
	}
}

// TestLSPClient_Close_Timeout covers the timeout/kill path in close.
func TestLSPClient_Close_Timeout(t *testing.T) {
	cmd := exec.Command("sleep", "10")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		t.Fatalf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		t.Fatalf("failed to start sleep: %v", err)
	}

	c := &lspClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewReader(stdout),
	}

	start := time.Now()
	c.close()
	elapsed := time.Since(start)

	// close should timeout after lspShutdownTimeout (5s) and kill the process
	if elapsed < 4*time.Second {
		t.Logf("close completed in %v (expected ~5s timeout)", elapsed)
	}
}

// --- lsp_manager.go tests ---

// TestLSPManager_GetServer_StartLSPError covers the startLSPClient error path in getServer.
func TestLSPManager_GetServer_StartLSPError(t *testing.T) {
	t.Setenv("PATH", "/nonexistent")

	m := &lspManager{
		cache:  make(map[string]*lspClient),
		lookup: func(s string) bool { return true },
	}
	_, err := m.getServer("go", "", "")
	if err == nil {
		t.Fatal("expected error when gopls not found")
	}
}

// TestLSPManager_GetServer_InitializeError covers the initialize error path in getServer.
func TestLSPManager_GetServer_InitializeError(t *testing.T) {
	dir := t.TempDir()
	// Create a fake gopls that exits immediately — startLSPClient succeeds
	// but the initialize call fails because the process is gone.
	fakeGopls := filepath.Join(dir, "gopls")
	if err := os.WriteFile(fakeGopls, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}

	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+oldPath)

	m := &lspManager{
		cache:  make(map[string]*lspClient),
		lookup: func(s string) bool { return true },
	}
	_, err := m.getServer("go", "", "")
	if err == nil {
		t.Fatal("expected initialize error")
	}
	if !strings.Contains(err.Error(), "lsp: initialize go") {
		t.Fatalf("expected initialize error, got: %v", err)
	}
}

// TestLSPManager_DetectServer_PyrightFallback covers the pyright fallback branch.
func TestLSPManager_DetectServer_PyrightFallback(t *testing.T) {
	m := &lspManager{
		cache:  make(map[string]*lspClient),
		lookup: func(s string) bool { return s == "pyright" },
	}
	info := m.detectServer("python")
	if info == nil {
		t.Fatal("expected server info for pyright fallback")
	}
	if info.bin != "pyright" {
		t.Fatalf("expected pyright, got %s", info.bin)
	}
}

// --- lsp_ops.go call-error tests ---
// These tests verify that when lspEnsureDocument's notify succeeds but
// the subsequent client.call fails, the error is properly returned.

// failOnThirdWriteWriter succeeds on the first two writes (notify header+body)
// and fails on the third write (call header).
type failOnThirdWriteWriter struct {
	buf   bytes.Buffer
	count int
}

func (w *failOnThirdWriteWriter) Write(p []byte) (int, error) {
	w.count++
	if w.count >= 3 {
		return 0, errors.New("write error")
	}
	return w.buf.Write(p)
}

func (failOnThirdWriteWriter) Close() error { return nil }

// TestLSPDefinition_CallError_WriteFails covers the client.call error path in lspDefinition.
func TestLSPDefinition_CallError_WriteFails(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	client := &lspClient{
		stdin: &failOnThirdWriteWriter{},
	}
	_, err := e.lspDefinition(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	if err == nil {
		t.Fatal("expected error from call failure")
	}
}

// TestLSPReferences_CallError_WriteFails covers the client.call error path in lspReferences.
func TestLSPReferences_CallError_WriteFails(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	client := &lspClient{
		stdin: &failOnThirdWriteWriter{},
	}
	_, err := e.lspReferences(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	if err == nil {
		t.Fatal("expected error from call failure")
	}
}

// TestLSPDocumentSymbols_CallError_WriteFails covers the client.call error path in lspDocumentSymbols.
func TestLSPDocumentSymbols_CallError_WriteFails(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &lspClient{
		stdin: &failOnThirdWriteWriter{},
	}
	e := &Executor{}
	_, err := e.lspDocumentSymbols(client, &domain.CodeIntelligenceConfig{
		Path: filePath,
	})
	if err == nil {
		t.Fatal("expected error from call failure")
	}
}

// TestLSPHover_CallError_WriteFails covers the client.call error path in lspHover.
func TestLSPHover_CallError_WriteFails(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	runner := &mockRunner{
		entries: map[string]string{
			"--json --line-number --max-count 1 main " + filePath: rgMatchJSON(filePath, "func main() {}", 1),
		},
	}
	e := NewExecutorWithRunner(runner)

	client := &lspClient{
		stdin: &failOnThirdWriteWriter{},
	}
	_, err := e.lspHover(client, &domain.CodeIntelligenceConfig{
		Symbol: "main",
		Path:   filePath,
	})
	if err == nil {
		t.Fatal("expected error from call failure")
	}
}

// TestLSPDiagnostics_CallError_WriteFails covers the client.call error path in lspDiagnostics.
func TestLSPDiagnostics_CallError_WriteFails(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "main.go")
	if err := os.WriteFile(filePath, []byte("package main\n"), 0644); err != nil {
		t.Fatal(err)
	}

	client := &lspClient{
		stdin: &failOnThirdWriteWriter{},
	}
	e := &Executor{}
	_, err := e.lspDiagnostics(client, &domain.CodeIntelligenceConfig{
		Path: filePath,
	})
	if err == nil {
		t.Fatal("expected error from call failure")
	}
}
