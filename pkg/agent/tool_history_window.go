/*
 * Copyright 2026 Kdeps, KvK 94834768
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package agent

import (
	"encoding/json"
	"fmt"
)

// toolLoopMessageBudget bounds the total estimated token size of the in-flight
// message array during a single turn's tool loop. Past this, the oldest
// complete tool round-trips are dropped so a long loop (many tool calls in one
// turn) does not re-send an ever-growing transcript on every round -- the
// dominant token cost in a long agent session. Auto-compaction only runs at
// turn boundaries, so without this the transcript grows unbounded within a turn
// up to MaxToolRounds (default 200) round trips.
const toolLoopMessageBudget = 24 * 1024

// toolRoundTripArgTruncateAt is the argument-string length past which a retained
// (older) assistant tool-call's long string arguments are replaced with a
// placeholder in windowed history. A large write_file / edit_file content
// argument is pure re-sent weight once its result is in the transcript.
const toolRoundTripArgTruncateAt = 256

// windowKeepRecentRoundTrips is the number of most-recent kept round-trips whose
// tool-call arguments are never truncated -- the model may still be acting on
// them.
const windowKeepRecentRoundTrips = 2

func msgRole(m map[string]any) string {
	r, _ := m["role"].(string)
	return r
}

func msgHasToolCalls(m map[string]any) bool {
	tc, ok := m["tool_calls"]
	if !ok || tc == nil {
		return false
	}
	switch v := tc.(type) {
	case []any:
		return len(v) > 0
	case []map[string]any:
		return len(v) > 0
	default:
		return true
	}
}

// isRoundTripStart reports whether m is the assistant message that opens a tool
// round-trip (assistant turn carrying one or more tool_calls).
func isRoundTripStart(m map[string]any) bool {
	return msgRole(m) == RoleAssistant && msgHasToolCalls(m)
}

func msgTokens(m map[string]any, model string) int {
	b, err := json.Marshal(m)
	if err != nil {
		return 0
	}
	return countTokensSilent(model, string(b))
}

func msgsTokens(ms []map[string]any, model string) int {
	total := 0
	for _, m := range ms {
		total += msgTokens(m, model)
	}
	return total
}

// historyBytes is a cheap upper-bound proxy for token size: the JSON-marshaled
// length of every message. Used to skip the tokenizer on small transcripts.
func historyBytes(ms []map[string]any) int {
	total := 0
	for _, m := range ms {
		if b, err := json.Marshal(m); err == nil {
			total += len(b)
		}
	}
	return total
}

// windowToolHistory bounds the in-flight message array during a turn's tool
// loop. When the transcript exceeds toolLoopMessageBudget it drops the oldest
// complete tool round-trips (an assistant-with-tool_calls message plus every
// tool result that follows it), keeps the newest round-trips that fit, and
// truncates long string arguments on the older kept round-trips. The leading
// conversation -- session history plus this turn's user question -- is always
// kept intact, and at least one full round-trip is always retained even if it
// alone exceeds the budget. A dropped span is announced by a short note
// prepended to the first kept round-trip's assistant content, so no extra
// message (and no consecutive same-role pair) is introduced.
//
// Returns the windowed history and the number of round-trips dropped.
func windowToolHistory(history []map[string]any, model string) ([]map[string]any, int) {
	if len(history) == 0 {
		return history, 0
	}
	// Cheap byte-length pre-check: at ~2 bytes/token minimum, a transcript under
	// budget*2 bytes cannot exceed the token budget, so skip the (per-round)
	// tokenizer pass entirely in the common case.
	if historyBytes(history) <= toolLoopMessageBudget*2 {
		return history, 0
	}
	if msgsTokens(history, model) <= toolLoopMessageBudget {
		return history, 0
	}

	head := 0
	for head < len(history) && !isRoundTripStart(history[head]) {
		head++
	}
	if head == len(history) {
		return history, 0 // nothing but conversation; not ours to trim
	}
	headMsgs := history[:head]
	rest := history[head:]

	// Segment the tail into round-trips: each starts at an assistant tool-call
	// message and runs through the tool results that follow it.
	var rts [][]map[string]any
	for i := 0; i < len(rest); {
		j := i + 1
		for j < len(rest) && msgRole(rest[j]) == roleTool {
			j++
		}
		rts = append(rts, rest[i:j])
		i = j
	}
	if len(rts) <= 1 {
		return history, 0 // can't drop the only round-trip
	}

	budget := toolLoopMessageBudget - msgsTokens(headMsgs, model)
	if budget < 0 {
		budget = 0
	}

	keptCount, used := 0, 0
	for k := len(rts) - 1; k >= 0; k-- {
		segTok := msgsTokens(rts[k], model)
		if keptCount > 0 && used+segTok > budget {
			break
		}
		used += segTok
		keptCount++
	}
	if keptCount < 1 {
		keptCount = 1
	}
	dropped := len(rts) - keptCount
	if dropped <= 0 {
		return history, 0
	}

	keptRTs := rts[len(rts)-keptCount:]
	for idx := range max(0, len(keptRTs)-windowKeepRecentRoundTrips) {
		truncateRoundTripArgs(keptRTs[idx])
	}
	prependDroppedNote(keptRTs[0], dropped)

	out := make([]map[string]any, 0, len(history))
	out = append(out, headMsgs...)
	for _, seg := range keptRTs {
		out = append(out, seg...)
	}
	return out, dropped
}

const roleTool = "tool"

func prependDroppedNote(seg []map[string]any, dropped int) {
	if len(seg) == 0 {
		return
	}
	prev, _ := seg[0][toolParamContent].(string)
	note := fmt.Sprintf(
		"[note: %d earlier tool round-trip(s) from this turn were dropped to fit the context window]",
		dropped)
	if prev != "" {
		note += "\n" + prev
	}
	seg[0][toolParamContent] = note
}

// truncateRoundTripArgs replaces long string values inside a retained assistant
// message's tool-call arguments with a "[N chars omitted]" placeholder, keeping
// the arguments valid JSON and preserving short identifying fields (file_path,
// url, query, ...).
func truncateRoundTripArgs(seg []map[string]any) {
	if len(seg) == 0 {
		return
	}
	raw, isArr := seg[0]["tool_calls"].([]any)
	if !isArr {
		return
	}
	for _, item := range raw {
		if fn := callArgsFn(item); fn != nil {
			shrinkArgFields(fn)
		}
	}
}

// callArgsFn returns the "function" map of one tool_calls entry, or nil.
func callArgsFn(item any) map[string]any {
	call, isMap := item.(map[string]any)
	if !isMap {
		return nil
	}
	fn, isMap := call["function"].(map[string]any)
	if !isMap {
		return nil
	}
	return fn
}

// shrinkArgFields parses fn["arguments"] (a JSON object string) and swaps any
// long string value for a placeholder, re-marshaling in place.
func shrinkArgFields(fn map[string]any) {
	args, isStr := fn["arguments"].(string)
	if !isStr {
		return
	}
	var parsed map[string]any
	if json.Unmarshal([]byte(args), &parsed) != nil {
		return
	}
	changed := false
	for k, v := range parsed {
		s, ok := v.(string)
		if !ok || len(s) <= toolRoundTripArgTruncateAt {
			continue
		}
		parsed[k] = fmt.Sprintf("[%d chars omitted; see the tool result]", len(s))
		changed = true
	}
	if changed {
		if b, err := json.Marshal(parsed); err == nil {
			fn["arguments"] = string(b)
		}
	}
}
