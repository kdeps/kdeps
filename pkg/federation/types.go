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

package federation

import (
	"time"

	"github.com/google/uuid"
)

// JSONSchema represents a JSON Schema for input/output validation.
type JSONSchema struct {
	Schema       string                 `json:"$schema,omitempty"`
	Type         string                 `json:"type,omitempty"`
	Properties   map[string]Property   `json:"properties,omitempty"`
	Required     []string               `json:"required,omitempty"`
	Definitions  map[string]JSONSchema  `json:"definitions,omitempty"`
	Items        *JSONSchema            `json:"items,omitempty"` // for arrays
	Enum         []interface{}          `json:"enum,omitempty"`
	Format       string                 `json:"format,omitempty"`
	Description  string                 `json:"description,omitempty"`
}

// Property describes a single property in a JSON Schema.
type Property struct {
	Type        string      `json:"type,omitempty"`
	Format      string      `json:"format,omitempty"`
	Description string      `json:"description,omitempty"`
	Items       *JSONSchema `json:"items,omitempty"`
	Enum        []interface{} `json:"enum,omitempty"`
}

// Action describes a capability exposed by an agent.
type Action struct {
	ActionID      string     `json:"actionId"`
	Title         string     `json:"title,omitempty"`
	Description   string     `json:"description,omitempty"`
	InputSchema   JSONSchema `json:"inputSchema"`
	OutputSchema  JSONSchema `json:"outputSchema"`
}

// Capability describes what an agent can do.
type Capability struct {
	URN             string    `json:"urn"`
	Title           string    `json:"title,omitempty"`
	Description     string    `json:"description,omitempty"`
	Version         string    `json:"version,omitempty"`
	Capabilities    []Action  `json:"capabilities"`
	AuthMethods     []string  `json:"authMethods,omitempty"`
	RequiredScopes  []string  `json:"requiredScopes,omitempty"`
	RateLimit       RateLimit `json:"rateLimit,omitempty"`
	Endpoint        string    `json:"endpoint,omitempty"`
	Contact         string    `json:"contact,omitempty"`
	Documentation   string    `json:"documentation,omitempty"`
	TrustLevel      string    `json:"trustLevel,omitempty"` // self-attested, verified, certified
}

// RateLimit describes throttling parameters.
type RateLimit struct {
	RequestsPerMinute int `json:"requestsPerMinute"`
	Burst             int `json:"burst,omitempty"`
}

// CallerIdentity identifies the invoking agent.
type CallerIdentity struct {
	URN        string `json:"urn"`
	PublicKey  string `json:"publicKey"`  // ed25519:base64...
}

// CalleeIdentity identifies the target agent.
type CalleeIdentity struct {
	URN string `json:"urn"`
}

// RequestPayload contains the invocation inputs and context.
type RequestPayload struct {
	Inputs     map[string]interface{} `json:"inputs"`
	Context    InvocationContext     `json:"context"`
}

// InvocationContext carries metadata about the request.
type InvocationContext struct {
	RequestID     string `json:"requestId"`
	CorrelationID string `json:"correlationId,omitempty"`
	CallerIP      string `json:"callerIp,omitempty"`
}

// InvocationRequest is sent by caller to callee.
type InvocationRequest struct {
	MessageID   uuid.UUID        `json:"messageId"`
	Timestamp   time.Time        `json:"timestamp"`
	Caller      CallerIdentity   `json:"caller"`
	Callee      CalleeIdentity   `json:"callee"`
	Payload     RequestPayload   `json:"payload"`
}

// ExecutionResult describes the outcome of remote execution.
type ExecutionResult struct {
	Status      string                 `json:"status"` // success, error, timeout
	DurationMs  int                    `json:"durationMs,omitempty"`
	Outputs     map[string]interface{} `json:"outputs,omitempty"`
	Error       *ExecutionError        `json:"error,omitempty"`
}

// ExecutionError describes a failure.
type ExecutionError struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Receipt is the signed response from callee.
type Receipt struct {
	MessageID   uuid.UUID        `json:"messageId"`
	Timestamp   time.Time        `json:"timestamp"`
	Callee      URN              `json:"callee"`
	Caller      URN              `json:"caller"`
	Execution   ExecutionResult  `json:"execution"`
	Tracing     *TracingInfo     `json:"tracing,omitempty"`
}

// TracingInfo contains OpenTelemetry identifiers.
type TracingInfo struct {
	TraceID  string `json:"traceId,omitempty"`
	SpanID   string `json:"spanId,omitempty"`
}
