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
	return loopWithGoalEvidence(false, descs...)
}

func loopWithGoalEvidence(requireEvidence bool, descs ...string) *Loop {
	l := &Loop{}
	l.enforcer = newGoalEnforcer(NewGoal("goal", descs), nil, 3, 25, requireEvidence)
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

// A missing id has exactly one sensible reading with a single active task:
// the active one. Defaulting instead of refusing avoids looping the model on
// a parameter it apparently can't supply.
func TestSettleTask_MissingIDDefaultsToActiveTask(t *testing.T) {
	l := loopWithGoal("a", "b")

	out, err := l.settleTask(map[string]any{"summary": "no id"}, GoalTaskDone)
	if err != nil {
		t.Fatalf("a missing id should default to the active task, got error: %v", err)
	}
	if !strings.Contains(out, "active task is now 2") {
		t.Fatalf("result should name the next active task: %q", out)
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

// RequireTaskEvidence: a task with zero tool calls has nothing to verify, so
// task_complete must not be blocked.
func TestSettleTask_EvidenceNotRequiredForZeroToolCallTask(t *testing.T) {
	l := loopWithGoalEvidence(true, "a", "b")

	_, err := l.settleTask(map[string]any{"id": float64(1), "summary": "answered directly"}, GoalTaskDone)
	if err != nil {
		t.Fatalf("a task with no tool calls must close without evidence, got: %v", err)
	}
}

// RequireTaskEvidence: a task that made only non-evidence-capable calls
// (e.g. web_search) must be refused until an evidence-capable tool runs.
func TestSettleTask_EvidenceRequiredRefusesUnverifiedCompletion(t *testing.T) {
	l := loopWithGoalEvidence(true, "a", "b")
	l.enforcer.recordCall(toolNameWebSearch, `{"query":"x"}`)

	_, err := l.settleTask(map[string]any{"id": float64(1), "summary": "done"}, GoalTaskDone)
	if err == nil {
		t.Fatal("a task with tool calls but no evidence-capable call must be refused")
	}
	if !strings.Contains(err.Error(), "none of them verify") {
		t.Fatalf("refusal should explain the missing evidence, got: %v", err)
	}
	if l.enforcer.goal.Cursor != 0 {
		t.Fatal("a refused completion must not move the cursor")
	}
}

// RequireTaskEvidence: once an evidence-capable tool has run, task_complete
// succeeds and the evidence text is recorded on the task.
func TestSettleTask_EvidenceRequiredAllowsVerifiedCompletion(t *testing.T) {
	l := loopWithGoalEvidence(true, "a", "b")
	l.enforcer.recordCall(toolNameBashExec, `{"command":"go test ./..."}`)

	out, err := l.settleTask(map[string]any{
		"id": float64(1), "summary": "done",
		"evidence": "ran go test ./..., all passed",
	}, GoalTaskDone)
	if err != nil {
		t.Fatalf("a task with an evidence-capable call must be allowed to close, got: %v", err)
	}
	if !strings.Contains(out, "active task is now 2") {
		t.Fatalf("result should name the next active task: %q", out)
	}
	if got := l.enforcer.goal.Tasks[0].Evidence; got != "ran go test ./..., all passed" {
		t.Fatalf("evidence should be recorded on the task, got %q", got)
	}
}

// RequireTaskEvidence never gates task_fail: giving up doesn't claim a
// result that needs verifying.
func TestSettleTask_EvidenceNotRequiredForTaskFail(t *testing.T) {
	l := loopWithGoalEvidence(true, "a", "b")
	l.enforcer.recordCall(toolNameWebSearch, `{"query":"x"}`)

	_, err := l.settleTask(map[string]any{"id": float64(1), "reason": "cannot do it"}, GoalTaskFailed)
	if err != nil {
		t.Fatalf("task_fail must never be gated by evidence, got: %v", err)
	}
}

// The default (RequireTaskEvidence: false) preserves today's behavior: no
// evidence needed even for a task with only non-evidence-capable calls.
func TestSettleTask_EvidenceFlagOffPreservesOldBehavior(t *testing.T) {
	l := loopWithGoal("a", "b") // requireEvidence defaults to false
	l.enforcer.recordCall(toolNameWebSearch, `{"query":"x"}`)

	_, err := l.settleTask(map[string]any{"id": float64(1), "summary": "done"}, GoalTaskDone)
	if err != nil {
		t.Fatalf("evidence gating must be opt-in; got unexpected refusal: %v", err)
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
