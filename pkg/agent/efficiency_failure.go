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
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/failbrief"
)

// Failure soft stops: a failed tool call or a failed LLM call is stopped by
// kdeps itself, not left for the model to notice. The error, plus whatever the
// model needs to fix it (the file lines around a failed edit, the folder of a
// missing path), goes to the model on the hidden model-facing channel and the
// model retries with that information. The human sees nothing unless
// /efficiency verbose is on. Past the per-turn failure budget the plain
// [TOOL FAILED] banner (tool calls) or the error itself (LLM calls) takes over.
// briefLimit is a failure-brief limit from its event's items, or 0 (the
// failbrief default) when the event is off.
func briefLimit(name string) int {
	n := effectiveItems(name, 0)
	if n == disabledSentinel || n < 1 {
		return 0
	}
	return n
}

// configureFailureBrief installs the failure-brief limits shared with
// workflow chats (pkg/failbrief).
func configureFailureBrief() {
	failbrief.Configure(failbrief.Settings{
		Failures:     func() int { return effectiveRounds(eventEfficiencyFailures) },
		ContextBytes: func() int { return briefLimit(eventFailureContextBytes) },
		DirEntries:   func() int { return briefLimit(eventFailureDirEntries) },
		ErrorBytes:   func() int { return briefLimit(eventFailureErrorBytes) },
	})
}

// llmUnfixableErrRe matches LLM-call errors the model cannot fix by being told
// about them: credentials, billing, quotas, unknown models, and request-shape
// rejections (a 422 on the request body is deterministic -- the same history
// produces the same rejection, so retrying only burns the failure budget).
var llmUnfixableErrRe = regexp.MustCompile(
	`(?i)401|403|422|unauthori[sz]ed|forbidden|invalid.?api.?key|api.?key|authenticat` +
		`|permission.?denied|billing|quota|insufficient.?(funds|credit|balance)` +
		`|model.?not.?found|no such model|unknown model|404` +
		`|failed to deserialize the json body`,
)

// efficiencyFailureStop soft-stops after a failed tool call: it queues a hidden
// brief with the error and the context needed to fix it, and returns true so
// the caller leaves the [TOOL FAILED] banner out of history. It returns false
// when the governor is off or the turn's failure budget is spent.
func (l *Loop) efficiencyFailureStop(tc domain.StreamedToolCall, result string) bool {
	n, limit, ok := l.efficiencyFailureSlot()
	if !ok {
		return false
	}
	l.eff.reason = tc.Name + " failed"
	var b strings.Builder
	fmt.Fprintf(&b,
		"%s Soft stop (failure %d/%s). Your last call failed and nothing changed: %s (%s). Error: %s",
		efficiencyMark, n, limit, tc.Name, summarizeToolArgs(tc.Arguments),
		failbrief.Quote(failbrief.ErrorText(result)))
	if ctx := failureFileContext(tc); ctx != "" {
		b.WriteString("\n\n")
		b.WriteString(ctx)
	}
	b.WriteString("\n\n" + failbrief.Needed + " Do not mention this notice.")
	l.queueEfficiencyBrief(b.String())
	l.eff.briefedFailure = true
	return true
}

// takeBriefedFailure reports and clears whether the last tool call was
// soft-stopped.
func (l *Loop) takeBriefedFailure() bool {
	briefed := l.eff.briefedFailure
	l.eff.briefedFailure = false
	return briefed
}

// attachToFailureBrief appends a soft-stopped call's status note (retry
// examples, edited-files ledger) to its brief, keeping it out of history and
// out of the human-visible tool status.
func (l *Loop) attachToFailureBrief(note string) {
	note = strings.TrimSpace(note)
	if note == "" || len(l.eff.briefs) == 0 {
		return
	}
	last := len(l.eff.briefs) - 1
	l.eff.briefs[last] += "\n\n" + note
}

// efficiencyLLMErrorStop soft-stops after a failed LLM call the model may be
// able to avoid (a rejected request, a malformed tool call). It queues a hidden
// brief and returns true so the caller retries the round. Transient errors
// (already retried), context overflow (compacted by the caller) and
// credential/billing/model errors are never retried this way.
func (l *Loop) efficiencyLLMErrorStop(err error) bool {
	if err == nil || isTransientError(err) || IsContextOverflowError(err) ||
		llmUnfixableErrRe.MatchString(err.Error()) {
		return false
	}
	n, limit, ok := l.efficiencyFailureSlot()
	if !ok {
		return false
	}
	l.eff.reason = "the model call failed"
	l.queueEfficiencyBrief(fmt.Sprintf(
		"%s Soft stop (failure %d/%s). Your last response could not be processed: %s. "+
			"Needed: answer again, and if you call a tool use exactly one valid call with JSON "+
			"arguments that match its schema. Do not mention this notice.",
		efficiencyMark, n, limit, failbrief.Quote(err.Error())))
	return true
}

// efficiencyFailureSlot claims one failure soft stop for this turn. It reports
// the failure number, the budget for display, and whether a stop is allowed.
func (l *Loop) efficiencyFailureSlot() (int, string, bool) {
	e := &l.eff
	if !e.active || e.ended {
		return 0, "", false
	}
	limit := failbrief.Failures()
	if e.failures >= limit {
		return 0, "", false
	}
	e.failures++
	shown := "unlimited"
	if limit != disabledSentinel {
		shown = strconv.Itoa(limit)
	}
	return e.failures, shown, true
}

// queueEfficiencyBrief adds a one-shot brief for the next model call.
func (l *Loop) queueEfficiencyBrief(note string) {
	l.eff.briefs = append(l.eff.briefs, note)
}

// failureFileContext is failbrief.FileContext for a call with its argument
// aliases resolved.
func failureFileContext(tc domain.StreamedToolCall) string {
	args := map[string]any{}
	if json.Unmarshal([]byte(tc.Arguments), &args) != nil {
		return ""
	}
	normalizeToolArgs(tc.Name, args)
	return failbrief.FileContext(AppFS, tc.Name, args)
}
