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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMintMatchID_SameInputsSameID(t *testing.T) {
	a := mintMatchID("/app/main.go", 10, "sha256:abc")
	b := mintMatchID("/app/main.go", 10, "sha256:abc")
	assert.Equal(t, a, b)
}

func TestMintMatchID_DifferentRevisionDifferentID(t *testing.T) {
	a := mintMatchID("/app/main.go", 10, "sha256:abc")
	b := mintMatchID("/app/main.go", 10, "sha256:def")
	assert.NotEqual(t, a, b)
}

func TestMintMatchID_DifferentLineDifferentID(t *testing.T) {
	a := mintMatchID("/app/main.go", 10, "sha256:abc")
	b := mintMatchID("/app/main.go", 11, "sha256:abc")
	assert.NotEqual(t, a, b)
}

func TestRememberMatch_ResolveMatchID_RoundTrip(t *testing.T) {
	id := mintMatchID("/app/x.go", 5, "sha256:xyz")
	rememberMatch(id, matchRef{path: "/app/x.go", line: 5, revision: "sha256:xyz"})
	ref, ok := resolveMatchID(id)
	assert.True(t, ok)
	assert.Equal(t, "/app/x.go", ref.path)
	assert.Equal(t, 5, ref.line)
	assert.Equal(t, "sha256:xyz", ref.revision)
}

func TestResolveMatchID_UnknownIDNotFound(t *testing.T) {
	_, ok := resolveMatchID("match-doesnotexist")
	assert.False(t, ok)
}

func TestRememberMatch_EvictsOldestPastDepth(t *testing.T) {
	var firstID string
	for i := range matchIDCacheDepth + 10 {
		id := mintMatchID("/app/evict.go", i, "sha256:rev")
		if i == 0 {
			firstID = id
		}
		rememberMatch(id, matchRef{path: "/app/evict.go", line: i, revision: "sha256:rev"})
	}
	_, ok := resolveMatchID(firstID)
	assert.False(t, ok, "the oldest entry must be evicted once the cache exceeds its depth")
}

func TestFirstMatchLine_FindsCaseInsensitiveMatch(t *testing.T) {
	content := "one\ntwo\nTHREE\nfour\n"
	assert.Equal(t, 3, firstMatchLine(content, "three"))
}

func TestFirstMatchLine_NotFoundReturnsZero(t *testing.T) {
	assert.Equal(t, 0, firstMatchLine("one\ntwo\n", "zzz"))
}

func TestAnnotateSearchMatches_AddsMatchIDLineRevision(t *testing.T) {
	f := writeSeenFile(t, "ann.go", "package x\n\nfunc Target() {}\n")
	results := []map[string]interface{}{{"path": f}}
	annotateSearchMatches(results, "Target")
	require := assert.New(t)
	require.NotEmpty(results[0]["match_id"])
	require.Equal(3, results[0]["line"])
	require.NotEmpty(results[0]["revision"])

	ref, ok := resolveMatchID(fmt.Sprint(results[0]["match_id"]))
	require.True(ok)
	require.Equal(f, ref.path)
	require.Equal(3, ref.line)
}

func TestAnnotateSearchMatches_NoMatchLeavesFieldsAbsent(t *testing.T) {
	f := writeSeenFile(t, "ann2.go", "package x\n\nfunc Other() {}\n")
	results := []map[string]interface{}{{"path": f}}
	annotateSearchMatches(results, "NotHere")
	_, hasID := results[0]["match_id"]
	assert.False(t, hasID)
}

func TestAnnotateSearchMatches_EmptyQueryIsNoOp(t *testing.T) {
	f := writeSeenFile(t, "ann3.go", "package x\n\nfunc Target() {}\n")
	results := []map[string]interface{}{{"path": f}}
	annotateSearchMatches(results, "")
	_, hasID := results[0]["match_id"]
	assert.False(t, hasID)
}
