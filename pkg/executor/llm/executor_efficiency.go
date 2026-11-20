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

package llm

import (
	"fmt"
	"strings"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/efficiency"
	"github.com/kdeps/kdeps/v3/pkg/failbrief"
	"github.com/kdeps/kdeps/v3/pkg/harness"
)

// newChatEfficiency is a chat call's efficiency ledger. It is active only
// with harness.efficiency and at least one output tool.
func newChatEfficiency(cfg *domain.ChatConfig, tools []domain.Tool) *efficiency.Ledger {
	names := make([]string, len(tools))
	for i, t := range tools {
		names[i] = t.Name
	}
	eff := &efficiency.Ledger{}
	eff.BeginTurn(cfg != nil && cfg.Efficiency && efficiency.HasOutputTool(names))
	return eff
}

// admitRound runs the governor on each call of a round. The first call that
// is not admitted decides the round.
func admitRound(eff *efficiency.Ledger, calls []map[string]interface{}) efficiency.Verdict {
	for _, c := range calls {
		name, args, _, _ := parseToolCallFunction(c)
		if v := eff.Admit(name, args); v != efficiency.Admit {
			return v
		}
	}
	return efficiency.Admit
}

// afterRound closes an open stop when an output call of the round succeeded.
func afterRound(eff *efficiency.Ledger, calls, results []map[string]interface{}) {
	for i, c := range calls {
		name, args, _, _ := parseToolCallFunction(c)
		ok := i < len(results) && results[i][fieldError] == nil
		eff.AfterOutput(name, args, ok)
	}
}

// repeatStop briefs a model that repeated a round it has the results of. It
// returns false when the governor is off or its stops are spent.
func (g *callGuard) repeatStop(eff *efficiency.Ledger, calls []map[string]interface{}) bool {
	if len(calls) == 0 {
		return false
	}
	name, args, _, _ := parseToolCallFunction(calls[0])
	key := toolCallKey(calls[0])
	if !eff.RepeatStop(name, args, fmt.Sprint(g.results[key]), g.issued[key]) {
		return false
	}
	for _, c := range calls {
		delete(g.issued, toolCallKey(c))
	}
	return true
}

// requestTools are the request's tools, without read tools while a stop is
// open.
func requestTools(eff *efficiency.Ledger, rc ChatRequestConfig) (ChatRequestConfig, []string) {
	names := make([]string, 0, len(rc.Tools))
	kept := make([]map[string]interface{}, 0, len(rc.Tools))
	for _, t := range rc.Tools {
		fn, _ := t[fieldFunction].(map[string]interface{})
		name, _ := fn[fieldName].(string)
		names = append(names, name)
		if eff.KeepTool(name) {
			kept = append(kept, t)
		}
	}
	if eff.Open {
		rc.Tools = kept
	}
	return rc, names
}

// summarizeHarness replaces the chat's harness sections in the system
// message with their one-line summaries when harness.promptMode says so.
// The first request always carries the full sections.
func summarizeHarness(cfg *domain.ChatConfig, messages []map[string]interface{}) []map[string]interface{} {
	if cfg == nil || len(cfg.Harness) == 0 || cfg.PromptMode == "" ||
		!harness.UseSummary(cfg.PromptMode, cfg.Backend) {
		return messages
	}
	full, err := harness.Sections(cfg.Harness)
	if err != nil || full == "" {
		return messages
	}
	short, err := harness.Summaries(cfg.Harness)
	if err != nil || short == "" || short == full {
		return messages
	}
	out := make([]map[string]interface{}, len(messages))
	copy(out, messages)
	for i, m := range out {
		content, ok := m[jsonFieldContent].(string)
		if m["role"] != roleSystem || !ok || !strings.Contains(content, full) {
			continue
		}
		cp := make(map[string]interface{}, len(m))
		for k, v := range m {
			cp[k] = v
		}
		cp[jsonFieldContent] = strings.Replace(content, full, short, 1)
		out[i] = cp
		break
	}
	return out
}

// effFollowUp is the next request of the tool loop: the harness in its
// prompt mode, the governor's hidden note as a last user message, and only
// output tools while a stop is open. A failed request the model may avoid
// is briefed and sent again, within the efficiency-failures budget.
func (e *Executor) effFollowUp(
	cfg *domain.ChatConfig,
	call chatCall,
	messages []map[string]interface{},
	rc ChatRequestConfig,
	eff *efficiency.Ledger,
) (map[string]interface{}, error) {
	base := summarizeHarness(cfg, messages)
	for {
		req, names := requestTools(eff, rc)
		msgs := base
		if note := eff.Note(names); note != "" {
			msgs = append(append([]map[string]interface{}{}, base...),
				map[string]interface{}{"role": "user", jsonFieldContent: note})
		}
		response, err := e.chatFollowUp(cfg, call, msgs, req)
		if err == nil {
			return response, nil
		}
		if efficiency.Unfixable(err) {
			return nil, err
		}
		n, limit, ok := eff.FailureSlot(failbrief.Failures())
		if !ok {
			return nil, err
		}
		eff.QueueBrief(efficiency.LLMErrorBrief(n, limit, failbrief.Quote(err.Error())))
	}
}

// roundReply is the reply that replaces a round the guard did not run.
type roundReply struct {
	response map[string]interface{}
	err      error
}

// governRound runs the efficiency governor (harness.efficiency) and the
// repeat guard on a round before it runs. skip is true when the round does
// not run: a dropped round asks again with the hidden note, and a repeated
// round is briefed while stops remain. done is true when the reply ends the
// loop: the model's final answer, or a failed request.
func (e *Executor) governRound(
	cfg *domain.ChatConfig,
	call chatCall,
	messages []map[string]interface{},
	rc ChatRequestConfig,
	eff *efficiency.Ledger,
	guard *callGuard,
	calls []map[string]interface{},
) (roundReply, bool, bool) {
	answer := func() (roundReply, bool, bool) {
		r, err := e.finalAnswer(cfg, call, messages, rc)
		return roundReply{r, err}, true, true
	}
	again := func() (roundReply, bool, bool) {
		r, err := e.effFollowUp(cfg, call, messages, rc, eff)
		return roundReply{r, err}, err != nil, true
	}
	switch admitRound(eff, calls) {
	case efficiency.End, efficiency.Abort:
		return answer()
	case efficiency.Drop:
		return again()
	case efficiency.Admit:
	}
	// A model that keeps re-issuing calls it already has the results of is
	// looping (common with small models). The governor briefs it while it has
	// stops left; then it has to answer.
	if guard.repeating(calls) {
		if !guard.repeatStop(eff, calls) {
			return answer()
		}
		return again()
	}
	return roundReply{}, false, false
}
