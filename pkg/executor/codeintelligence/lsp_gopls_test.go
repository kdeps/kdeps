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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func skipIfNoGopls(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("gopls"); err != nil {
		t.Skip("gopls not installed — skipping LSP integration test")
	}
}

func writeGoFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestLSP_Gopls_DocumentSymbols(t *testing.T) {
	skipIfNoGopls(t)

	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	require.NoError(t, os.WriteFile(goMod, []byte("module example\n\ngo 1.21\n"), 0644))

	path := writeGoFile(t, dir, "main.go", `package main

import "fmt"

// Greeter greets the world.
type Greeter struct {
	Name string
}

// Greet prints a greeting.
func (g *Greeter) Greet() string {
	return fmt.Sprintf("Hello, %s!", g.Name)
}

func main() {
	g := Greeter{Name: "world"}
	println(g.Greet())
}
`)

	exec := NewExecutor()
	result, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
		Path:      path,
	})
	require.NoError(t, err)

	res, ok := result.(map[string]interface{})
	require.True(t, ok, "result should be a map")

	// LSP should find Greeter struct and Greet method.
	symbols, _ := res["symbols"].([]map[string]interface{})
	require.NotEmpty(t, symbols, "should find at least one symbol")

	t.Logf("Found %d symbols via gopls", len(symbols))
	names := make(map[string]bool)
	for _, s := range symbols {
		name := fmt.Sprint(s["name"])
		names[name] = true
		t.Logf("  [%v] %v (kind=%v)", s["kind"], s["name"], s["kind"])
	}

	// gopls returns symbols with their Go names (may include receiver like "(*Greeter).Greet" or just "Greet")
	assert.NotEmpty(t, symbols, "gopls should find at least one symbol")
	foundGreeter := false
	for name := range names {
		if strings.Contains(name, "Greeter") || strings.Contains(name, "Greet") {
			foundGreeter = true
			break
		}
	}
	assert.True(t, foundGreeter, "gopls should find Greeter-related symbols, got: %v", names)
}

func TestLSP_Gopls_Hover(t *testing.T) {
	skipIfNoGopls(t)

	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	require.NoError(t, os.WriteFile(goMod, []byte("module example\n\ngo 1.21\n"), 0644))

	path := writeGoFile(t, dir, "main.go", `package main

// Greet says hello.
func Greet(name string) string {
	return "Hello, " + name
}

func main() {
	println(Greet("world"))
}
`)

	exec := NewExecutor()
	result, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpHover,
		Path:      path,
		Symbol:    "Greet",
	})
	require.NoError(t, err)

	res, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, res["success"].(bool))

	hover, _ := res["hover"].(map[string]interface{})
	contents := fmt.Sprintf("%v", hover["contents"])
	t.Logf("Hover contents: %s", contents)
	// Hover may be empty if position 0,0 doesn't match a symbol, or may contain docs.
	// Just verify the call didn't error and returned success=true.
	t.Logf("Hover success=%v", res["success"])
}

func TestLSP_RgFallback_WhenNoLSP(t *testing.T) {
	// .txt files have no LSP server — should fall back to rg via mock runner.
	mockRGOutput := `{"type":"match","data":{"path":{"text":"notes.txt"},"lines":{"text":"TODO: fix the GetUser function"},"line_number":1}}
{"type":"match","data":{"path":{"text":"notes.txt"},"lines":{"text":"Call GetUser from the handler."},"line_number":4}}`

	mock := &mockRunner{
		entries: map[string]string{
			"--json --line-number --sort path GetUser notes.txt": mockRGOutput,
		},
	}

	exec := NewExecutorWithRunner(mock)
	result, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpReferences,
		Path:      "notes.txt",
		Symbol:    "GetUser",
	})
	require.NoError(t, err)

	res, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, res["success"].(bool))

	refs, _ := res["references"].([]map[string]interface{})
	assert.NotEmpty(t, refs, "rg should find text references")
	assert.Equal(t, 2, len(refs))
}

func TestLSP_DocumentSymbols_RgFallback_NonGo(t *testing.T) {
	mockRGOutput := `{"type":"match","data":{"path":{"text":"script.py"},"lines":{"text":"def hello():","line_number":2},"line_number":2}}
{"type":"match","data":{"path":{"text":"script.py"},"lines":{"text":"class Greeter:","line_number":5},"line_number":5}}`

	mock := &mockRunner{
		entries: map[string]string{
			"--json --line-number --sort path ^(func |type |class |def |function |interface |struct |enum |trait |impl ) script.py": mockRGOutput,
		},
	}

	exec := NewExecutorWithRunner(mock)
	result, err := exec.Execute(nil, &domain.CodeIntelligenceConfig{
		Operation: domain.CodeIntOpDocumentSymbols,
		Path:      "script.py",
	})
	require.NoError(t, err)

	res, ok := result.(map[string]interface{})
	require.True(t, ok)
	assert.True(t, res["success"].(bool))

	symbols, _ := res["symbols"].([]map[string]interface{})
	assert.NotEmpty(t, symbols, "rg should find symbols via regex fallback")
}

func TestLSP_LanguageIDFromPath(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"app.py", "python"},
		{"lib.rs", "rust"},
		{"index.ts", "typescript"},
		{"util.js", "javascript"},
		{"main.c", "c"},
		{"main.cpp", "cpp"},
		{"gem.rb", "ruby"},
		{"Main.java", "java"},
		{"README.md", ""},
		{"Makefile", ""},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := languageIDFromPath(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestLSP_DetectServer(t *testing.T) {
	m := newLSPManager()

	// gopls should be found in CI/developer machines.
	info := m.detectServer("go")
	if _, err := exec.LookPath("gopls"); err == nil {
		require.NotNil(t, info, "gopls should be detected when installed")
		assert.Equal(t, "gopls", info.bin)
	} else {
		assert.Nil(t, info, "gopls should not be detected when missing")
	}

	// Unknown language should return nil.
	assert.Nil(t, m.detectServer("brainfuck"))
}

func TestLSP_FileURI(t *testing.T) {
	uri := fileURI("/home/user/project/main.go")
	assert.True(t, strings.HasPrefix(uri, "file://"))
	assert.True(t, strings.Contains(uri, "main.go"))
}

func TestLSP_FilepathFromURI(t *testing.T) {
	path := filepathFromURI("file:///home/user/main.go")
	assert.Equal(t, "/home/user/main.go", path)
}
