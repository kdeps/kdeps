package agent

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

func TestFeedToolResult_EditsAreDiffsAndOtherToolsAreActions(t *testing.T) {
	var f feed.Feed
	setTUIFeed(&f)
	t.Cleanup(func() { setTUIFeed(nil) })

	feedToolResult("edit_file", `{"path":"pkg/x.go"}`, "edited pkg/x.go")
	feedToolResult("bash_exec", `{"command":"git status --short"}`, "ok")

	out := ansi.Strip(f.Render(80, 10))
	assert.Contains(t, out, "[diff] edit_file")
	assert.Contains(t, out, "[action] bash_exec")
}

func TestFeedToolResult_LongOutputIsCutToTheBodyLines(t *testing.T) {
	var f feed.Feed
	setTUIFeed(&f)
	t.Cleanup(func() { setTUIFeed(nil) })

	feedToolResult("bash_exec", "{}", strings.Repeat("line\n", 50))
	assert.Equal(t, 1+defaultFeedBodyLines, strings.Count(f.Render(80, 100), "\n")+1)
}

// TestFeedToolResult_TheEventSetsTheBodyLength checks the tui-activity-lines
// event's items cut the body, and a disabled event leaves it whole.
func TestFeedToolResult_TheEventSetsTheBodyLength(t *testing.T) {
	var f feed.Feed
	setTUIFeed(&f)
	t.Cleanup(func() { setTUIFeed(nil) })

	withEvent(t, Event{Name: eventTuiActivityLines, Items: 2, Run: actionTruncate})
	feedToolResult("bash_exec", "{}", strings.Repeat("line\n", 50))
	assert.Equal(t, 1+2, strings.Count(f.Render(80, 100), "\n")+1)
}

func TestFeedToolResult_ADisabledEventLeavesTheBodyWhole(t *testing.T) {
	var f feed.Feed
	setTUIFeed(&f)
	t.Cleanup(func() { setTUIFeed(nil) })

	withEvent(t, Event{Name: eventTuiActivityLines, Items: 2, Run: actionTruncate, Disabled: true})
	feedToolResult("bash_exec", "{}", strings.Repeat("line\n", 50))
	assert.Equal(t, 1+50, strings.Count(f.Render(80, 200), "\n")+1)
}

func TestFeedToolResult_NoFeedIsANoOp(t *testing.T) {
	setTUIFeed(nil)
	assert.NotPanics(t, func() { feedToolResult("bash_exec", "{}", "x") })
}

// TestToolTarget_NamesWhatTheToolActedOn checks the activity title carries the
// command, path or query, and stays empty when there is none.
func TestToolTarget_NamesWhatTheToolActedOn(t *testing.T) {
	assert.Equal(t, "git status --short", toolTarget(`{"command":"git  status\n --short"}`))
	assert.Equal(t, "pkg/x.go", toolTarget(`{"path":"pkg/x.go"}`))
	assert.Equal(t, "", toolTarget(`{"other":"x"}`))
	assert.Equal(t, "", toolTarget("not json"))
	long := toolTarget(`{"query":"` + strings.Repeat("a", 100) + `"}`)
	assert.Equal(t, toolTargetMax+1, len([]rune(long)), "cut to the max, with an ellipsis")
}

// TestFeedToolResult_RedrawsTheViewAfterAnEntry checks an entry added off the UI
// loop asks the running view to draw again.
func TestFeedToolResult_RedrawsTheViewAfterAnEntry(t *testing.T) {
	var f feed.Feed
	setTUIFeed(&f)
	redraws := 0
	tuiFeedSink.Lock()
	tuiFeedSink.redraw = func() { redraws++ }
	tuiFeedSink.Unlock()
	t.Cleanup(func() {
		tuiFeedSink.Lock()
		tuiFeedSink.redraw = nil
		tuiFeedSink.Unlock()
		setTUIFeed(nil)
	})

	feedToolResult("bash_exec", "{}", "ok")
	assert.Equal(t, 1, redraws, "the view is asked to draw after the entry")
}
