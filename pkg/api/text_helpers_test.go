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

func TestToInt(t *testing.T) {
	assert.Equal(t, 5, toInt(float64(5)))
	assert.Equal(t, 5, toInt(5))
	assert.Equal(t, 5, toInt(int64(5)))
	assert.Equal(t, 5, toInt("5"))
	assert.Equal(t, 0, toInt("not-a-number"))
	assert.Equal(t, 0, toInt(true), "an unsupported type falls back to 0")
}

// --- symbol extent detection ---
