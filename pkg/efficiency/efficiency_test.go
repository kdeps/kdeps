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

package efficiency_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/efficiency"
)

func configure(t *testing.T, lim efficiency.Limits) {
	t.Helper()
	efficiency.Configure(efficiency.Settings{
		Enabled: func() bool { return true },
		Verbose: func() bool { return false },
		Limits:  func() efficiency.Limits { return lim },
	})
	t.Cleanup(func() { efficiency.Configure(efficiency.Settings{}) })
}

func limits() efficiency.Limits {
	return efficiency.Limits{
		Reads: 2, Tighten: 1, Actions: 50, Stops: 2, Drops: 2,
		Category: map[string]int{"web": 1, "bash": efficiency.Unlimited},
	}
}

func TestUnconfigured(t *testing.T) {
	efficiency.Configure(efficiency.Settings{})
	assert.False(t, efficiency.Enabled())
	assert.False(t, efficiency.Verbose())
	assert.Equal(t, efficiency.Unlimited, efficiency.CurrentLimits().Reads)
}

func TestClassify(t *testing.T) {
	cases := []struct {
		name, args string
		want       efficiency.Kind
	}{
		{"read_file", `{}`, efficiency.Read},
		{"web_search", `{}`, efficiency.Read},
		{"write_file", `{}`, efficiency.Output},
		{"edit_file", `{"command":"view"}`, efficiency.Read},
		{"edit_file", `{"command":"str_replace"}`, efficiency.Output},
		{"bash_exec", `{"command":"ls -la | wc -l"}`, efficiency.Read},
		{"bash_exec", `{"command":"git status && git log"}`, efficiency.Read},
		{"bash_exec", `{"command":"git commit -m x"}`, efficiency.Output},
		{"bash_exec", `{"command":"ls > out"}`, efficiency.Output},
		{"bash_exec", `{"command":"go build ./..."}`, efficiency.Output},
		{"bash_exec", `not json`, efficiency.Output},
		{"calculator", `{}`, efficiency.State},
		{"memory_save", `{}`, efficiency.State},
		{"custom", `{}`, efficiency.Output},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, efficiency.Classify(c.name, c.args), "%s %s", c.name, c.args)
	}
	assert.False(t, efficiency.ReadOnlyBash("git"))
	assert.Equal(t, "web", efficiency.Category("wikipedia"))
	assert.Equal(t, "code", efficiency.Category("sql_query"))
	assert.Equal(t, "file", efficiency.Category("list_files"))
	assert.Empty(t, efficiency.Category("custom"))
	assert.True(t, efficiency.HasOutputTool([]string{"read_file", "write_file"}))
	assert.False(t, efficiency.HasOutputTool([]string{"read_file"}))
}

func TestReadRunStopsThenEnds(t *testing.T) {
	configure(t, limits())
	var e efficiency.Ledger
	e.BeginTurn(true)
	assert.Equal(t, efficiency.Admit, e.Admit("read_file", `{}`))
	assert.Equal(t, efficiency.Admit, e.Admit("list_files", `{}`))
	assert.Equal(t, efficiency.Drop, e.Admit("read_file", `{}`), "third read crosses reads=2")
	assert.True(t, e.Open)
	assert.Equal(t, 1, e.ReadsLimit(limits()), "tightened by one per stop")
	assert.False(t, e.KeepTool("read_file"))
	assert.True(t, e.KeepTool("write_file"))

	note := e.Note([]string{"read_file", "write_file"})
	assert.Contains(t, note, efficiency.Mark+" Soft stop 1/2")
	assert.Contains(t, note, "(write_file)")

	assert.Equal(t, efficiency.Admit, e.Admit("calculator", `{}`), "state calls pass")
	assert.Equal(t, efficiency.Drop, e.Admit("read_file", `{}`), "second refused read")
	assert.Equal(t, efficiency.Drop, e.Admit("read_file", `{}`), "past drops: the stop escalates")
	assert.Equal(t, 2, e.Stops)
	assert.Equal(t, efficiency.Drop, e.Admit("read_file", `{}`))
	assert.Equal(t, efficiency.End, e.Admit("read_file", `{}`), "past the stop budget the turn ends")
	assert.True(t, e.Ended)
	assert.Equal(t, efficiency.Abort, e.Admit("write_file", `{}`))
}

func TestOutputResolvesStop(t *testing.T) {
	configure(t, limits())
	var e efficiency.Ledger
	e.BeginTurn(true)
	e.Admit("read_file", `{}`)
	e.Admit("read_file", `{}`)
	require.Equal(t, efficiency.Drop, e.Admit("read_file", `{}`))
	assert.Equal(t, efficiency.Admit, e.Admit("write_file", `{}`))
	e.AfterOutput("write_file", `{}`, false)
	assert.True(t, e.Open, "a failed output keeps the stop open")
	e.AfterOutput("write_file", `{}`, true)
	assert.False(t, e.Open)
	note := e.Note(nil)
	assert.Contains(t, note, "Resumed after soft stop")
	assert.Empty(t, e.Note(nil), "the resume brief is sent once")

	e.BeginTurn(true)
	assert.Equal(t, 0, e.Stops, "counters reset each turn")
}

func TestCategoryAndActions(t *testing.T) {
	lim := limits()
	lim.Reads = efficiency.Unlimited
	configure(t, lim)
	var e efficiency.Ledger
	e.BeginTurn(true)
	assert.Equal(t, efficiency.Admit, e.Admit("web_search", `{}`))
	assert.Equal(t, efficiency.Drop, e.Admit("web_search", `{}`), "web budget is 1")
	assert.Contains(t, e.Reason, "1 web calls")

	lim.Category = nil
	lim.Actions = 1
	configure(t, lim)
	e.BeginTurn(true)
	assert.Equal(t, efficiency.Admit, e.Admit("write_file", `{}`))
	assert.Equal(t, efficiency.Drop, e.Admit("read_file", `{}`))
	assert.Contains(t, e.Reason, "tool calls this turn")
}

func TestInactive(t *testing.T) {
	configure(t, limits())
	var e efficiency.Ledger
	e.BeginTurn(false)
	for range 5 {
		assert.Equal(t, efficiency.Admit, e.Admit("read_file", `{}`))
	}
	assert.False(t, e.RepeatStop("read_file", "", "x", 3))
	assert.Empty(t, e.Note(nil))
	_, _, ok := e.FailureSlot(5)
	assert.False(t, ok)
	e.Ended = true
	e.Active = true
	assert.Equal(t, efficiency.Abort, e.Admit("read_file", `{}`))
}

func TestRepeatStop(t *testing.T) {
	configure(t, limits())
	var e efficiency.Ledger
	e.BeginTurn(true)
	assert.True(t, e.RepeatStop("web_search", "q=go", "same\nresult", 3))
	note := e.Note(nil)
	assert.Contains(t, note, "you called web_search (q=go) 3 times")
	assert.Contains(t, note, `"same result"`)
	assert.True(t, e.RepeatStop("web_search", "q=go", "same", 3))
	assert.False(t, e.RepeatStop("web_search", "q=go", "same", 3), "past the stop budget")
	assert.True(t, e.Ended)
}

func TestBriefsAndFailures(t *testing.T) {
	configure(t, limits())
	var e efficiency.Ledger
	e.BeginTurn(true)
	e.AttachToBrief("ignored: no brief yet")
	e.QueueBrief("first")
	e.AttachToBrief("  status  ")
	e.AttachToBrief(" ")
	assert.Equal(t, "first\n\nstatus", e.Note(nil))

	n, shown, ok := e.FailureSlot(1)
	assert.True(t, ok)
	assert.Equal(t, 1, n)
	assert.Equal(t, "1", shown)
	_, _, ok = e.FailureSlot(1)
	assert.False(t, ok)
	_, shown, _ = (&efficiency.Ledger{Active: true}).FailureSlot(efficiency.Unlimited)
	assert.Equal(t, "unlimited", shown)

	e.BriefedFailure = true
	assert.True(t, e.TakeBriefedFailure())
	assert.False(t, e.TakeBriefedFailure())

	assert.Contains(t, efficiency.LLMErrorBrief(2, "5", `"bad"`), "failure 2/5")
	assert.True(t, efficiency.Unfixable(errors.New("401 unauthorized")))
	assert.False(t, efficiency.Unfixable(errors.New("tool call was malformed")))
	assert.False(t, efficiency.Unfixable(nil))
}

func TestSnippetAndScrub(t *testing.T) {
	assert.Equal(t, "a b c", efficiency.Snippet("a\n  b\tc "))
	assert.True(t, strings.HasSuffix(efficiency.Snippet(strings.Repeat("x", 500)), "..."))

	msgs := []map[string]interface{}{
		{"role": "user", "content": "task\n\n" + efficiency.Mark + " Soft stop 1/3"},
		{"role": "user", "content": efficiency.Mark + " only a note"},
		{"role": "assistant", "content": "ok"},
	}
	out := efficiency.Scrub(msgs)
	require.Len(t, out, 2)
	assert.Equal(t, "task", out[0]["content"])
	assert.Contains(t, msgs[0]["content"], efficiency.Mark, "input untouched")
	clean := []map[string]interface{}{{"role": "user", "content": "x"}}
	assert.Equal(t, clean, efficiency.Scrub(clean))
	assert.Nil(t, efficiency.Scrub(nil))
}
