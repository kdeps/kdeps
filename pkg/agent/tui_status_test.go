package agent

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTUIEnabled_OnlyWithTheFlag(t *testing.T) {
	t.Setenv(tuiEnv, "")
	assert.False(t, tuiEnabled())
	t.Setenv(tuiEnv, "1")
	assert.True(t, tuiEnabled())
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
