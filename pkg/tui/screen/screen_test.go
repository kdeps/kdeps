package screen

import (
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	lines := strings.Split(m.View(), "\n")
	require.Len(t, lines, 30)
	for _, line := range lines {
		assert.Equal(t, 100, lipgloss.Width(line))
	}
}

func TestToggleFeed_GivesMainTheWholeWidth(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
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
	m := resized(t, New(&f))
	assert.Contains(t, m.View(), "fetch.yaml")

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlF})
	assert.NotContains(t, next.(Model).View(), "fetch.yaml", "a closed feed is not drawn")
}

func TestView_BeforeResizeIsEmpty(t *testing.T) {
	assert.Equal(t, "", New(nil).View())
}

func typed(t *testing.T, m Model, msg tea.Msg) (Model, tea.Cmd) {
	t.Helper()
	next, cmd := m.Update(msg)
	return next.(Model), cmd
}

func TestPrompt_TypingBackspaceAndSubmit(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("hi")})
	m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyBackspace})
	m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("ey")})
	assert.Equal(t, "hey", m.Prompt())

	var got []string
	m.OnSubmit = func(line string) { got = append(got, line) }
	m, cmd := typed(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd, "Enter on a non-empty prompt submits it")
	cmd()
	assert.Equal(t, []string{"hey"}, got)
	assert.Equal(t, "", m.Prompt(), "the prompt clears after submit")
}

func TestPrompt_EmptyEnterDoesNothing(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	_, cmd := typed(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	assert.Nil(t, cmd)
}

func TestOutput_LinesAppearInTheMainPane(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: "first line\nsecond line"})
	assert.Contains(t, m.View(), "first line")
	assert.Contains(t, m.View(), "second line")
}

func TestCtrlC_Quits(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	_, cmd := typed(t, m, tea.KeyMsg{Type: tea.KeyCtrlC})
	require.NotNil(t, cmd)
	assert.Equal(t, tea.Quit(), cmd())
}

func TestStatusText_ReplacesTheStatusBox(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, StatusTextMsg{Text: "[turn 9k]"})
	assert.Contains(t, m.View(), "[turn 9k]")
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
	assert.Contains(t, m.View(), "row 99", "the tail is shown first")
	assert.NotContains(t, m.View(), "row 50")

	m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyPgUp})
	assert.Contains(t, m.View(), "row 75", "a page up scrolls back into history")
	assert.NotContains(t, m.View(), "row 99")

	m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyEnd})
	assert.Contains(t, m.View(), "row 99", "End returns to the newest rows")
}

func TestScroll_ClampsAtTheTop(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(5)})
	for range 50 {
		m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyUp})
	}
	assert.Contains(t, m.View(), "row 0", "scrolling past the top shows the first rows")
}

func TestScroll_TabMovesFocusToTheFeed(t *testing.T) {
	var f feed.Feed
	for i := range 100 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	m := resized(t, New(&f))
	m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyTab})
	m, _ = typed(t, m, tea.KeyMsg{Type: tea.KeyPgUp})
	assert.NotContains(t, m.View(), "step 99", "the feed scrolled, not the main pane")
}

func TestScroll_WheelScrollsTheBoxUnderThePointer(t *testing.T) {
	var f feed.Feed
	for i := range 100 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	m := resized(t, New(&f))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})

	feedX := m.width - m.width/3 + 2
	m, _ = typed(t, m, tea.MouseMsg{X: feedX, Y: 3, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelUp})
	assert.NotContains(t, m.View(), "step 99", "the wheel over the feed scrolls the feed")
	assert.Contains(t, m.View(), "row 99", "the main pane did not move")
}
