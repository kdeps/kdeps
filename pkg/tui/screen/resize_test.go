package screen

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v3/pkg/tui/feed"
	"github.com/kdeps/kdeps/v3/pkg/tui/layout"
)

// dragBorder presses on the main/feed border and drags it to x.
func dragBorder(t *testing.T, m Model, x int) Model {
	t.Helper()
	frame := m.frame()
	y := frame.Feed.Y + 2
	m, _ = typed(t, m, tea.MouseClickMsg{X: frame.Feed.X, Y: y, Button: tea.MouseLeft})
	m, _ = typed(t, m, tea.MouseMotionMsg{X: x, Y: y, Button: tea.MouseLeft})
	m, _ = typed(t, m, tea.MouseReleaseMsg{X: x, Y: y, Button: tea.MouseLeft})
	return m
}

func TestResize_DraggingTheBorderWidensTheFeed(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	before := m.frame().Feed.W
	m = dragBorder(t, m, 40)
	frame := m.frame()
	assert.Equal(t, 60, frame.Feed.W, "the feed's left edge follows the pointer")
	assert.Greater(t, frame.Feed.W, before)
	assert.Equal(t, 100, frame.Main.W+frame.Feed.W, "the panes still fill the width")
	assert.False(t, m.resizing, "release ends the drag")

	lines := strings.Split(ansi.Strip(m.render()), "\n")
	assert.Len(t, lines, 30, "the resized frame still fills the terminal")
}

func TestResize_KeepsBothPanesUsable(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m = dragBorder(t, m, 0)
	assert.Equal(t, 20, m.frame().Main.W, "the main pane keeps its minimum")
	m = dragBorder(t, m, 99)
	assert.Equal(t, 30, m.frame().Feed.W, "the feed keeps its minimum")
}

func TestResize_ClickInsideAPaneDoesNotResize(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	frame := m.frame()
	m, _ = typed(t, m, tea.MouseClickMsg{X: frame.Main.X + 2, Y: frame.Main.Y + 1, Button: tea.MouseLeft})
	m, _ = typed(t, m, tea.MouseMotionMsg{X: 10, Y: frame.Main.Y + 1, Button: tea.MouseLeft})
	assert.Equal(t, frame.Feed.W, m.frame().Feed.W)
	assert.False(t, m.focusFeed, "the click still focuses the main pane")
}

func TestResize_WidthSurvivesClosingTheFeed(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m = dragBorder(t, m, 50)
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	assert.Equal(t, 0, m.frame().Feed.W)
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	assert.Equal(t, 50, m.frame().Feed.W, "reopening keeps the dragged width")
}

func TestFeedWidth_Clamps(t *testing.T) {
	assert.Equal(t, 30, layout.FeedWidth(100, 5))
	assert.Equal(t, 80, layout.FeedWidth(100, 95))
	assert.Equal(t, 45, layout.FeedWidth(100, 45))
	assert.Equal(t, 33, layout.ComputeWith(100, 30, true, 0).Feed.W, "0 keeps the default share")
}
