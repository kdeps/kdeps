package statuswindow

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestView_TwoRowsWithSpinnerOnTheSecond(t *testing.T) {
	m := New()
	next, _ := m.Update(StatusMsg{Turn: "[turn 5.6k]", Session: "[session sent 1k]", Spinner: "⠋"})
	view := next.View()
	assert.Equal(t, "[turn 5.6k]\n[session sent 1k]  ⠋", view)
}

func TestView_NoSpinnerNoTrailer(t *testing.T) {
	next, _ := New().Update(StatusMsg{Turn: "t", Session: "s"})
	assert.Equal(t, "t\ns", next.View())
}

func TestUpdate_LogMsgPrintsAboveAndKeepsRows(t *testing.T) {
	m := New()
	next, _ := m.Update(StatusMsg{Turn: "t", Session: "s"})
	after, cmd := next.Update(LogMsg{Line: "openrouter: refused"})
	require.NotNil(t, cmd, "a log line returns a print command")
	assert.Equal(t, "t\ns", after.View(), "printing a line leaves the rows unchanged")
}

func TestUpdate_UnknownMsgIsIgnored(t *testing.T) {
	m := New()
	next, cmd := m.Update(tea.KeyMsg{})
	assert.Nil(t, cmd)
	assert.Equal(t, m.View(), next.View())
}
