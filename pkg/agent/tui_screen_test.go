package agent

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
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

func TestSendScreenPill_ReachesTheView(t *testing.T) {
	var got []screen.PillMsg
	screenPillSend.Lock()
	screenPillSend.send = func(msg screen.PillMsg) { got = append(got, msg) }
	screenPillSend.Unlock()
	t.Cleanup(func() {
		screenPillSend.Lock()
		screenPillSend.send = nil
		screenPillSend.Unlock()
	})

	sendScreenPill("backend", "openrouter")
	sendScreenPill("tool", "bash_exec")
	assert.Equal(t, []screen.PillMsg{{Name: "backend", Value: "openrouter"}, {Name: "tool", Value: "bash_exec"}}, got)
}

func TestSessionPillValue_NewThenShortID(t *testing.T) {
	assert.Equal(t, "new", sessionPillValue(""))
	assert.Equal(t, "abcdefgh", sessionPillValue("abcdefghijkl"))
	assert.Equal(t, "abc", sessionPillValue("abc"))
}

func TestPromptHistoryLimit_FromTheEvent(t *testing.T) {
	withEvent(t, Event{Name: eventPromptHistoryLimit, Items: 3, Run: actionCap})
	assert.Equal(t, 3, promptHistoryLimit())
}

func TestPromptHistory_SharedWithTheLineEditor(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("USERPROFILE", dir)
	path := historyPath()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte("from the line editor\n\n  \n"), 0o600))

	m := newScreenModel(&feed.Feed{})
	next, _ := m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	got := next.(screen.Model)
	assert.Equal(t, "from the line editor", got.Prompt())

	cleared, _ := got.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	got = cleared.(screen.Model)
	assert.Equal(t, "", got.Prompt(), "ctrl+c clears the recalled line")
	got = typeScreen(t, got, "from the screen")
	_, cmd := got.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd)
	cmd()

	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "from the line editor\n\n  \nfrom the screen\n", string(body))
}

func typeScreen(t *testing.T, m screen.Model, s string) screen.Model {
	t.Helper()
	for _, r := range s {
		next, _ := m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
		m = next.(screen.Model)
	}
	return m
}

func TestPromptHistoryLimit_ADisabledEventKeepsEverything(t *testing.T) {
	withEvent(t, Event{Name: eventPromptHistoryLimit, Items: 3, Run: actionCap, Disabled: true})
	assert.Equal(t, 0, promptHistoryLimit())
}
