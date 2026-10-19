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

package api

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestInferEditCommand_NullFieldsStayAbsent(t *testing.T) {
	assert.Equal(t, "view", inferEditCommand(map[string]any{
		"symbol": "Foo", "new_str": nil,
	}))
	assert.Equal(t, "replace_symbol", inferEditCommand(map[string]any{
		"symbol": "Foo", "new_str": "func Foo() {}",
	}))
	assert.Equal(t, "", inferEditCommand(map[string]any{"insert_line": nil}))
	assert.Equal(t, "insert", inferEditCommand(map[string]any{"insert_line": float64(0)}))
}

func TestToInt(t *testing.T) {
	assert.Equal(t, 5, toInt(float64(5)))
	assert.Equal(t, 5, toInt(5))
	assert.Equal(t, 5, toInt(int64(5)))
	assert.Equal(t, 5, toInt("5"))
	assert.Equal(t, 0, toInt("not-a-number"))
	assert.Equal(t, 0, toInt(true), "an unsupported type falls back to 0")
}

func TestRevisionMatches_AcceptsBareHexAndWrappers(t *testing.T) {
	got := FileRevision([]byte("abc"))
	hex := strings.TrimPrefix(got, "sha256:")

	assert.True(t, revisionMatches(got, got))
	assert.True(t, revisionMatches(hex, got), "bare hex without the sha256: prefix")
	assert.True(t, revisionMatches("  "+strings.ToUpper(hex)+" ", got), "case and space")
	assert.True(t, revisionMatches("[revision "+got+"]", got), "copied marker")
	assert.True(t, revisionMatches(hex[:8], got), "shortened prefix")
	assert.False(t, revisionMatches("sha256:000000000000", got))
	assert.False(t, revisionMatches(hex[:3], got), "too short to identify a revision")
	assert.False(t, revisionMatches("", got))
}

func TestCheckRevision_BareHexIsNotStale(t *testing.T) {
	content := "hello\n"
	hex := strings.TrimPrefix(FileRevision([]byte(content)), "sha256:")
	require.NoError(t, checkRevision("/f", content, map[string]any{"revision": hex}))
	require.Error(t, checkRevision("/f", content, map[string]any{"revision": "deadbeefcafe"}))
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
