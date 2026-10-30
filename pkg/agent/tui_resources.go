package agent

import (
	"path/filepath"
	"sync"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

// pendingResources are resource loads that happened before a feed was set.
// They are added when the feed attaches, so nothing loaded earlier is lost.
//
//nolint:gochecknoglobals // one pending list per process
var pendingResources struct {
	sync.Mutex
	paths []string
}

// feedResourceLoaded adds a loaded resource file to the feed, or holds it
// until a feed is set.
func feedResourceLoaded(path string) {
	tuiFeedSink.Lock()
	f := tuiFeedSink.f
	tuiFeedSink.Unlock()
	if f == nil {
		pendingResources.Lock()
		pendingResources.paths = append(pendingResources.paths, path)
		pendingResources.Unlock()
		return
	}
	addResourceEntry(f, path)
}

// addResourceEntry writes one resource-loaded entry.
func addResourceEntry(f *feed.Feed, path string) {
	f.Add(feed.Entry{Kind: feed.KindResource, Title: filepath.Base(path), Body: path})
}

// drainPendingResources adds the held loads to f and clears them.
func drainPendingResources(f *feed.Feed) {
	pendingResources.Lock()
	paths := pendingResources.paths
	pendingResources.paths = nil
	pendingResources.Unlock()
	for _, p := range paths {
		addResourceEntry(f, p)
	}
}
