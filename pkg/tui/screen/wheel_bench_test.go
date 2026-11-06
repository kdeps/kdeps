package screen

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

// BenchmarkWheel_BusyFeed is one wheel event with a long session on screen:
// the path that pinned a core in a live session.
func BenchmarkWheel_BusyFeed(b *testing.B) {
	f := &feed.Feed{}
	body := strings.Repeat(strings.Repeat("output text ", 30)+"\n", 20)
	for range 300 {
		f.Add(feed.Entry{Kind: feed.KindAction, Title: "bash_exec go test ./...", Body: body})
	}
	m := New(f)
	next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	m = next.(Model)
	next, _ = m.Update(OutputMsg{Text: strings.Repeat("a reply line\n", 2000)})
	m = next.(Model)
	b.ResetTimer()
	for range b.N {
		next, _ = m.Update(tea.MouseWheelMsg{X: 150, Y: 10, Button: tea.MouseWheelUp})
		m = next.(Model)
		_ = m.render()
	}
}
