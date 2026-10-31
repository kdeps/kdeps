// Package feed is the right-hand panel: one continuous list of what happened
// in the session, newest at the bottom. Entries are resources that loaded,
// action output, and file diffs.
package feed

import (
	"strings"

	"charm.land/lipgloss/v2"
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

// Feed holds the entries in the order they happened.
type Feed struct {
	entries []Entry
	version int // bumped by every change, so a renderer can cache a frame
}

// Add appends an entry.
func (f *Feed) Add(e Entry) {
	f.entries = append(f.entries, e)
	f.version++
}

// Version changes whenever the feed changes.
func (f *Feed) Version() int { return f.version }

// Len returns the number of entries.
func (f *Feed) Len() int { return len(f.entries) }

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
	var lines []string
	for _, e := range f.entries {
		lines = append(lines, entryLines(e, width)...)
	}
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
	n := 0
	for _, e := range f.entries {
		n += len(entryLines(e, width))
	}
	return n
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

// truncate cuts s to width cells.
func truncate(s string, width int) string {
	if lipgloss.Width(s) <= width {
		return s
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width-1 {
		runes = runes[:len(runes)-1]
	}
	return string(runes) + "…"
}
