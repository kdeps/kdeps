package codegraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRulesYAMLConfiguresKartographer(t *testing.T) {
	require.NoError(t, Configure())
	assert.NotEmpty(t, rulesYAML)
}
