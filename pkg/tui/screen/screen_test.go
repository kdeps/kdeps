package screen

import (
	"strconv"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

func resized(t *testing.T, m Model) Model {
	t.Helper()
	next, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	return next.(Model)
}

func TestView_FillsTheTerminal(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m.Status = "status"
	lines := strings.Split(m.render(), "\n")
	require.Len(t, lines, 30)
	for _, line := range lines {
		assert.Equal(t, 100, lipgloss.Width(line))
	}
}

func TestToggleFeed_GivesMainTheWholeWidth(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	assert.True(t, m.FeedOpen())

	next, _ := m.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	closed := next.(Model)
	assert.False(t, closed.FeedOpen())

	next, _ = closed.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	assert.True(t, next.(Model).FeedOpen())
}

func TestView_FeedShowsItsEntries(t *testing.T) {
	var f feed.Feed
	f.Add(feed.Entry{Kind: feed.KindResource, Title: "fetch.yaml"})
	m := resized(t, New(&f))
	assert.Contains(t, m.render(), "fetch.yaml")

	next, _ := m.Update(tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl})
	assert.NotContains(t, next.(Model).render(), "fetch.yaml", "a closed feed is not drawn")
}

func TestView_BeforeResizeIsEmpty(t *testing.T) {
	assert.Equal(t, "", New(nil).render())
}

func typed(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func TestPrompt_TypingBackspaceAndSubmit(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'h', Text: "h"})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'i', Text: "i"})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'e', Text: "e"})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'y', Text: "y"})
	assert.Equal(t, "hey", m.Prompt())

	var got []string
	m.OnSubmit = func(line string) { got = append(got, line) }
	m, cmd := typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter on a non-empty prompt submits it")
	cmd()
	assert.Equal(t, []string{"hey"}, got)
	assert.Equal(t, "", m.Prompt(), "the prompt clears after submit")
}

func TestPrompt_EmptyEnterDoesNothing(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	_, cmd := typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestOutput_LinesAppearInTheMainPane(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: "first line\nsecond line"})
	assert.Contains(t, m.render(), "first line")
	assert.Contains(t, m.render(), "second line")
}

func TestCtrlC_Quits(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	_, cmd := typed(t, m, tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	require.NotNil(t, cmd)
	assert.Equal(t, tea.Quit(), cmd())
}

func TestStatusText_ReplacesTheStatusBox(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, StatusTextMsg{Text: "[turn 9k]"})
	assert.Contains(t, m.render(), "[turn 9k]")
}

func manyLines(n int) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = "row " + strconv.Itoa(i)
	}
	return strings.Join(lines, "\n")
}

func TestScroll_UpShowsOlderRowsAndEndReturnsToTheNewest(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	assert.Contains(t, m.render(), "row 99", "the tail is shown first")
	assert.NotContains(t, m.render(), "row 50")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	assert.Contains(t, m.render(), "row 75", "a page up scrolls back into history")
	assert.NotContains(t, m.render(), "row 99")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnd})
	assert.Contains(t, m.render(), "row 99", "End returns to the newest rows")
}

func TestScroll_ClampsAtTheTop(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(5)})
	for range 50 {
		m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	}
	assert.Contains(t, m.render(), "row 0", "scrolling past the top shows the first rows")
}

func TestScroll_TabMovesFocusToTheFeed(t *testing.T) {
	var f feed.Feed
	for i := range 100 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	m := resized(t, New(&f))
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	assert.NotContains(t, m.render(), "step 99", "the feed scrolled, not the main pane")
}

func TestScroll_WheelScrollsTheBoxUnderThePointer(t *testing.T) {
	var f feed.Feed
	for i := range 100 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	m := resized(t, New(&f))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})

	feedX := m.width - m.width/3 + 2
	m, _ = typed(t, m, tea.MouseWheelMsg{X: feedX, Y: 3, Button: tea.MouseWheelUp})
	assert.NotContains(t, m.render(), "step 99", "the wheel over the feed scrolls the feed")
	assert.Contains(t, m.render(), "row 99", "the main pane did not move")
}

// TestFrameCache_ChangesWhenTheScreenChanges checks a frame is reused only
// while nothing it is drawn from has changed.
func TestFrameCache_ChangesWhenTheScreenChanges(t *testing.T) {
	var f feed.Feed
	m := resized(t, New(&f))
	first := m.render()
	assert.Equal(t, first, m.render(), "an unchanged screen reuses its frame")

	f.Add(feed.Entry{Kind: feed.KindAction, Title: "new step"})
	assert.Contains(t, m.render(), "new step", "a feed change redraws the frame")

	m, _ = typed(t, m, OutputMsg{Text: "fresh output"})
	assert.Contains(t, m.render(), "fresh output", "new output redraws the frame")

	m = typeText(t, m, "x")
	assert.Contains(t, m.render(), "> x|", "a prompt edit redraws the frame")
}

// TestFrameCache_OverlayIsNeverCached checks an open overlay is drawn fresh
// every time, since it changes without any cached key moving.
func TestFrameCache_OverlayIsNeverCached(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m = m.WithOverlay(fakeOverlay{text: "ONE"}, nil)
	assert.Contains(t, m.render(), "ONE")
	m = m.WithOverlay(fakeOverlay{text: "TWO"}, nil)
	assert.Contains(t, m.render(), "TWO", "a new overlay is drawn, not the cached frame")
}

// TestFrame_FillsTheTerminalExactly checks the drawn frame is the terminal's
// size at a few sizes: one row per terminal row, each row as wide as the terminal.
func TestFrame_FillsTheTerminalExactly(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {120, 40}, {40, 12}} {
		m := New(&feed.Feed{})
		m, _ = typed(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		lines := strings.Split(m.render(), "\n")
		assert.Len(t, lines, size[1], "rows at %dx%d", size[0], size[1])
		for i, line := range lines {
			assert.Equal(t, size[0], ansi.StringWidth(line), "row %d at %dx%d", i, size[0], size[1])
		}
	}
}

// TestStatusRow_CountersThenKeyHelp checks the status row shows the counters
// first, then the key help.
func TestStatusRow_CountersThenKeyHelp(t *testing.T) {
	m := New(&feed.Feed{})
	m.Status = "[turn 5.6k]"
	row := ansi.Strip(m.statusText())
	assert.True(t, strings.HasPrefix(row, "[turn 5.6k]   ? help"), "counters first: %q", row)
}

// TestSidebar_HasAnActivityTitle checks the side panel opens with its title.
func TestSidebar_HasAnActivityTitle(t *testing.T) {
	var f feed.Feed
	f.Add(feed.Entry{Kind: feed.KindResource, Title: "fetch.yaml"})
	m := resized(t, New(&f))
	assert.Contains(t, ansi.Strip(m.render()), "│ activity", "the side panel opens with its title")
}

// TestHeader_ShowsTheProductAndTheTitle checks the header row names the product
// and carries the title, with nothing else in it.
func TestHeader_ShowsTheProductAndTheTitle(t *testing.T) {
	m := New(&feed.Feed{})
	m.Title = "myproject"
	header := ansi.Strip(m.headerText())
	assert.Equal(t, " kdeps ╱╱╱ myproject", header)
}
