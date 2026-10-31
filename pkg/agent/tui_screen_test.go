package agent

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/screen"
)

func TestScreenStatusSink_OnlyWhileTheScreenRuns(t *testing.T) {
	assert.False(t, sendScreenStatus("x"), "no sink: the caller draws the rows itself")

	var got []string
	setScreenStatusSink(func(text string) { got = append(got, text) })
	t.Cleanup(func() { setScreenStatusSink(nil) })

	assert.True(t, sendScreenStatus("[turn 1k]  [session 2k]"))
	assert.Equal(t, []string{"[turn 1k]  [session 2k]"}, got)
}

func TestDrawLiveStatus_IdleSendsTheCountersToTheScreen(t *testing.T) {
	resetLiveStatus(t)
	var got []string
	setScreenStatusSink(func(text string) { got = append(got, text) })
	t.Cleanup(func() { setScreenStatusSink(nil) })

	drawLiveStatus(io.Discard, "")
	if assert.Len(t, got, 1, "the counters show before any turn runs") {
		assert.Contains(t, got[0], "session")
	}
}

func TestDrawLiveStatus_LeadsWithTheModel(t *testing.T) {
	resetLiveStatus(t)
	var got []string
	setScreenStatusSink(func(text string) { got = append(got, text) })
	t.Cleanup(func() {
		setScreenStatusSink(nil)
		setScreenModel("")
	})

	setScreenModel("llama-3.2-1b")
	drawLiveStatus(io.Discard, "")
	if assert.Len(t, got, 1) {
		assert.True(t, strings.HasPrefix(got[0], "llama-3.2-1b  "), "the model leads the status: %q", got[0])
	}

	setScreenModel("")
	drawLiveStatus(io.Discard, "")
	if assert.Len(t, got, 2) {
		assert.False(t, strings.Contains(got[1], "llama-3.2-1b"), "a cleared model leaves no label")
	}
}

func TestSetScreenModel_SendsTheNameToTheView(t *testing.T) {
	var got []screen.ModelMsg
	screenModelSend.Lock()
	screenModelSend.send = func(msg screen.ModelMsg) { got = append(got, msg) }
	screenModelSend.Unlock()
	t.Cleanup(func() {
		screenModelSend.Lock()
		screenModelSend.send = nil
		screenModelSend.Unlock()
		setScreenModel("")
	})

	setScreenModel("gpt-x")
	assert.Equal(t, []screen.ModelMsg{{Name: "gpt-x"}}, got)
	assert.Equal(t, "gpt-x", screenModelLabel())
}

func TestScreenPillNames_FromEnv(t *testing.T) {
	t.Setenv("KDEPS_TUI_PILLS", "")
	assert.Empty(t, screenPillNames(), "an empty setting shows no pills")
	t.Setenv("KDEPS_TUI_PILLS", "tool, backend,bogus")
	assert.Equal(t, []string{"backend", "tool"}, screenPillNames(),
		"listed pills in their fixed order, unknown names ignored")
	t.Setenv("KDEPS_TUI_PILLS", "")
	require.NoError(t, os.Unsetenv("KDEPS_TUI_PILLS"))
	assert.Equal(t, []string{"backend", "session", "tool"}, screenPillNames(), "unset shows all three")
}

func TestSendScreenPill_OnlyEnabledPillsReachTheView(t *testing.T) {
	var got []screen.PillMsg
	screenPillSend.Lock()
	screenPillSend.send = func(msg screen.PillMsg) { got = append(got, msg) }
	screenPillSend.enabled = map[string]bool{"backend": true}
	screenPillSend.Unlock()
	t.Cleanup(func() {
		screenPillSend.Lock()
		screenPillSend.send = nil
		screenPillSend.enabled = nil
		screenPillSend.Unlock()
	})

	sendScreenPill("backend", "openrouter")
	sendScreenPill("tool", "bash_exec")
	assert.Equal(t, []screen.PillMsg{{Name: "backend", Value: "openrouter"}}, got)
}

func TestSessionPillValue_NewThenShortID(t *testing.T) {
	assert.Equal(t, "new", sessionPillValue(""))
	assert.Equal(t, "abcdefgh", sessionPillValue("abcdefghijkl"))
	assert.Equal(t, "abc", sessionPillValue("abc"))
}
