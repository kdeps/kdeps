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

// Package events provides a structured, machine-readable execution event stream
// for kdeps workflow runs. Events are typed and classified so external systems
// (dashboards, recovery loops, orchestrators) can react without parsing logs.
package events

import "time"

// EventName identifies the lifecycle point of an event.
type EventName string

const (
	// EventWorkflowStarted fires when a workflow begins executing resources.
	EventWorkflowStarted EventName = "workflow.started"
	// EventWorkflowCompleted fires when a workflow finishes successfully.
	EventWorkflowCompleted EventName = "workflow.completed"
	// EventWorkflowFailed fires when a workflow terminates with an error.
	EventWorkflowFailed EventName = "workflow.failed"

	// EventResourceStarted fires immediately before a resource executes.
	EventResourceStarted EventName = "resource.started"
	// EventResourceSkipped fires when a resource is skipped (skip condition or route mismatch).
	EventResourceSkipped EventName = "resource.skipped"
	// EventResourceCompleted fires after a resource executes successfully.
	EventResourceCompleted EventName = "resource.completed"
	// EventResourceFailed fires when a resource execution returns an error.
	EventResourceFailed EventName = "resource.failed"
	// EventResourceRetrying fires when a resource is about to be retried.
	EventResourceRetrying EventName = "resource.retrying"
)

// FailureClass classifies why a resource or workflow failed.
// Mirrors the taxonomy from claw-code's LaneFailureClass.
type FailureClass string

const (
	// FailureClassValidation indicates an input validation failure.
	FailureClassValidation FailureClass = "validation"
	// FailureClassProvider indicates an LLM/model provider failure.
	FailureClassProvider FailureClass = "provider"
	// FailureClassToolRuntime indicates an exec, browser, or scraper failure.
	FailureClassToolRuntime FailureClass = "tool_runtime"
	// FailureClassCompile indicates a syntax or expression compile failure.
	FailureClassCompile FailureClass = "compile"
	// FailureClassTimeout indicates a timeout or deadline exceeded.
	FailureClassTimeout FailureClass = "timeout"
	// FailureClassPreflight indicates an authentication or authorization failure.
	FailureClassPreflight FailureClass = "preflight"
	// FailureClassInfra indicates a network or connectivity failure.
	FailureClassInfra FailureClass = "infra"
)

// Event is a single lifecycle event emitted during workflow execution.
// Fields are omitted from JSON when empty to keep the stream compact.
type Event struct {
	Event        EventName    `json:"event"`
	WorkflowID   string       `json:"workflowId,omitempty"`
	ActionID     string       `json:"actionId,omitempty"`
	ResourceType string       `json:"resourceType,omitempty"`
	EmittedAt    time.Time    `json:"emittedAt"`
	FailureClass FailureClass `json:"failureClass,omitempty"`
	Detail       string       `json:"detail,omitempty"`
	Data         any          `json:"data,omitempty"`
}

// WorkflowStarted returns a workflow.started event.
func WorkflowStarted(workflowID string) Event {
	return Event{
		Event:      EventWorkflowStarted,
		WorkflowID: workflowID,
		EmittedAt:  time.Now().UTC(),
	}
}

// WorkflowCompleted returns a workflow.completed event.
func WorkflowCompleted(workflowID string) Event {
	return Event{
		Event:      EventWorkflowCompleted,
		WorkflowID: workflowID,
		EmittedAt:  time.Now().UTC(),
	}
}

// WorkflowFailed returns a workflow.failed event with classified failure.
func WorkflowFailed(workflowID string, err error) Event {
	return Event{
		Event:        EventWorkflowFailed,
		WorkflowID:   workflowID,
		EmittedAt:    time.Now().UTC(),
		FailureClass: ClassifyError(err),
		Detail:       err.Error(),
	}
}

// ResourceStarted returns a resource.started event.
func ResourceStarted(workflowID, actionID, resourceType string) Event {
	return Event{
		Event:        EventResourceStarted,
		WorkflowID:   workflowID,
		ActionID:     actionID,
		ResourceType: resourceType,
		EmittedAt:    time.Now().UTC(),
	}
}

// ResourceSkipped returns a resource.skipped event.
func ResourceSkipped(workflowID, actionID, resourceType string) Event {
	return Event{
		Event:        EventResourceSkipped,
		WorkflowID:   workflowID,
		ActionID:     actionID,
		ResourceType: resourceType,
		EmittedAt:    time.Now().UTC(),
	}
}

// ResourceCompleted returns a resource.completed event.
func ResourceCompleted(workflowID, actionID, resourceType string) Event {
	return Event{
		Event:        EventResourceCompleted,
		WorkflowID:   workflowID,
		ActionID:     actionID,
		ResourceType: resourceType,
		EmittedAt:    time.Now().UTC(),
	}
}

// ResourceFailed returns a resource.failed event with classified failure.
func ResourceFailed(workflowID, actionID, resourceType string, err error) Event {
	return Event{
		Event:        EventResourceFailed,
		WorkflowID:   workflowID,
		ActionID:     actionID,
		ResourceType: resourceType,
		EmittedAt:    time.Now().UTC(),
		FailureClass: ClassifyError(err),
		Detail:       err.Error(),
	}
}

// ResourceRetrying returns a resource.retrying event.
func ResourceRetrying(workflowID, actionID, resourceType string, attempt int) Event {
	return Event{
		Event:        EventResourceRetrying,
		WorkflowID:   workflowID,
		ActionID:     actionID,
		ResourceType: resourceType,
		EmittedAt:    time.Now().UTC(),
		Data:         map[string]int{"attempt": attempt},
	}
}
