package agent

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

func TestResourceFeed_HeldLoadsAppearWhenTheFeedAttaches(t *testing.T) {
	pendingResources.Lock()
	pendingResources.paths = nil
	pendingResources.Unlock()
	setTUIFeed(nil)

	feedResourceLoaded("/w/resources/fetch.yaml")
	var f feed.Feed
	setTUIFeed(&f)
	t.Cleanup(func() { setTUIFeed(nil) })

	assert.Contains(t, ansi.Strip(f.Render(80, 10)), "[resource] fetch.yaml")
}

func TestResourceFeed_LoadsWithAFeedAppearAtOnce(t *testing.T) {
	var f feed.Feed
	setTUIFeed(&f)
	t.Cleanup(func() { setTUIFeed(nil) })

	feedResourceLoaded("/w/resources/parse.yaml")
	assert.Contains(t, ansi.Strip(f.Render(80, 10)), "[resource] parse.yaml")
}
