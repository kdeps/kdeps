package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSchemaEnumMap_ContainsExpectedKeys(t *testing.T) {
	t.Parallel()
	m := schemaEnumMap()
	assert.Contains(t, m, "run.chat.backend")
	assert.Contains(t, m, "run.httpClient.method")
	assert.Contains(t, m, "apiVersion")
	assert.Contains(t, m, "kind")
	assert.NotEmpty(t, m["run.chat.backend"])
}

func TestLookupEnumByPath_ExactMatch(t *testing.T) {
	t.Parallel()
	m := map[string][]interface{}{
		"run.chat.backend": {"ollama", "openai"},
	}
	vals := lookupEnumByPath("run.chat.backend", m, func(s string) string { return s })
	assert.Equal(t, []interface{}{"ollama", "openai"}, vals)
}

func TestLookupEnumByPath_NoMatch(t *testing.T) {
	t.Parallel()
	m := map[string][]interface{}{
		"run.chat.backend": {"ollama"},
	}
	vals := lookupEnumByPath("unknown.field", m, func(s string) string { return s })
	assert.Nil(t, vals)
}

func TestLookupResourceSchemaEnums_Backend(t *testing.T) {
	t.Parallel()
	m := schemaEnumMap()
	vals := lookupResourceSchemaEnums("backend", m)
	assert.NotEmpty(t, vals)
}

func TestLookupResourceSchemaEnums_Unknown(t *testing.T) {
	t.Parallel()
	m := schemaEnumMap()
	vals := lookupResourceSchemaEnums("unknown", m)
	assert.Nil(t, vals)
}

func TestLookupWorkflowSchemaEnums_Methods(t *testing.T) {
	t.Parallel()
	m := schemaEnumMap()
	vals := lookupWorkflowSchemaEnums(methodsField, m)
	assert.NotEmpty(t, vals)
}

func TestLookupWorkflowSchemaEnums_Other(t *testing.T) {
	t.Parallel()
	m := schemaEnumMap()
	vals := lookupWorkflowSchemaEnums("notmethods", m)
	assert.Nil(t, vals)
}

func TestIsEnumField(t *testing.T) {
	t.Parallel()
	sv, err := NewSchemaValidatorForTesting()
	require.NoError(t, err)
	assert.True(t, sv.IsEnumField("run.chat.backend", "resource"))
	assert.False(t, sv.IsEnumField("run.chat.backend", "unknown"))
	assert.False(t, sv.IsEnumField("nonexistent.field", "resource"))
}
