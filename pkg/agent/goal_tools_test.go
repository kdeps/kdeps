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
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func loopWithGoal(descs ...string) *Loop {
	l := &Loop{}
	l.enforcer = newGoalEnforcer(NewGoal("goal", descs), nil, 3, 25)
	return l
}

// Completion is a fact the code owns: the id must match the active task.
func TestSettleTask_AdvancesOnMatchingID(t *testing.T) {
	l := loopWithGoal("a", "b")

	out, err := l.settleTask(map[string]any{"id": float64(1), "summary": "did a"}, GoalTaskDone)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "active task is now 2") {
		t.Fatalf("result should name the next active task: %q", out)
	}
	active := l.enforcer.goal.Active()
	if active == nil || active.ID != 2 {
		t.Fatalf("expected task 2 active, got %+v", active)
	}
	if l.enforcer.goal.Tasks[0].Note != "did a" {
		t.Fatalf("summary should be recorded, got %q", l.enforcer.goal.Tasks[0].Note)
	}
}

func TestSettleTask_RejectsWrongID(t *testing.T) {
	l := loopWithGoal("a", "b")

	_, err := l.settleTask(map[string]any{"id": float64(2), "summary": "skipping ahead"}, GoalTaskDone)
	if err == nil {
		t.Fatal("settling a non-active task must be refused")
	}
	if !strings.Contains(err.Error(), "active task is 1") {
		t.Fatalf("error should name the real active task, got %v", err)
	}
	if l.enforcer.goal.Cursor != 0 {
		t.Fatal("a refused call must not move the cursor")
	}
}

func TestSettleTask_MissingIDIsRefused(t *testing.T) {
	l := loopWithGoal("a")
	if _, err := l.settleTask(map[string]any{"summary": "no id"}, GoalTaskDone); err == nil {
		t.Fatal("a missing id must be refused")
	}
}

func TestSettleTask_ReportsGoalCompletion(t *testing.T) {
	l := loopWithGoal("only task")

	out, err := l.settleTask(map[string]any{"id": float64(1), "summary": "done"}, GoalTaskDone)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "goal is complete") {
		t.Fatalf("final task should report completion: %q", out)
	}
	if !l.enforcer.goal.Complete() {
		t.Fatal("goal should be complete")
	}
}

func TestSettleTask_FailAdvancesToo(t *testing.T) {
	l := loopWithGoal("a", "b")
	if _, err := l.settleTask(map[string]any{"id": float64(1), "reason": "impossible"}, GoalTaskFailed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.enforcer.goal.Tasks[0].Status != GoalTaskFailed {
		t.Fatalf("task should be failed, got %s", l.enforcer.goal.Tasks[0].Status)
	}
	if active := l.enforcer.goal.Active(); active == nil || active.ID != 2 {
		t.Fatalf("failing a task must still advance, got %+v", active)
	}
}

func TestSettleTask_NoGoalIsAnError(t *testing.T) {
	l := &Loop{}
	if _, err := l.settleTask(map[string]any{"id": float64(1)}, GoalTaskDone); err == nil {
		t.Fatal("settling without an active goal must error")
	}
}

func TestToolArgInt(t *testing.T) {
	cases := map[string]any{
		"float":  float64(3),
		"int":    4,
		"int64":  int64(5),
		"string": "6",
		"padded": " 7 ",
	}
	want := map[string]int{"float": 3, "int": 4, "int64": 5, "string": 6, "padded": 7}
	for key, raw := range cases {
		got, ok := toolArgInt(map[string]any{key: raw}, key)
		if !ok || got != want[key] {
			t.Errorf("toolArgInt(%s) = %d, %v; want %d, true", key, got, ok, want[key])
		}
	}
	if _, ok := toolArgInt(map[string]any{"x": "nope"}, "x"); ok {
		t.Error("a non-numeric argument must not parse")
	}
}

func TestIsTaskStateTool(t *testing.T) {
	if !isTaskStateTool(toolNameTaskComplete) || !isTaskStateTool(toolNameTaskFail) {
		t.Fatal("task state tools must be recognized")
	}
	if isTaskStateTool(toolNameWebSearch) {
		t.Fatal("ordinary tools must not count as state transitions")
	}
}

// Reasoning capture: DeepSeek-family models require an assistant turn's
// reasoning_content to be replayed, so the loop must record it per turn and
// hand it over exactly once.
func TestLoop_ReasoningCaptureAndTake(t *testing.T) {
	l := &Loop{}
	cfg := &domain.ChatConfig{}
	l.captureReasoning(cfg)
	if cfg.ReasoningOut == nil {
		t.Fatal("chat config must expose a reasoning sink")
	}

	*cfg.ReasoningOut = "step-by-step reasoning"
	if got := l.takeReasoning(); got != "step-by-step reasoning" {
		t.Fatalf("takeReasoning() = %q", got)
	}
	// Consumed: a second turn must not inherit the previous turn's reasoning.
	if got := l.takeReasoning(); got != "" {
		t.Fatalf("reasoning should be cleared after being taken, got %q", got)
	}
}
