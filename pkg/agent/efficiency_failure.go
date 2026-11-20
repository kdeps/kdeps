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
	"strings"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/efficiency"
	"github.com/kdeps/kdeps/v3/pkg/failbrief"
	"github.com/kdeps/kdeps/v3/pkg/overflow"
)

// Failure soft stops: a failed tool call or a failed LLM call is stopped by
// kdeps itself, not left for the model to notice. The error, plus what the
// model needs to fix it (the file lines around a failed edit, the folder of a
// missing path), goes to the model on the hidden model-facing channel and the
// model retries. Past the per-turn failure budget the plain [TOOL FAILED]
// banner (tool calls) or the error itself (LLM calls) takes over.

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

// efficiencyFailureStop soft-stops after a failed tool call: it queues a hidden
// brief with the error and the context needed to fix it, and returns true so
// the caller leaves the [TOOL FAILED] banner out of history.
func (l *Loop) efficiencyFailureStop(tc domain.StreamedToolCall, result string) bool {
	n, limit, ok := l.eff.FailureSlot(failbrief.Failures())
	if !ok {
		return false
	}
	l.eff.Reason = tc.Name + " failed"
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
	l.eff.QueueBrief(b.String())
	l.eff.BriefedFailure = true
	return true
}

// takeBriefedFailure reports and clears whether the last call was soft-stopped.
func (l *Loop) takeBriefedFailure() bool { return l.eff.TakeBriefedFailure() }

// attachToFailureBrief appends a soft-stopped call's status note to its brief.
func (l *Loop) attachToFailureBrief(note string) { l.eff.AttachToBrief(note) }

// efficiencyLLMErrorStop soft-stops after a failed LLM call the model may be
// able to avoid. Transient errors (already retried), context overflow
// (compacted by the caller) and credential/billing/model errors are never
// retried this way.
func (l *Loop) efficiencyLLMErrorStop(err error) bool {
	if err == nil || isTransientError(err) || overflow.Is(err) || efficiency.Unfixable(err) {
		return false
	}
	n, limit, ok := l.eff.FailureSlot(failbrief.Failures())
	if !ok {
		return false
	}
	l.eff.Reason = "the model call failed"
	l.eff.QueueBrief(efficiency.LLMErrorBrief(n, limit, failbrief.Quote(err.Error())))
	return true
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
