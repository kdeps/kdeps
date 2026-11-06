// Package feed is the right-hand panel: one continuous list of what happened
// in the session, newest at the bottom. Entries are resources that loaded,
// action output, and file diffs.
package feed

import (
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Kind is what an entry is about.
type Kind string

const (
	KindResource Kind = "resource" // a YAML resource that loaded
	KindAction   Kind = "action"   // the output of an action
	KindDiff     Kind = "diff"     // a file edit
)

// Entry is one item in the feed.
type Entry struct {
	Kind  Kind
	Title string
	Body  string
}

// Feed holds the entries in the order they happened. The agent adds entries
// while the view reads them, so every method takes the lock.
type Feed struct {
	mu      sync.Mutex
	entries []Entry
	version int // bumped by every change, so a renderer can cache a frame

	// Rendered lines per width, kept across calls (the view measures the feed
	// with and without its scroll-bar column). Entries never change once
	// added, so only the entries added since are rendered.
	rendered map[int]*renderedLines
}

// renderedLines is every entry rendered at one width so far.
type renderedLines struct {
	lines []string
	done  int // entries already in lines
}

// Add appends an entry.
func (f *Feed) Add(e Entry) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.entries = append(f.entries, e)
	f.version++
}

// Version changes whenever the feed changes.
func (f *Feed) Version() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.version
}

// Len returns the number of entries.
func (f *Feed) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.entries)
}

// renderedLocked is every entry's lines at width, from the cache. f.mu is held.
func (f *Feed) renderedLocked(width int) []string {
	if f.rendered == nil {
		f.rendered = map[int]*renderedLines{}
	}
	r := f.rendered[width]
	if r == nil {
		if len(f.rendered) >= maxRenderedWidths {
			clear(f.rendered) // a resize drag leaves widths nobody asks for again
		}
		r = &renderedLines{}
		f.rendered[width] = r
	}
	for _, e := range f.entries[r.done:] {
		r.lines = append(r.lines, entryLines(e, width)...)
	}
	r.done = len(f.entries)
	return r.lines
}

// maxRenderedWidths bounds the per-width cache; a buffer size, not policy.
const maxRenderedWidths = 4

// Render returns the newest entries that fit in width x height. Each entry is
// a header line and its body; the oldest are dropped first.
func (f *Feed) Render(width, height int) string {
	return f.RenderWindow(width, height, 0)
}

// RenderWindow returns height lines ending offset lines above the newest one,
// so offset 0 is the tail and a larger offset scrolls back through the feed.
// The offset is clamped to the feed's length.
func (f *Feed) RenderWindow(width, height, offset int) string {
	if width < 1 || height < 1 {
		return ""
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	lines := f.renderedLocked(width)
	if top := len(lines) - height; offset > top {
		offset = max(top, 0)
	}
	if offset < 0 {
		offset = 0
	}
	end := len(lines) - offset
	if start := max(end-height, 0); start < end {
		return strings.Join(lines[start:end], "\n")
	}
	return ""
}

// Lines returns how many rendered lines the feed has at width.
func (f *Feed) Lines(width int) int {
	if width < 1 {
		return 0
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.renderedLocked(width))
}

// entryHeader is the "[kind] title" line, cut to width. The kind is bold so
// the eye finds the kind of each entry first; the title is cut as plain text.
func entryHeader(e Entry, width int) string {
	kind := "[" + string(e.Kind) + "]"
	if lipgloss.Width(kind) >= width {
		return truncate(kind, width)
	}
	return lipgloss.NewStyle().Bold(true).Render(kind) + truncate(" "+e.Title, width-lipgloss.Width(kind))
}

// entryLines is one entry as a header and its body, each cut to width.
func entryLines(e Entry, width int) []string {
	out := []string{entryHeader(e, width)}
	if e.Body == "" {
		return out
	}
	for _, line := range strings.Split(e.Body, "\n") {
		out = append(out, truncate("  "+line, width))
	}
	return out
}

// truncate cuts s to width cells, ending in an ellipsis when it cuts. One
// pass: it runs for every feed line the view lays out.
func truncate(s string, width int) string {
	return ansi.Truncate(s, width, "…")
}
