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

func testEnforcer(t *testing.T, descs ...string) *goalEnforcer {
	t.Helper()
	return newGoalEnforcer(NewGoal("goal", descs), nil, 3, 25)
}

// A round that only returns blocked or duplicate content is not progress.
func TestEnforcer_ObserveResult(t *testing.T) {
	e := testEnforcer(t, "a")

	if !e.observeResult(toolNameWebSearch, "fresh findings") {
		t.Fatal("new content should count as progress")
	}
	if e.observeResult(toolNameWebSearch, "fresh findings") {
		t.Fatal("an identical payload is not new progress")
	}
	if e.observeResult(toolNameWebSearch, "convergence (5 calls): ALL web/search calls blocked") {
		t.Fatal("a convergence block is not progress")
	}
	if e.observeResult(toolNameBashExec, `{"error":"boom"}`) {
		t.Fatal("an error result is not progress")
	}
	if !e.blockedTools[toolNameBashExec] {
		t.Fatal("a failing tool should be recorded for the narrow step")
	}
}

// Unproductive rounds must climb the ladder and end in fail-forward, never
// stalling in place.
func TestEnforcer_EscalatesToFailForward(t *testing.T) {
	e := testEnforcer(t, "a", "b")

	levels := []int{
		e.endRound(false, false),
		e.endRound(false, false),
		e.endRound(false, false),
		e.endRound(false, false),
	}
	want := []int{escalateReanchor, escalateNarrow, escalateForceClose, escalateFailForward}
	for i, got := range levels {
		if got != want[i] {
			t.Fatalf("round %d: escalation = %d, want %d (all: %v)", i+1, got, want[i], levels)
		}
	}

	e.failForward("stuck")
	active := e.goal.Active()
	if active == nil || active.ID != 2 {
		t.Fatalf("fail-forward must advance to task 2, got %+v", active)
	}
	if e.goal.Tasks[0].Status != GoalTaskFailed {
		t.Fatalf("task 1 should be failed, got %s", e.goal.Tasks[0].Status)
	}
}

func TestEnforcer_ProductiveRoundResetsCounter(t *testing.T) {
	e := testEnforcer(t, "a")
	e.endRound(false, false)
	e.endRound(false, false)
	if lvl := e.endRound(true, false); lvl != escalateNone {
		t.Fatalf("a productive round should clear escalation, got %d", lvl)
	}
	if e.unproductive != 0 {
		t.Fatalf("unproductive counter should reset, got %d", e.unproductive)
	}
}

// Work belonging to a settled task must be refused rather than re-executed.
func TestEnforcer_RefusesBacktrack(t *testing.T) {
	e := testEnforcer(t, "a", "b")
	e.recordCall(toolNameWebSearch, `{"query":"x"}`)

	if _, refused := e.refuseBacktrack(toolNameWebSearch, `{"query":"x"}`); refused {
		t.Fatal("a call belonging to the ACTIVE task must not be refused")
	}

	e.goal.Advance(GoalTaskDone, "done")
	e.resetTask()

	msg, refused := e.refuseBacktrack(toolNameWebSearch, `{"query":"x"}`)
	if !refused {
		t.Fatal("a call from a settled task must be refused")
	}
	if !strings.Contains(msg, "REFUSED") || !strings.Contains(msg, "settled") {
		t.Fatalf("refusal should explain itself, got %q", msg)
	}
	if _, other := e.refuseBacktrack(toolNameWebSearch, `{"query":"different"}`); other {
		t.Fatal("an unrelated call must still be allowed")
	}
}

// Cosmetic edits to the arguments must not get settled work past the ban.
func TestEnforcer_BacktrackIgnoresArgumentCosmetics(t *testing.T) {
	e := testEnforcer(t, "a", "b")
	e.recordCall(toolNameWebSearch, `{"query":"World News"}`)
	e.goal.Advance(GoalTaskDone, "done")
	e.resetTask()

	if _, refused := e.refuseBacktrack(toolNameWebSearch, `{"query":"world   news"}`); !refused {
		t.Fatal("a whitespace/case variant of settled work must still be refused")
	}
}

// Violations must cost something: tool withdrawn, then all tools, then the task.
func TestEnforcer_PenaltyLadder(t *testing.T) {
	e := testEnforcer(t, "a", "b")
	e.recordCall(toolNameWebSearch, `{"query":"x"}`)
	e.goal.Advance(GoalTaskDone, "done")
	e.resetTask()

	// Strike 1: refused, nothing withdrawn yet.
	msg, _ := e.refuseBacktrack(toolNameWebSearch, `{"query":"x"}`)
	if !strings.Contains(msg, "Strike 1") {
		t.Fatalf("first violation should be reported as a strike: %q", msg)
	}
	if e.blockedTools[toolNameWebSearch] {
		t.Fatal("a single strike should not withdraw the tool yet")
	}

	// Strike 2: the misused tool is withdrawn for this task.
	_, _ = e.refuseBacktrack(toolNameWebSearch, `{"query":"x"}`)
	if !e.blockedTools[toolNameWebSearch] {
		t.Fatal("second strike must withdraw the offending tool")
	}

	// Strike 3: force-close — only the task-state tools remain legal.
	_, _ = e.refuseBacktrack(toolNameWebSearch, `{"query":"x"}`)
	if lvl := e.endRound(true, false); lvl != escalateForceClose {
		t.Fatalf("third strike must force-close even on a productive round, got %d", lvl)
	}
	if _, refused := e.refuseOffTask(toolNameBashExec); !refused {
		t.Fatal("under force-close, ordinary tools must be refused")
	}
	if _, refused := e.refuseOffTask(toolNameTaskComplete); refused {
		t.Fatal("task_complete must always remain available to close the task")
	}
}

func TestEnforcer_StrikesFailTaskForward(t *testing.T) {
	e := testEnforcer(t, "a", "b")
	for range penaltyFailForward {
		e.recordViolation(toolNameWebSearch)
	}
	if lvl := e.endRound(true, false); lvl != escalateFailForward {
		t.Fatalf("a task that keeps violating must be abandoned, got %d", lvl)
	}
}

// Advancing wipes the record so a new task is not punished for the last one.
func TestEnforcer_StrikesResetOnAdvance(t *testing.T) {
	e := testEnforcer(t, "a", "b")
	e.recordViolation(toolNameWebSearch)
	e.recordViolation(toolNameWebSearch)
	if e.strikes != 2 {
		t.Fatalf("expected 2 strikes, got %d", e.strikes)
	}
	e.goal.Advance(GoalTaskDone, "done")
	e.resetTask()
	if e.strikes != 0 {
		t.Fatalf("a new task must start clean, got %d strikes", e.strikes)
	}
	if e.blockedTools[toolNameWebSearch] {
		t.Fatal("tool bans must not carry into the next task")
	}
}

func TestEnforcer_DirectiveStatesRulesAndStrikes(t *testing.T) {
	e := testEnforcer(t, "a", "b")
	if d := e.directive(); !strings.Contains(d, "RULES") || !strings.Contains(d, "REFUSED") {
		t.Fatalf("directive must state the rules up front:\n%s", d)
	}
	e.recordViolation(toolNameWebSearch)
	if d := e.directive(); !strings.Contains(d, "Strikes against this task: 1") {
		t.Fatalf("directive must report the running strike count:\n%s", d)
	}
}

func TestEnforcer_DirectiveNamesActiveTaskAndSettledWork(t *testing.T) {
	e := testEnforcer(t, "gather sources", "write summary")
	e.goal.Advance(GoalTaskDone, "found three sources")

	d := e.directive()
	if !strings.Contains(d, "ACTIVE TASK 2") || !strings.Contains(d, "write summary") {
		t.Fatalf("directive must name the active task:\n%s", d)
	}
	if !strings.Contains(d, "do NOT redo") || !strings.Contains(d, "gather sources") {
		t.Fatalf("directive must list settled work:\n%s", d)
	}
	if !strings.Contains(d, toolNameTaskComplete) {
		t.Fatalf("directive must tell the model how to close the task:\n%s", d)
	}
}

func TestNarrowTools_DropsFailingButKeepsTaskTools(t *testing.T) {
	cfg := &domain.ChatConfig{Tools: []domain.Tool{
		{Name: toolNameWebSearch},
		{Name: toolNameBashExec},
		{Name: toolNameTaskComplete},
		{Name: toolNameTaskFail},
	}}
	got := narrowTools(cfg, map[string]bool{
		toolNameBashExec:     true,
		toolNameTaskComplete: true, // must survive regardless
	})

	names := map[string]bool{}
	for _, tool := range got.Tools {
		names[tool.Name] = true
	}
	if names[toolNameBashExec] {
		t.Fatal("a failing tool should be dropped")
	}
	for _, keep := range []string{toolNameWebSearch, toolNameTaskComplete, toolNameTaskFail} {
		if !names[keep] {
			t.Fatalf("%s must be kept so the task can still be closed", keep)
		}
	}
}

// The directive is a separate scenario item so rewriting it each round never
// invalidates the cached system preamble.
func TestWithGoalDirective_ReplacesAndPreservesPreamble(t *testing.T) {
	cfg := &domain.ChatConfig{Scenario: []domain.ScenarioItem{
		{Role: "system", Prompt: "cached preamble", CacheControl: "ephemeral"},
	}}

	first := withGoalDirective(cfg, "GOAL: x\nACTIVE TASK 1 of 2: a")
	second := withGoalDirective(first, "GOAL: x\nACTIVE TASK 2 of 2: b")

	if len(second.Scenario) != 2 {
		t.Fatalf("directives must replace, not accumulate: %+v", second.Scenario)
	}
	if second.Scenario[0].Prompt != "cached preamble" || second.Scenario[0].CacheControl != "ephemeral" {
		t.Fatalf("cached preamble must be untouched: %+v", second.Scenario[0])
	}
	if !strings.Contains(second.Scenario[1].Prompt, "ACTIVE TASK 2") {
		t.Fatalf("latest directive should win: %+v", second.Scenario[1])
	}
	if second.Scenario[1].CacheControl != "" {
		t.Fatal("the per-round directive must not be cached")
	}
}

// A model that reports the outcome in prose instead of calling task_complete
// must still advance the plan, otherwise the turn ends with tasks outstanding.
func TestSettleActiveFromText_AdvancesAndClassifies(t *testing.T) {
	l := &Loop{}
	l.enforcer = newGoalEnforcer(NewGoal("goal", []string{"a", "b", "c"}), nil, 3, 25)

	if !l.settleActiveFromText("Done: wrote the file.", nil) {
		t.Fatal("a text answer should settle task 1 and continue")
	}
	if l.enforcer.goal.Tasks[0].Status != GoalTaskDone {
		t.Fatalf("plain report should mark the task done, got %s", l.enforcer.goal.Tasks[0].Status)
	}

	// A refusal settles the task as failed rather than done.
	if !l.settleActiveFromText("I cannot complete this, no access to the repo.", nil) {
		t.Fatal("a refusal should still advance")
	}
	if l.enforcer.goal.Tasks[1].Status != GoalTaskFailed {
		t.Fatalf("a refusal should mark the task failed, got %s", l.enforcer.goal.Tasks[1].Status)
	}

	// Settling the final task ends the turn instead of looping forever.
	if l.settleActiveFromText("all done", nil) {
		t.Fatal("the last task must not request another round")
	}
	if !l.enforcer.goal.Complete() {
		t.Fatal("goal should be complete")
	}
}

func TestSettleActiveFromText_NoEnforcerIsNoop(t *testing.T) {
	l := &Loop{}
	if l.settleActiveFromText("anything", nil) {
		t.Fatal("without a goal there is nothing to settle")
	}
}

func TestStripTools(t *testing.T) {
	cfg := &domain.ChatConfig{Tools: []domain.Tool{{Name: toolNameWebSearch}}}
	if got := stripTools(cfg); len(got.Tools) != 0 {
		t.Fatalf("expected tools removed, got %+v", got.Tools)
	}
}
