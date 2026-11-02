package knowledge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// The recall rules are the "recall" item of the "knowledge" asset set.
func TestRecallRulesLoadFromTheAssetSet(t *testing.T) {
	assert.Equal(t, 3, recallRules.MinWordLength)
	assert.Equal(t, 3, recallRules.MinStemLength)
	assert.True(t, stopWords["the"])
}
