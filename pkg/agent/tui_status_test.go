package agent

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/tui/screen"
)

func TestTUIEnabled_DefaultFollowsModeAndTerminal(t *testing.T) {
	origIn, origOut, origMode := stdinIsTerminal, stdoutIsTerminal, agentLoopMode.Load()
	t.Cleanup(func() {
		stdinIsTerminal, stdoutIsTerminal = origIn, origOut
		agentLoopMode.Store(origMode)
	})

	cases := []struct {
		name      string
		env       string
		agentLoop bool
		in, out   bool
		want      bool
	}{
		{"agent loop in a terminal", "", true, true, true, true},
		{"workflow or agency run in a terminal", "", false, true, true, false},
		{"agent loop with piped stdin", "", true, false, true, false},
		{"agent loop with redirected stdout", "", true, true, false, false},
		{"forced off", "0", true, true, true, false},
		{"forced on outside the agent loop", "1", false, false, false, true},
		{"unknown value uses the default", "yes", true, true, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(tuiEnv, c.env)
			agentLoopMode.Store(c.agentLoop)
			stdinIsTerminal = func() bool { return c.in }
			stdoutIsTerminal = func() bool { return c.out }
			assert.Equal(t, c.want, tuiEnabled())
		})
	}
}

func TestDrawLiveStatus_WindowOwnsTheRowsWhenRunning(t *testing.T) {
	var window bytes.Buffer
	stop := startTUIStatus(&window)
	require.True(t, tuiRunning())

	var rows bytes.Buffer
	drawLiveStatus(&rows, "⠋")
	assert.Empty(t, rows.String(), "with the window up, the rows are not drawn to the terminal writer")

	stop()
	assert.False(t, tuiRunning())
	assert.False(t, setTUIStatus("t", "s", ""), "no window after stop")
}

func TestStartTUIStatus_StopIsSafeTwice(t *testing.T) {
	var out bytes.Buffer
	stop := startTUIStatus(&out)
	done := make(chan struct{})
	go func() {
		stop()
		stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("stop hung")
	}
}

// TestFullScreenEnabled_TrueWhileTheViewRuns checks the view stays "on" while
// it runs, although it has redirected stdout into a pipe (a /model picker in
// the view must open as an overlay, not as a second program).
func TestFullScreenEnabled_TrueWhileTheViewRuns(t *testing.T) {
	t.Setenv(tuiEnv, "0")
	assert.False(t, FullScreenEnabled())
	closed := make(chan struct{})
	setScreenOverlaySink(func(screen.OpenOverlayMsg) {}, closed)
	t.Cleanup(func() { setScreenOverlaySink(nil, nil) })
	assert.True(t, FullScreenEnabled(), "a running view is on whatever stdout is now")
}
