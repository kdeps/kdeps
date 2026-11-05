package agent

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTUIEnabled_DefaultFollowsTerminal(t *testing.T) {
	origIn, origOut := stdinIsTerminal, stdoutIsTerminal
	t.Cleanup(func() { stdinIsTerminal, stdoutIsTerminal = origIn, origOut })

	cases := []struct {
		env     string
		in, out bool
		want    bool
	}{
		{"", true, true, true},    // interactive terminal: full-screen by default
		{"", false, true, false},  // piped stdin: line REPL
		{"", true, false, false},  // redirected stdout: line REPL
		{"0", true, true, false},  // forced off
		{"1", false, false, true}, // forced on
		{"yes", true, true, true}, // unknown value: default rule
	}
	for _, c := range cases {
		t.Setenv(tuiEnv, c.env)
		stdinIsTerminal = func() bool { return c.in }
		stdoutIsTerminal = func() bool { return c.out }
		assert.Equal(t, c.want, tuiEnabled(), "KDEPS_TUI=%q stdin=%v stdout=%v", c.env, c.in, c.out)
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
