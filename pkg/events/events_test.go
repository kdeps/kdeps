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

package events_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/events"
)

// --- Event constructors ---

func TestWorkflowStarted(t *testing.T) {
	ev := events.WorkflowStarted("my-workflow")
	if ev.Event != events.EventWorkflowStarted {
		t.Errorf("want %q, got %q", events.EventWorkflowStarted, ev.Event)
	}
	if ev.WorkflowID != "my-workflow" {
		t.Errorf("want workflowId %q, got %q", "my-workflow", ev.WorkflowID)
	}
	if ev.EmittedAt.IsZero() {
		t.Error("EmittedAt should be set")
	}
}

func TestWorkflowCompleted(t *testing.T) {
	ev := events.WorkflowCompleted("wf")
	if ev.Event != events.EventWorkflowCompleted {
		t.Errorf("want %q, got %q", events.EventWorkflowCompleted, ev.Event)
	}
}

func TestWorkflowFailed(t *testing.T) {
	err := errors.New("timeout: deadline exceeded")
	ev := events.WorkflowFailed("wf", err)
	if ev.Event != events.EventWorkflowFailed {
		t.Errorf("want %q, got %q", events.EventWorkflowFailed, ev.Event)
	}
	if ev.FailureClass != events.FailureClassTimeout {
		t.Errorf("want failure class %q, got %q", events.FailureClassTimeout, ev.FailureClass)
	}
	if ev.Detail == "" {
		t.Error("Detail should be set from error")
	}
}

func TestResourceStarted(t *testing.T) {
	ev := events.ResourceStarted("wf", "step-1", "exec")
	if ev.Event != events.EventResourceStarted {
		t.Errorf("want %q, got %q", events.EventResourceStarted, ev.Event)
	}
	if ev.ActionID != "step-1" {
		t.Errorf("want actionId %q, got %q", "step-1", ev.ActionID)
	}
	if ev.ResourceType != "exec" {
		t.Errorf("want resourceType %q, got %q", "exec", ev.ResourceType)
	}
}

func TestResourceSkipped(t *testing.T) {
	ev := events.ResourceSkipped("wf", "step-2", "llm")
	if ev.Event != events.EventResourceSkipped {
		t.Errorf("want %q, got %q", events.EventResourceSkipped, ev.Event)
	}
}

func TestResourceCompleted(t *testing.T) {
	ev := events.ResourceCompleted("wf", "step-3", "python")
	if ev.Event != events.EventResourceCompleted {
		t.Errorf("want %q, got %q", events.EventResourceCompleted, ev.Event)
	}
}

func TestResourceFailed(t *testing.T) {
	err := errors.New("connection refused: dial tcp")
	ev := events.ResourceFailed("wf", "step-4", "http", err)
	if ev.Event != events.EventResourceFailed {
		t.Errorf("want %q, got %q", events.EventResourceFailed, ev.Event)
	}
	if ev.FailureClass != events.FailureClassInfra {
		t.Errorf("want failure class %q, got %q", events.FailureClassInfra, ev.FailureClass)
	}
}

func TestResourceRetrying(t *testing.T) {
	ev := events.ResourceRetrying("wf", "step-5", "exec", 2)
	if ev.Event != events.EventResourceRetrying {
		t.Errorf("want %q, got %q", events.EventResourceRetrying, ev.Event)
	}
	data, ok := ev.Data.(map[string]int)
	if !ok {
		t.Fatalf("Data should be map[string]int, got %T", ev.Data)
	}
	if data["attempt"] != 2 {
		t.Errorf("want attempt 2, got %d", data["attempt"])
	}
}

// --- EmittedAt is always UTC ---

func TestEventEmittedAtIsUTC(t *testing.T) {
	before := time.Now().UTC()
	ev := events.WorkflowStarted("wf")
	after := time.Now().UTC()
	if ev.EmittedAt.Before(before) || ev.EmittedAt.After(after) {
		t.Errorf("EmittedAt %v not in [%v, %v]", ev.EmittedAt, before, after)
	}
}

// --- ClassifyError ---

func TestClassifyError(t *testing.T) {
	cases := []struct {
		err  string
		want events.FailureClass
	}{
		{"context deadline exceeded", events.FailureClassTimeout},
		{"timeout: 60s expired", events.FailureClassTimeout},
		{"llm provider returned 503", events.FailureClassProvider},
		{"openai API error", events.FailureClassProvider},
		{"validation: required field missing", events.FailureClassValidation},
		{"invalid input: must be non-empty", events.FailureClassValidation},
		{"preflight check failed: unauthorized", events.FailureClassPreflight},
		{"expression compilation failed: syntax error", events.FailureClassCompile},
		{"connection refused: dial tcp 127.0.0.1", events.FailureClassInfra},
		{"unknown resource execution failure", events.FailureClassToolRuntime},
	}
	for _, tc := range cases {
		got := events.ClassifyError(errors.New(tc.err))
		if got != tc.want {
			t.Errorf("ClassifyError(%q) = %q, want %q", tc.err, got, tc.want)
		}
	}
}

func TestClassifyNilError(t *testing.T) {
	if got := events.ClassifyError(nil); got != "" {
		t.Errorf("ClassifyError(nil) should return empty string, got %q", got)
	}
}

// --- NopEmitter ---

func TestNopEmitter(_ *testing.T) {
	var em events.NopEmitter
	em.Emit(events.WorkflowStarted("wf")) // must not panic
	em.Close()
}

// --- NDJSONEmitter ---

func TestNDJSONEmitter_WritesValidJSON(t *testing.T) {
	var buf bytes.Buffer
	em := events.NewNDJSONEmitter(&buf)

	em.Emit(events.WorkflowStarted("my-wf"))
	em.Emit(events.ResourceStarted("my-wf", "step-1", "exec"))
	em.Close()

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 NDJSON lines, got %d: %q", len(lines), buf.String())
	}

	for i, line := range lines {
		var ev events.Event
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Errorf("line %d is not valid JSON: %v\n%s", i+1, err, line)
		}
	}
}

func TestNDJSONEmitter_FieldNames(t *testing.T) {
	var buf bytes.Buffer
	em := events.NewNDJSONEmitter(&buf)
	em.Emit(events.ResourceFailed("wf", "step-1", "browser", errors.New("timeout: expired")))

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, key := range []string{"event", "workflowId", "actionId", "resourceType", "emittedAt", "failureClass", "detail"} {
		if _, ok := m[key]; !ok {
			t.Errorf("expected key %q in JSON output", key)
		}
	}
}

func TestNDJSONEmitter_OmitsEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	em := events.NewNDJSONEmitter(&buf)
	em.Emit(events.WorkflowStarted("wf"))

	var m map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &m); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	for _, key := range []string{"actionId", "resourceType", "failureClass", "detail", "data"} {
		if _, ok := m[key]; ok {
			t.Errorf("unexpected key %q in JSON (should be omitted when empty)", key)
		}
	}
}

// --- ChanEmitter ---

func TestChanEmitter_SendsAndReceives(t *testing.T) {
	em := events.NewChanEmitter(10)
	ev := events.WorkflowStarted("wf")
	em.Emit(ev)

	select {
	case got := <-em.C():
		if got.Event != events.EventWorkflowStarted {
			t.Errorf("got %q, want %q", got.Event, events.EventWorkflowStarted)
		}
	default:
		t.Error("no event received from channel")
	}
	em.Close()
}

func TestChanEmitter_DropsWhenFull(_ *testing.T) {
	em := events.NewChanEmitter(1)
	em.Emit(events.WorkflowStarted("wf"))
	em.Emit(events.WorkflowStarted("wf2")) // buffer full — must not block
	em.Close()
}

// --- MultiEmitter ---

func TestMultiEmitter_FansOut(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	em1 := events.NewNDJSONEmitter(&buf1)
	em2 := events.NewNDJSONEmitter(&buf2)
	multi := events.NewMultiEmitter(em1, em2)

	multi.Emit(events.WorkflowStarted("wf"))
	multi.Close()

	if buf1.Len() == 0 {
		t.Error("emitter 1 received no events")
	}
	if buf2.Len() == 0 {
		t.Error("emitter 2 received no events")
	}
}
