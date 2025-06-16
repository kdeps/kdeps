package resolver

import (
	"encoding/json"
	"testing"

	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/assert"
)

func TestConstructToolCallsFromJSONAndDeduplication(t *testing.T) {
	logger := logging.NewTestLogger()

	// case 1: empty string returns nil
	result := constructToolCallsFromJSON("", logger)
	assert.Nil(t, result)

	// case 2: invalid json returns nil
	result = constructToolCallsFromJSON("{bad json}", logger)
	assert.Nil(t, result)

	// case 3: single valid object
	single := `{"name":"echo","arguments":{"msg":"hi"}}`
	result = constructToolCallsFromJSON(single, logger)
	assert.Len(t, result, 1)
	assert.Equal(t, "echo", result[0].FunctionCall.Name)

	// case 4: array with duplicate items (should deduplicate)
	arr := `[
        {"name":"echo","arguments":{"msg":"hi"}},
        {"name":"echo","arguments":{"msg":"hi"}},
        {"name":"sum","arguments":{"a":1,"b":2}}
    ]`
	result = constructToolCallsFromJSON(arr, logger)
	// before dedup, duplicates exist; after dedup should be 2 unique
	dedup := deduplicateToolCalls(result, logger)
	assert.Len(t, dedup, 2)

	// ensure deduplication preserved original ordering (echo then sum)
	names := []string{dedup[0].FunctionCall.Name, dedup[1].FunctionCall.Name}
	assert.Equal(t, []string{"echo", "sum"}, names)

	// additional sanity: encode/decode arguments roundtrip
	var args map[string]interface{}
	_ = json.Unmarshal([]byte(dedup[1].FunctionCall.Arguments), &args)
	assert.Equal(t, float64(1), args["a"]) // json numbers unmarshal to float64
}
