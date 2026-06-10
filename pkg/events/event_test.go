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
	"errors"
	"testing"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/events"
)

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

func TestEventEmittedAtIsUTC(t *testing.T) {
	before := time.Now().UTC()
	ev := events.WorkflowStarted("wf")
	after := time.Now().UTC()
	if ev.EmittedAt.Before(before) || ev.EmittedAt.After(after) {
		t.Errorf("EmittedAt %v not in [%v, %v]", ev.EmittedAt, before, after)
	}
}
