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
	"os"
	"strings"
	"testing"
)

// TestMain configures process-wide state so the package's tests are
// deterministic and match CI (a clean machine):
//
//   - HOME is pointed at a throwaway dir so real skill dirs (~/.agents/skills,
//     ~/.kdeps/skills) and the local model registry (~/.kdeps) on the dev
//     machine do not leak into tests that assert on discovered skills/registry.
//   - KDEPS_TURO=off disables the turo reducer, which would otherwise rewrite
//     system-prompt and history text on machines where the binary is installed
//     (CI has none) and break assertions that check for literal content.
//   - The convergence limiters are global and cumulative across the whole test
//     binary. Raising them keeps unrelated web/bash/file/code tool tests from
//     exhausting a shared budget and blocking each other. No test asserts the
//     blocking behavior, so this only removes cross-test pollution.
func TestMain(m *testing.M) {
	home, err := os.MkdirTemp("", "kdeps-agent-test-home")
	if err == nil {
		os.Setenv("HOME", home)
	}
	os.Setenv("KDEPS_TURO", "off")
	SetConvergenceLimits(1_000_000, 1_000_000, 1_000_000, 1_000_000)
	code := m.Run()
	if home != "" {
		os.RemoveAll(home)
	}
	os.Exit(code)
}

// resetTuroState restores turo runtime settings after a test mutates the
// package globals, so tests do not leak state into one another.
func resetTuroState(t *testing.T) {
	t.Helper()
	turoState.mu.Lock()
	pLevel, pFiller, pSyn, pGloss, pOff, pInit :=
		turoState.level, turoState.filler, turoState.synonyms, turoState.gloss, turoState.off, turoState.init
	turoState.mu.Unlock()
	t.Cleanup(func() {
		turoState.mu.Lock()
		turoState.level, turoState.filler, turoState.synonyms, turoState.gloss, turoState.off, turoState.init =
			pLevel, pFiller, pSyn, pGloss, pOff, pInit
		turoState.mu.Unlock()
		turoReduceCache.Clear()
	})
}

func TestTuroLevel_DefaultAndSet(t *testing.T) {
	resetTuroState(t)

	if !SetTuroLevel("ultra") {
		t.Fatal("SetTuroLevel(ultra) returned false")
	}
	if got := TuroLevel(); got != "ultra" {
		t.Fatalf("expected level ultra, got %q", got)
	}
	if !SetTuroLevel("LITE") {
		t.Fatal("SetTuroLevel is not case-insensitive")
	}
	if got := TuroLevel(); got != "lite" {
		t.Fatalf("expected level lite, got %q", got)
	}
	if SetTuroLevel("turbo") {
		t.Fatal("SetTuroLevel accepted an invalid level")
	}
	if got := TuroLevel(); got != "lite" {
		t.Fatalf("invalid level should not change state, got %q", got)
	}
}

func TestTuroRuntimeOff_Toggle(t *testing.T) {
	resetTuroState(t)

	SetTuroRuntimeOff(true)
	if !TuroRuntimeOff() {
		t.Fatal("expected runtime off")
	}
	SetTuroRuntimeOff(false)
	if TuroRuntimeOff() {
		t.Fatal("expected runtime on")
	}
}

func TestTuroStage_ToggleAndUnknown(t *testing.T) {
	resetTuroState(t)

	for _, stage := range []string{"filler", "synonyms", "gloss"} {
		if !SetTuroStage(stage, false) {
			t.Fatalf("SetTuroStage(%q, false) returned false", stage)
		}
		if TuroStage(stage) {
			t.Fatalf("%s should be off", stage)
		}
		if !SetTuroStage(stage, true) {
			t.Fatalf("SetTuroStage(%q, true) returned false", stage)
		}
		if !TuroStage(stage) {
			t.Fatalf("%s should be on", stage)
		}
	}
	if SetTuroStage("bogus", true) {
		t.Fatal("SetTuroStage accepted an unknown stage")
	}
	if TuroStage("bogus") {
		t.Fatal("TuroStage reported an unknown stage as on")
	}
}

func TestCapToolResult(t *testing.T) {
	small := "line1\nline2\n"
	if got := capToolResult(small); got != small {
		t.Fatalf("small result should pass through unchanged, got %q", got)
	}
	big := strings.Repeat("x\n", maxToolResultBytes) // well over the cap
	got := capToolResult(big)
	if len(got) > maxToolResultBytes+120 {
		t.Fatalf("capped result too large: %d bytes", len(got))
	}
	if !strings.Contains(got, "truncated") {
		t.Fatalf("expected truncation marker, got tail %q", got[len(got)-80:])
	}
}

func TestReduceToolOutputForLLM_PassthroughWhenInactive(t *testing.T) {
	// turo is disabled in tests (KDEPS_TURO=off via TestMain), so this must be a
	// no-op passthrough that leaves tool result content untouched.
	msgs := []AgentMessage{
		{Role: RoleUser, ResultContent: ""},
		{Role: RoleToolResult, ResultContent: `{"result":"raw tool output"}`},
	}
	got := reduceToolOutputForLLM(context.Background(), msgs)
	if got[1].ResultContent != `{"result":"raw tool output"}` {
		t.Fatalf("expected tool output unchanged when turo inactive, got %q", got[1].ResultContent)
	}
}

func TestBuildMessagesJSONReduced_AppliesTransform(t *testing.T) {
	s := NewSession(0)
	s.Append("hello", "world")

	got := s.BuildMessagesJSONReduced(strings.ToUpper)
	if !strings.Contains(got, "HELLO") || !strings.Contains(got, "WORLD") {
		t.Fatalf("expected transformed content, got %q", got)
	}
	// nil transform falls back to the plain builder.
	if s.BuildMessagesJSONReduced(nil) != s.BuildMessagesJSON() {
		t.Fatal("nil transform should equal BuildMessagesJSON")
	}
}

func TestToolTuning_TuroPersistRoundTrip(t *testing.T) {
	resetTuroState(t)
	r := &REPL{loop: &Loop{config: Config{}}}

	SetTuroLevel("wenyan")
	SetTuroRuntimeOff(true)
	SetTuroStage("gloss", false)
	snap := r.toolTuningSnapshot()
	if snap.TuroLevel != "wenyan" || !snap.TuroOff || snap.TuroGloss {
		t.Fatalf("snapshot did not capture turo state: %+v", snap)
	}

	// Change state, then restore from the snapshot.
	SetTuroLevel("ultra")
	SetTuroRuntimeOff(false)
	SetTuroStage("gloss", true)
	r.applyToolTuning(snap)
	if TuroLevel() != "wenyan" || !TuroRuntimeOff() || TuroStage("gloss") {
		t.Fatalf("apply did not restore turo state: level=%s off=%v gloss=%v",
			TuroLevel(), TuroRuntimeOff(), TuroStage("gloss"))
	}

	// An empty TuroLevel (never configured) must not clobber defaults.
	resetTuroState(t)
	SetTuroStage("filler", true)
	r.applyToolTuning(ToolTuning{TuroLevel: ""})
	if !TuroStage("filler") {
		t.Fatal("empty TuroLevel should leave turo defaults untouched")
	}
}
