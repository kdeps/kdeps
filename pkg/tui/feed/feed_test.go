package feed

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestRender_ShowsEntriesInOrderNewestAtTheBottom(t *testing.T) {
	var f Feed
	f.Add(Entry{Kind: KindResource, Title: "fetch.yaml"})
	f.Add(Entry{Kind: KindAction, Title: "bash_exec", Body: "exit 0"})
	out := f.Render(40, 10)
	assert.Equal(t, "[resource] fetch.yaml\n[action] bash_exec\n  exit 0", out)
}

func TestRender_KeepsTheNewestWhenItDoesNotFit(t *testing.T) {
	var f Feed
	for _, name := range []string{"a", "b", "c", "d"} {
		f.Add(Entry{Kind: KindResource, Title: name})
	}
	out := f.Render(40, 2)
	assert.Equal(t, "[resource] c\n[resource] d", out)
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
