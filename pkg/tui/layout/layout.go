// Package layout places the full-screen REPL: a main pane, an optional right
// feed, and two fixed boxes at the bottom (status, then prompt). The bottom
// boxes keep their height; the main pane and the feed take what is left.
package layout

// Rect is a box on the screen, in cells.
type Rect struct{ X, Y, W, H int }

// Frame is where each box goes for one terminal size.
type Frame struct {
	Header Rect // one row above the boxes
	Main   Rect
	Feed   Rect // zero when the feed is closed
	Status Rect
	Prompt Rect
}

const (
	headerH   = 1 // the header row above the boxes
	statusH   = 3 // one line of text inside a bordered box
	promptH   = 3
	minFeedW  = 30 // a feed narrower than this is not worth showing
	feedShare = 3  // the feed takes one third of the width
	minMainW  = 20
)

// Compute returns the boxes for a width x height terminal. The feed is shown
// only when feedOpen is set and the terminal is wide enough for both panes.
func Compute(width, height int, feedOpen bool) Frame {
	if width < 1 {
		width = 1
	}
	if height < headerH+statusH+promptH+1 {
		height = headerH + statusH + promptH + 1
	}
	mainH := height - headerH - statusH - promptH
	f := Frame{
		Header: Rect{X: 0, Y: 0, W: width, H: headerH},
		Status: Rect{X: 0, Y: headerH + mainH, W: width, H: statusH},
		Prompt: Rect{X: 0, Y: headerH + mainH + statusH, W: width, H: promptH},
	}
	feedW := width / feedShare
	if feedOpen && feedW >= minFeedW && width-feedW >= minMainW {
		f.Main = Rect{X: 0, Y: headerH, W: width - feedW, H: mainH}
		f.Feed = Rect{X: width - feedW, Y: headerH, W: feedW, H: mainH}
		return f
	}
	f.Main = Rect{X: 0, Y: headerH, W: width, H: mainH}
	return f
}
