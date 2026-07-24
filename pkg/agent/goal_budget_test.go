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

import "testing"

// withCleanBudgets isolates a test from the process-wide convergence caches.
func withCleanBudgets(t *testing.T) {
	t.Helper()
	ResetConvergence()
	SetConvergenceLimits(maxWebToolCalls, maxBashToolCalls, maxFileToolCalls, maxCodeToolCalls)
	t.Cleanup(func() {
		ResetConvergence()
		SetConvergenceLimits(maxWebToolCalls, maxBashToolCalls, maxFileToolCalls, maxCodeToolCalls)
	})
}

func TestCategoryForTool(t *testing.T) {
	cases := map[string]toolBudgetCategory{
		toolNameWebSearch:    budgetWeb,
		toolNameWebScraper:   budgetWeb,
		toolNameBashExec:     budgetBash,
		toolNameReadFile:     budgetFile,
		toolNameSearchLocal:  budgetCode,
		toolNameTaskComplete: budgetNone, // never moves a budget
		"totally_unknown":    budgetNone,
	}
	for name, want := range cases {
		if got := categoryForTool(name); got != want {
			t.Errorf("categoryForTool(%q) = %v, want %v", name, got, want)
		}
	}
}

// Below the sample threshold the tuner must not move anything: a couple of
// unlucky calls are not evidence.
func TestBudgetTuner_NoChangeBeforeMinSamples(t *testing.T) {
	withCleanBudgets(t)
	b := newBudgetTuner()

	for range budgetMinSamples - 1 {
		if _, changed := b.record(toolNameWebSearch, false); changed {
			t.Fatal("must not adjust before the minimum sample count")
		}
	}
	if _, maxAfter := globalWebCache.count(); maxAfter != maxWebToolCalls {
		t.Fatalf("cap moved early: %d", maxAfter)
	}
}

// A category returning nothing useful gets cut short instead of burning the
// rest of the turn's budget.
func TestBudgetTuner_CutsOnLowYield(t *testing.T) {
	withCleanBudgets(t)
	b := newBudgetTuner()

	var final int
	for range budgetMinSamples {
		final, _ = b.record(toolNameWebSearch, false)
	}
	if final >= maxWebToolCalls {
		t.Fatalf("a failing category should be cut below %d, got %d", maxWebToolCalls, final)
	}
	if _, capNow := globalWebCache.count(); capNow != final {
		t.Fatalf("cache cap not updated: got %d, want %d", capNow, final)
	}
}

// The cut must never fall below the calls already made, or work in flight would
// be retroactively blocked.
func TestBudgetTuner_CutNeverBlocksCompletedCalls(t *testing.T) {
	withCleanBudgets(t)
	b := newBudgetTuner()
	// Consume budget so calls > 0.
	for i := range 3 {
		_, _ = globalWebCache.trackCall(string(rune('a'+i)), func() (string, error) { return "x", nil })
	}
	calls, _ := globalWebCache.count()

	var final int
	for range budgetMinSamples {
		final, _ = b.record(toolNameWebSearch, false)
	}
	if final < calls {
		t.Fatalf("cut below calls already made (%d < %d) would block completed work", final, calls)
	}
}

// A category still returning new content near its cap earns more room, bounded
// by the ceiling.
func TestBudgetTuner_ExtendsOnHighYieldNearCap(t *testing.T) {
	withCleanBudgets(t)
	SetConvergenceLimits(5, maxBashToolCalls, maxFileToolCalls, maxCodeToolCalls)
	b := newBudgetTuner()

	// Consume up to the cap so the extension condition applies.
	for i := range 4 {
		_, _ = globalWebCache.trackCall(string(rune('a'+i)), func() (string, error) { return "x", nil })
	}
	var final int
	for range budgetMinSamples {
		final, _ = b.record(toolNameWebSearch, true)
	}
	if final <= 5 {
		t.Fatalf("a paying category near its cap should be extended, got %d", final)
	}
	if final > 5*budgetCeilingFactor {
		t.Fatalf("extension exceeded the ceiling: %d", final)
	}
}

// Growth is bounded no matter how long the category keeps paying off.
func TestBudgetTuner_RespectsCeiling(t *testing.T) {
	withCleanBudgets(t)
	SetConvergenceLimits(4, maxBashToolCalls, maxFileToolCalls, maxCodeToolCalls)
	b := newBudgetTuner()

	final := 0
	for i := range 200 {
		// Keep consuming so the near-cap condition stays true.
		_, _ = globalWebCache.trackCall(string(rune(i)), func() (string, error) { return "x", nil })
		final, _ = b.record(toolNameWebSearch, true)
	}
	if final > 4*budgetCeilingFactor {
		t.Fatalf("cap %d exceeded ceiling %d", final, 4*budgetCeilingFactor)
	}
}

func TestBudgetTuner_IgnoresUncategorizedTools(t *testing.T) {
	withCleanBudgets(t)
	b := newBudgetTuner()
	for range budgetMinSamples + 2 {
		if _, changed := b.record(toolNameTaskComplete, false); changed {
			t.Fatal("task tools must never move a convergence budget")
		}
	}
}

func TestBudgetTuner_NilSafe(t *testing.T) {
	var b *budgetTuner
	if _, changed := b.record(toolNameWebSearch, true); changed {
		t.Fatal("a nil tuner must be inert")
	}
}

func TestCategoryName(t *testing.T) {
	if categoryName(budgetWeb) != "web" || categoryName(budgetCode) != "code" {
		t.Fatal("category names should be stable")
	}
	if categoryName(budgetNone) != "" {
		t.Fatal("an uncategorized budget has no name")
	}
}
