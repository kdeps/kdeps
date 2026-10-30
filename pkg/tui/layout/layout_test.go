package layout

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompute_BottomBoxesStayFixed(t *testing.T) {
	for _, h := range []int{20, 40, 60} {
		f := Compute(120, h, true)
		assert.Equal(t, 3, f.Status.H)
		assert.Equal(t, 3, f.Prompt.H)
		assert.Equal(t, h-6, f.Main.H, "the main pane and the feed take the height above the bottom boxes")
		assert.Equal(t, h-6, f.Feed.H)
		assert.Equal(t, h-6, f.Status.Y, "status sits right below the main area")
		assert.Equal(t, h-3, f.Prompt.Y, "prompt is the last box")
		assert.Equal(t, h, f.Prompt.Y+f.Prompt.H, "the boxes end at the bottom of the terminal")
	}
}

func TestCompute_FeedTakesOneThirdAndMainTheRest(t *testing.T) {
	f := Compute(120, 40, true)
	assert.Equal(t, 40, f.Feed.W)
	assert.Equal(t, 80, f.Main.W)
	assert.Equal(t, 120, f.Main.W+f.Feed.W, "main and feed fill the width")
	assert.Equal(t, 80, f.Feed.X)
}

func TestCompute_ClosingTheFeedGivesMainTheWholeWidth(t *testing.T) {
	f := Compute(120, 40, false)
	assert.Equal(t, 120, f.Main.W)
	assert.Equal(t, Rect{}, f.Feed)
}

func TestCompute_NarrowTerminalHidesTheFeed(t *testing.T) {
	f := Compute(60, 40, true)
	assert.Equal(t, Rect{}, f.Feed, "a feed under the minimum width is not drawn")
	assert.Equal(t, 60, f.Main.W)
}
