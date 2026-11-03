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

func TestTaskWords_ComeFromTheRecallRules(t *testing.T) {
	assert.Contains(t, recallRules.TaskWords, "please")
	assert.True(t, taskWords["fix"])
	assert.Equal(t, []string{"stripe"}, topicWords("please fix the stripe"), "task words and stopwords are left out")
}
