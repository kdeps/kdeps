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
	"strings"
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func refineLoop(t *testing.T, fn func(*domain.Workflow, interface{}) (interface{}, error)) *Loop {
	t.Helper()
	eng := executor.NewEngine(nil)
	eng.SetExecuteFunc(fn)
	return &Loop{
		engine:   eng,
		workflow: newTestWorkflowForSession(),
		config:   Config{Backend: "openai", PromptRefine: true},
	}
}

func TestRefinePrompt_ReturnsRewrite(t *testing.T) {
	l := refineLoop(t, func(wf *domain.Workflow, _ interface{}) (interface{}, error) {
		if wf.Metadata.TargetActionID != refineActionID {
			t.Fatalf("action id = %q", wf.Metadata.TargetActionID)
		}
		return "Add a --dry-run flag to the sync command and add a unit test that covers it.", nil
	})
	got := refinePrompt(context.Background(), l, "add dry run flag to sync and also write a test that covers it")
	if !strings.Contains(got, "--dry-run") {
		t.Fatalf("expected the rewrite, got %q", got)
	}
}

func TestRefinePrompt_SkipsTrivialInputWithoutEngineCall(t *testing.T) {
	calls := 0
	l := refineLoop(t, func(*domain.Workflow, interface{}) (interface{}, error) {
		calls++
		return "rewritten", nil
	})
	in := "what does this function do?"
	if got := refinePrompt(context.Background(), l, in); got != in {
		t.Fatalf("got %q, want unchanged", got)
	}
	if calls != 0 {
		t.Fatalf("expected no engine call for trivial input, got %d", calls)
	}
}

func TestRefinePrompt_FallsBackOnEngineError(t *testing.T) {
	l := refineLoop(t, func(*domain.Workflow, interface{}) (interface{}, error) {
		return nil, errors.New("boom")
	})
	in := "do the thing with the config and restart it"
	if got := refinePrompt(context.Background(), l, in); got != in {
		t.Fatalf("got %q, want unchanged", got)
	}
}

func TestRefinePrompt_FallsBackOnErrorMapAndEmpty(t *testing.T) {
	in := "do the thing with the config and restart it"
	for name, reply := range map[string]interface{}{
		"error map": map[string]any{"error": "model unavailable"},
		"empty":     "",
		"blank":     "   \n  ",
	} {
		t.Run(name, func(t *testing.T) {
			l := refineLoop(t, func(*domain.Workflow, interface{}) (interface{}, error) {
				return reply, nil
			})
			if got := refinePrompt(context.Background(), l, in); got != in {
				t.Fatalf("got %q, want unchanged", got)
			}
		})
	}
}

func TestRefinePrompt_NoEngine(t *testing.T) {
	in := "do the thing with the config and restart it"
	if got := refinePrompt(context.Background(), &Loop{config: Config{PromptRefine: true}}, in); got != in {
		t.Fatalf("got %q, want unchanged", got)
	}
}

func TestRefinePrompt_RunawayExpansionRejected(t *testing.T) {
	in := "fix the bug in the parser that drops trailing commas"
	l := refineLoop(t, func(*domain.Workflow, interface{}) (interface{}, error) {
		return strings.Repeat("blah blah expansion ", 80), nil
	})
	if got := refinePrompt(context.Background(), l, in); got != in {
		t.Fatalf("expected the runaway rewrite rejected, got %q", got)
	}
}

func TestCleanRefined(t *testing.T) {
	cases := map[string]string{
		`"add a flag"`:                  "add a flag",
		"Rewritten request: add a flag": "add a flag",
		"'add a flag'":                  "add a flag",
		"add a flag":                    "add a flag",
	}
	for in, want := range cases {
		if got := cleanRefined(in); got != want {
			t.Fatalf("cleanRefined(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestSetPromptRefine(t *testing.T) {
	l := &Loop{}
	l.SetPromptRefine(true)
	if !l.PromptRefineEnabled() {
		t.Fatal("expected enabled")
	}
	l.SetPromptRefine(false)
	if l.PromptRefineEnabled() {
		t.Fatal("expected disabled")
	}
}
