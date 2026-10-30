// Package screen is the full-screen REPL: it sizes the layout to the terminal,
// draws the main pane, the feed and the two bottom boxes, and toggles the feed.
package screen

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
	"github.com/kdeps/kdeps/v2/pkg/tui/layout"
)

// toggleFeedKey shows or hides the right-hand feed.
const toggleFeedKey = "ctrl+f"

// feedBorder is the cells the feed's rounded border takes on each axis.
const feedBorder = 2

// Model is the full-screen view.
type Model struct {
	width, height int
	feedOpen      bool
	Main          string
	Status        string
	Prompt        string
	Feed          *feed.Feed
}

// New returns a screen with the feed open.
func New(f *feed.Feed) Model {
	if f == nil {
		f = &feed.Feed{}
	}
	return Model{feedOpen: true, Feed: f}
}

// Init has no startup command.
func (m Model) Init() tea.Cmd { return nil }

// Update sizes the screen on a resize and toggles the feed on its key.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case tea.KeyMsg:
		if v.String() == toggleFeedKey {
			m.feedOpen = !m.feedOpen
		}
	}
	return m, nil
}

// FeedOpen reports whether the feed is shown.
func (m Model) FeedOpen() bool { return m.feedOpen }

// View draws the frame at the current terminal size. Before the first resize
// it draws nothing.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	feedText := ""
	if m.Feed != nil && frame.Feed.W > 0 {
		feedText = m.Feed.Render(frame.Feed.W-feedBorder, frame.Feed.H-feedBorder)
	}
	return layout.Render(frame, m.Main, feedText, m.Status, m.Prompt)
}
