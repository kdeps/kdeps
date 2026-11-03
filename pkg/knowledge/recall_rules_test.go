package knowledge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The recall rules are the "recall" item of the "knowledge" asset set.
func TestRecallRulesLoadFromTheAssetSet(t *testing.T) {
	assert.Equal(t, 3, recallRules().MinWordLength)
	assert.Equal(t, 3, recallRules().MinStemLength)
	assert.True(t, recallRules().stop["the"])
}

func TestTaskWords_ComeFromTheRecallRules(t *testing.T) {
	assert.Contains(t, recallRules().TaskWords, "please")
	assert.True(t, recallRules().task["fix"])
	assert.Equal(t, []string{"stripe"}, topicWords("please fix the stripe"), "task words and stopwords are left out")
}

func TestParsePolicy_RejectsMissingRecallCounts(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "assets", knowledgeSet, policyItem+".yaml"))
	require.NoError(t, err)
	p, err := parsePolicy(data)
	require.NoError(t, err, "the shipped policy is complete")
	assert.Positive(t, p.BytesPerToken)

	_, err = parsePolicy([]byte("version: 1.0.0\nmax_code_files: 20000\n"))
	require.Error(t, err, "a policy from before the recall counts is rejected")
	assert.Contains(t, err.Error(), "recall_max_hops")
}
