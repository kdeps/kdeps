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

// TestEditFile_StrReplace_NearMissHintShowsClosestLines covers the "did not
// appear verbatim" error's near-miss suffix: the closest actual line(s) in
// the file, so the model sees exactly where its reproduction diverged
// (here, a dropped tab on line 2) instead of a generic "go view the file"
// message with nothing to act on.
func TestEditFile_StrReplace_NearMissHintShowsClosestLines(t *testing.T) {
	f := writeSeenFile(t, "v.go", "func f() {\n\treturn 1\n}\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f,
		"old_str": "func f() {\nreturn 1", "new_str": "func f() {\n\treturn 2",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Closest match")
	assert.Contains(t, err.Error(), "line 1")
	assert.Contains(t, err.Error(), "\treturn 1", "the actual (tab-indented) line must be shown verbatim")
}

// TestEditFile_StrReplace_NearMissHintAbsentWhenNoOverlap covers old_str that
// shares nothing with the file: no near-miss suffix should be added, since
// there's nothing useful to show.
func TestEditFile_StrReplace_NearMissHintAbsentWhenNoOverlap(t *testing.T) {
	f := writeSeenFile(t, "v.go", "func f() {\n\treturn 1\n}\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f,
		"old_str": "totally different content", "new_str": "x",
	})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "Closest match")
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
	assert.True(t, strings.HasPrefix(res, "2\tb\n3\tc\n4\td"))
	assert.Contains(t, res, "[revision sha256:")
}

func TestEditFile_ViewRange_ToEnd(t *testing.T) {
	f := filepath.Join(t.TempDir(), "r2.txt")
	require.NoError(t, os.WriteFile(f, []byte("a\nb\nc\n"), 0o600))
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "view", "file_path": f, "view_range": []any{float64(2), float64(-1)},
	})
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(res, "2\tb\n3\tc"))
	assert.Contains(t, res, "[revision sha256:")
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

func TestEditFile_Insert_BeforeAnchor(t *testing.T) {
	f := writeSeenFile(t, "ib.txt", "a\nTARGET\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "insert_before_anchor": "TARGET", "new_str": "X",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nX\nTARGET\nb\n", string(got))
}

func TestEditFile_Insert_AfterAnchor(t *testing.T) {
	f := writeSeenFile(t, "ia.txt", "a\nTARGET\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "insert_after_anchor": "TARGET", "new_str": "X",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nTARGET\nX\nb\n", string(got))
}

func TestEditFile_Insert_AfterMultilineAnchor(t *testing.T) {
	f := writeSeenFile(t, "iam.txt", "a\nT1\nT2\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "insert_after_anchor": "T1\nT2", "new_str": "X",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nT1\nT2\nX\nb\n", string(got))
}

func TestEditFile_Insert_AnchorAmbiguousRejected(t *testing.T) {
	f := writeSeenFile(t, "iamb.txt", "dup\nmid\ndup\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "insert_after_anchor": "dup", "new_str": "X",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "appears 2 times")
}

func TestEditFile_Insert_AnchorNotFoundRejected(t *testing.T) {
	f := writeSeenFile(t, "iamnf.txt", "a\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "insert_after_anchor": "zzz", "new_str": "X",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not appear verbatim")
}

func TestEditFile_Insert_NoPositionGivenRejected(t *testing.T) {
	f := writeSeenFile(t, "ino.txt", "a\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "new_str": "X",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "insert_line")
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

func TestClosestMatch_PicksHighestScoringWindow(t *testing.T) {
	content := "one\ntwo\nthree\nfour\nfive\n"
	snippet, atLine := closestMatch(content, "two\nthree")
	assert.Equal(t, "two\nthree", snippet)
	assert.Equal(t, 2, atLine)
}

func TestClosestMatch_IgnoresSurroundingWhitespaceDifferences(t *testing.T) {
	content := "func f() {\n\treturn 1\n}\n"
	// old_str's second line is missing the tab -- still the closest window.
	snippet, atLine := closestMatch(content, "func f() {\nreturn 1")
	assert.Equal(t, "func f() {\n\treturn 1", snippet)
	assert.Equal(t, 1, atLine)
}

func TestClosestMatch_NoOverlapReturnsEmpty(t *testing.T) {
	snippet, atLine := closestMatch("a\nb\nc\n", "x\ny")
	assert.Equal(t, "", snippet)
	assert.Equal(t, 0, atLine)
}

func TestClosestMatch_OldStrLongerThanFileReturnsEmpty(t *testing.T) {
	snippet, atLine := closestMatch("a\n", "a\nb\nc")
	assert.Equal(t, "", snippet)
	assert.Equal(t, 0, atLine)
}

func TestClosestMatch_EmptyOldStrReturnsEmpty(t *testing.T) {
	// strings.Split("", "\n") is []string{""} (len 1, not 0), so this exercises
	// the "no window scores above zero" path rather than the len==0 guard --
	// worth pinning down since an empty old_str is otherwise rejected earlier
	// in editStrReplace, before nearMissHint is ever reached.
	snippet, atLine := closestMatch("a\nb\n", "")
	assert.Equal(t, "", snippet)
	assert.Equal(t, 0, atLine)
}

func TestClosestMatch_OldStrOverHintCapReturnsEmpty(t *testing.T) {
	huge := strings.Repeat("x\n", nearMissHintMaxLines+1)
	snippet, atLine := closestMatch("a\nb\nc\n", huge)
	assert.Equal(t, "", snippet)
	assert.Equal(t, 0, atLine)
}

func TestNearMissHint_EmptyWhenNoMatch(t *testing.T) {
	assert.Equal(t, "", nearMissHint("a\nb\nc\n", "x\ny"))
}

func TestNearMissHint_IncludesLineNumberAndSnippet(t *testing.T) {
	got := nearMissHint("one\ntwo\nthree\n", "two\nthree")
	assert.Contains(t, got, "line 2")
	assert.Contains(t, got, "two\nthree")
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

// --- revision ---

func TestEditFile_Revision_MatchingRevisionSucceeds(t *testing.T) {
	f := writeSeenFile(t, "rev.txt", "a\nb\n")
	tool := editFileTool(t)
	view, err := tool.Execute(map[string]any{"command": "view", "file_path": f})
	require.NoError(t, err)
	rev := extractRevision(t, view)

	_, err = tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "b", "new_str": "B", "revision": rev,
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nB\n", string(got))
}

func TestEditFile_Revision_StaleRevisionRejected(t *testing.T) {
	f := writeSeenFile(t, "rev2.txt", "a\nb\n")
	tool := editFileTool(t)
	view, err := tool.Execute(map[string]any{"command": "view", "file_path": f})
	require.NoError(t, err)
	staleRev := extractRevision(t, view)

	// File changes out-of-band after the view.
	require.NoError(t, os.WriteFile(f, []byte("a\nb\nc\n"), 0o600))
	markFileSeen(f)

	_, err = tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "b", "new_str": "B", "revision": staleRev,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "changed since revision")
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nb\nc\n", string(got), "file untouched on a stale revision")
}

func TestEditFile_Revision_OmittedSkipsCheck(t *testing.T) {
	f := writeSeenFile(t, "rev3.txt", "a\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "b", "new_str": "B",
	})
	require.NoError(t, err)
}

// extractRevision pulls the "[revision sha256:...]" token out of an edit_file
// result string.
func extractRevision(t *testing.T, out string) string {
	t.Helper()
	i := strings.Index(out, "[revision ")
	require.NotEqual(t, -1, i, "output missing revision marker: %q", out)
	rest := out[i+len("[revision "):]
	j := strings.Index(rest, "]")
	require.NotEqual(t, -1, j)
	return rest[:j]
}

// --- occurrence ---

func TestEditFile_Occurrence_PicksTheRequestedMatch(t *testing.T) {
	f := writeSeenFile(t, "occ.txt", "x = 1\ny = 2\nx = 1\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f,
		"old_str": "x = 1", "new_str": "x = 9", "occurrence": float64(2),
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "x = 1\ny = 2\nx = 9\n", string(got))
}

func TestEditFile_Occurrence_OutOfRangeRejected(t *testing.T) {
	f := writeSeenFile(t, "occ2.txt", "x = 1\ny = 2\nx = 1\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f,
		"old_str": "x = 1", "new_str": "x = 9", "occurrence": float64(3),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestEditFile_Occurrence_MissingHintSuggestsOccurrence(t *testing.T) {
	f := writeSeenFile(t, "occ3.txt", "x = 1\ny = 2\nx = 1\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "x = 1", "new_str": "x = 9",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "occurrence: 1-2")
}

// --- anchors ---

func TestEditFile_View_Anchor_ShowsSurroundingContext(t *testing.T) {
	body := strings.Join([]string{"l1", "l2", "l3", "TARGET", "l5", "l6", "l7"}, "\n") + "\n"
	f := writeSeenFile(t, "anchor.txt", body)
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "view", "file_path": f, "anchor": "TARGET",
		"context_before": float64(1), "context_after": float64(1),
	})
	require.NoError(t, err)
	assert.Contains(t, res, "3\tl3")
	assert.Contains(t, res, "4\tTARGET")
	assert.Contains(t, res, "5\tl5")
	assert.NotContains(t, res, "\tl2")
	assert.NotContains(t, res, "\tl6")
}

func TestEditFile_View_Anchor_AmbiguousRejected(t *testing.T) {
	f := writeSeenFile(t, "anchor2.txt", "dup\nmid\ndup\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{"command": "view", "file_path": f, "anchor": "dup"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "appears 2 times")
}

func TestEditFile_View_Anchor_NotFoundRejected(t *testing.T) {
	f := writeSeenFile(t, "anchor3.txt", "a\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{"command": "view", "file_path": f, "anchor": "zzz"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "did not appear verbatim")
}

func TestEditFile_StrReplace_AnchorRangeReplacesBetween(t *testing.T) {
	body := "func f() {\n\tSTART\n\tmid1\n\tmid2\n\tEND\n}\n"
	f := writeSeenFile(t, "anchor4.txt", body)
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f,
		"start_anchor": "START", "end_anchor": "END", "new_str": "REPLACED",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "func f() {\n\tREPLACED\n}\n", string(got))
}

func TestEditFile_StrReplace_EndAnchorRequiredWithStartAnchor(t *testing.T) {
	f := writeSeenFile(t, "anchor5.txt", "START\nmid\nEND\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "start_anchor": "START", "new_str": "x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "end_anchor is required")
}

// --- patch ---

func TestEditFile_Patch_SingleHunkApplies(t *testing.T) {
	f := writeSeenFile(t, "p1.txt", "one\ntwo\nthree\n")
	tool := editFileTool(t)
	patch := "@@ -1,3 +1,3 @@\n one\n-two\n+TWO\n three\n"
	_, err := tool.Execute(map[string]any{"command": "patch", "file_path": f, "patch": patch})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "one\nTWO\nthree\n", string(got))
}

func TestEditFile_Patch_MultiHunkAppliesAtomically(t *testing.T) {
	f := writeSeenFile(t, "p2.txt", "a\nb\nc\nd\ne\n")
	tool := editFileTool(t)
	patch := "@@ -1,1 +1,1 @@\n-a\n+A\n@@ -5,1 +5,1 @@\n-e\n+E\n"
	_, err := tool.Execute(map[string]any{"command": "patch", "file_path": f, "patch": patch})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "A\nb\nc\nd\nE\n", string(got))
}

func TestEditFile_Patch_HunkNotFoundNamesHunk(t *testing.T) {
	f := writeSeenFile(t, "p3.txt", "one\ntwo\nthree\n")
	tool := editFileTool(t)
	patch := "@@ -1,1 +1,1 @@\n-nope\n+NOPE\n"
	_, err := tool.Execute(map[string]any{"command": "patch", "file_path": f, "patch": patch})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "hunk 1")
	assert.Contains(t, err.Error(), "did not match")
}

func TestEditFile_Patch_AmbiguousHunkRejected(t *testing.T) {
	f := writeSeenFile(t, "p4.txt", "dup\nmid\ndup\n")
	tool := editFileTool(t)
	patch := "@@ -1,1 +1,1 @@\n-dup\n+DUP\n"
	_, err := tool.Execute(map[string]any{"command": "patch", "file_path": f, "patch": patch})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "matches 2 times")
}

func TestEditFile_Patch_NoHunksRejected(t *testing.T) {
	f := writeSeenFile(t, "p5.txt", "a\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{"command": "patch", "file_path": f, "patch": "not a diff"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no hunks found")
}

func TestEditFile_Patch_RequiresPriorRead(t *testing.T) {
	f := filepath.Join(t.TempDir(), "p6.txt")
	require.NoError(t, os.WriteFile(f, []byte("a\nb\n"), 0o600))
	tool := editFileTool(t)
	patch := "@@ -1,1 +1,1 @@\n-a\n+A\n"
	_, err := tool.Execute(map[string]any{"command": "patch", "file_path": f, "patch": patch})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

// --- symbol extent detection ---

func TestFindSymbolExtent_GoFunction(t *testing.T) {
	content := "package x\n\nfunc Foo() {\n\treturn\n}\n\nfunc Bar() {}\n"
	start, end, err := findSymbolExtent(content, "Foo", "")
	require.NoError(t, err)
	assert.Equal(t, 3, start)
	assert.Equal(t, 5, end)
}

func TestFindSymbolExtent_GoMethodWithReceiver(t *testing.T) {
	content := "package x\n\nfunc (r *T) Method() {\n\tdoStuff()\n}\n"
	start, end, err := findSymbolExtent(content, "Method", "")
	require.NoError(t, err)
	assert.Equal(t, 3, start)
	assert.Equal(t, 5, end)
}

func TestFindSymbolExtent_NestedBraces(t *testing.T) {
	content := "func Outer() {\n\tif true {\n\t\tdoStuff()\n\t}\n\tfor i := 0; i < 3; i++ {\n\t\tdoMore()\n\t}\n}\n"
	start, end, err := findSymbolExtent(content, "Outer", "")
	require.NoError(t, err)
	assert.Equal(t, 1, start)
	assert.Equal(t, 8, end)
}

func TestFindSymbolExtent_BraceInsideStringLiteralIgnored(t *testing.T) {
	content := "func Weird() {\n\ts := \"{ not a brace }\"\n\t_ = s\n}\n"
	start, end, err := findSymbolExtent(content, "Weird", "")
	require.NoError(t, err)
	assert.Equal(t, 1, start)
	assert.Equal(t, 4, end)
}

func TestFindSymbolExtent_JSFunction(t *testing.T) {
	content := "function loadData() {\n  fetch('/x');\n}\n"
	start, end, err := findSymbolExtent(content, "loadData", "")
	require.NoError(t, err)
	assert.Equal(t, 1, start)
	assert.Equal(t, 3, end)
}

func TestFindSymbolExtent_PythonDef(t *testing.T) {
	content := "def foo():\n    x = 1\n    return x\n\ndef bar():\n    pass\n"
	start, end, err := findSymbolExtent(content, "foo", "")
	require.NoError(t, err)
	assert.Equal(t, 1, start)
	assert.Equal(t, 3, end)
}

func TestFindSymbolExtent_PythonClass(t *testing.T) {
	content := "class Foo:\n    def __init__(self):\n        self.x = 1\n\nclass Bar:\n    pass\n"
	start, end, err := findSymbolExtent(content, "Foo", "")
	require.NoError(t, err)
	assert.Equal(t, 1, start)
	assert.Equal(t, 3, end)
}

func TestFindSymbolExtent_NotFoundRejected(t *testing.T) {
	_, _, err := findSymbolExtent("func Foo() {}\n", "Bar", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFindSymbolExtent_AmbiguousRejected(t *testing.T) {
	content := "func (a *A) Handle() {}\nfunc (b *B) Handle() {}\n"
	_, _, err := findSymbolExtent(content, "Handle", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
	assert.Contains(t, err.Error(), "line(s) 1, 2")
}

func TestFindSymbolExtent_SymbolKindNarrowsMatch(t *testing.T) {
	// A type and a function share a name -- unqualified this is ambiguous,
	// but symbol_kind picks the intended one.
	content := "type Handler struct{}\n\nfunc Handler() {}\n"
	start, _, err := findSymbolExtent(content, "Handler", "type")
	require.NoError(t, err)
	assert.Equal(t, 1, start)

	start, _, err = findSymbolExtent(content, "Handler", "function")
	require.NoError(t, err)
	assert.Equal(t, 3, start)
}

// --- replace_symbol ---

func TestEditFile_ReplaceSymbol_GoFunction(t *testing.T) {
	body := "package x\n\nfunc Foo() {\n\treturn 1\n}\n\nfunc Bar() {}\n"
	f := writeSeenFile(t, "sym1.go", body)
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "replace_symbol", "file_path": f, "symbol": "Foo",
		"new_str": "func Foo() {\n\treturn 2\n}",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "package x\n\nfunc Foo() {\n\treturn 2\n}\n\nfunc Bar() {}\n", string(got))
}

func TestEditFile_ReplaceSymbol_PythonDef(t *testing.T) {
	body := "def foo():\n    return 1\n\ndef bar():\n    pass\n"
	f := writeSeenFile(t, "sym2.py", body)
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "replace_symbol", "file_path": f, "symbol": "foo",
		"new_str": "def foo():\n    return 2",
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "def foo():\n    return 2\n\ndef bar():\n    pass\n", string(got))
}

func TestEditFile_ReplaceSymbol_NotFoundRejected(t *testing.T) {
	f := writeSeenFile(t, "sym3.go", "func Foo() {}\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "replace_symbol", "file_path": f, "symbol": "Missing", "new_str": "x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestEditFile_ReplaceSymbol_AmbiguousRejected(t *testing.T) {
	f := writeSeenFile(t, "sym4.go", "func (a *A) Handle() {}\nfunc (b *B) Handle() {}\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "replace_symbol", "file_path": f, "symbol": "Handle", "new_str": "x",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ambiguous")
}

func TestEditFile_ReplaceSymbol_RequiresPriorRead(t *testing.T) {
	f := filepath.Join(t.TempDir(), "sym5.go")
	require.NoError(t, os.WriteFile(f, []byte("func Foo() {}\n"), 0o600))
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "replace_symbol", "file_path": f, "symbol": "Foo", "new_str": "func Foo() { return }",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read")
}

func TestEditFile_ReplaceSymbol_RevisionMismatchRejected(t *testing.T) {
	f := writeSeenFile(t, "sym6.go", "func Foo() {}\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "replace_symbol", "file_path": f, "symbol": "Foo", "new_str": "func Foo() { return }",
		"revision": "sha256:0000000000",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "changed since revision")
}

func TestEditFile_ReplaceSymbol_DryRunDoesNotWrite(t *testing.T) {
	f := writeSeenFile(t, "sym7.go", "func Foo() {\n\treturn 1\n}\n")
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "replace_symbol", "file_path": f, "symbol": "Foo",
		"new_str": "func Foo() {\n\treturn 2\n}", "dry_run": true,
	})
	require.NoError(t, err)
	assert.Contains(t, res, "[dry_run] no changes written.")
	got, _ := os.ReadFile(f)
	assert.Equal(t, "func Foo() {\n\treturn 1\n}\n", string(got))
}

func TestEditFile_ReplaceSymbol_UndoRestoresPrevious(t *testing.T) {
	f := writeSeenFile(t, "sym8.go", "func Foo() {\n\treturn 1\n}\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "replace_symbol", "file_path": f, "symbol": "Foo",
		"new_str": "func Foo() {\n\treturn 2\n}",
	})
	require.NoError(t, err)
	_, err = tool.Execute(map[string]any{"command": "undo_edit", "file_path": f})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "func Foo() {\n\treturn 1\n}\n", string(got))
}

func TestEditFile_View_Symbol_ShowsDeclaration(t *testing.T) {
	body := "package x\n\nfunc Foo() {\n\treturn 1\n}\n\nfunc Bar() {}\n"
	f := writeSeenFile(t, "sym9.go", body)
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "view", "file_path": f, "symbol": "Foo",
		"context_before": float64(0), "context_after": float64(0),
	})
	require.NoError(t, err)
	assert.Contains(t, res, "3\tfunc Foo() {")
	assert.Contains(t, res, "5\t}")
	assert.NotContains(t, res, "Bar")
}

// --- dry_run ---

func TestEditFile_DryRun_StrReplaceDoesNotWrite(t *testing.T) {
	f := writeSeenFile(t, "dr1.txt", "a\nb\nc\n")
	tool := editFileTool(t)
	res, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "b", "new_str": "B", "dry_run": true,
	})
	require.NoError(t, err)
	assert.Contains(t, res, "[dry_run] no changes written.")
	assert.Contains(t, res, "B")
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nb\nc\n", string(got), "dry_run must not touch the file")
}

func TestEditFile_DryRun_DoesNotPushUndoHistory(t *testing.T) {
	f := writeSeenFile(t, "dr2.txt", "a\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "b", "new_str": "B", "dry_run": true,
	})
	require.NoError(t, err)
	_, err = tool.Execute(map[string]any{"command": "undo_edit", "file_path": f})
	require.Error(t, err, "a dry_run must not leave anything to undo")
}

func TestEditFile_DryRun_InsertDoesNotWrite(t *testing.T) {
	f := writeSeenFile(t, "dr3.txt", "a\nb\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "insert", "file_path": f, "insert_line": float64(1), "new_str": "X", "dry_run": true,
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "a\nb\n", string(got))
}

func TestEditFile_DryRun_PatchDoesNotWrite(t *testing.T) {
	f := writeSeenFile(t, "dr4.txt", "one\ntwo\n")
	tool := editFileTool(t)
	patch := "@@ -1,1 +1,1 @@\n-one\n+ONE\n"
	_, err := tool.Execute(map[string]any{
		"command": "patch", "file_path": f, "patch": patch, "dry_run": true,
	})
	require.NoError(t, err)
	got, _ := os.ReadFile(f)
	assert.Equal(t, "one\ntwo\n", string(got))
}

func TestEditFile_DryRun_NoOpStillRejected(t *testing.T) {
	f := writeSeenFile(t, "dr5.txt", "keep\n")
	tool := editFileTool(t)
	_, err := tool.Execute(map[string]any{
		"command": "str_replace", "file_path": f, "old_str": "keep", "new_str": "keep", "dry_run": true,
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
