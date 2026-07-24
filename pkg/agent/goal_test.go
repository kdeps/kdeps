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
	"encoding/json"
	"strings"
	"testing"
)

func TestNewGoal_FirstTaskActive(t *testing.T) {
	g := NewGoal("ship the feature", []string{"write code", "run tests"})
	active := g.Active()
	if active == nil || active.ID != 1 {
		t.Fatalf("expected task 1 active, got %+v", active)
	}
	if g.Tasks[1].Status != GoalTaskPending {
		t.Fatalf("expected task 2 pending, got %s", g.Tasks[1].Status)
	}
}

func TestNewGoal_EmptyDecompositionFallsBackToPrompt(t *testing.T) {
	g := NewGoal("just do it", nil)
	if len(g.Tasks) != 1 || g.Tasks[0].Desc != "just do it" {
		t.Fatalf("expected a single task covering the prompt, got %+v", g.Tasks)
	}
}

// The cursor is the structural guarantee against circling: it must never move
// backward, whatever sequence of outcomes is applied.
func TestGoal_CursorIsMonotonic(t *testing.T) {
	g := NewGoal("goal", []string{"a", "b", "c"})
	prev := g.Cursor
	for _, status := range []GoalTaskStatus{GoalTaskDone, GoalTaskFailed, GoalTaskDone} {
		g.Advance(status, "note")
		if g.Cursor < prev {
			t.Fatalf("cursor moved backward: %d -> %d", prev, g.Cursor)
		}
		prev = g.Cursor
	}
	if !g.Complete() {
		t.Fatal("goal should be complete after settling every task")
	}
	// Advancing past the end is a no-op, not a panic or a rewind.
	if next := g.Advance(GoalTaskDone, ""); next != nil {
		t.Fatalf("advancing a complete goal should return nil, got %+v", next)
	}
	if g.Cursor < prev {
		t.Fatal("cursor rewound after completion")
	}
}

func TestGoal_ProgressAndCompletedDescs(t *testing.T) {
	g := NewGoal("goal", []string{"a", "b"})
	g.Advance(GoalTaskDone, "did a")

	settled, total := g.Progress()
	if settled != 1 || total != 2 {
		t.Fatalf("expected 1/2 settled, got %d/%d", settled, total)
	}
	done := g.CompletedDescs()
	if len(done) != 1 || !strings.Contains(done[0], "a") {
		t.Fatalf("expected the finished task listed, got %v", done)
	}
}

func TestGoal_JSONRoundTrip(t *testing.T) {
	g := NewGoal("goal", []string{"a", "b"})
	g.Advance(GoalTaskDone, "did a")

	data, err := json.Marshal(g)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var restored Goal
	if unmarshalErr := json.Unmarshal(data, &restored); unmarshalErr != nil {
		t.Fatalf("unmarshal: %v", unmarshalErr)
	}
	if restored.Cursor != g.Cursor || len(restored.Tasks) != len(g.Tasks) {
		t.Fatalf("round trip lost state: cursor=%d tasks=%d", restored.Cursor, len(restored.Tasks))
	}
	if restored.Tasks[0].Note != "did a" {
		t.Fatalf("round trip lost the completion note: %+v", restored.Tasks[0])
	}
}
