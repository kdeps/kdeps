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

	"github.com/kdeps/kdeps/v2/pkg/events"
)

func TestNopEmitter(_ *testing.T) {
	var em events.NopEmitter
	em.Emit(events.WorkflowStarted("wf")) // must not panic
	em.Close()
}

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

func TestNDJSONEmitter_Emit(t *testing.T) {
	var buf bytes.Buffer
	em := events.NewNDJSONEmitter(&buf)
	em.Emit(events.WorkflowStarted("wf-emit-test"))
	if buf.Len() == 0 {
		t.Fatal("expected non-empty output after Emit")
	}
	var ev events.Event
	if err := json.Unmarshal(buf.Bytes(), &ev); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	if ev.WorkflowID != "wf-emit-test" {
		t.Errorf("workflowId = %q, want %q", ev.WorkflowID, "wf-emit-test")
	}
}

func TestNDJSONEmitter_Close(_ *testing.T) {
	var buf bytes.Buffer
	em := events.NewNDJSONEmitter(&buf)
	em.Close() // must not panic
}

func TestMultiEmitter_EmitAndClose(t *testing.T) {
	sink1 := events.NewChanEmitter(4)
	sink2 := events.NewChanEmitter(4)
	multi := events.NewMultiEmitter(sink1, sink2)

	multi.Emit(events.WorkflowStarted("wf"))
	multi.Close()

	select {
	case got := <-sink1.C():
		if got.Event != events.EventWorkflowStarted {
			t.Errorf("sink1 got %q, want %q", got.Event, events.EventWorkflowStarted)
		}
	default:
		t.Error("sink1 received no event")
	}

	select {
	case got := <-sink2.C():
		if got.Event != events.EventWorkflowStarted {
			t.Errorf("sink2 got %q, want %q", got.Event, events.EventWorkflowStarted)
		}
	default:
		t.Error("sink2 received no event")
	}
}

func TestChanEmitter_Emit(t *testing.T) {
	em := events.NewChanEmitter(4)
	ev := events.WorkflowCompleted("wf")
	em.Emit(ev)
	select {
	case got := <-em.C():
		if got.Event != events.EventWorkflowCompleted {
			t.Errorf("got %q, want %q", got.Event, events.EventWorkflowCompleted)
		}
	default:
		t.Error("channel empty after Emit")
	}
	em.Close()
}

func TestChanEmitter_Emit_DropWhenFull(t *testing.T) {
	em := events.NewChanEmitter(0) // zero-buffer: all sends drop
	em.Emit(events.WorkflowStarted("wf"))
	// must not block or panic; channel stays empty
	select {
	case <-em.C():
		t.Error("expected empty channel but got an event")
	default:
	}
	em.Close()
}

func TestChanEmitter_Close(t *testing.T) {
	em := events.NewChanEmitter(2)
	em.Emit(events.WorkflowStarted("wf"))
	em.Close()
	// After Close the channel should be closed (range terminates).
	count := 0
	for range em.C() {
		count++
	}
	if count != 1 {
		t.Errorf("expected 1 event before close, got %d", count)
	}
}

func TestChanEmitter_C(t *testing.T) {
	em := events.NewChanEmitter(1)
	ch := em.C()
	if ch == nil {
		t.Fatal("C() returned nil channel")
	}
	em.Close()
}

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
