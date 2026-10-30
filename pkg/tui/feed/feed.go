// Package feed is the right-hand panel: one continuous list of what happened
// in the session, newest at the bottom. Entries are resources that loaded,
// action output, and file diffs.
package feed

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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
}

// Add appends an entry.
func (f *Feed) Add(e Entry) {
	f.entries = append(f.entries, e)
}

// Len returns the number of entries.
func (f *Feed) Len() int { return len(f.entries) }

// Render returns the newest entries that fit in width x height. Each entry is
// a header line and its body; the oldest are dropped first.
func (f *Feed) Render(width, height int) string {
	if width < 1 || height < 1 {
		return ""
	}
	var lines []string
	for _, e := range f.entries {
		lines = append(lines, entryLines(e, width)...)
	}
	if len(lines) > height {
		lines = lines[len(lines)-height:]
	}
	return strings.Join(lines, "\n")
}

// entryLines is one entry as a header and its body, each cut to width.
func entryLines(e Entry, width int) []string {
	header := "[" + string(e.Kind) + "] " + e.Title
	out := []string{truncate(header, width)}
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
