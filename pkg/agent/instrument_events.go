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
	"crypto/sha256"
	"encoding/json"
	"sync"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/instrument"
)

// sentBefore remembers the system prompt and tool list of the last model
// request, so a request records them only when they change: the file stays
// readable for the model that debugs the session.
//
//nolint:gochecknoglobals // one recording per process
var sentBefore struct {
	sync.Mutex
	session        *instrument.Session // the session these refer to; a new one starts over
	seq            int
	scenario, tool [sha256.Size]byte
	scenarioAt     int
	toolsAt        int
}

// unchangedSince is the field value for a part sent as in an earlier request.
func unchangedSince(seq int) map[string]int { return map[string]int{"unchanged_since_request": seq} }

// recordModelRequest records what is about to be sent to the model in an
// instrumented session: everything the model sees (the system prompt and
// tool list only when they differ from the previous request's).
func recordModelRequest(cfg *domain.ChatConfig, attempt int) {
	if instrument.Active() == nil || cfg == nil {
		return
	}
	scenario := make([]map[string]string, 0, len(cfg.Scenario))
	for _, s := range cfg.Scenario {
		scenario = append(scenario, map[string]string{"role": s.Role, "prompt": s.Prompt})
	}
	tools := make([]string, 0, len(cfg.Tools))
	for _, t := range cfg.Tools {
		tools = append(tools, t.Name)
	}
	fields := map[string]any{
		"backend": cfg.Backend, "model": cfg.Model, "attempt": attempt + 1,
		"messages": cfg.Messages, "prompt": cfg.Prompt,
	}
	sh, _ := json.Marshal(scenario)
	th, _ := json.Marshal(tools)
	sentBefore.Lock()
	if s := instrument.Active(); s != sentBefore.session {
		sentBefore.session, sentBefore.seq, sentBefore.scenarioAt, sentBefore.toolsAt = s, 0, 0, 0
	}
	sentBefore.seq++
	fields["request"] = sentBefore.seq
	if h := sha256.Sum256(sh); h != sentBefore.scenario || sentBefore.scenarioAt == 0 {
		sentBefore.scenario, sentBefore.scenarioAt = h, sentBefore.seq
		fields["scenario"] = scenario
	} else {
		fields["scenario"] = unchangedSince(sentBefore.scenarioAt)
	}
	if h := sha256.Sum256(th); h != sentBefore.tool || sentBefore.toolsAt == 0 {
		sentBefore.tool, sentBefore.toolsAt = h, sentBefore.seq
		fields["tools"] = tools
	} else {
		fields["tools"] = unchangedSince(sentBefore.toolsAt)
	}
	sentBefore.Unlock()
	instrument.Event("model_request", fields)
}

// recordModelResponse records what the model answered, or its error.
func recordModelResponse(cfg *domain.ChatConfig, content string, calls []domain.StreamedToolCall, err error,
	took time.Duration,
) {
	if instrument.Active() == nil || cfg == nil {
		return
	}
	fields := map[string]any{
		"backend": cfg.Backend, "model": cfg.Model, "duration_ms": took.Milliseconds(), "content": content,
	}
	if len(calls) > 0 {
		list := make([]map[string]string, 0, len(calls))
		for _, c := range calls {
			list = append(list, map[string]string{"id": c.ID, "name": c.Name, "arguments": c.Arguments})
		}
		fields["tool_calls"] = list
	}
	if err != nil {
		fields["error"] = err.Error()
	}
	instrument.Event("model_response", fields)
}

// recordToolCall records a tool call the agent loop ran: the name the model
// used, the tool it resolved to, its arguments and result, and how long it took.
func recordToolCall(called, tool, args, result string, took time.Duration) {
	if instrument.Active() == nil {
		return
	}
	fields := map[string]any{"name": tool, "arguments": args, "result": result, "duration_ms": took.Milliseconds()}
	if called != tool {
		fields["called_as"] = called
	}
	if isToolErrorResult(result) {
		fields["error"] = result
	}
	instrument.Event("tool_call", fields)
}

// recordPermissionDecision records one permission gate decision: the tool,
// the mode it needs, the mode in force, and whether the call was allowed.
// The governance audit record: it shows which rule decided each call.
func recordPermissionDecision(tool string, required, mode PermissionMode, reason string, blocked bool) {
	if instrument.Active() == nil {
		return
	}
	decision := "allow"
	if blocked {
		decision = "deny"
	}
	fields := map[string]any{
		"name":     tool,
		"required": string(required),
		"mode":     string(mode),
		"decision": decision,
	}
	if reason != "" {
		fields["reason"] = reason
	}
	instrument.Event("permission_decision", fields)
}

// recordPrompt records a request you typed.
func recordPrompt(text string) {
	instrument.Event("prompt", map[string]any{"text": text})
}
