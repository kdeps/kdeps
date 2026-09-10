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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

func editFileTool(t *testing.T) *kdepstools.Tool {
	t.Helper()
	reg := kdepstools.NewRegistry()
	registerEditFile(reg)
	tool := reg.Get("edit_file")
	require.NotNil(t, tool)
	return tool
}

// writeSeenFile creates a temp file and marks it read this turn, so a
// str_replace/insert test does not have to view it first.
func writeSeenFile(t *testing.T, name, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(f, []byte(content), 0o600))
	markFileSeen(f)
	return f
}

func TestEditFile_StrReplace(t *testing.T) {
	f := writeSeenFile(t, "m.txt", "alpha\nbeta\ngamma\n")
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f,
		"old_str": "beta", "new_str": "BETA",
	})
	require.NoError(t, err)
	assert.Contains(t, res, "2\tBETA")
	got, _ := os.ReadFile(f)
	assert.Equal(t, "alpha\nBETA\ngamma\n", string(got))
}

func TestEditFile_StrReplace_MustMatchVerbatim(t *testing.T) {
	f := writeSeenFile(t, "v.go", "func f() {\n\treturn 1\n}\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f,
		// multi-line old_str with the tab dropped on line 2 - not a substring
		"old_str": "func f() {\nreturn 1", "new_str": "func f() {\n\treturn 2",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not appear verbatim")
	got, _ := os.ReadFile(f)
	assert.Equal(t, "func f() {\n\treturn 1\n}\n", string(got), "file untouched on a miss")
}

func TestEditFile_StrReplace_MustBeUnique(t *testing.T) {
	f := writeSeenFile(t, "d.txt", "x = 1\ny = 2\nx = 1\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f,
		"old_str": "x = 1", "new_str": "x = 9",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "appears 2 times")
	assert.Contains(t, err.Error(), "line(s) 1, 3")
}

func TestEditFile_StrReplace_RequiresPriorRead(t *testing.T) {
	// No markFileSeen -> the gate fires.
	f := filepath.Join(t.TempDir(), "ng.txt")
	require.NoError(t, os.WriteFile(f, []byte("a\nb\n"), 0o600))
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "a", "new_str": "A",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nb\n", string(got))
}

func TestEditFile_View_MarksSeen_ThenStrReplaceWorks(t *testing.T) {
	f := filepath.Join(t.TempDir(), "s.txt")
	require.NoError(t, os.WriteFile(f, []byte("one\ntwo\nthree\n"), 0o600))
	tool := editFileTool(t)

	view, err := tool.Execute(map[string]any{"command": "view", "file_path": f})
	require.NoError(t, err)
	assert.Contains(t, view, "2\ttwo")

	_, err = tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "two", "new_str": "TWO",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "one\nTWO\nthree\n", string(got))
}

func TestEditFile_ViewRange(t *testing.T) {
	f := filepath.Join(t.TempDir(), "r.txt")
	require.NoError(t, os.WriteFile(f, []byte("a\nb\nc\nd\ne\n"), 0o600))
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "view", "file_path": f, "view_range": []any{float64(2), float64(4)},
	})
	require.NoError(t, err)
	assert.Equal(t, "2\tb\n3\tc\n4\td", res)
}

func TestEditFile_ViewRange_ToEnd(t *testing.T) {
	f := filepath.Join(t.TempDir(), "r2.txt")
	require.NoError(t, os.WriteFile(f, []byte("a\nb\nc\n"), 0o600))
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "view", "file_path": f, "view_range": []any{float64(2), float64(-1)},
	})
	require.NoError(t, err)
	assert.Equal(t, "2\tb\n3\tc", res)
}

func TestEditFile_Insert(t *testing.T) {
	f := writeSeenFile(t, "i.txt", "a\nb\nc\n")
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "insert_line": float64(2), "new_str": "B2",
	})
	require.NoError(t, err)
	assert.Contains(t, res, "3\tB2")
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nb\nB2\nc\n", string(got))
}

func TestEditFile_Insert_Top(t *testing.T) {
	f := writeSeenFile(t, "it.txt", "a\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "insert_line": float64(0), "new_str": "HEAD",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "HEAD\na\nb\n", string(got))
}

func TestEditFile_Insert_OutOfRange(t *testing.T) {
	f := writeSeenFile(t, "io.txt", "a\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "insert_line": float64(9), "new_str": "x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestEditFile_UndoEdit(t *testing.T) {
	f := writeSeenFile(t, "u.txt", "a\nb\nc\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "b", "new_str": "B",
	})
	require.NoError(t, err)

	_, err = tool.Execute(map[string]any{"command": "undo_edit", "file_path": f})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nb\nc\n", string(got))
}

func TestEditFile_UndoEdit_NoHistory(t *testing.T) {
	f := writeSeenFile(t, "un.txt", "a\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{"command": "undo_edit", "file_path": f})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no edit history")
}

func TestEditFile_UnknownCommand(t *testing.T) {
	f := writeSeenFile(t, "x.txt", "a\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{"command": "rewrite", "file_path": f, "new_str": "z"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestEditFile_InfersStrReplace(t *testing.T) {
	f := writeSeenFile(t, "inf.txt", "a\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"file_path": f, "old_str": "b", "new_str": "B", // no command
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nB\n", string(got))
}

// The kdeps param names still work via the alias table.
func TestEditFile_KdepsParamAliases(t *testing.T) {
	f := writeSeenFile(t, "al.txt", "a\nb\n")
	reg := kdepstools.NewRegistry()
	registerEditFile(reg)
	registerToolAliases(reg)
	tool := reg.Get("edit_file")
	args := map[string]any{
		"command": "str_replace", "path": f,
		"old_string": "b", "new_string": "B",
	}
	normalizeToolArgs("edit_file", args)
	_, err := tool.Execute(args)
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nB\n", string(got))
}

func TestEditFile_SnippetLineNumbersAccurate(t *testing.T) {
	body := "l1\nl2\nl3\nl4\nl5\nl6\nl7\nl8\nl9\nl10\n"
	f := writeSeenFile(t, "acc.txt", body)
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "l6", "new_str": "SIX",
	})
	require.NoError(t, err)
	// The changed line must be numbered 6, and its neighbours 5 and 7.
	assert.Contains(t, res, "5\tl5")
	assert.Contains(t, res, "6\tSIX")
	assert.Contains(t, res, "7\tl7")
	assert.NotContains(t, res, "\tl6")
}

func TestSplitKeepCount(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, splitKeepCount("a\nb\n"))
	assert.Equal(t, []string{"a", "b"}, splitKeepCount("a\nb"))
	assert.Equal(t, []string{}, splitKeepCount(""))
}

func TestEditFile_NoChangeRejected(t *testing.T) {
	f := writeSeenFile(t, "nc.txt", "keep\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "keep", "new_str": "keep",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "identical")
}

func TestEditFile_ChainedEditsNoReReadNeeded(t *testing.T) {
	f := writeSeenFile(t, "chain.txt", "a\nb\nc\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "a", "new_str": "A",
	})
	require.NoError(t, err)
	// second edit without re-reading: commitEdit re-marked the file seen
	_, err = tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "c", "new_str": "C",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "A\nb\nC\n", string(got))
}
