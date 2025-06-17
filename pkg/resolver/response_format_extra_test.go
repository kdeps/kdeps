package resolver

import (
	"encoding/base64"
	"testing"

	"github.com/google/uuid"
	"github.com/kdeps/kdeps/pkg/logging"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
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

// Simple struct for structToMap / formatValue tests
type demo struct {
	FieldA string
	FieldB int
}

func TestFormatValueVariousTypes(t *testing.T) {
	// nil becomes "null"
	assert.Contains(t, formatValue(nil), "null")

	// map[string]interface{}
	m := map[string]interface{}{"k1": "v1"}
	out := formatValue(m)
	assert.Contains(t, out, "[\"k1\"]")
	assert.Contains(t, out, "v1")

	// pointer to struct
	d := &demo{FieldA: "abc", FieldB: 123}
	out2 := formatValue(d)
	assert.Contains(t, out2, "FieldA")
	assert.Contains(t, out2, "abc")
}

func TestValidatePklFileExtension(t *testing.T) {
	fs := afero.NewMemMapFs()
	dr := &DependencyResolver{Fs: fs, ResponsePklFile: "/file.pkl", ResponseTargetFile: "/out.json"}
	assert.NoError(t, dr.validatePklFileExtension())

	dr.ResponsePklFile = "/file.txt"
	assert.Error(t, dr.validatePklFileExtension())
}

func TestEnsureResponseTargetFileNotExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	path := "/out.json"
	_ = afero.WriteFile(fs, path, []byte("x"), 0o644)

	dr := &DependencyResolver{Fs: fs, ResponseTargetFile: path}
	assert.NoError(t, dr.ensureResponseTargetFileNotExists())
	exists, _ := afero.Exists(fs, path)
	assert.False(t, exists)
}
