package agent

import (
	"strings"
	"sync"

	parser "github.com/kdeps/kdeps/v2/pkg/parser/yaml"
	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

// feedBodyLines is how many lines of a tool result the feed shows per entry.
const feedBodyLines = 6

// fileEditTools are the tools whose result is a file change, shown as a diff.
//
//nolint:gochecknoglobals // fixed set
var fileEditTools = map[string]bool{"edit_file": true, "write_file": true}

// tuiFeedSink is the feed the full-screen view shows, or nil when that view is
// not running.
//
//nolint:gochecknoglobals // one feed per process, set by the full-screen view
var tuiFeedSink struct {
	sync.Mutex
	f *feed.Feed
}

// setTUIFeed points the tool results at f. A nil feed turns feeding off.
func setTUIFeed(f *feed.Feed) {
	tuiFeedSink.Lock()
	tuiFeedSink.f = f
	tuiFeedSink.Unlock()
	if f != nil {
		drainPendingResources(f)
	}
}

// InstallResourceFeed makes every loaded resource file show in the feed. Called
// once at process start, like InstallStagePermission.
func InstallResourceFeed() {
	parser.SetResourceLoaded(feedResourceLoaded)
}

// feedToolResult adds one tool call to the feed: a file edit as a diff, any
// other tool as action output. It does nothing when no feed is set.
func feedToolResult(tool, result string) {
	tuiFeedSink.Lock()
	defer tuiFeedSink.Unlock()
	if tuiFeedSink.f == nil {
		return
	}
	kind := feed.KindAction
	if fileEditTools[tool] {
		kind = feed.KindDiff
	}
	body := strings.TrimRight(result, "\n")
	if lines := strings.Split(body, "\n"); len(lines) > feedBodyLines {
		body = strings.Join(lines[:feedBodyLines], "\n")
	}
	tuiFeedSink.f.Add(feed.Entry{Kind: kind, Title: tool, Body: body})
}
