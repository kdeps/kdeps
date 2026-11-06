package screen

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

// clockFilter is an InputFilter on a clock the test moves.
func clockFilter() (*InputFilter, *time.Time) {
	at := time.Unix(1000, 0)
	return &InputFilter{now: func() time.Time { return at }}, &at
}

func wheelUp(x int) tea.MouseWheelMsg {
	return tea.MouseWheelMsg{X: x, Button: tea.MouseWheelUp}
}

func TestInputFilter_FoldsWheelNotchesWithinAFrame(t *testing.T) {
	f, at := clockFilter()
	assert.Equal(t, WheelMsg{X: 5, Steps: 1}, f.Filter(nil, wheelUp(5)), "the first notch goes through at once")
	assert.Nil(t, f.Filter(nil, wheelUp(5)))
	assert.Nil(t, f.Filter(nil, wheelUp(5)))
	*at = at.Add(inputSampleGap)
	assert.Equal(t, WheelMsg{X: 5, Steps: 3}, f.Filter(nil, wheelUp(5)), "the held notches arrive as one step count")
}

func TestInputFilter_DirectionChangeDropsTheOtherWay(t *testing.T) {
	f, at := clockFilter()
	f.Filter(nil, wheelUp(1))
	f.Filter(nil, wheelUp(1))
	assert.Nil(t, f.Filter(nil, tea.MouseWheelMsg{Button: tea.MouseWheelDown}))
	*at = at.Add(inputSampleGap)
	got := f.Filter(nil, tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	assert.Equal(t, -2, got.(WheelMsg).Steps)
}

func TestInputFilter_ThinsMotionAndPassesEverythingElse(t *testing.T) {
	f, at := clockFilter()
	motion := tea.MouseMotionMsg{X: 1}
	assert.Equal(t, motion, f.Filter(nil, motion))
	assert.Nil(t, f.Filter(nil, motion))
	*at = at.Add(inputSampleGap)
	assert.Equal(t, motion, f.Filter(nil, motion))
	key := tea.KeyPressMsg{Code: 'a', Text: "a"}
	assert.Equal(t, key, f.Filter(nil, key), "keys are never held back")
	side := tea.MouseWheelMsg{Button: tea.MouseWheelLeft}
	assert.Equal(t, side, f.Filter(nil, side))
}

func TestWheelMsg_ScrollsByEveryNotch(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(200)})
	m, _ = typed(t, m, WheelMsg{X: 2, Steps: 3})
	assert.Equal(t, 3*wheelStep, m.mainOff)
	m, _ = typed(t, m, WheelMsg{X: 2, Steps: 0})
	assert.Equal(t, 3*wheelStep, m.mainOff, "no notches, no move")
}

func TestResize_ReleaseSetsTheFinalWidth(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	frame := m.frame()
	y := frame.Feed.Y + 2
	m, _ = typed(t, m, tea.MouseClickMsg{X: frame.Feed.X, Y: y, Button: tea.MouseLeft})
	m, _ = typed(t, m, tea.MouseReleaseMsg{X: 45, Y: y, Button: tea.MouseLeft})
	assert.Equal(t, 55, m.frame().Feed.W, "a dropped motion sample does not lose where the drag ended")
}

// TestMainRows_AreWrappedOncePerChange checks the scroll keys and the frame
// reuse one wrap of the main pane until it changes.
func TestMainRows_AreWrappedOncePerChange(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(50)})
	first := m.wrappedMain(40)
	again := m.wrappedMain(40)
	require.NotEmpty(t, first)
	assert.Same(t, &first[0], &again[0], "an unchanged pane is not wrapped again")
	m, _ = typed(t, m, OutputMsg{Text: "one more"})
	after := m.wrappedMain(40)
	assert.Equal(t, "one more", strings.TrimSpace(after[len(after)-1]), "new output is wrapped in")
}
