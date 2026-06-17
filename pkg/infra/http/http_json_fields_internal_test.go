package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSuccessResponseMap(t *testing.T) {
	t.Parallel()
	m := successResponseMap("payload", map[string]interface{}{"id": "req-1"})
	assert.Equal(t, true, m[jsonFieldSuccess])
	assert.Equal(t, "payload", m[jsonFieldData])
	meta, ok := m[jsonFieldMeta].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "req-1", meta["id"])
}

func TestAPIResultSuccessValue_True(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{jsonFieldSuccess: true}
	assert.True(t, apiResultSuccessValue(m))
}

func TestAPIResultSuccessValue_False(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{jsonFieldSuccess: false}
	assert.False(t, apiResultSuccessValue(m))
}

func TestAPIResultSuccessValue_Missing(t *testing.T) {
	t.Parallel()
	assert.False(t, apiResultSuccessValue(map[string]interface{}{}))
}

func TestAPIResultData(t *testing.T) {
	t.Parallel()
	m := map[string]interface{}{jsonFieldData: "result"}
	assert.Equal(t, "result", apiResultData(m))
}

func TestIsMetaHeadersKey(t *testing.T) {
	t.Parallel()
	assert.True(t, isMetaHeadersKey(metaHeadersKey))
	assert.False(t, isMetaHeadersKey("other"))
}

func TestAnyMapToInterfaceMap(t *testing.T) {
	t.Parallel()
	src := map[string]any{"a": 1, "b": "two"}
	dst := anyMapToInterfaceMap(src)
	assert.Equal(t, 1, dst["a"])
	assert.Equal(t, "two", dst["b"])
}

func TestResponseMetaFields(t *testing.T) {
	t.Parallel()
	m := responseMetaFields("req-abc")
	assert.Equal(t, "req-abc", m[jsonFieldRequestID])
	assert.NotNil(t, m[jsonFieldTimestamp])
}

func TestTypeNameOf(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "string", typeNameOf("hello"))
	assert.Equal(t, "int", typeNameOf(42))
	assert.Equal(t, "map[string]interface {}", typeNameOf(map[string]interface{}{}))
}
