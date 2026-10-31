package agent

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui"
	"github.com/kdeps/kdeps/v2/pkg/tui/screen"
)

// fakeScreenOverlay is an overlay that is done as soon as it is opened.
type fakeScreenOverlay struct{}

func (f fakeScreenOverlay) Init() tea.Cmd                       { return nil }
func (f fakeScreenOverlay) Update(tea.Msg) (tea.Model, tea.Cmd) { return f, nil }
func (f fakeScreenOverlay) View() tea.View                      { return tea.NewView("") }
func (f fakeScreenOverlay) Done() bool                          { return true }

func TestRunInScreen_NoViewReturnsFalse(t *testing.T) {
	setScreenOverlaySink(nil, nil)
	_, ok := RunInScreen(fakeScreenOverlay{})
	assert.False(t, ok)
}

func TestRunInScreen_ReturnsTheFinishedOverlay(t *testing.T) {
	setScreenOverlaySink(func(msg screen.OpenOverlayMsg) {
		msg.Done(msg.Overlay)
	}, make(chan struct{}))
	t.Cleanup(func() { setScreenOverlaySink(nil, nil) })

	got, ok := RunInScreen(fakeScreenOverlay{})
	require.True(t, ok)
	assert.True(t, got.Done())
}

func TestRunInScreen_ViewEndingUnblocksTheCaller(t *testing.T) {
	closed := make(chan struct{})
	setScreenOverlaySink(func(screen.OpenOverlayMsg) {}, closed)
	t.Cleanup(func() { setScreenOverlaySink(nil, nil) })

	go func() {
		time.Sleep(20 * time.Millisecond)
		close(closed)
	}()
	_, ok := RunInScreen(fakeScreenOverlay{})
	assert.False(t, ok, "a view that ends before the overlay is done releases the caller")
}

func TestSessionPickerItems_EmptyStoreOffersNothing(t *testing.T) {
	store := NewSessionStore(t.TempDir())
	t.Cleanup(func() { _ = store.Close() })
	assert.Nil(t, sessionPickerItems(store))
}

func TestSessionPickerItems_ListsStartFreshThenSavedSessions(t *testing.T) {
	store := NewSessionStore(t.TempDir())
	t.Cleanup(func() { _ = store.Close() })
	s := NewSession(0)
	s.Append("hi", "hello")
	_, err := store.Save(s)
	require.NoError(t, err)

	items := sessionPickerItems(store)
	require.Len(t, items, 2)
	assert.Equal(t, newSessionSentinel, items[0].ID)
}

func TestResumeChosenSession_StartFreshLoadsNothing(t *testing.T) {
	store := NewSessionStore(t.TempDir())
	t.Cleanup(func() { _ = store.Close() })
	s := NewSession(0)
	s.Append("hi", "hello")
	_, err := store.Save(s)
	require.NoError(t, err)

	var r REPL
	picker := tui.NewPickerOverlay("Resume", sessionPickerItems(store), nil)
	next, _ := picker.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	cmd := r.resumeChosenSession(store)(next.(tui.PickerOverlay))
	assert.Nil(t, cmd, "the first row starts a new session and loads nothing")
}
