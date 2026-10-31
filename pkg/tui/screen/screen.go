// Package screen is the full-screen REPL: it sizes the layout to the terminal,
// draws the main pane, the feed and the two bottom boxes, and toggles the feed.
package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

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

// layoutBottomRows is the height of the status and prompt boxes together.
const layoutBottomRows = 6

// wheelStep is how many rows one mouse-wheel notch scrolls.
const wheelStep = 3

// Model is the full-screen view.
type Model struct {
	width, height int
	feedOpen      bool
	main          []string
	mainOff       int  // rows scrolled up from the newest main line
	feedOff       int  // rows scrolled up from the newest feed row
	focusFeed     bool // arrow keys scroll the feed instead of the main pane
	Status        string
	input         []rune
	cursor        int      // rune index of the prompt cursor
	history       []string // submitted lines, oldest first
	histAt        int      // history position while recalling; len(history) when not recalling
	histDraft     string   // unsent prompt text kept while recalling
	Feed          *feed.Feed
	// OnSubmit runs one submitted line. It runs off the UI loop.
	OnSubmit func(line string)
	// Complete lists the slash commands that start with a prefix. Tab completes
	// a slash command when the prompt starts with "/". Nil disables completion.
	Complete    func(prefix string) []string
	overlay     Overlay
	overlayDone func(Overlay) tea.Cmd
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
	if open, ok := msg.(OpenOverlayMsg); ok {
		return m.WithOverlay(open.Overlay, open.Done), nil
	}
	if m.overlay != nil {
		// The overlay gets every message, resizes included.
		if ws, ok := msg.(tea.WindowSizeMsg); ok {
			m.width, m.height = ws.Width, ws.Height
		}
		return m.updateOverlay(msg)
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
	case OutputMsg:
		m = m.appendOutput(v.Text)
	case StatusTextMsg:
		m.Status = v.Text
	case tea.MouseWheelMsg:
		return m.wheel(v), nil
	case tea.KeyPressMsg:
		return m.key(v)
	case tea.PasteMsg:
		// The prompt is one line: a pasted newline must not submit it.
		return m.insertText(strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(v.Content)), nil
	}
	return m, nil
}

// pageRows is how many rows a page key scrolls: the height of the boxes.
func (m Model) pageRows() int {
	return max(m.height-layoutBottomRows, 1)
}

// scroll moves the focused box by delta rows (positive is up, into history).
func (m Model) scroll(delta int) Model {
	if m.focusFeed {
		m.feedOff = max(m.feedOff+delta, 0)
	} else {
		m.mainOff = max(m.mainOff+delta, 0)
	}
	return m
}

// wheel scrolls the box under the mouse pointer.
func (m Model) wheel(v tea.MouseWheelMsg) Model {
	var step int
	switch v.Button {
	case tea.MouseWheelUp:
		step = wheelStep
	case tea.MouseWheelDown:
		step = -wheelStep
	default:
		return m
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	prev := m.focusFeed
	m.focusFeed = frame.Feed.W > 0 && v.X >= frame.Feed.X
	m = m.scroll(step)
	m.focusFeed = prev
	return m
}

// key handles one keypress: the feed toggle, prompt editing, and submit.
func (m Model) key(v tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if next, handled := m.editKey(v.String()); handled {
		return next, nil
	}
	switch v.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+f":
		m.feedOpen = !m.feedOpen
	case "tab":
		m.focusFeed = !m.focusFeed
	case "up":
		return m.scroll(1), nil
	case "down":
		return m.scroll(-1), nil
	case "pgup":
		return m.scroll(m.pageRows()), nil
	case "pgdown":
		return m.scroll(-m.pageRows()), nil
	case "end":
		if m.focusFeed {
			m.feedOff = 0
		} else {
			m.mainOff = 0
		}
	case "enter":
		var line string
		m, line = m.submitPrompt()
		if line != "" && m.OnSubmit != nil {
			submit := m.OnSubmit
			return m, func() tea.Msg {
				submit(line)
				return nil
			}
		}
	default:
		if v.Text != "" {
			m = m.insertText(v.Text)
		}
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

// window returns height rows of lines ending offset rows above the newest, so
// offset 0 is the tail. An offset past the top clamps to the top.
func window(lines []string, height, offset int) string {
	if height < 1 || len(lines) == 0 {
		return ""
	}
	if top := len(lines) - height; offset > top {
		offset = max(top, 0)
	}
	end := len(lines) - max(offset, 0)
	return strings.Join(lines[max(end-height, 0):end], "\n")
}

// FeedOpen reports whether the feed is shown.
func (m Model) FeedOpen() bool { return m.feedOpen }

// View draws the frame on the alternate screen with mouse cell motion, so the
// wheel scrolls the boxes and the shell history stays out of sight.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion
	return v
}

// render draws the frame at the current terminal size. Before the first resize
// it draws nothing.
func (m Model) render() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	mainText := window(wrapRows(m.main, frame.Main.W-feedBorder), frame.Main.H-feedBorder, m.mainOff)
	if m.overlay != nil {
		mainText = m.overlay.View().Content
	}
	feedText := ""
	if m.Feed != nil && frame.Feed.W > 0 {
		feedText = m.Feed.RenderWindow(frame.Feed.W-feedBorder, frame.Feed.H-feedBorder, m.feedOff)
	}
	return layout.Render(frame, mainText, feedText, m.Status, "> "+m.promptText())
}

// wrapRows splits each line into the rows it takes at width cells, so the
// scroll offset counts screen rows and a long line scrolls in full.
func wrapRows(lines []string, width int) []string {
	if width < 1 {
		return lines
	}
	rows := make([]string, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
	}
	return rows
}
