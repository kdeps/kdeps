package agent

import (
	"encoding/json"
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
func feedToolResult(tool, args, result string) {
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
	title := tool
	if target := toolTarget(args); target != "" {
		title = tool + " " + target
	}
	tuiFeedSink.f.Add(feed.Entry{Kind: kind, Title: title, Body: body})
}

// toolTargetKeys returns the argument names that say what a tool acted on, most
// telling first.
func toolTargetKeys() []string {
	return []string{"command", "path", "file_path", "query", "url", "pattern", "name"}
}

// toolTargetMax is how many runes of the target the activity title keeps.
const toolTargetMax = 60

// toolTarget is the part of a tool call's JSON arguments that says what it acted
// on (the command, the path, the query), cut to toolTargetMax runes. It is empty
// when the arguments do not parse or name no target.
func toolTarget(args string) string {
	var fields map[string]any
	if err := json.Unmarshal([]byte(args), &fields); err != nil {
		return ""
	}
	for _, key := range toolTargetKeys() {
		value, ok := fields[key].(string)
		if !ok || value == "" {
			continue
		}
		runes := []rune(strings.Join(strings.Fields(value), " "))
		if len(runes) > toolTargetMax {
			return string(runes[:toolTargetMax]) + "…"
		}
		return string(runes)
	}
	return ""
}
