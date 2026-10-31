package screen

import (
	"bytes"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/teatest/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

// TestTeatest_RunsTheScreenAsAProgram drives the full-screen model as a real
// bubbletea program: it draws the boxes, takes typed input, shows output, and
// quits on Ctrl+C.
func TestTeatest_RunsTheScreenAsAProgram(t *testing.T) {
	var f feed.Feed
	f.Add(feed.Entry{Kind: feed.KindResource, Title: "fetch.yaml"})
	tm := teatest.NewTestModel(t, New(&f), teatest.WithInitialTermSize(100, 30))

	tm.Send(OutputMsg{Text: "hello from the agent"})
	tm.Type("hey")

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("hello from the agent")) && bytes.Contains(b, []byte("> hey"))
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	final, ok := tm.FinalModel(t).(Model)
	require.True(t, ok)
	assert.Equal(t, "hey", final.Prompt())
}

// TestTeatest_FeedToggleChangesTheDrawnFrame checks the frame in a running
// program: the feed box shows its entries, and Ctrl+F removes it.
func TestTeatest_FeedToggleChangesTheDrawnFrame(t *testing.T) {
	var f feed.Feed
	f.Add(feed.Entry{Kind: feed.KindResource, Title: "fetch.yaml"})
	tm := teatest.NewTestModel(t, New(&f), teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("fetch.yaml"))
	}, teatest.WithDuration(5*time.Second))

	tm.Send(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	tm.Send(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	tm.WaitFinished(t, teatest.WithFinalTimeout(5*time.Second))

	final, ok := tm.FinalModel(t).(Model)
	require.True(t, ok)
	assert.False(t, final.FeedOpen(), "ctrl+f closed the feed in the running program")
}
