package llm

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tmc/langchaingo/llms"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestResetSessionTokens_ReplacesCumulativeAndClearsLive(t *testing.T) {
	origIn, origOut := atomic.LoadInt64(&TokenInputs), atomic.LoadInt64(&TokenOutputs)
	origLiveIn, origLiveOut := atomic.LoadInt64(&liveInputs), atomic.LoadInt64(&liveOutputs)
	t.Cleanup(func() {
		atomic.StoreInt64(&TokenInputs, origIn)
		atomic.StoreInt64(&TokenOutputs, origOut)
		atomic.StoreInt64(&liveInputs, origLiveIn)
		atomic.StoreInt64(&liveOutputs, origLiveOut)
	})
	atomic.StoreInt64(&TokenInputs, 5000)
	atomic.StoreInt64(&TokenOutputs, 800)
	atomic.StoreInt64(&liveInputs, 12)
	atomic.StoreInt64(&liveOutputs, 3)

	ResetSessionTokens(40, 9)

	assert.Equal(t, int64(40), SessionInputTokens())
	assert.Equal(t, int64(9), SessionOutputTokens())
	assert.Equal(t, int64(0), atomic.LoadInt64(&liveInputs))
	assert.Equal(t, int64(0), atomic.LoadInt64(&liveOutputs))
}

func TestSessionTokens_IncludeLiveThenReplaceWithUsage(t *testing.T) {
	origIn, origOut := atomic.LoadInt64(&TokenInputs), atomic.LoadInt64(&TokenOutputs)
	origLiveIn, origLiveOut := atomic.LoadInt64(&liveInputs), atomic.LoadInt64(&liveOutputs)
	t.Cleanup(func() {
		atomic.StoreInt64(&TokenInputs, origIn)
		atomic.StoreInt64(&TokenOutputs, origOut)
		atomic.StoreInt64(&liveInputs, origLiveIn)
		atomic.StoreInt64(&liveOutputs, origLiveOut)
	})
	atomic.StoreInt64(&TokenInputs, 0)
	atomic.StoreInt64(&TokenOutputs, 0)
	atomic.StoreInt64(&liveInputs, 0)
	atomic.StoreInt64(&liveOutputs, 0)

	resp, err := generateAccounted(&domain.ChatConfig{}, nil, func() (*llms.ContentResponse, error) {
		addLive(&liveOutputs, 7)
		assert.Equal(t, int64(7), SessionOutputTokens(), "generated must move before the call returns")
		return &llms.ContentResponse{Choices: []*llms.ContentChoice{{
			Content:        "hello",
			GenerationInfo: map[string]any{"PromptTokens": 40, "CompletionTokens": 9},
		}}}, nil
	})
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, int64(40), SessionInputTokens())
	assert.Equal(t, int64(9), SessionOutputTokens())
	assert.Equal(t, int64(0), atomic.LoadInt64(&liveInputs))
	assert.Equal(t, int64(0), atomic.LoadInt64(&liveOutputs))
}
