package screen

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/tui/feed"
	"github.com/kdeps/kdeps/v3/pkg/tui/layout"
)

// fakeOverlay closes on Enter and reports the key it saw.
type fakeOverlay struct {
	text string
	done bool
	last string
}

func (f fakeOverlay) Init() tea.Cmd { return nil }

func (f fakeOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		f.last = k.String()
		if k.Code == tea.KeyEnter {
			f.done = true
		}
	}
	return f, nil
}

func (f fakeOverlay) View() tea.View { return tea.NewView(f.text) }
func (f fakeOverlay) Done() bool     { return f.done }

func TestOverlay_TakesTheKeysAndTheMainArea(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: "main text"})
	m = m.WithOverlay(fakeOverlay{text: "PICK ONE"}, nil)

	assert.True(t, m.HasOverlay())
	assert.Contains(t, ansi.Strip(m.render()), "PICK ONE", "the overlay is drawn in the main area")
	assert.NotContains(t, ansi.Strip(m.render()), "main text")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'x', Text: "x"})
	assert.Equal(t, "", m.Prompt(), "keys go to the overlay, not the prompt")
}

func TestOverlay_DoneClosesItAndHandsBackTheResult(t *testing.T) {
	var got Overlay
	m := resized(t, New(&feed.Feed{}))
	m = m.WithOverlay(fakeOverlay{text: "PICK"}, func(o Overlay) tea.Cmd { got = o; return nil })

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.False(t, m.HasOverlay(), "a done overlay closes")
	require.NotNil(t, got, "the caller gets the finished overlay")
	assert.True(t, got.Done())
	assert.Equal(t, "enter", got.(fakeOverlay).last)
	assert.True(t, strings.Contains(ansi.Strip(m.render()), "> "), "the prompt is back")
}

// sizeOverlay records the last size it was given.
type sizeOverlay struct{ w, h int }

func (o sizeOverlay) Init() tea.Cmd { return nil }
func (o sizeOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		o.w, o.h = ws.Width, ws.Height
	}
	return o, nil
}
func (o sizeOverlay) View() tea.View { return tea.NewView("") }
func (o sizeOverlay) Done() bool     { return false }

// TestOverlay_IsSizedToTheMainPane checks an overlay is laid out for the main
// pane it is drawn in, on open and on a resize, not for the whole terminal.
func TestOverlay_IsSizedToTheMainPane(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	pane := layout.InnerWidth(m.frame().Main.W)
	m = m.WithOverlay(sizeOverlay{}, nil)
	got := m.overlay.(sizeOverlay)
	assert.Equal(t, pane, got.w, "opened at the pane's inner width")
	assert.Less(t, got.h, 30)

	next, _ := m.Update(tea.WindowSizeMsg{Width: 140, Height: 40})
	m = next.(Model)
	got = m.overlay.(sizeOverlay)
	assert.Equal(t, layout.InnerWidth(m.frame().Main.W), got.w, "a resize passes the new pane width")
	assert.Less(t, got.w, 140)
}
