package feed

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestRender_ShowsEntriesInOrderNewestAtTheBottom(t *testing.T) {
	var f Feed
	f.Add(Entry{Kind: KindResource, Title: "fetch.yaml"})
	f.Add(Entry{Kind: KindAction, Title: "bash_exec", Body: "exit 0"})
	out := f.Render(40, 10)
	assert.Equal(t, "[resource] fetch.yaml\n[action] bash_exec\n  exit 0", ansi.Strip(out))
}

func TestRender_KeepsTheNewestWhenItDoesNotFit(t *testing.T) {
	var f Feed
	for _, name := range []string{"a", "b", "c", "d"} {
		f.Add(Entry{Kind: KindResource, Title: name})
	}
	out := f.Render(40, 2)
	assert.Equal(t, "[resource] c\n[resource] d", ansi.Strip(out))
}

func TestRender_CutsLinesToTheWidth(t *testing.T) {
	var f Feed
	f.Add(Entry{Kind: KindDiff, Title: "edit.go", Body: strings.Repeat("x", 100)})
	for _, line := range strings.Split(f.Render(20, 10), "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), 20)
	}
}

func TestRender_EmptyFeedAndZeroSize(t *testing.T) {
	var f Feed
	assert.Equal(t, "", f.Render(40, 10))
	f.Add(Entry{Kind: KindAction, Title: "x"})
	assert.Equal(t, "", f.Render(0, 10))
}

func TestRenderWindow_ScrollsBackAndClamps(t *testing.T) {
	var f Feed
	for _, name := range []string{"a", "b", "c", "d"} {
		f.Add(Entry{Kind: KindResource, Title: name})
	}
	assert.Equal(t, "[resource] c\n[resource] d", ansi.Strip(f.RenderWindow(40, 2, 0)))
	assert.Equal(t, "[resource] a\n[resource] b", ansi.Strip(f.RenderWindow(40, 2, 2)))
	assert.Equal(t, "[resource] a\n[resource] b", ansi.Strip(f.RenderWindow(40, 2, 99)), "offset past the top clamps")
	assert.Equal(t, 4, f.Lines(40))
}

// TestEntryHeader_BoldsOnlyTheKind checks the kind label is styled and the
// title is plain text, and that a narrow width still cuts the header.
func TestEntryHeader_BoldsOnlyTheKind(t *testing.T) {
	out := entryHeader(Entry{Kind: KindAction, Title: "read_file config.yaml"}, 40)
	assert.Equal(t, "[action] read_file config.yaml", ansi.Strip(out))
	assert.Contains(t, out, "\x1b[1m[action]", "the kind is bold")
	assert.NotContains(t, out, "\x1b[1m read_file", "the title is not bold")
	narrow := entryHeader(Entry{Kind: KindAction, Title: "read_file config.yaml"}, 15)
	assert.Equal(t, "[action] read_…", ansi.Strip(narrow))
}
