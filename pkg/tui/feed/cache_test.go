package feed

import (
	"strings"
	"sync"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestTruncate_CutsInOnePassWithAnEllipsis(t *testing.T) {
	long := strings.Repeat("word ", 2000)
	got := truncate(long, 40)
	assert.Equal(t, 40, ansi.StringWidth(got))
	assert.True(t, strings.HasSuffix(got, "…"))
	assert.Equal(t, "short", truncate("short", 40))
}

func TestLines_RendersOnlyNewEntriesPerWidth(t *testing.T) {
	var f Feed
	f.Add(Entry{Kind: KindAction, Title: "a", Body: "one\ntwo"})
	assert.Equal(t, 3, f.Lines(30))
	assert.Equal(t, 3, f.Lines(31), "a second width has its own cache")
	f.Add(Entry{Kind: KindResource, Title: "b"})
	assert.Equal(t, 4, f.Lines(30), "a later entry is added to the cached lines")
	assert.Equal(t, 4, f.Lines(31))
	assert.Contains(t, f.RenderWindow(30, 10, 0), "b")
	for w := 32; w < 40; w++ {
		f.Lines(w)
	}
	assert.LessOrEqual(t, len(f.rendered), maxRenderedWidths, "old widths are dropped")
}

// TestFeed_AddWhileTheViewReads checks the agent adding entries while the view
// lays the feed out is safe (run with -race).
func TestFeed_AddWhileTheViewReads(t *testing.T) {
	var f Feed
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 200 {
			f.Add(Entry{Kind: KindAction, Title: "t", Body: "x"})
		}
	})
	for range 200 {
		f.Lines(30)
		_ = f.RenderWindow(30, 5, 0)
		_ = f.Version()
	}
	wg.Wait()
	assert.Equal(t, 200, f.Len())
	assert.Equal(t, 400, f.Lines(30))
}
