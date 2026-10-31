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

package executor

import (
	"context"
	"encoding/json"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/instrument"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/events"
)

// SetEmitter configures the event emitter for this engine.
// Call before Execute to receive structured lifecycle events.
// In an instrumented session the events are recorded there too.
func (e *Engine) SetEmitter(em events.Emitter) {
	kdeps_debug.Log("enter: SetEmitter")
	if em == nil {
		e.emitter = withInstrument(events.NopEmitter{})
		return
	}
	e.emitter = withInstrument(em)
}

// withInstrument adds the instrumented session's recorder to em, when a
// session is recording.
func withInstrument(em events.Emitter) events.Emitter {
	if instrument.Active() == nil {
		return em
	}
	if _, nop := em.(events.NopEmitter); nop {
		return instrumentEmitter{}
	}
	return events.NewMultiEmitter(em, instrumentEmitter{})
}

// instrumentEmitter records workflow events in the instrumented session.
type instrumentEmitter struct{}

func (instrumentEmitter) Emit(ev events.Event) {
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	var fields map[string]any
	if json.Unmarshal(data, &fields) != nil {
		return
	}
	instrument.Event("workflow", fields)
}

func (instrumentEmitter) Close() {}

// SetRegistry sets the executor registry.
func (e *Engine) SetRegistry(registry *Registry) {
	kdeps_debug.Log("enter: SetRegistry")
	e.registry = registry
}

// SetDebugMode enables or disables debug mode.
func (e *Engine) SetDebugMode(enabled bool) {
	kdeps_debug.Log("enter: SetDebugMode")
	e.debugMode = enabled
}

// SetNewExecutionContextForAgency overrides the execution-context factory so
// every context created by this engine carries the provided agentPaths map.
// This allows resources using the `agent` type to call sibling agents by name.
func (e *Engine) SetNewExecutionContextForAgency(agentPaths map[string]string) {
	kdeps_debug.Log("enter: SetNewExecutionContextForAgency")
	e.newExecutionContext = func(
		parent context.Context, workflow *domain.Workflow, sessionID string,
	) (*ExecutionContext, error) {
		ctx, err := defaultExecutionContext(parent, workflow, sessionID)
		if err != nil {
			return nil, err
		}
		ctx.AgentPaths = agentPaths
		ctx.AgentMemoryStore = e.agentMemoryStore
		return ctx, nil
	}
}

// SetAgentMemoryStore configures the agent's persistent memory store on this engine.
// When set, every ExecutionContext created by this engine will carry the store,
// making memory_save, memory_search, memory_list, and memory_delete expression
// functions available in workflow resources.
func (e *Engine) SetAgentMemoryStore(ms AgentMemoryStore) {
	kdeps_debug.Log("enter: SetAgentMemoryStore")
	e.agentMemoryStore = ms
}
