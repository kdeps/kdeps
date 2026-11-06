package screen

import (
	"strconv"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
	"github.com/kdeps/kdeps/v2/pkg/tui/layout"
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
	assert.Contains(t, ansi.Strip(m.render()), "fetch.yaml")

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
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Equal(t, []string{"hey"}, got, "Enter hands the line over on the UI loop")
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
	assert.Contains(t, ansi.Strip(m.render()), "first line")
	assert.Contains(t, ansi.Strip(m.render()), "second line")
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
	assert.Contains(t, ansi.Strip(m.render()), "[turn 9k]")
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
	assert.Contains(t, ansi.Strip(m.render()), "row 99", "the tail is shown first")
	assert.NotContains(t, ansi.Strip(m.render()), "row 50")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	assert.Contains(t, ansi.Strip(m.render()), "row 75", "a page up scrolls back into history")
	assert.NotContains(t, ansi.Strip(m.render()), "row 99")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnd})
	assert.Contains(t, ansi.Strip(m.render()), "row 99", "End returns to the newest rows")
}

func TestCtrlEnd_FollowsTheMainPaneWhileTheFeedIsFocused(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	require.Greater(t, m.mainOff, 0)
	m = typeText(t, m, "hi")
	m, _ = typed(t, m, keyPress(tea.KeyTab))
	require.True(t, m.focusFeed)

	off := m.mainOff
	m, _ = typed(t, m, keyPress(tea.KeyEnd))
	assert.Equal(t, off, m.mainOff, "end follows the focused feed")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnd, Mod: tea.ModCtrl})
	assert.Equal(t, 0, m.mainOff, "ctrl+end follows the main pane")
	assert.True(t, m.focusFeed, "the feed stays focused")
	assert.Equal(t, "hi", m.Prompt())
	assert.Contains(t, ansi.Strip(m.render()), "row 99")
}

func TestScroll_ClampsAtTheTop(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(5)})
	for range 50 {
		m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	}
	assert.Contains(t, ansi.Strip(m.render()), "row 0", "scrolling past the top shows the first rows")
	assert.Equal(t, 0, m.mainOff, "the stored offset stops at the top")
}

func TestScroll_ExtraPageUpDoesNotStick(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	for range 10 {
		m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	}
	assert.Contains(t, ansi.Strip(m.render()), "row 0", "repeated page-up reaches the oldest row")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	assert.NotContains(t, ansi.Strip(m.render()), "row 0", "one page-down from the top moves forward")
}

func TestScroll_HalfPageMovesTheFocusedPane(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	m = typeText(t, m, "stay")
	half := m.halfRows()
	require.Greater(t, half, 1, "half a page is more than one row")
	require.Less(t, half, m.pageRows())

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl})
	assert.Equal(t, "stay", m.Prompt(), "ctrl+up does not type")
	assert.Equal(t, half, m.mainOff)
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "row "+strconv.Itoa(99-half))
	assert.NotContains(t, view, "row 99", "half a page leaves the newest row")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModCtrl})
	assert.Equal(t, 0, m.mainOff, "ctrl+down returns by the same half page")
	assert.Contains(t, ansi.Strip(m.render()), "row 99")

	var f feed.Feed
	for i := range 100 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	m = resized(t, New(&f))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModCtrl})
	assert.Equal(t, 0, m.mainOff, "the main pane stays when the feed is focused")
	assert.Equal(t, half, m.feedOff)
	assert.NotContains(t, ansi.Strip(m.render()), "step 99")
}

func TestHelp_ListsHalfPage(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("/model", "/session")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "ctrl+up/down", "the help lists half a page")
	assert.Contains(t, view, "/model  /session", "the command list still fits")
}

func TestScroll_NotFollowingKeepsTheRowsAndShowsTheHint(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m.Status = "[turn 1]"
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	assert.NotContains(t, ansi.Strip(m.render()), "not following")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "row 75")
	assert.Contains(t, view, "not following")
	assert.True(t, strings.Contains(view, "[turn 1]   not following"), "the counters stay ahead of the hint")

	m, _ = typed(t, m, OutputMsg{Text: "NEW"})
	view = ansi.Strip(m.render())
	assert.Contains(t, view, "row 75", "new output leaves the scrolled rows in place")
	assert.NotContains(t, view, "NEW")
	assert.Contains(t, view, "not following")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnd})
	view = ansi.Strip(m.render())
	assert.Contains(t, view, "NEW", "End follows the newest row")
	assert.NotContains(t, view, "not following")
}

func TestScroll_SendingFollowsTheNewestRow(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	assert.Contains(t, ansi.Strip(m.render()), "not following")

	m = typeText(t, m, "hello")
	m, _ = typed(t, m, keyPress(tea.KeyEnter))
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "hello", "sending shows the prompt at the tail")
	assert.NotContains(t, view, "not following")
}

func TestScroll_HomeJumpsToTheOldestRow(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyHome})
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "row 0", "Home shows the oldest row")
	assert.NotContains(t, view, "row 99", "Home leaves the newest row")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	for i := range 100 {
		m.Feed.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	for range 10 {
		m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	}
	assert.Contains(t, ansi.Strip(m.render()), "step 0")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgDown})
	assert.NotContains(t, ansi.Strip(m.render()), "step 0", "the feed offset stops at the top too")
}

func shiftUp() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyUp, Mod: tea.ModShift}
}

func shiftDown() tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: tea.KeyDown, Mod: tea.ModShift}
}

func TestScroll_ShiftUpDownMovesOneOutput(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	for i := range 30 {
		m, _ = typed(t, m, OutputMsg{Text: "item " + strconv.Itoa(i)})
	}
	m = typeText(t, m, "keep")
	assert.Contains(t, ansi.Strip(m.render()), "item 29")

	m, _ = typed(t, m, shiftUp())
	view := ansi.Strip(m.render())
	assert.NotContains(t, view, "item 29", "one shift+up hides the newest output")
	assert.Contains(t, view, "item 28")
	assert.Equal(t, "keep", m.Prompt(), "the prompt is not edited")
	assert.Equal(t, 1, m.mainOff)

	m, _ = typed(t, m, shiftUp())
	assert.NotContains(t, ansi.Strip(m.render()), "item 28")
	assert.Equal(t, 2, m.mainOff)

	m, _ = typed(t, m, shiftDown())
	view = ansi.Strip(m.render())
	assert.Contains(t, view, "item 28")
	assert.NotContains(t, view, "item 29")
	assert.Equal(t, 1, m.mainOff)
}

func TestScroll_ShiftUpTreatsABlockAsOneOutput(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(40)})
	m, _ = typed(t, m, OutputMsg{Text: "AAA\nBBB\nCCC\nDDD\nEEE"})
	m, _ = typed(t, m, OutputMsg{Text: "NEW"})
	assert.Contains(t, ansi.Strip(m.render()), "NEW")

	m, _ = typed(t, m, shiftUp())
	assert.Equal(t, 1, m.mainOff, "the newest one-line output is one step")
	assert.NotContains(t, ansi.Strip(m.render()), "NEW")
	assert.Contains(t, ansi.Strip(m.render()), "EEE")

	m, _ = typed(t, m, shiftUp())
	view := ansi.Strip(m.render())
	assert.Equal(t, 6, m.mainOff, "the five-line block is one step")
	assert.NotContains(t, view, "AAA")
	assert.NotContains(t, view, "EEE")
	assert.Contains(t, view, "row 39")

	m, _ = typed(t, m, shiftDown())
	assert.Equal(t, 1, m.mainOff, "shift+down lands on the output below")
	assert.Contains(t, ansi.Strip(m.render()), "AAA")
	assert.NotContains(t, ansi.Strip(m.render()), "NEW")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnd})
	for range 3 {
		m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyUp})
	}
	assert.Equal(t, 3, m.mainOff, "arrow up still moves one row")
	m, _ = typed(t, m, shiftUp())
	assert.Equal(t, 6, m.mainOff, "shift+up finishes the output it started inside")
	m, _ = typed(t, m, shiftDown())
	assert.Equal(t, 1, m.mainOff, "shift+down returns to the boundary under that output")
}

func TestScroll_ShiftUpMovesMainWhileTheFeedIsFocused(t *testing.T) {
	var f feed.Feed
	for i := range 40 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	m := resized(t, New(&f))
	for i := range 30 {
		m, _ = typed(t, m, OutputMsg{Text: "item " + strconv.Itoa(i)})
	}
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	assert.True(t, m.focusFeed)
	assert.Equal(t, 0, m.feedOff)

	m, _ = typed(t, m, shiftUp())
	assert.Equal(t, 0, m.feedOff, "the feed offset stays")
	assert.Equal(t, 1, m.mainOff, "the main pane moved by one output")
	assert.Contains(t, ansi.Strip(m.render()), "step 39", "the feed view did not scroll")
}

func TestScroll_ItemEndsDropWithTrimmedRows(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	for i := range maxMainLines + 3 {
		m, _ = typed(t, m, OutputMsg{Text: "n " + strconv.Itoa(i)})
	}
	assert.Equal(t, maxMainLines, len(m.main))
	assert.Equal(t, maxMainLines, len(m.outEnds))
	assert.Equal(t, maxMainLines, m.outEnds[len(m.outEnds)-1])

	m, _ = typed(t, m, shiftUp())
	assert.Equal(t, 1, m.mainOff, "one trimmed output is still one step")
}

func TestHelp_ListsOneOutputScroll(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("/model", "/session")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "shift+up/down", "the help lists one-output scroll")
	assert.Contains(t, view, "/model  /session", "the command list still fits")
	assert.Contains(t, view, "ctrl+o")
	assert.Contains(t, view, "ctrl+y")
}

func TestCopy_VisibleMainWhenNothingIsSelected(t *testing.T) {
	copyKey := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl | tea.ModShift, Text: "c"}
	m := resized(t, New(&feed.Feed{}))
	m.killed = []rune("stay")
	m, _ = typed(t, m, copyKey)
	assert.Equal(t, "stay", string(m.killed), "an empty pane leaves the yank buffer")
	assert.Equal(t, "", m.Prompt())

	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'c', Text: "c"})
	assert.Equal(t, "c", m.Prompt(), "c still types")
	assert.Equal(t, "stay", string(m.killed))

	m = m.setPrompt("")
	m, _ = typed(t, m, copyKey)
	got := string(m.killed)
	assert.Equal(t, "", m.Prompt(), "copy does not type c")
	assert.NotContains(t, got, "\n")
	assert.Contains(t, got, "row 99")
	assert.NotContains(t, got, "row 50")
	assert.False(t, m.killing)

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	m, _ = typed(t, m, copyKey)
	got = string(m.killed)
	assert.Contains(t, got, "row 75", "a scrolled pane copies the rows on screen")
	assert.NotContains(t, got, "row 99")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl, Text: "y"})
	assert.Contains(t, m.Prompt(), "row 75")
	assert.NotContains(t, m.Prompt(), "\n")

	m = typeText(t, m.setPrompt(""), "hello")
	m, _ = typed(t, m, ctrl('a'))
	right := tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}
	m, _ = typed(t, m, right)
	m, _ = typed(t, m, right)
	m, _ = typed(t, m, copyKey)
	assert.Equal(t, "he", string(m.killed), "a selection wins over the pane")
	assert.Equal(t, "hello", m.Prompt())
}

func TestFold_CtrlSpaceHidesTheRestOfTheOutput(t *testing.T) {
	fold := tea.KeyPressMsg{Code: ' ', Mod: tea.ModCtrl, Text: " "}
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(30)})
	m, _ = typed(t, m, OutputMsg{Text: "AAA\nBBB\nCCC"})
	m = typeText(t, m, "keep")
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "AAA")
	assert.Contains(t, view, "CCC")

	m, _ = typed(t, m, fold)
	view = ansi.Strip(m.render())
	assert.Contains(t, view, "AAA", "the first line stays")
	assert.NotContains(t, view, "BBB")
	assert.NotContains(t, view, "CCC")
	assert.Contains(t, view, "row 29")
	assert.Equal(t, "keep", m.Prompt(), "ctrl+space does not type a space")

	m, _ = typed(t, m, fold)
	view = ansi.Strip(m.render())
	assert.Contains(t, view, "BBB")
	assert.Contains(t, view, "CCC", "the same chord opens the output")

	m, _ = typed(t, m, OutputMsg{Text: "ONLY"})
	m, _ = typed(t, m, fold)
	view = ansi.Strip(m.render())
	assert.Contains(t, view, "ONLY", "a one-line output does not fold")
	assert.Contains(t, view, "CCC")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: ' ', Text: " "})
	assert.Equal(t, "keep ", m.Prompt(), "space still types")
}

func TestFold_TogglesTheOutputAtTheBottom(t *testing.T) {
	fold := tea.KeyPressMsg{Code: ' ', Mod: tea.ModCtrl}
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(40)})
	m, _ = typed(t, m, OutputMsg{Text: "AAA\nBBB\nCCC"})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	m, _ = typed(t, m, fold)
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "row 0", "the older output folds to its first line")
	assert.NotContains(t, view, "row 10")
	assert.Contains(t, view, "CCC", "the output below the window stays whole")
}

func TestScroll_TabMovesFocusToTheFeed(t *testing.T) {
	var f feed.Feed
	for i := range 100 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	m := resized(t, New(&f))
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	assert.NotContains(t, ansi.Strip(m.render()), "step 99", "the feed scrolled, not the main pane")
}

func TestSuspend_CtrlZLeavesThePrompt(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m = typeText(t, m, "keep")
	m, cmd := typed(t, m, tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl, Text: "z"})
	assert.Equal(t, "keep", m.Prompt(), "ctrl+z does not type")
	require.NotNil(t, cmd)
	_, ok := cmd().(tea.SuspendMsg)
	assert.True(t, ok, "ctrl+z suspends the view")
	m, cmd = typed(t, m, tea.KeyPressMsg{Code: 'z', Text: "z"})
	assert.Equal(t, "keepz", m.Prompt())
	assert.Nil(t, cmd)
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
	assert.NotContains(t, ansi.Strip(m.render()), "step 99", "the wheel over the feed scrolls the feed")
	assert.Contains(t, ansi.Strip(m.render()), "row 99", "the main pane did not move")
}

func TestClick_FocusesThePaneUnderThePointer(t *testing.T) {
	var f feed.Feed
	for i := range 100 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	m := resized(t, New(&f))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	frame := layout.Compute(m.width, m.height, m.feedOpen)

	m, _ = typed(t, m, tea.MouseClickMsg{X: frame.Feed.X + 2, Y: frame.Feed.Y + 1, Button: tea.MouseLeft})
	assert.True(t, m.focusFeed, "a click in the feed focuses it")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	view := ansi.Strip(m.render())
	assert.NotContains(t, view, "step 99", "the focused feed scrolled")
	assert.Contains(t, view, "row 99", "the main pane did not scroll")

	m, _ = typed(t, m, tea.MouseClickMsg{X: frame.Main.X + 2, Y: frame.Main.Y + 1, Button: tea.MouseLeft})
	assert.False(t, m.focusFeed, "a click in the main pane focuses it")
	m, _ = typed(t, m, tea.MouseClickMsg{X: frame.Feed.X + 2, Y: frame.Feed.Y + 1, Button: tea.MouseRight})
	assert.False(t, m.focusFeed, "a right click does not move focus")
}

func TestDrag_SelectsPromptText(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	for _, c := range "hello" {
		m, _ = typed(t, m, tea.KeyPressMsg{Code: c, Text: string(c)})
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	textX := frame.Prompt.X + 1 + layout.BoxPadX
	y := frame.Prompt.Y + 1

	m, _ = typed(t, m, tea.MouseClickMsg{X: textX + 2, Y: y, Button: tea.MouseLeft})
	// The caret sits on the first cell while the drag starts, so the last
	// letter is one cell further right than a click with the caret at the end.
	m, _ = typed(t, m, tea.MouseMotionMsg{X: textX + 8, Y: y, Button: tea.MouseLeft})
	m, _ = typed(t, m, tea.MouseReleaseMsg{X: textX + 8, Y: y, Button: tea.MouseLeft})
	lo, hi := m.selBounds()
	assert.Equal(t, "hello", string(m.input[lo:hi]), "dragging across the prompt selects it")
	assert.False(t, m.selecting, "release ends the drag")

	m, _ = typed(t, m, tea.MouseClickMsg{X: frame.Main.X + 2, Y: frame.Main.Y + 1, Button: tea.MouseLeft})
	assert.False(t, m.hasSel(), "a click in the main pane clears the selection")
}

func TestClick_MovesThePromptCursor(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	for _, c := range "hello" {
		m, _ = typed(t, m, tea.KeyPressMsg{Code: c, Text: string(c)})
	}
	assert.Equal(t, 5, m.cursor)
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	textX := frame.Prompt.X + 1 + layout.BoxPadX
	y := frame.Prompt.Y + 1

	m, _ = typed(t, m, tea.MouseClickMsg{X: textX + 2, Y: y, Button: tea.MouseLeft})
	assert.Equal(t, 0, m.cursor, "a click on the first letter moves the cursor there")

	m.cursor = len(m.input)
	m.anchor = m.cursor
	m, _ = typed(t, m, tea.MouseClickMsg{X: textX + 4, Y: y, Button: tea.MouseLeft})
	assert.Equal(t, 2, m.cursor, "a click on a later letter lands on that letter")

	m.cursor = len(m.input)
	m.anchor = m.cursor
	m, _ = typed(t, m, tea.MouseClickMsg{X: textX + 7, Y: y, Button: tea.MouseLeft})
	assert.Equal(t, 5, m.cursor, "a click on the bar at the end stays at the end")

	m, _ = typed(t, m, tea.MouseClickMsg{X: textX, Y: y, Button: tea.MouseLeft})
	assert.Equal(t, 0, m.cursor, "a click on the prompt marker moves to the start")
}

func TestClick_DoubleAndTripleClickSelect(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	for _, c := range "say hello world" {
		m, _ = typed(t, m, tea.KeyPressMsg{Code: c, Text: string(c)})
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	textX := frame.Prompt.X + 1 + layout.BoxPadX
	y := frame.Prompt.Y + 1
	click := tea.MouseClickMsg{X: textX + 2 + 4, Y: y, Button: tea.MouseLeft}

	m, _ = typed(t, m, click)
	assert.Equal(t, 4, m.cursor, "the first click lands on h")
	assert.False(t, m.hasSel())

	m, _ = typed(t, m, click)
	lo, hi := m.selBounds()
	assert.Equal(t, "hello", string(m.input[lo:hi]), "the second click selects the word")
	assert.False(t, m.selecting)

	m, _ = typed(t, m, click)
	lo, hi = m.selBounds()
	assert.Equal(t, "say hello world", string(m.input[lo:hi]), "the third click selects the line")

	m, _ = typed(t, m, click)
	assert.False(t, m.hasSel(), "the click after a triple click places the cursor")

	m.lastClick = time.Now().Add(-time.Second)
	m, _ = typed(t, m, click)
	assert.False(t, m.hasSel(), "a late second click does not select the word")
}

func TestPromptWindow_KeepsTheCursorOnScreen(t *testing.T) {
	m := typeText(t, resized(t, New(&feed.Feed{})), strings.Repeat("a", 200))
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	inner := layout.InnerWidth(frame.Prompt.W)
	shown := ansi.Strip(m.promptLine(inner))
	assert.LessOrEqual(t, ansi.StringWidth(shown), inner)
	assert.Contains(t, shown, "|")
	assert.NotContains(t, shown, "> ", "a long line scrolls the marker off")
	assert.NotContains(t, shown, strings.Repeat("a", 200))

	textX := frame.Prompt.X + 1 + layout.BoxPadX
	y := frame.Prompt.Y + 1
	clicked, _ := typed(t, m, tea.MouseClickMsg{X: textX + 2, Y: y, Button: tea.MouseLeft})
	assert.NotEqual(t, 0, clicked.cursor, "a click in the scrolled window is not the start")
	clicked, _ = typed(t, m, tea.MouseClickMsg{X: textX + inner - 1, Y: y, Button: tea.MouseLeft})
	assert.Equal(t, 200, clicked.cursor, "a click on the drawn bar stays at the end")

	next, _ := m.Update(ctrl('a'))
	m = next.(Model)
	shown = ansi.Strip(m.promptLine(inner))
	assert.Contains(t, shown, "> |aaa", "the start of the line comes back with the cursor")
	m, _ = typed(t, m, tea.MouseClickMsg{X: textX + 2, Y: y, Button: tea.MouseLeft})
	assert.Equal(t, 0, m.cursor, "a click on the bar at the start stays there")
}

// TestFrameCache_ChangesWhenTheScreenChanges checks a frame is reused only
// while nothing it is drawn from has changed.
func TestFrameCache_ChangesWhenTheScreenChanges(t *testing.T) {
	var f feed.Feed
	m := resized(t, New(&f))
	first := m.render()
	assert.Equal(t, first, m.render(), "an unchanged screen reuses its frame")

	f.Add(feed.Entry{Kind: feed.KindAction, Title: "new step"})
	assert.Contains(t, ansi.Strip(m.render()), "new step", "a feed change redraws the frame")

	m, _ = typed(t, m, OutputMsg{Text: "fresh output"})
	assert.Contains(t, ansi.Strip(m.render()), "fresh output", "new output redraws the frame")

	m = typeText(t, m, "x")
	assert.Contains(t, ansi.Strip(m.render()), "> x|", "a prompt edit redraws the frame")
}

// TestFrameCache_HeaderAndProfileRedraw checks a model chip, a pill, the
// title, or the terminal color profile is not served from the previous frame.
func TestFrameCache_HeaderAndProfileRedraw(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	stale := func(m Model) Model {
		t.Helper()
		_ = m.render()
		m.cache.text = "STALE"
		return m
	}

	m = stale(m)
	m, _ = typed(t, m, tea.ColorProfileMsg{Profile: colorprofile.ANSI})
	assert.NotContains(t, m.render(), "STALE", "a color profile change redraws")

	m = stale(m)
	m, _ = typed(t, m, ModelMsg{Name: "llama"})
	got := m.render()
	assert.NotContains(t, got, "STALE")
	assert.Contains(t, ansi.Strip(got), "llama")

	m = stale(m)
	m.Title = "proj"
	got = m.render()
	assert.NotContains(t, got, "STALE")
	assert.Contains(t, ansi.Strip(got), "proj")

	m = stale(m)
	m, _ = typed(t, m, PillMsg{Name: "backend", Value: "openrouter"})
	got = m.render()
	assert.NotContains(t, got, "STALE")
	assert.Contains(t, ansi.Strip(got), "openrouter")
}

// TestFrameCache_OverlayIsNeverCached checks an open overlay is drawn fresh
// every time, since it changes without any cached key moving.
func TestFrameCache_OverlayIsNeverCached(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m = m.WithOverlay(fakeOverlay{text: "ONE"}, nil)
	assert.Contains(t, ansi.Strip(m.render()), "ONE")
	m = m.WithOverlay(fakeOverlay{text: "TWO"}, nil)
	assert.Contains(t, ansi.Strip(m.render()), "TWO", "a new overlay is drawn, not the cached frame")
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

func TestScrollBar_ShowsOnScrollAndHidesOnItsMessage(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	before := strings.Count(m.render(), "│")
	assert.False(t, m.mainBar, "the bar stays hidden until the pane scrolls")

	m, cmd := typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	require.NotNil(t, cmd, "a scroll of a tall pane arms the hide")
	assert.True(t, m.mainBar)
	assert.Greater(t, strings.Count(m.render(), "│"), before, "the bar is a column of the pane")
	lines := strings.Split(m.render(), "\n")
	require.Len(t, lines, 30)
	for i, line := range lines {
		assert.Equal(t, 100, ansi.StringWidth(line), "row %d keeps the terminal width", i)
	}

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	stale := m.mainBarSeq - 1
	m, _ = typed(t, m, barHideMsg{seq: stale})
	assert.True(t, m.mainBar, "a hide from an older scroll does not drop the bar")
	m, _ = typed(t, m, barHideMsg{seq: m.mainBarSeq})
	assert.False(t, m.mainBar, "the matching hide drops the bar")
}

func TestScrollBar_ShortPaneAndTypingStayPlain(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(5)})
	m, cmd := typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	assert.Nil(t, cmd, "a pane that fits has no bar to hide")
	assert.False(t, m.mainBar)

	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'b', Text: "b"})
	assert.Equal(t, "b", m.Prompt(), "b still types")
	assert.False(t, m.mainBar)
}

func TestScrollBar_FeedScrollDoesNotMarkTheMainPane(t *testing.T) {
	var f feed.Feed
	for i := range 100 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "step " + strconv.Itoa(i)})
	}
	m := resized(t, New(&f))
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	m, cmd := typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	require.NotNil(t, cmd)
	assert.True(t, m.feedBar, "scrolling the feed marks the feed")
	assert.False(t, m.mainBar, "the main pane was not scrolled")
	lines := strings.Split(m.render(), "\n")
	require.Len(t, lines, 30)
	for i, line := range lines {
		assert.Equal(t, 100, ansi.StringWidth(line), "row %d keeps the terminal width", i)
	}
}

func TestScrollBar_ThumbSitsAtTheEndYouAreOn(t *testing.T) {
	marks := barMarks(10, 40, 0)
	require.Len(t, marks, 10)
	assert.Equal(t, barThumb(), marks[9], "offset zero is the newest row, so the thumb is at the bottom")
	assert.Equal(t, barTrack(), marks[0])

	marks = barMarks(10, 40, 30)
	assert.Equal(t, barThumb(), marks[0], "the oldest row puts the thumb at the top")
	assert.Nil(t, barMarks(10, 10, 0), "a pane that fits has no bar")
	assert.Nil(t, barMarks(10, 4, 0))
}

func TestScrollBar_HideRedrawsTheFrame(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyPgUp})
	_ = m.render()
	m.cache.text = "STALE"
	m, _ = typed(t, m, barHideMsg{seq: m.mainBarSeq})
	assert.NotContains(t, m.render(), "STALE", "hiding the bar is a new frame")
}

func TestHelp_ListsTheScrollBar(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	assert.Contains(t, ansi.Strip(m.render()), "scroll one row, bar")
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

// TestHeader_ShowsTheModelChip checks the header carries the model name as a
// chip, between the rule and the title.
func TestHeader_ShowsTheModelChip(t *testing.T) {
	m := New(&feed.Feed{})
	m.Title = "proj"
	m, _ = typed(t, m, ModelMsg{Name: "llama-3.2"})
	assert.Equal(t, " kdeps ╱╱╱  llama-3.2  ╱╱╱ proj", ansi.Strip(m.headerText()))
}

// TestPills_MergeTheValuesIntoOneChip checks the set pills show as one chip in
// backend, session, tool order, and a cleared value leaves the others.
func TestPills_MergeTheValuesIntoOneChip(t *testing.T) {
	m := New(&feed.Feed{})
	m, _ = typed(t, m, PillMsg{Name: "tool", Value: "bash_exec"})
	m, _ = typed(t, m, PillMsg{Name: "backend", Value: "openrouter"})
	assert.Equal(t, "    openrouter · bash_exec ", ansi.Strip(m.pillsText()))
	m, _ = typed(t, m, PillMsg{Name: "tool", Value: ""})
	assert.Equal(t, "    openrouter ", ansi.Strip(m.pillsText()))
}

// TestPills_CtrlTHidesTheChip checks ctrl+t hides the header pills and shows
// them again. The model chip, the title, and the prompt stay. t still types.
func TestPills_CtrlTHidesTheChip(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m.Title = "proj"
	m, _ = typed(t, m, ModelMsg{Name: "llama-3.2"})
	m, _ = typed(t, m, PillMsg{Name: "backend", Value: "openrouter"})
	m = typeText(t, m, "keep")
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "openrouter")
	assert.Contains(t, view, "llama-3.2")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl, Text: "t"})
	view = ansi.Strip(m.render())
	assert.NotContains(t, view, "openrouter")
	assert.Contains(t, view, "llama-3.2")
	assert.Contains(t, view, "proj")
	assert.Equal(t, "keep", m.Prompt())
	m, _ = typed(t, m, PillMsg{Name: "backend", Value: "ollama"})
	assert.NotContains(t, ansi.Strip(m.render()), "ollama", "a hidden chip stays hidden")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 't', Mod: tea.ModCtrl, Text: "t"})
	assert.Contains(t, ansi.Strip(m.render()), "ollama")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 't', Text: "t"})
	assert.Equal(t, "keept", m.Prompt())
}

// TestMessageArea_ShowsTheSubmittedLineBeforeTheReply checks a submitted line
// is echoed into the main pane, as a line of its own.
func TestMessageArea_ShowsTheSubmittedLineBeforeTheReply(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m.OnSubmit = func(string) {}
	m = typeText(t, m, "list files")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	assert.Contains(t, strings.Join(m.main, "\n"), "list files", "the user's line is in the message area")
	assert.Contains(t, ansi.Strip(m.main[len(m.main)-1]), "› list files")
}

// TestRedrawMsg_ShowsAFeedChangeMadeOffTheLoop checks a redraw message after
// the feed changes draws the new entry.
func TestRedrawMsg_ShowsAFeedChangeMadeOffTheLoop(t *testing.T) {
	var f feed.Feed
	m := resized(t, New(&f))
	f.Add(feed.Entry{Kind: feed.KindAction, Title: "bash_exec"})
	m, _ = typed(t, m, RedrawMsg{})
	assert.Contains(t, ansi.Strip(m.render()), "bash_exec", "the new entry is drawn after the redraw")
}
