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

package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

func TestFindEditTargets_Exact(t *testing.T) {
	m, s := findEditTargets("alpha beta gamma", "beta")
	require.Len(t, m, 1)
	assert.Equal(t, matchExact, s)
	assert.Equal(t, 6, m[0].start)
	assert.Equal(t, 10, m[0].end)
}

func TestFindEditTargets_TrailingWhitespaceAndCRLF(t *testing.T) {
	// File has a trailing space and CRLF; old_string has neither. The matched
	// span is the line content without its \r\n terminator.
	content := "func x() {\r\n    return 1 \r\n}\r\n"
	old := "    return 1\n"
	m, s := findEditTargets(content, old)
	require.Len(t, m, 1)
	assert.Equal(t, matchTrailingWS, s)
	assert.Equal(t, "    return 1 ", content[m[0].start:m[0].end])
}

func TestFindEditTargets_MissingFinalNewline(t *testing.T) {
	content := "line one\nline two"
	m, s := findEditTargets(content, "line two\n")
	require.Len(t, m, 1)
	assert.Equal(t, matchTrailingWS, s)
	assert.Equal(t, "line two", content[m[0].start:m[0].end])
}

func TestFindEditTargets_Indentation(t *testing.T) {
	content := "if a {\n        doThing()\n        doOther()\n}\n"
	old := "doThing()\ndoOther()" // model dropped all indentation
	m, s := findEditTargets(content, old)
	require.Len(t, m, 1)
	assert.Equal(t, matchIndentation, s)
	assert.Equal(t, "        doThing()\n        doOther()", content[m[0].start:m[0].end])
}

func TestFindEditTargets_BlankOldNeverLooseMatches(t *testing.T) {
	// A whitespace-only old_string with no exact occurrence never resolves
	// via the loose strategies (it would match any gap in the file).
	content := "a\nx\ny\nb\n"
	m, s := findEditTargets(content, "\n\n")
	assert.Empty(t, m)
	assert.Equal(t, matchNone, s)

	m, s = findEditTargets(content, "    \n    ")
	assert.Empty(t, m)
	assert.Equal(t, matchNone, s)
}

func TestFindEditTargets_ExactWinsWhenLooseWouldBeAmbiguous(t *testing.T) {
	// "x = 1" is a literal substring of both lines, so the exact strategy
	// returns two matches and the looser strategies are never consulted.
	content := "x = 1 \ny = 2\nx = 1\n"
	m, s := findEditTargets(content, "x = 1")
	require.Len(t, m, 2)
	assert.Equal(t, matchExact, s)
}

func TestFindEditTargets_LooseAmbiguous(t *testing.T) {
	// "a = 1\n" is not a literal substring (trailing space / tab before the
	// newline), so it resolves via the trailing-whitespace strategy - twice.
	m, s := findEditTargets("a = 1 \nb\na = 1\t\n", "a = 1\n")
	require.Len(t, m, 2)
	assert.Equal(t, matchTrailingWS, s)
}

func TestReindentReplacement(t *testing.T) {
	// old written flush-left, file indented 8 -> replacement gains 8 spaces.
	got := reindentReplacement("newThing()\nmore()", "doThing()", "        doThing()")
	assert.Equal(t, "        newThing()\n        more()", got)

	// old over-indented by 4 relative to file -> replacement loses 4.
	got = reindentReplacement("    a()\n    b()", "    x()", "x()")
	assert.Equal(t, "a()\nb()", got)

	// equal indent -> untouched.
	got = reindentReplacement("    a()", "    x()", "    y()")
	assert.Equal(t, "    a()", got)
}

func TestSpliceMatches(t *testing.T) {
	content := "aXbXc"
	m := exactMatches(content, "X")
	assert.Equal(t, "a-b-c", spliceMatches(content, m, "-"))
}

func TestMatchEOLStyle(t *testing.T) {
	assert.Equal(t, "a\r\nb", matchEOLStyle("a\nb", "x\r\ny"))
	assert.Equal(t, "a\nb", matchEOLStyle("a\nb", "x\ny"))
	assert.Equal(t, "a\r\nb", matchEOLStyle("a\r\nb", "x\r\ny"))
}

func TestNearMissHint(t *testing.T) {
	content := "package main\n\nfunc main() {\n\tprintln(\"hi\")\n}\n"
	hint := nearMissHint(content, "\tprintln(\"hello\")")
	assert.Contains(t, hint, "Closest match is near line 4")
	assert.Contains(t, hint, `println("hi")`)
}

// --- end-to-end through the registered tool ---

func editFileTool(t *testing.T) *kdepstools.Tool {
	t.Helper()
	reg := kdepstools.NewRegistry()
	registerEditFile(reg)
	tool := reg.Get("edit_file")
	require.NotNil(t, tool)
	return tool
}

func TestEditFile_LooseIndentationEdit(t *testing.T) {
	f := filepath.Join(t.TempDir(), "m.go")
	require.NoError(t, os.WriteFile(f,
		[]byte("func f() {\n\tif ok {\n\t\ta()\n\t\tb()\n\t}\n}\n"), 0o600))

	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"file_path":  f,
		"old_string": "a()\nb()", // model dropped the two tabs of indentation
		"new_string": "a()\nc()\nb()",
	})
	require.NoError(t, err)
	assert.Contains(t, res, "indentation")

	got, _ := os.ReadFile(f)
	assert.Equal(t,
		"func f() {\n\tif ok {\n\t\ta()\n\t\tc()\n\t\tb()\n\t}\n}\n", string(got),
		"new_string is reindented to the file's two-tab level")
}

func TestEditFile_CRLFPreserved(t *testing.T) {
	f := filepath.Join(t.TempDir(), "w.txt")
	require.NoError(t, os.WriteFile(f, []byte("one\r\ntwo\r\nthree\r\n"), 0o600))

	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"file_path":  f,
		"old_string": "two\n",
		"new_string": "TWO\n",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "one\r\nTWO\r\nthree\r\n", string(got))
}

func TestEditFile_ReplaceAll(t *testing.T) {
	f := filepath.Join(t.TempDir(), "r.txt")
	// Trailing space / tab means "x = 1\n" is not a literal substring; it
	// resolves loosely on all three lines.
	require.NoError(t, os.WriteFile(f, []byte("x = 1 \nx = 1\t\nx = 1 \n"), 0o600))

	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"file_path":   f,
		"old_string":  "x = 1\n",
		"new_string":  "x = 2\n",
		"replace_all": true,
	})
	require.NoError(t, err)
	assert.Contains(t, res, "3 passages")
	got, _ := os.ReadFile(f)
	assert.Equal(t, "x = 2\nx = 2\nx = 2\n", string(got))
}

func TestEditFile_AmbiguousWithoutReplaceAll(t *testing.T) {
	f := filepath.Join(t.TempDir(), "a.txt")
	require.NoError(t, os.WriteFile(f, []byte("v = 1 \nv = 1\t\n"), 0o600))

	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"file_path":  f,
		"old_string": "v = 1\n",
		"new_string": "v = 2\n",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matches 2 passages")
}

func TestEditFile_NotFoundHint(t *testing.T) {
	f := filepath.Join(t.TempDir(), "h.go")
	require.NoError(t, os.WriteFile(f,
		[]byte("func main() {\n\tprintln(\"hi\")\n}\n"), 0o600))

	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"file_path":  f,
		"old_string": "println(\"nope\")",
		"new_string": "println(\"yep\")",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Closest match is near line 2")
}

func TestEditFile_BlankOldString(t *testing.T) {
	f := filepath.Join(t.TempDir(), "b.txt")
	require.NoError(t, os.WriteFile(f, []byte("data"), 0o600))
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"file_path":  f,
		"old_string": "",
		"new_string": "x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "old_string is required")
}

func TestEditFile_ExactStillUnique(t *testing.T) {
	f := filepath.Join(t.TempDir(), "u.txt")
	require.NoError(t, os.WriteFile(f, []byte("cat cat"), 0o600))
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"file_path":  f,
		"old_string": "cat",
		"new_string": "dog",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matches 2 passages")
}

func TestEditFile_LooseDiffUsesActualFileText(t *testing.T) {
	f := filepath.Join(t.TempDir(), "d.txt")
	require.NoError(t, os.WriteFile(f, []byte("hello world   \n"), 0o600))
	tool := editFileTool(t)
	var buf strings.Builder
	tool.OutputWriter = &buf
	res, err := tool.Execute(map[string]any{
		"file_path":  f,
		"old_string": "hello world\n", // model omitted the trailing spaces
		"new_string": "hi there\n",
	})
	require.NoError(t, err)
	assert.Contains(t, res, "trailing-whitespace")
	// The diff shows the matched line as it really is in the file.
	assert.Contains(t, buf.String(), "hello world   ")
	got, _ := os.ReadFile(f)
	assert.Equal(t, "hi there\n", string(got))
}

func TestEditFile_MultilineReplacementReindented(t *testing.T) {
	f := filepath.Join(t.TempDir(), "n.go")
	require.NoError(t, os.WriteFile(f,
		[]byte("func g() {\n    step()\n    done()\n}\n"), 0o600))
	tool := editFileTool(t)
	// old_string is two lines with no indentation -> resolves via the
	// indentation strategy, and new_string is reindented to match.
	_, err := tool.Execute(map[string]any{
		"file_path":  f,
		"old_string": "step()\ndone()",
		"new_string": "first()\nsecond()\ndone()",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t,
		"func g() {\n    first()\n    second()\n    done()\n}\n", string(got))
}
