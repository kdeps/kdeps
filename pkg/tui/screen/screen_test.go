package screen

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

func resized(t *testing.T, m Model, w, h int) Model {
	t.Helper()
	next, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return next.(Model)
}

func TestView_FillsTheTerminal(t *testing.T) {
	m := resized(t, New(&feed.Feed{}), 100, 30)
	m.Main, m.Status, m.Prompt = "main", "status", "> "
	lines := strings.Split(m.View(), "\n")
	require.Len(t, lines, 30)
	for _, line := range lines {
		assert.Equal(t, 100, lipgloss.Width(line))
	}
}

func TestToggleFeed_GivesMainTheWholeWidth(t *testing.T) {
	m := resized(t, New(&feed.Feed{}), 100, 30)
	assert.True(t, m.FeedOpen())

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	closed := next.(Model)
	assert.False(t, closed.FeedOpen())

	next, _ = closed.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	assert.True(t, next.(Model).FeedOpen())
}

func TestView_FeedShowsItsEntries(t *testing.T) {
	var f feed.Feed
	f.Add(feed.Entry{Kind: feed.KindResource, Title: "fetch.yaml"})
	m := resized(t, New(&f), 100, 30)
	assert.Contains(t, m.View(), "fetch.yaml")

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	assert.NotContains(t, next.(Model).View(), "fetch.yaml", "a closed feed is not drawn")
}

func TestView_BeforeResizeIsEmpty(t *testing.T) {
	assert.Equal(t, "", New(nil).View())
}
