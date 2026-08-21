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
	"context"
	"errors"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func TestParsePlanTasks(t *testing.T) {
	cases := map[string]struct {
		reply string
		want  int
	}{
		"plain":            {`{"tasks":["a","b"]}`, 2},
		"fenced":           {"```json\n{\"tasks\":[\"a\",\"b\",\"c\"]}\n```", 3},
		"prose wrapped":    {`Sure! {"tasks":["a"]} hope that helps`, 1},
		"blank entries":    {`{"tasks":["a","   ",""]}`, 1},
		"not json":         {"I cannot do that", 0},
		"missing tasks":    {`{"goal":"x"}`, 0},
		"empty task array": {`{"tasks":[]}`, 0},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if got := parsePlanTasks(c.reply); len(got) != c.want {
				t.Fatalf("parsePlanTasks(%q) = %v, want %d tasks", c.reply, got, c.want)
			}
		})
	}
}

func TestParsePlanTasks_CapsAtMax(t *testing.T) {
	reply := `{"tasks":["1","2","3","4","5","6","7","8","9","10","11","12","13","14"]}`
	if got := parsePlanTasks(reply); len(got) != maxPlanTasks {
		t.Fatalf("expected the list capped at %d, got %d", maxPlanTasks, len(got))
	}
}

// The trivial fast-path is what keeps always-on enforcement from adding a
// planning call to ordinary chat.
func TestLooksTrivial(t *testing.T) {
	trivial := []string{
		"what is 2+2?", "who wrote this file?",
		"hello", "hi", "thanks", "ok",
		"build the auth API", // short, single-line, no multi-step marker -- no longer requires a trailing "?"
	}
	for _, s := range trivial {
		if !looksTrivial(s) {
			t.Errorf("%q should skip decomposition", s)
		}
	}
	notTrivial := []string{
		"fix the bug and then run the tests?",      // multi-step marker
		"summarize this\nand also update the docs", // multi-line
		"is this question long enough to be treated as a real request that spans well beyond the trivial length limit used for a single quick question?",
	}
	for _, s := range notTrivial {
		if looksTrivial(s) {
			t.Errorf("%q should be decomposed", s)
		}
	}
}

// Planning must never be able to block a turn: with no engine it degrades to a
// single task instead of failing.
func TestPlanGoal_FallsBackWithoutEngine(t *testing.T) {
	g := planGoal(context.Background(), &Loop{}, "do the thing")
	if g == nil {
		t.Fatal("planGoal must never return nil")
	}
	if len(g.Tasks) != 1 || g.Tasks[0].Desc != "do the thing" {
		t.Fatalf("expected a single fallback task, got %+v", g.Tasks)
	}
}

func TestConfirmPlan_NoLocalServer(t *testing.T) {
	for _, backend := range []string{"", "file", "gguf"} {
		l := &Loop{config: Config{Backend: backend}}
		candidate := []string{"a", "b"}
		got := confirmPlan(l, "do something", candidate)
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("backend %q: expected the unchanged candidate, got %v", backend, got)
		}
	}
}

func TestConfirmPlan_ApprovesAsIs(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(*domain.Workflow, interface{}) (interface{}, error) {
		return `{"tasks":["a","b"]}`, nil
	})
	l := &Loop{engine: eng, workflow: newTestWorkflowForSession(), config: Config{Backend: "openai"}}

	got := confirmPlan(l, "do something", []string{"a", "b"})
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("expected the approved list unchanged, got %v", got)
	}
}

func TestConfirmPlan_ReturnsCorrectedList(t *testing.T) {
	var gotActionID string
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		gotActionID = wf.Metadata.TargetActionID
		return `{"tasks":["a","missing step","b"]}`, nil
	})
	l := &Loop{engine: eng, workflow: newTestWorkflowForSession(), config: Config{Backend: "openai"}}

	got := confirmPlan(l, "do something", []string{"a", "b"})
	if len(got) != 3 || got[1] != "missing step" {
		t.Fatalf("expected the corrected list, got %v", got)
	}
	if gotActionID != goalConfirmActionID {
		t.Errorf("action id = %q, want %q", gotActionID, goalConfirmActionID)
	}
}

func TestConfirmPlan_FallsBackOnEngineError(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(*domain.Workflow, interface{}) (interface{}, error) {
		return nil, errors.New("boom")
	})
	l := &Loop{engine: eng, workflow: newTestWorkflowForSession(), config: Config{Backend: "openai"}}

	candidate := []string{"a", "b"}
	got := confirmPlan(l, "do something", candidate)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("expected the unchanged candidate on error, got %v", got)
	}
}

func TestConfirmPlan_FallsBackOnUnparsableReply(t *testing.T) {
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(*domain.Workflow, interface{}) (interface{}, error) {
		return "not json at all", nil
	})
	l := &Loop{engine: eng, workflow: newTestWorkflowForSession(), config: Config{Backend: "openai"}}

	candidate := []string{"a", "b"}
	got := confirmPlan(l, "do something", candidate)
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("expected the unchanged candidate on unparsable reply, got %v", got)
	}
}

// PlanGoal must only pay for the confirmation call when there is more than
// one task to get wrong -- a single-task plan has nothing to reorder.
func TestPlanGoal_SkipsConfirmationForSingleTask(t *testing.T) {
	calls := 0
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(*domain.Workflow, interface{}) (interface{}, error) {
		calls++
		return `{"tasks":["only step"]}`, nil
	})
	l := &Loop{engine: eng, workflow: newTestWorkflowForSession(), config: Config{Backend: "openai"}}

	g := planGoal(context.Background(), l, "first do step one; then do step two; then verify everything works")
	if len(g.Tasks) != 1 {
		t.Fatalf("expected a single task, got %+v", g.Tasks)
	}
	if calls != 1 {
		t.Errorf("expected exactly one engine call (decompose only, no confirm), got %d", calls)
	}
}

// A multi-task decomposition must be run through the independent
// confirmation pass before becoming the Goal.
func TestPlanGoal_ConfirmsMultiTaskPlan(t *testing.T) {
	var actionIDs []string
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		actionIDs = append(actionIDs, wf.Metadata.TargetActionID)
		if wf.Metadata.TargetActionID == goalConfirmActionID {
			return `{"tasks":["step one","step two","step three"]}`, nil
		}
		return `{"tasks":["step one","step two"]}`, nil
	})
	l := &Loop{engine: eng, workflow: newTestWorkflowForSession(), config: Config{Backend: "openai"}}

	g := planGoal(context.Background(), l, "first do step one; then do step two; then verify everything works")
	if len(g.Tasks) != 3 {
		t.Fatalf("expected the confirmed 3-task plan, got %+v", g.Tasks)
	}
	if len(actionIDs) != 2 || actionIDs[0] != goalPlanActionID || actionIDs[1] != goalConfirmActionID {
		t.Fatalf("expected [plan, confirm] action id order, got %v", actionIDs)
	}
}

func TestExtractJSONObject(t *testing.T) {
	if got := extractJSONObject(`noise {"a":1} tail`); got != `{"a":1}` {
		t.Fatalf("got %q", got)
	}
	if got := extractJSONObject("no object here"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestFormatLoopResult_ErrorMap(t *testing.T) {
	t.Parallel()
	if got := formatLoopResult(map[string]any{"error": "connection refused"}); got != "" {
		t.Fatalf("error map: got %q, want empty", got)
	}
	if got := formatLoopResult(map[string]interface{}{"error": "boom"}); got != "" {
		t.Fatalf("error map interface: got %q, want empty", got)
	}
}

func TestFormatLoopResult_MessageMap(t *testing.T) {
	t.Parallel()
	got := formatLoopResult(map[string]any{
		"message": map[string]any{"role": "assistant", "content": `{"tasks":["a","b"]}`},
	})
	want := `{"tasks":["a","b"]}`
	if got != want {
		t.Fatalf("message map: got %q, want %q", got, want)
	}
}
