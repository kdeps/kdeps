package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

func TestFeedToolResult_EditsAreDiffsAndOtherToolsAreActions(t *testing.T) {
	var f feed.Feed
	setTUIFeed(&f)
	t.Cleanup(func() { setTUIFeed(nil) })

	feedToolResult("edit_file", "edited pkg/x.go")
	feedToolResult("bash_exec", "ok")

	out := f.Render(80, 10)
	assert.Contains(t, out, "[diff] edit_file")
	assert.Contains(t, out, "[action] bash_exec")
}

func TestFeedToolResult_LongOutputIsCutToTheBodyLines(t *testing.T) {
	var f feed.Feed
	setTUIFeed(&f)
	t.Cleanup(func() { setTUIFeed(nil) })

	feedToolResult("bash_exec", strings.Repeat("line\n", 50))
	assert.Equal(t, 1+feedBodyLines, strings.Count(f.Render(80, 100), "\n")+1)
}

func TestFeedToolResult_NoFeedIsANoOp(t *testing.T) {
	setTUIFeed(nil)
	assert.NotPanics(t, func() { feedToolResult("bash_exec", "x") })
}
