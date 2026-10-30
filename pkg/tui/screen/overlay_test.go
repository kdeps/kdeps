package screen

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	if k, ok := msg.(tea.KeyMsg); ok {
		f.last = k.String()
		if k.Type == tea.KeyEnter {
			f.done = true
		}
	}
	return f, nil
}

func (f fakeOverlay) View() string { return f.text }
func (f fakeOverlay) Done() bool   { return f.done }

func TestOverlay_TakesTheKeysAndTheMainArea(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: "main text"})
	m = m.WithOverlay(fakeOverlay{text: "PICK ONE"}, nil)

	assert.True(t, m.HasOverlay())
	assert.Contains(t, m.View(), "PICK ONE", "the overlay is drawn in the main area")
	assert.NotContains(t, m.View(), "main text")

	m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	assert.Equal(t, "", m.Prompt(), "keys go to the overlay, not the prompt")
}

func TestOverlay_DoneClosesItAndHandsBackTheResult(t *testing.T) {
	var got Overlay
	m := resized(t, New(&feed.Feed{}))
	m = m.WithOverlay(fakeOverlay{text: "PICK"}, func(o Overlay) { got = o })

	m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	require.False(t, m.HasOverlay(), "a done overlay closes")
	require.NotNil(t, got, "the caller gets the finished overlay")
	assert.True(t, got.Done())
	assert.Equal(t, "enter", got.(fakeOverlay).last)
	assert.True(t, strings.Contains(m.View(), "> "), "the prompt is back")
}
