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

package federation

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validURNString is a well-formed URN used across types tests.
const validURNString = "urn:agent:example.com/testns:myagent@v1.2.3#sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

// mustParseURN parses a URN string and panics on error, for use in test setup.
func mustParseURN(s string) URN {
	u, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return *u
}

func TestInvocationRequestMarshalJSON(t *testing.T) {
	msgID := uuid.New()
	ts := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	req := InvocationRequest{
		MessageID: msgID,
		Timestamp: ts,
		Caller: CallerIdentity{
			URN:       "urn:agent:caller.example.com/org:caller-agent@v1.0.0#sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			PublicKey: "ed25519:dGVzdHB1YmxpY2tleQ==",
		},
		Callee: CalleeIdentity{
			URN: validURNString,
		},
		Payload: RequestPayload{
			Inputs: map[string]interface{}{
				"query":  "hello world",
				"limit":  float64(10),
				"active": true,
			},
			Context: InvocationContext{
				RequestID:     "req-abc-123",
				CorrelationID: "corr-xyz-456",
				CallerIP:      "192.168.1.1",
			},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err, "Marshal should not error")
	require.NotEmpty(t, data)

	var got InvocationRequest
	err = json.Unmarshal(data, &got)
	require.NoError(t, err, "Unmarshal should not error")

	assert.Equal(t, req.MessageID, got.MessageID)
	assert.True(t, req.Timestamp.Equal(got.Timestamp))
	assert.Equal(t, req.Caller.URN, got.Caller.URN)
	assert.Equal(t, req.Caller.PublicKey, got.Caller.PublicKey)
	assert.Equal(t, req.Callee.URN, got.Callee.URN)
	assert.Equal(t, req.Payload.Inputs["query"], got.Payload.Inputs["query"])
	assert.Equal(t, req.Payload.Inputs["limit"], got.Payload.Inputs["limit"])
	assert.Equal(t, req.Payload.Inputs["active"], got.Payload.Inputs["active"])
	assert.Equal(t, req.Payload.Context.RequestID, got.Payload.Context.RequestID)
	assert.Equal(t, req.Payload.Context.CorrelationID, got.Payload.Context.CorrelationID)
	assert.Equal(t, req.Payload.Context.CallerIP, got.Payload.Context.CallerIP)
}

func TestInvocationRequestJSONFieldNames(t *testing.T) {
	req := InvocationRequest{
		MessageID: uuid.New(),
		Timestamp: time.Now().UTC(),
		Caller: CallerIdentity{
			URN:       "urn:agent:caller.example.com/org:caller-agent@v1.0.0#sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			PublicKey: "ed25519:abc",
		},
		Callee: CalleeIdentity{
			URN: validURNString,
		},
		Payload: RequestPayload{
			Inputs:  map[string]interface{}{"k": "v"},
			Context: InvocationContext{RequestID: "r1"},
		},
	}

	data, err := json.Marshal(req)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	assert.Contains(t, raw, "messageId")
	assert.Contains(t, raw, "timestamp")
	assert.Contains(t, raw, "caller")
	assert.Contains(t, raw, "callee")
	assert.Contains(t, raw, "payload")
}

func TestReceiptMarshalJSON(t *testing.T) {
	msgID := uuid.New()
	ts := time.Date(2026, 3, 25, 14, 30, 0, 0, time.UTC)

	calleeURN := mustParseURN(validURNString)
	callerURN := mustParseURN(
		"urn:agent:caller.example.com/org:caller-agent@v1.0.0#sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)

	receipt := Receipt{
		MessageID: msgID,
		Timestamp: ts,
		Callee:    calleeURN,
		Caller:    callerURN,
		Execution: ExecutionResult{
			Status:     "success",
			DurationMs: 42,
			Outputs: map[string]interface{}{
				"result": "ok",
				"count":  float64(7),
			},
		},
		Tracing: &TracingInfo{
			TraceID: "trace-abc",
			SpanID:  "span-xyz",
		},
	}

	data, err := json.Marshal(&receipt)
	require.NoError(t, err, "Marshal should not error")

	var got Receipt
	err = json.Unmarshal(data, &got)
	require.NoError(t, err, "Unmarshal should not error")

	assert.Equal(t, receipt.MessageID, got.MessageID)
	assert.True(t, receipt.Timestamp.Equal(got.Timestamp))

	// URN round-trip: compare via String() since ContentHash bytes may differ.
	assert.Equal(t, calleeURN.String(), got.Callee.String())
	assert.Equal(t, callerURN.String(), got.Caller.String())

	assert.Equal(t, receipt.Execution.Status, got.Execution.Status)
	assert.Equal(t, receipt.Execution.DurationMs, got.Execution.DurationMs)
	assert.Equal(t, receipt.Execution.Outputs["result"], got.Execution.Outputs["result"])
	assert.Equal(t, receipt.Execution.Outputs["count"], got.Execution.Outputs["count"])

	require.NotNil(t, got.Tracing)
	assert.Equal(t, "trace-abc", got.Tracing.TraceID)
	assert.Equal(t, "span-xyz", got.Tracing.SpanID)
}

func TestReceiptTracingOmittedWhenNil(t *testing.T) {
	calleeURN := mustParseURN(validURNString)
	callerURN := mustParseURN(
		"urn:agent:caller.example.com/org:caller-agent@v1.0.0#sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)

	receipt := Receipt{
		MessageID: uuid.New(),
		Timestamp: time.Now().UTC(),
		Callee:    calleeURN,
		Caller:    callerURN,
		Execution: ExecutionResult{Status: "success"},
		Tracing:   nil,
	}

	data, err := json.Marshal(&receipt)
	require.NoError(t, err)

	// "tracing" key should be absent when nil (omitempty).
	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))
	assert.NotContains(t, raw, "tracing")
}

func TestReceiptURNSerializedAsString(t *testing.T) {
	calleeURN := mustParseURN(validURNString)
	callerURN := mustParseURN(
		"urn:agent:caller.example.com/org:caller-agent@v1.0.0#sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
	)

	receipt := Receipt{
		MessageID: uuid.New(),
		Timestamp: time.Now().UTC(),
		Callee:    calleeURN,
		Caller:    callerURN,
		Execution: ExecutionResult{Status: "success"},
	}

	data, err := json.Marshal(&receipt)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	// Both callee and caller must be JSON strings (not objects).
	calleeVal, ok := raw["callee"]
	require.True(t, ok, "callee key must be present")
	_, isString := calleeVal.(string)
	assert.True(t, isString, "callee should serialize as a JSON string")

	callerVal, ok := raw["caller"]
	require.True(t, ok, "caller key must be present")
	_, isString = callerVal.(string)
	assert.True(t, isString, "caller should serialize as a JSON string")

	assert.Equal(t, calleeURN.String(), calleeVal.(string))
	assert.Equal(t, callerURN.String(), callerVal.(string))
}

func TestCapabilityMarshalJSON(t *testing.T) {
	capability := Capability{
		URN:         validURNString,
		Title:       "Smart Summarizer",
		Description: "Summarizes text using LLMs",
		Version:     "v2.0.0",
		Capabilities: []Action{
			{
				ActionID:    "summarize",
				Title:       "Summarize Text",
				Description: "Produces a summary of the provided text",
				InputSchema: JSONSchema{
					Schema: "http://json-schema.org/draft-07/schema#",
					Type:   "object",
					Properties: map[string]Property{
						"text":      {Type: "string", Description: "Text to summarize"},
						"max_words": {Type: "integer", Description: "Maximum summary word count"},
					},
					Required: []string{"text"},
				},
				OutputSchema: JSONSchema{
					Type: "object",
					Properties: map[string]Property{
						"summary": {Type: "string"},
					},
				},
			},
		},
		AuthMethods:    []string{"bearer"},
		RequiredScopes: []string{"summarize:run"},
		RateLimit: RateLimit{
			RequestsPerMinute: 60,
			Burst:             10,
		},
		Endpoint:      "https://agents.example.com/summarizer",
		TrustLevel:    "verified",
		Contact:       "ops@example.com",
		Documentation: "https://docs.example.com/summarizer",
	}

	data, err := json.Marshal(capability)
	require.NoError(t, err)

	var got Capability
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, capability.URN, got.URN)
	assert.Equal(t, capability.Title, got.Title)
	assert.Equal(t, capability.Description, got.Description)
	assert.Equal(t, capability.Version, got.Version)
	assert.Equal(t, capability.TrustLevel, got.TrustLevel)
	assert.Equal(t, capability.Endpoint, got.Endpoint)
	assert.Equal(t, capability.Contact, got.Contact)
	assert.Equal(t, capability.Documentation, got.Documentation)
	assert.Equal(t, capability.AuthMethods, got.AuthMethods)
	assert.Equal(t, capability.RequiredScopes, got.RequiredScopes)
	assert.Equal(t, capability.RateLimit.RequestsPerMinute, got.RateLimit.RequestsPerMinute)
	assert.Equal(t, capability.RateLimit.Burst, got.RateLimit.Burst)

	require.Len(t, got.Capabilities, 1)
	action := got.Capabilities[0]
	assert.Equal(t, "summarize", action.ActionID)
	assert.Equal(t, "Summarize Text", action.Title)
	assert.Equal(t, "object", action.InputSchema.Type)
	assert.Contains(t, action.InputSchema.Properties, "text")
	assert.Equal(t, []string{"text"}, action.InputSchema.Required)
	assert.Contains(t, action.OutputSchema.Properties, "summary")
}

func TestExecutionResultWithError(t *testing.T) {
	result := ExecutionResult{
		Status:     "error",
		DurationMs: 5,
		Error: &ExecutionError{
			Code:    "SCHEMA_MISMATCH",
			Message: "required field 'text' is missing",
			Details: map[string]interface{}{
				"field":    "text",
				"expected": "string",
			},
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var got ExecutionResult
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, "error", got.Status)
	assert.Equal(t, 5, got.DurationMs)
	require.NotNil(t, got.Error)
	assert.Equal(t, "SCHEMA_MISMATCH", got.Error.Code)
	assert.Equal(t, "required field 'text' is missing", got.Error.Message)
	assert.Equal(t, "text", got.Error.Details["field"])
	assert.Equal(t, "string", got.Error.Details["expected"])
}

func TestExecutionResultErrorOmittedWhenNil(t *testing.T) {
	result := ExecutionResult{
		Status:     "success",
		DurationMs: 12,
		Outputs:    map[string]interface{}{"result": "done"},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var raw map[string]interface{}
	require.NoError(t, json.Unmarshal(data, &raw))

	// "error" should be absent when nil (omitempty).
	assert.NotContains(t, raw, "error")
	assert.Contains(t, raw, "outputs")
}

func TestJSONSchemaNestedProperties(t *testing.T) {
	schema := JSONSchema{
		Schema: "http://json-schema.org/draft-07/schema#",
		Type:   "object",
		Properties: map[string]Property{
			"name": {
				Type:        "string",
				Description: "The name field",
			},
			"tags": {
				Type: "array",
				Items: &JSONSchema{
					Type: "string",
				},
			},
			"status": {
				Type: "string",
				Enum: []interface{}{"active", "inactive", "pending"},
			},
		},
		Required: []string{"name"},
		Definitions: map[string]JSONSchema{
			"TagList": {
				Type:  "array",
				Items: &JSONSchema{Type: "string"},
			},
		},
		Items: &JSONSchema{
			Type: "string",
		},
		Description: "Top-level object schema",
		Format:      "uri",
	}

	data, err := json.Marshal(schema)
	require.NoError(t, err)

	var got JSONSchema
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, schema.Schema, got.Schema)
	assert.Equal(t, schema.Type, got.Type)
	assert.Equal(t, schema.Description, got.Description)
	assert.Equal(t, schema.Format, got.Format)
	assert.Equal(t, schema.Required, got.Required)

	// Properties round-trip.
	require.Contains(t, got.Properties, "name")
	assert.Equal(t, "string", got.Properties["name"].Type)
	assert.Equal(t, "The name field", got.Properties["name"].Description)

	require.Contains(t, got.Properties, "tags")
	require.NotNil(t, got.Properties["tags"].Items)
	assert.Equal(t, "string", got.Properties["tags"].Items.Type)

	require.Contains(t, got.Properties, "status")
	assert.Len(t, got.Properties["status"].Enum, 3)

	// Definitions round-trip.
	require.Contains(t, got.Definitions, "TagList")
	assert.Equal(t, "array", got.Definitions["TagList"].Type)
	require.NotNil(t, got.Definitions["TagList"].Items)
	assert.Equal(t, "string", got.Definitions["TagList"].Items.Type)

	// Items round-trip.
	require.NotNil(t, got.Items)
	assert.Equal(t, "string", got.Items.Type)
}

func TestActionMarshalJSON(t *testing.T) {
	action := Action{
		ActionID:    "classify",
		Title:       "Classify Input",
		Description: "Assigns a category to the input",
		InputSchema: JSONSchema{
			Type:     "object",
			Required: []string{"text"},
			Properties: map[string]Property{
				"text": {Type: "string"},
			},
		},
		OutputSchema: JSONSchema{
			Type: "object",
			Properties: map[string]Property{
				"category": {Type: "string"},
				"score":    {Type: "number"},
			},
		},
	}

	data, err := json.Marshal(action)
	require.NoError(t, err)

	var got Action
	err = json.Unmarshal(data, &got)
	require.NoError(t, err)

	assert.Equal(t, action.ActionID, got.ActionID)
	assert.Equal(t, action.Title, got.Title)
	assert.Equal(t, action.Description, got.Description)
	assert.Equal(t, "object", got.InputSchema.Type)
	assert.Equal(t, []string{"text"}, got.InputSchema.Required)
	assert.Contains(t, got.OutputSchema.Properties, "category")
	assert.Contains(t, got.OutputSchema.Properties, "score")
}
