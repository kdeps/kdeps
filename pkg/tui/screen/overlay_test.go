package screen

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
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
	assert.Contains(t, m.render(), "PICK ONE", "the overlay is drawn in the main area")
	assert.NotContains(t, m.render(), "main text")

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
	assert.True(t, strings.Contains(m.render(), "> "), "the prompt is back")
}
