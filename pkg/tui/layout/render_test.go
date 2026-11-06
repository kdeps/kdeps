package layout

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_FillsTheTerminalExactly(t *testing.T) {
	f := Compute(60, 20, true)
	out := Render(f, "", "main text", "feed text", "status", "> ")
	lines := strings.Split(out, "\n")
	assert.Len(t, lines, 20, "one line per terminal row")
	for i, line := range lines {
		assert.Equal(t, 60, lipgloss.Width(line), "row %d is the full width", i)
	}
}

func TestRender_ClosedFeedLeavesMainTheWidth(t *testing.T) {
	f := Compute(60, 20, false)
	out := Render(f, "", "main", "unused", "status", "> ")
	assert.NotContains(t, out, "unused")
	assert.Contains(t, out, "main")
}

func TestRender_BottomBoxesKeepTheirHeight(t *testing.T) {
	f := Compute(60, 20, true)
	short := Render(f, "", "a", "b", "status", "> ")
	long := Render(f, "", strings.Repeat("x\n", 100), "b", "status", "> ")
	assert.Equal(t, strings.Count(short, "\n"), strings.Count(long, "\n"),
		"a long main pane does not push the status or prompt down")
}

// TestRender_TextIsPaddedInsideTheBorder checks each box keeps one blank column
// between its border and its text, so text never touches a border.
func TestRender_TextIsPaddedInsideTheBorder(t *testing.T) {
	frame := Compute(60, 10, true)
	out := Render(frame, "", "hi", "", "", "")
	first := strings.Split(out, "\n")[2]
	assert.True(t, strings.HasPrefix(first, "│ hi"), "text starts one column inside the border: %q", first)
}

// TestRender_LongTextNeverGrowsABox checks a box stays as tall as its frame
// rows, even when its text is longer than the box is wide.
func TestRender_LongTextNeverGrowsABox(t *testing.T) {
	frame := Compute(60, 12, true)
	long := strings.Repeat("status words ", 40)
	out := Render(frame, "", "main", "feed", long, "> ")
	lines := strings.Split(out, "\n")
	assert.Equal(t, 12, len(lines), "the frame is exactly the terminal height")
	for i, line := range lines {
		assert.Equal(t, 60, lipgloss.Width(line), "row %d is the terminal width", i)
	}
}

// TestClampRows_CutsLinesAndRowsToTheFrame checks an overlong line and extra
// rows are cut, so the frame never exceeds the terminal.
func TestClampRows_CutsLinesAndRowsToTheFrame(t *testing.T) {
	out := clampRows("abcdefgh\nxy\nthird\nfourth", 3, 5)
	assert.Equal(t, "abcde\nxy\nthird", out)
}

// TestRender_LongStatusKeepsItsBorder checks a status line wider than the box
// is cut on its row, so the box keeps its bottom border and the prompt box
// below stays in place.
func TestRender_LongStatusKeepsItsBorder(t *testing.T) {
	f := Compute(60, 20, true)
	out := Render(f, "", "main", "feed", strings.Repeat("status ", 40), "> ")
	lines := strings.Split(out, "\n")
	require.Len(t, lines, 20)
	bottom := f.Status.Y + f.Status.H - 1
	assert.True(t, strings.HasPrefix(lines[bottom], "╰"), "status box keeps its bottom border: %q", lines[bottom])
	assert.Contains(t, lines[f.Status.Y+1], "…", "the cut status ends in an ellipsis")
	assert.True(t, strings.HasPrefix(lines[f.Prompt.Y], "╭"), "prompt box starts on its own row")
}

// TestRender_PaddingPastTheBoxIsDroppedQuietly checks a line that is only too
// wide because of trailing spaces gets no ellipsis.
func TestRender_PaddingPastTheBoxIsDroppedQuietly(t *testing.T) {
	assert.Equal(t, "abc", strings.TrimRight(fitInside("abc"+strings.Repeat(" ", 20), 10, 1), " "))
	assert.NotContains(t, fitInside("abc"+strings.Repeat(" ", 20), 10, 1), "…")
	assert.Contains(t, fitInside(strings.Repeat("x", 20), 10, 1), "…")
}
