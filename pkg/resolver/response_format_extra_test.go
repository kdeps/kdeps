package resolver

import (
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/stretchr/testify/require"
)

func TestFormatMapAndValueHelpers(t *testing.T) {
	simpleMap := map[interface{}]interface{}{uuid.New().String(): "value"}
	formatted := formatMap(simpleMap)
	require.Contains(t, formatted, "new Mapping {")
	require.Contains(t, formatted, "value")

	// Value wrappers
	require.Equal(t, "null", formatValue(nil))

	// Map[string]interface{}
	m := map[string]interface{}{"key": "val"}
	formattedMap := formatValue(m)
	require.Contains(t, formattedMap, "\"key\"")
	require.Contains(t, formattedMap, "val")

	// Struct pointer should deref
	type sample struct{ A string }
	s := &sample{A: "x"}
	formattedStruct := formatValue(s)
	require.Contains(t, formattedStruct, "A")
	require.Contains(t, formattedStruct, "x")

	// structToMap should reflect fields
	stMap := structToMap(sample{A: "y"})
	require.Equal(t, "y", stMap["A"])
}

func TestDecodeErrorMessageExtra(t *testing.T) {
	logger := logging.NewTestLogger()
	src := "hello"
	encoded := base64.StdEncoding.EncodeToString([]byte(src))
	// Should decode base64
	out := decodeErrorMessage(encoded, logger)
	require.Equal(t, src, out)

	// Non-base64 should return original
	require.Equal(t, src, decodeErrorMessage(src, logger))
}
