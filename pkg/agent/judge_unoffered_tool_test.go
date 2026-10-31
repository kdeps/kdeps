package agent

import (
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// unofferedToolStreamer always calls a tool the judge was not offered.
type unofferedToolStreamer struct{ calls int }

func (s *unofferedToolStreamer) StreamChat(
	_ context.Context, _ *domain.ChatConfig, _ io.Writer,
) (string, []domain.StreamedToolCall, error) {
	s.calls++
	return "", []domain.StreamedToolCall{{Name: "bash_exec", Arguments: "{}"}}, nil
}

func TestRunJudge_EndsWhenItCallsAToolItWasNotOffered(t *testing.T) {
	s := &unofferedToolStreamer{}
	loop := freeModelLoop(t, "m", s)
	loop.config.MaxToolRounds = 50

	verdict := runJudge(t.Context(), loop, JudgeSpec{Name: "critic", MaxRounds: 50}, "input", "output")
	require.True(t, verdict.Approved, "an unsettled judge approves, as before")
	assert.LessOrEqual(t, s.calls, 2, "the judge stops at the first missing tool, not after its rounds")
}
