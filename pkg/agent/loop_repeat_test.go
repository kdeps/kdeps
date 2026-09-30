package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestTrackRepeat_CountsOnlyIdenticalCallAndResult(t *testing.T) {
	tc := domain.StreamedToolCall{Name: "bash_exec", Arguments: `{"command":"tail log"}`}

	n, sig := trackRepeat(tc, "r1", "", 0)
	assert.Equal(t, 1, n)
	n, sig = trackRepeat(tc, "r1", sig, n)
	assert.Equal(t, 2, n, "same call, same result: repeat")
	n, sig = trackRepeat(tc, "r2", sig, n)
	assert.Equal(t, 1, n, "same call, changed result: progress, counter resets")
	n, _ = trackRepeat(domain.StreamedToolCall{Name: "bash_exec", Arguments: `{"command":"ls"}`}, "r2", sig, n)
	assert.Equal(t, 1, n, "different call resets")
}

func TestIdenticalRepeatLimit_FollowsToolLimit(t *testing.T) {
	resetConvergenceLimitsAfter(t)
	def := effectiveRounds(eventIdenticalCalls)

	_, fileLimit := FileConvergenceCalls()
	assert.Equal(t, fileLimit, identicalRepeatLimit("read_file"), "read_file follows the file limit")
	assert.Equal(t, def, identicalRepeatLimit("some_other_tool"), "uncategorized tools keep the event threshold")

	SetConvergenceLimits(0, -1, 7, 0)
	assert.Equal(t, disabledSentinel, identicalRepeatLimit("bash_exec"), "unlimited bash limit never stops")
	assert.Equal(t, 7, identicalRepeatLimit("read_file"), "custom file limit applies")
}
