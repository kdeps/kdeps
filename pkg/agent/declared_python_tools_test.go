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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package agent

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

func TestValidateToolDefinition_OneImplementation(t *testing.T) {
	require.NoError(t, validateToolDefinition([]byte("description: d\npython:\n  script: print(1)\n"), "t"))
	require.Error(t, validateToolDefinition([]byte("description: d\npython: {}\n"), "t"), "python block needs a script")
	err := validateToolDefinition([]byte("description: d\npython:\n  script: print(1)\nexec:\n  command: git\n"), "t")
	require.ErrorContains(t, err, "at most one of")
}

// pythonTool returns a built-in python: tool, skipping when no python
// interpreter is installed.
func pythonTool(t *testing.T, name string, env map[string]string) *kdepstools.Tool {
	t.Helper()
	if _, err := exec.LookPath("python3"); err != nil {
		if _, err = exec.LookPath("python"); err != nil {
			t.Skip("no python interpreter")
		}
	}
	require.NotNil(t, toolDefinitions[name].Python, "%s has a python: block", name)
	return declaredTool(t, name, env)
}

func TestCalculator_RejectsHugeExponent(t *testing.T) {
	_, err := pythonTool(t, "calculator", nil).Execute(map[string]any{"expression": "9**9**9"})
	require.ErrorContains(t, err, "too large")
}

func TestMD5File_Declared(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a.txt")
	require.NoError(t, os.WriteFile(path, []byte("hello\n"), 0o600))
	tool := pythonTool(t, "md5_file", nil)
	out, err := tool.Execute(map[string]any{"file_path": path})
	require.NoError(t, err)
	assert.Equal(t, path+"  b1946ac92492d2347c6235b4d2611184", out)

	_, err = tool.Execute(map[string]any{"file_path": filepath.Join(t.TempDir(), "missing")})
	require.ErrorContains(t, err, "md5_file:")
	_, err = tool.Execute(map[string]any{})
	require.ErrorContains(t, err, "file_path is required")
}

func TestTailFile_Declared(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "log.txt")
	var sb strings.Builder
	for i := 1; i <= 25; i++ {
		fmt.Fprintf(&sb, "line %d\n", i)
	}
	require.NoError(t, os.WriteFile(path, []byte(sb.String()), 0o600))
	tool := pythonTool(t, "tail_file", nil)

	out, err := tool.Execute(map[string]any{"file_path": path})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(out, "6\tline 6\n"), out)
	assert.True(t, strings.HasSuffix(out, "25\tline 25\n[last 20/25 lines shown]"), out)

	out, err = tool.Execute(map[string]any{"file_path": path, "lines": float64(2)})
	require.NoError(t, err)
	assert.Equal(t, "24\tline 24\n25\tline 25\n[last 2/25 lines shown]", out)

	out, err = tool.Execute(map[string]any{"file_path": path, "lines": "100"})
	require.NoError(t, err)
	assert.NotContains(t, out, "lines shown", "the whole file fits")

	bin := filepath.Join(dir, "b.bin")
	require.NoError(t, os.WriteFile(bin, []byte{'a', 0, 'b'}, 0o600))
	_, err = tool.Execute(map[string]any{"file_path": bin})
	require.ErrorContains(t, err, "binary")
}

func TestListFiles_Declared(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "sub"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), nil, 0o600))
	tool := pythonTool(t, "list_files", nil)
	out, err := tool.Execute(map[string]any{"path": dir})
	require.NoError(t, err)
	abs, err := filepath.Abs(dir)
	require.NoError(t, err)
	assert.Equal(t, abs+":\n  [file] a.txt\n  [dir] sub", out)

	_, err = tool.Execute(map[string]any{"path": filepath.Join(dir, "missing")})
	require.ErrorContains(t, err, "list_files: read")
}

func TestGitOps_Declared(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	tool := pythonTool(t, "git_ops", nil)
	dir := t.TempDir()
	run := func(args map[string]any) map[string]any {
		t.Helper()
		args["workingDir"] = dir
		out, err := tool.Execute(args)
		require.NoError(t, err)
		var res map[string]any
		require.NoError(t, json.Unmarshal([]byte(out), &res), out)
		assert.Equal(t, true, res["success"])
		return res
	}
	run(map[string]any{"operation": "init"})
	for _, kv := range [][2]string{{"user.email", "t@example.com"}, {"user.name", "T"}} {
		cmd := exec.Command("git", "config", kv[0], kv[1])
		cmd.Dir = dir
		require.NoError(t, cmd.Run())
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("x\n"), 0o600))

	assert.Equal(t, []any{"a.txt"}, run(map[string]any{"operation": "status"})["untracked"])
	assert.Equal(t, true, run(map[string]any{"operation": "add", "paths": `'a.txt'`})["staged"])
	assert.Equal(t, []any{"a.txt"}, run(map[string]any{"operation": "status"})["staged"])
	dry := run(map[string]any{"operation": "commit", "message": "first", "dryRun": true})
	assert.Equal(t, false, dry["committed"])
	assert.Equal(t, true, run(map[string]any{"operation": "commit", "message": "first"})["committed"])

	log := run(map[string]any{"operation": "log", "maxCount": float64(5)})
	assert.InDelta(t, 1, log["count"], 0)
	commits, _ := log["commits"].([]any)
	require.Len(t, commits, 1)
	assert.Equal(t, "first", commits[0].(map[string]any)["message"])

	_, err := tool.Execute(map[string]any{"operation": "rebase", "workingDir": dir})
	require.ErrorContains(t, err, "unsupported operation 'rebase'")
	_, err = tool.Execute(map[string]any{"operation": "commit", "workingDir": dir})
	require.ErrorContains(t, err, "commit message is required")
	_, err = tool.Execute(map[string]any{"operation": "show", "args": "nosuchrev", "workingDir": dir})
	require.ErrorContains(t, err, "git show failed")
}

func TestWebSearch_Declared(t *testing.T) {
	var gotQuery, gotUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery, gotUA = r.URL.Query().Get("q"), r.UserAgent()
		_, _ = io.WriteString(w, `<html><body>
<div class="result web-result"><h2><a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fgo.dev%2F&amp;rut=x">The <b>Go</b> Language</a></h2>
<a class="result__snippet" href="#">Build simple, <b>secure</b>
 software.</a></div>
<table><tr><td><a rel="nofollow" href="https://example.com/" class='result-link'>Example</a></td></tr>
<tr><td class='result-snippet'>Lite page snippet.</td></tr></table>
</body></html>`)
	}))
	t.Cleanup(srv.Close)
	tool := pythonTool(t, "web_search", map[string]string{"KDEPS_WEB_SEARCH_URL": srv.URL + "/html/"})
	out, err := tool.Execute(map[string]any{"query": "go lang"})
	require.NoError(t, err)
	assert.Equal(t, "go lang", gotQuery)
	assert.Equal(t, "kdeps/agent", gotUA)
	assert.Equal(t, "Title: The Go Language\nDescription: Build simple, secure software.\nURL: https://go.dev/\n\n"+
		"Title: Example\nDescription: Lite page snippet.\nURL: https://example.com/", out)
}

func TestWebScraper_Declared(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = io.WriteString(w, `<html><head><title>Docs</title><meta name="description" content="About it">
<script>var hidden = 1;</script></head><body><h1>Intro</h1><p>First   para.</p>
<p>Second <a href="/next">link</a>.</p></body></html>`)
	}))
	t.Cleanup(srv.Close)
	tool := pythonTool(t, "web_scraper", nil)
	out, err := tool.Execute(map[string]any{"url": srv.URL})
	require.NoError(t, err)
	assert.Equal(t, "Page URL: "+srv.URL+"\nPage Title: Docs\nPage Description: About it\nHeaders:\nIntro\n"+
		"Content:\nFirst para.\nSecond link.\nLink: /next", out)

	_, err = tool.Execute(map[string]any{"url": "file:///etc/passwd"})
	require.ErrorContains(t, err, "not an http(s) URL")
}

func TestWikipedia_Declared(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.URL.Query()
		if got.Get("gsrsearch") == "nothing" {
			_, _ = io.WriteString(w, `{"batchcomplete":true}`)
			return
		}
		_, _ = io.WriteString(w, `{"query":{"pages":[`+
			`{"index":2,"title":"Second","extract":"Two."},{"index":1,"title":"First","extract":"One."}]}}`)
	}))
	t.Cleanup(srv.Close)
	tool := pythonTool(t, "wikipedia", map[string]string{"KDEPS_WIKIPEDIA_URL": srv.URL})
	out, err := tool.Execute(map[string]any{"query": "golang"})
	require.NoError(t, err)
	assert.Equal(t, "First\nOne.\n\nSecond\nTwo.", out, "pages keep the search rank")
	assert.Equal(t, "golang", got.Get("gsrsearch"))
	assert.Equal(t, "extracts", got.Get("prop"))

	out, err = tool.Execute(map[string]any{"query": "nothing"})
	require.NoError(t, err)
	assert.Equal(t, "no wikipedia pages found", out)
}
