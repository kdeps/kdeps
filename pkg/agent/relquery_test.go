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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelJoin_EmptyInputs(t *testing.T) {
	assert.Equal(t, []relRow{}, relJoin(nil, nil, "id", "id"))
	assert.Equal(t, []relRow{}, relJoin([]relRow{{"id": 1}}, nil, "id", "id"))
	assert.Equal(t, []relRow{}, relJoin(nil, []relRow{{"id": 1}}, "id", "id"))
}

func TestRelJoin_NoMatches(t *testing.T) {
	left := []relRow{{"id": 1, "name": "a"}}
	right := []relRow{{"id": 2, "label": "b"}}
	assert.Equal(t, []relRow{}, relJoin(left, right, "id", "id"))
}

func TestRelJoin_SingleMatch(t *testing.T) {
	left := []relRow{{"id": 1, "name": "a"}}
	right := []relRow{{"id": 1, "label": "b"}}
	out := relJoin(left, right, "id", "id")
	require.Len(t, out, 1)
	assert.Equal(t, 1, out[0]["left_id"])
	assert.Equal(t, "a", out[0]["left_name"])
	assert.Equal(t, 1, out[0]["right_id"])
	assert.Equal(t, "b", out[0]["right_label"])
}

func TestRelJoin_MultipleMatches(t *testing.T) {
	left := []relRow{{"id": 1}, {"id": 1}}
	right := []relRow{{"id": 1}, {"id": 1}}
	out := relJoin(left, right, "id", "id")
	assert.Len(t, out, 4) // cartesian product of matches, per relational join semantics
}

func TestRelJoin_NumericStringCoercion(t *testing.T) {
	// left holds an int, right holds a string -- fmt.Sprint should still match.
	left := []relRow{{"key": 42}}
	right := []relRow{{"ref": "42"}}
	out := relJoin(left, right, "key", "ref")
	assert.Len(t, out, 1)
}

func TestRelJoin_MissingOrNilField(t *testing.T) {
	left := []relRow{{"other": 1}, {"id": nil}}
	right := []relRow{{"id": 1}}
	assert.Equal(t, []relRow{}, relJoin(left, right, "id", "id"))
}

func TestRelJoin_FieldNameCollisionPrefixed(t *testing.T) {
	// Both relations use "name" -- prefixing must prevent a collision.
	left := []relRow{{"id": 1, "name": "left-name"}}
	right := []relRow{{"id": 1, "name": "right-name"}}
	out := relJoin(left, right, "id", "id")
	require.Len(t, out, 1)
	assert.Equal(t, "left-name", out[0]["left_name"])
	assert.Equal(t, "right-name", out[0]["right_name"])
}

func TestRelUnion_EmptyInputs(t *testing.T) {
	assert.Equal(t, []relRow{}, relUnion(nil, nil))
}

func TestRelUnion_Concatenates(t *testing.T) {
	a := []relRow{{"id": 1}}
	b := []relRow{{"id": 2}}
	out := relUnion(a, b)
	assert.Len(t, out, 2)
}

func TestRelUnion_DedupesIdenticalRows(t *testing.T) {
	a := []relRow{{"id": 1, "name": "x"}}
	b := []relRow{{"id": 1, "name": "x"}, {"id": 2, "name": "y"}}
	out := relUnion(a, b)
	assert.Len(t, out, 2) // the duplicate {"id":1,"name":"x"} from b is dropped
}

func TestRelUnion_KeepsRowsThatDifferByOneField(t *testing.T) {
	a := []relRow{{"id": 1, "name": "x"}}
	b := []relRow{{"id": 1, "name": "y"}}
	out := relUnion(a, b)
	assert.Len(t, out, 2) // not duplicates -- "name" differs
}

func TestMemoryRelation_NilStore(t *testing.T) {
	assert.Equal(t, []relRow{}, memoryRelation(nil))
}

func TestMemoryRelation_ConvertsEntries(t *testing.T) {
	ms := NewMemoryStore(t.TempDir())
	ms.SetCwd("/Users/test/Projects/foo")
	require.NoError(t, ms.Set("k1", "v1"))

	rows := memoryRelation(ms)
	require.Len(t, rows, 1)
	assert.Equal(t, "k1", rows[0]["key"])
	assert.Equal(t, "v1", rows[0]["value"])
}

// TestMemoryRelation_TruncatesToLimitKeepingMostRecent guards the
// >relMemoryLimit branch: with more entries than the limit, only the most
// recently updated ones survive, matching the recency bias used elsewhere in
// the memory subsystem (selectKeptEntries, RecentKeys).
func TestMemoryRelation_TruncatesToLimitKeepingMostRecent(t *testing.T) {
	ms := NewMemoryStore(t.TempDir())
	ms.SetCwd("/Users/test/Projects/truncation")

	total := relMemoryLimit() + 5
	// Filled in memory, not through Set (which writes the store to disk on
	// every call), each with a distinct, increasing UpdatedAt.
	for i := range total {
		key := fmt.Sprintf("key-%04d", i)
		ms.entries[key] = MemoryEntry{Key: key, Value: "v", CreatedAt: int64(i), UpdatedAt: int64(i)}
	}

	rows := memoryRelation(ms)
	require.Len(t, rows, relMemoryLimit())
	// The oldest 5 entries (key-0000..key-0004) must have been dropped.
	seen := make(map[string]bool, len(rows))
	for _, r := range rows {
		seen[r["key"].(string)] = true
	}
	assert.False(t, seen["key-0000"], "the oldest entry must be dropped")
	assert.True(t, seen[fmt.Sprintf("key-%04d", total-1)], "the newest entry must survive")
}

func TestToolCallRelation_ConvertsRecords(t *testing.T) {
	log := []ToolCallRecord{
		{Name: "bash_exec", Args: `{"command":"ls"}`, Result: "ok", Timestamp: 100},
	}
	rows := toolCallRelation(log)
	require.Len(t, rows, 1)
	assert.Equal(t, "bash_exec", rows[0]["name"])
	assert.Equal(t, `{"command":"ls"}`, rows[0]["args"])
	assert.Equal(t, "ok", rows[0]["result"])
	assert.Equal(t, int64(100), rows[0]["timestamp"])
}

func TestToolCallRelation_EmptyLog(t *testing.T) {
	assert.Equal(t, []relRow{}, toolCallRelation(nil))
}
