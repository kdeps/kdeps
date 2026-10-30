// Package screen is the full-screen REPL: it sizes the layout to the terminal,
// draws the main pane, the feed and the two bottom boxes, and toggles the feed.
package screen

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
	"github.com/kdeps/kdeps/v2/pkg/tui/layout"
)

// OutputMsg adds text to the main pane. Each line of text is one row.
type OutputMsg struct{ Text string }

// StatusTextMsg replaces the text of the status box.
type StatusTextMsg struct{ Text string }

// maxMainLines bounds the main pane's history.
const maxMainLines = 2000

// feedBorder is the cells the feed's rounded border takes on each axis.
const feedBorder = 2

// Model is the full-screen view.
type Model struct {
	width, height int
	feedOpen      bool
	main          []string
	Status        string
	input         []rune
	Feed          *feed.Feed
	// OnSubmit runs one submitted line. It runs off the UI loop.
	OnSubmit func(line string)
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

// Update sizes the screen on a resize, toggles the feed on its key, edits the
// prompt, and appends output to the main pane.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case OutputMsg:
		m = m.appendOutput(v.Text)
	case StatusTextMsg:
		m.Status = v.Text
	case tea.KeyMsg:
		return m.key(v)
	}
	return m, nil
}

// key handles one keypress: the feed toggle, prompt editing, and submit.
func (m Model) key(v tea.KeyMsg) (tea.Model, tea.Cmd) {
	//nolint:exhaustive // only these keys have an action; the rest are ignored
	switch v.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyCtrlF:
		m.feedOpen = !m.feedOpen
	case tea.KeyBackspace:
		if n := len(m.input); n > 0 {
			m.input = m.input[:n-1]
		}
	case tea.KeyEnter:
		line := strings.TrimSpace(string(m.input))
		m.input = nil
		if line != "" && m.OnSubmit != nil {
			submit := m.OnSubmit
			return m, func() tea.Msg {
				submit(line)
				return nil
			}
		}
	case tea.KeySpace:
		m.input = append(m.input, ' ')
	case tea.KeyRunes:
		m.input = append(m.input, v.Runes...)
	}
	return m, nil
}

// appendOutput adds text to the main pane, one row per line.
func (m Model) appendOutput(text string) Model {
	m.main = append(m.main, strings.Split(strings.TrimRight(text, "\n"), "\n")...)
	if extra := len(m.main) - maxMainLines; extra > 0 {
		m.main = m.main[extra:]
	}
	return m
}

// Prompt is the text the user has typed so far.
func (m Model) Prompt() string { return string(m.input) }

// mainText is the last rows of output that fit the main pane.
func (m Model) mainText(rows int) string {
	if rows < 1 || len(m.main) == 0 {
		return ""
	}
	start := len(m.main) - rows
	if start < 0 {
		start = 0
	}
	return strings.Join(m.main[start:], "\n")
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
	mainText := m.mainText(frame.Main.H - feedBorder)
	feedText := ""
	if m.Feed != nil && frame.Feed.W > 0 {
		feedText = m.Feed.Render(frame.Feed.W-feedBorder, frame.Feed.H-feedBorder)
	}
	return layout.Render(frame, mainText, feedText, m.Status, "> "+m.Prompt())
}
