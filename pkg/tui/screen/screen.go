// Package screen is the full-screen REPL: it sizes the layout to the terminal,
// draws the main pane, the feed and the two bottom boxes, and toggles the feed.
package screen

import (
	"math"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/charmbracelet/x/ansi"

	"github.com/kdeps/kdeps/v2/pkg/tui"
	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
	"github.com/kdeps/kdeps/v2/pkg/tui/layout"
)

// OutputMsg adds text to the main pane. Each line of text is one row.
type OutputMsg struct{ Text string }

// StatusTextMsg replaces the text of the status box.
type StatusTextMsg struct{ Text string }

// ModelMsg sets the model name shown as a chip in the header row.
type ModelMsg struct{ Name string }

// RedrawMsg asks for a redraw with no other change, for example after the feed
// changed off the UI loop.
type RedrawMsg struct{}

// PillMsg sets one pill's value. An empty value hides the pill.
type PillMsg struct{ Name, Value string }

// maxMainLines bounds the main pane's history.
const maxMainLines = 2000

// foldMinLines is the shortest output that can fold. A one-line output stays.
const foldMinLines = 2

// feedBorder is the rows the boxes' rounded borders take on each axis.
const feedBorder = 2

// headerRule is the diagonal rule drawn between the product name and the title.
const headerRule = "╱╱╱"

// layoutBottomRows is the height of the status and prompt boxes together.
const layoutBottomRows = 6

// wheelStep is how many rows one mouse-wheel notch scrolls.
const wheelStep = 3

// halfPageDivisor splits a page scroll in two.
const halfPageDivisor = 2

// Model is the full-screen view.
// frameCache holds the last drawn frame. The model copies share it, so a
// frame is reused until the key below changes.
type frameCache struct {
	valid bool
	key   frameKey
	text  string
}

// frameKey is everything a frame depends on, except an open overlay (which
// has no key and is never cached). profile is the terminal color profile:
// a frame drawn for one profile is not served for another. model, title
// and pills are the header, so a chip change is not served from the last frame.
type frameKey struct {
	width, height, mainOff, feedOff int
	mainVer, feedVer                int
	feedOpen, focusFeed, pillsOff   bool
	status, prompt                  string
	model, title, pills             string
	profile                         colorprofile.Profile
}

type Model struct {
	cache         *frameCache
	mainVer       int
	width, height int
	feedOpen      bool
	main          []string
	mainOff       int    // rows scrolled up from the newest main line
	outEnds       []int  // exclusive end index in main after each output
	outFolded     []bool // a folded output shows its first line only
	feedOff       int    // rows scrolled up from the newest feed row
	focusFeed     bool   // arrow keys scroll the feed instead of the main pane
	Status        string
	// Title is shown in the header row, after the product name (for example the folder).
	Title string
	// HistoryMax bounds the submitted lines ctrl+p and ctrl+n recall; 0 keeps all.
	HistoryMax int
	// Model is the active model name, shown as a chip in the header row.
	Model     string
	pills     map[string]string
	pillsOff  bool // ctrl+t hides the header pills; the model chip stays
	input     []rune
	cursor    int      // rune index of the prompt cursor
	anchor    int      // the other end of a selection; equal to cursor when none
	selecting bool     // a left-button drag in the prompt is extending the selection
	killed    []rune   // text the last kill removed; ctrl+y inserts it
	killing   bool     // the previous key was a kill, so the next kill joins it
	history   []string // submitted lines, oldest first
	histAt    int      // history position while recalling; len(history) when not recalling
	histDraft string   // unsent prompt text kept while recalling
	Feed      *feed.Feed
	// OnSubmit runs one submitted line. It runs off the UI loop.
	OnSubmit func(line string)
	// launchEditor opens the prompt in an external editor. Nil uses the default.
	launchEditor func(text string) tea.Cmd
	// Complete lists the slash commands that start with a prefix. Tab completes
	// a slash command when the prompt starts with "/". Nil disables completion.
	Complete func(prefix string) []string
	// remember runs off the UI loop when a submitted line is new.
	remember    func(string)
	profile     colorprofile.Profile
	overlay     Overlay
	overlayDone func(Overlay) tea.Cmd
}

// New returns a screen with the feed open.
func New(f *feed.Feed) Model {
	if f == nil {
		f = &feed.Feed{}
	}
	return Model{feedOpen: true, Feed: f, cache: &frameCache{}}
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
			m = m.clampScroll()
		}
		return m.updateOverlay(msg)
	}
	switch v := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = v.Width, v.Height
		m = m.clampScroll()
	case OutputMsg:
		m = m.appendOutput(v.Text)
	case StatusTextMsg:
		m.Status = v.Text
	case ModelMsg:
		m.Model = v.Name
	case RedrawMsg:
		// Nothing to change: the update itself makes the view draw again.
	case PillMsg:
		m.pills = withPill(m.pills, v.Name, v.Value)
	case tea.ColorProfileMsg:
		m.profile = v.Profile
	case tea.MouseWheelMsg:
		return m.wheel(v), nil
	case tea.MouseClickMsg:
		return m.click(v), nil
	case tea.MouseMotionMsg:
		return m.drag(tea.Mouse(v)), nil
	case tea.MouseReleaseMsg:
		m.selecting = false
	case tea.KeyPressMsg:
		return m.key(v)
	case tea.PasteMsg:
		// The prompt is one line: a pasted newline must not submit it.
		return m.insertText(strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(v.Content)), nil
	case editorResultMsg:
		if !v.ok {
			return m, nil
		}
		before := string(m.input)
		return m.setPrompt(v.text).keepEdit(before), nil
	}
	return m, nil
}

// pageRows is how many rows a page key scrolls: the height of the boxes.
func (m Model) pageRows() int {
	return max(m.height-layoutBottomRows, 1)
}

// halfRows is half a page, and at least one row, so the move keeps some of
// the rows that were already on screen.
func (m Model) halfRows() int {
	return max(m.pageRows()/halfPageDivisor, 1)
}

// scroll moves the focused box by delta rows (positive is up, into history).
// The offset stops at the oldest row, so a later move in the other direction
// leaves that row instead of repeating it.
func (m Model) scroll(delta int) Model {
	if m.focusFeed {
		m.feedOff += delta
	} else {
		m.mainOff += delta
	}
	return m.clampScroll()
}

// jumpOldest moves the focused box to its oldest row.
func (m Model) jumpOldest() Model {
	if m.focusFeed {
		m.feedOff = math.MaxInt
	} else {
		m.mainOff = math.MaxInt
	}
	return m.clampScroll()
}

// clampScroll limits both offsets to the rows each box can actually show.
func (m Model) clampScroll() Model {
	if m.width == 0 || m.height == 0 {
		return m
	}
	mainH, mainN := m.mainSpan()
	m.mainOff = clampOffset(m.mainOff, mainN, mainH)
	if feedH, feedN, ok := m.feedSpan(); ok {
		m.feedOff = clampOffset(m.feedOff, feedN, feedH)
	}
	return m
}

// mainSpan is the main pane's content height and how many wrapped rows it holds.
func (m Model) mainSpan() (int, int) {
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	return frame.Main.H - feedBorder, len(wrapRows(m.shownLines(), layout.InnerWidth(frame.Main.W)))
}

// feedSpan is the feed's content height and row count. The bool is false when
// the feed is hidden, so a closed feed keeps its offset for the next open.
func (m Model) feedSpan() (int, int, bool) {
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	if m.Feed == nil || frame.Feed.W == 0 {
		return 0, 0, false
	}
	return frame.Feed.H - feedBorder - 1, m.Feed.Lines(layout.InnerWidth(frame.Feed.W)), true
}

// clampOffset limits off to the range the window can show.
func clampOffset(off, lines, height int) int {
	if off < 0 || height < 1 {
		return 0
	}
	if top := lines - height; off > top {
		if top < 0 {
			return 0
		}
		return top
	}
	return off
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

// click focuses the pane under a left click. A click in the prompt moves the
// cursor instead, so the arrow keys keep scrolling whichever pane was focused.
func (m Model) click(v tea.MouseClickMsg) Model {
	if v.Button != tea.MouseLeft || m.width == 0 || m.height == 0 {
		return m
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	switch {
	case frame.Feed.W > 0 && inRect(frame.Feed, v.X, v.Y):
		m.focusFeed = true
		m.selecting = false
		m.anchor = m.cursor
	case inRect(frame.Main, v.X, v.Y):
		m.focusFeed = false
		m.selecting = false
		m.anchor = m.cursor
	case v.Y == frame.Prompt.Y+1 && inRect(frame.Prompt, v.X, v.Y):
		m = m.placeCursor(v.X - (frame.Prompt.X + 1 + layout.BoxPadX))
		m.anchor = m.cursor
		m.selecting = true
	}
	return m
}

// drag extends a prompt selection while the left button stays down.
func (m Model) drag(v tea.Mouse) Model {
	if !m.selecting || v.Button != tea.MouseLeft || m.width == 0 || m.height == 0 {
		return m
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	if v.Y != frame.Prompt.Y+1 || !inRect(frame.Prompt, v.X, v.Y) {
		return m
	}
	return m.placeCursor(v.X - (frame.Prompt.X + 1 + layout.BoxPadX))
}

// inRect reports whether the cell is inside the box.
func inRect(r layout.Rect, x, y int) bool {
	return r.W > 0 && r.H > 0 && x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

// placeCursor moves the cursor to the prompt cell under rel. rel is measured
// from the left edge of the prompt text, which starts with the "> " marker.
// The drawn bar is one cell, so a click on a letter lands on that letter.
func (m Model) placeCursor(rel int) Model {
	m.killing = false
	prefix := lipgloss.Width(styled(tui.CurrentPickerColors().Accent).Render(">")) + 1
	cell := rel - prefix
	if cell < 0 {
		m.cursor = 0
		return m
	}
	pos := 0
	for i := 0; i <= len(m.input); i++ {
		if i == m.cursor {
			if cell == pos {
				m.cursor = i
				return m
			}
			pos++
		}
		if i == len(m.input) {
			break
		}
		w := lipgloss.Width(string(m.input[i]))
		if w < 1 {
			w = 1
		}
		if cell < pos+w {
			m.cursor = i
			return m
		}
		pos += w
	}
	m.cursor = len(m.input)
	return m
}

// key handles one keypress: the feed toggle, prompt editing, and submit.
func (m Model) key(v tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if v.Text == "?" && len(m.input) == 0 {
		return m.WithOverlay(helpOverlay{commands: m.commandList()}, nil), nil
	}
	if next, cmd, handled := m.commandKey(v); handled {
		return next, cmd
	}
	if next, handled := m.selectKey(v); handled {
		return next, nil
	}
	if next, handled := m.wordKey(v); handled {
		return next, nil
	}
	if next, handled := m.editKey(v.String()); handled {
		return next, nil
	}
	if next, handled := m.pillKey(v); handled {
		return next, nil
	}
	// Ctrl+Up and Ctrl+Down scroll half a page. u and d still type.
	if v.Mod.Contains(tea.ModCtrl) && (v.Code == tea.KeyUp || v.Code == tea.KeyDown) {
		delta := m.halfRows()
		if v.Code == tea.KeyDown {
			delta = -delta
		}
		return m.scroll(delta), nil
	}
	if next, handled := m.foldKey(v); handled {
		return next, nil
	}
	switch v.String() {
	case "ctrl+c":
		// Ctrl+C clears a typed prompt first, so a half-written line is not
		// lost to a stray keypress; with an empty prompt it quits.
		if len(m.input) > 0 {
			return m.setPrompt(""), nil
		}
		return m, tea.Quit
	case "ctrl+f":
		m.feedOpen = !m.feedOpen
		m = m.clampScroll()
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
	case "home":
		return m.jumpOldest(), nil
	case "ctrl+end":
		// End follows whichever pane is focused. Ctrl+End always follows
		// the main pane, including while the feed has the keys.
		m.mainOff = 0
		return m, nil
	case "end":
		if m.focusFeed {
			m.feedOff = 0
		} else {
			m.mainOff = 0
		}
	case "enter":
		return m.submitLine()
	default:
		if v.Text != "" {
			m = m.insertText(v.Text)
		}
	}
	return m, nil
}

// submitLine records the prompt, shows it in the message area, and runs the
// submit and history callbacks off the UI loop.
func (m Model) submitLine() (tea.Model, tea.Cmd) {
	line, recorded := "", false
	m, line, recorded = m.submitPrompt()
	if line == "" {
		return m, nil
	}
	m.mainOff = 0
	m = m.appendOutput(userLine(line))
	submit, remember := m.OnSubmit, m.remember
	if submit == nil && (!recorded || remember == nil) {
		return m, nil
	}
	return m, func() tea.Msg {
		if recorded && remember != nil {
			remember(line)
		}
		if submit != nil {
			submit(line)
		}
		return nil
	}
}

// appendOutput adds text to the main pane, one row per line. While the pane
// is scrolled away from the newest row, the new rows stay below the window.
func (m Model) appendOutput(text string) Model {
	stuck := m.mainOff > 0
	before := 0
	if stuck {
		_, before = m.mainSpan()
	}
	m.mainVer++
	m.main = append(m.main, strings.Split(strings.TrimRight(text, "\n"), "\n")...)
	m.outEnds = append(m.outEnds, len(m.main))
	m.outFolded = append(m.outFolded, false)
	if extra := len(m.main) - maxMainLines; extra > 0 {
		m.main = m.main[extra:]
		m.outEnds, m.outFolded = trimOutputs(m.outEnds, m.outFolded, extra)
	}
	if stuck {
		_, after := m.mainSpan()
		m.mainOff += after - before
	}
	return m.clampScroll()
}

// commandKey handles ctrl chords that must not type their letter.
// Ctrl+Y yanks, Ctrl+D deletes forward, Ctrl+O opens the editor, and
// Ctrl+Z suspends the view.
func (m Model) commandKey(v tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	if !v.Mod.Contains(tea.ModCtrl) {
		return m, nil, false
	}
	switch {
	case v.Code == 'y' || v.String() == "ctrl+y":
		return m.yank(), nil, true
	case (v.Code == 'd' || v.String() == "ctrl+d") && !v.Mod.Contains(tea.ModAlt) && !v.Mod.Contains(tea.ModShift):
		// Ctrl+D deletes the character under the cursor. An empty prompt
		// quits, the same way the line editor treats end of input.
		if len(m.input) == 0 {
			return m, tea.Quit, true
		}
		m = m.deleteAfter()
		m.killing = false
		return m, nil, true
	case v.Code == 'o' || v.String() == "ctrl+o":
		return m, m.openEditor(), true
	case (v.Code == 'z' || v.String() == "ctrl+z") && !v.Mod.Contains(tea.ModAlt) && !v.Mod.Contains(tea.ModShift):
		return m, tea.Suspend, true
	default:
		return m, nil, false
	}
}

// foldKey toggles the output at the bottom of the main pane. Space still types.
func (m Model) foldKey(v tea.KeyPressMsg) (Model, bool) {
	if !v.Mod.Contains(tea.ModCtrl) || v.Mod.Contains(tea.ModAlt) || v.Mod.Contains(tea.ModShift) {
		return m, false
	}
	if v.Code != ' ' && v.Code != tea.KeySpace && v.String() != "ctrl+space" {
		return m, false
	}
	return m.toggleFold(), true
}

// toggleFold folds or opens the output at the bottom of the main pane.
// A one-line output stays open. The prompt is left as it is.
func (m Model) toggleFold() Model {
	idx := m.outputAtBottom()
	if idx < 0 || m.outputLines(idx) < foldMinLines {
		return m
	}
	m.outFolded[idx] = !m.outFolded[idx]
	m.mainVer++
	return m.clampScroll()
}

// outputAtBottom is the output that owns the newest visible main row.
func (m Model) outputAtBottom() int {
	if len(m.outEnds) == 0 {
		return -1
	}
	if m.width == 0 || m.height == 0 {
		return len(m.outEnds) - 1
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	width := layout.InnerWidth(frame.Main.W)
	bottom := len(wrapRows(m.shownLines(), width)) - 1 - max(m.mainOff, 0)
	if bottom < 0 {
		bottom = 0
	}
	start, acc := 0, 0
	for i, end := range m.outEnds {
		chunk := m.outputChunk(i, start, end)
		n := len(wrapRows(chunk, width))
		if n < 1 {
			n = 1
		}
		if acc+n > bottom {
			return i
		}
		acc += n
		start = end
	}
	return len(m.outEnds) - 1
}

// outputLines is how many stored lines an output has, before folding.
func (m Model) outputLines(i int) int {
	start := 0
	if i > 0 {
		start = m.outEnds[i-1]
	}
	end := m.outEnds[i]
	if end > len(m.main) {
		end = len(m.main)
	}
	if start > end {
		return 0
	}
	return end - start
}

// outputChunk is the lines an output contributes. A folded output keeps its first line.
func (m Model) outputChunk(i, start, end int) []string {
	if end > len(m.main) {
		end = len(m.main)
	}
	if start > end {
		start = end
	}
	chunk := m.main[start:end]
	if i < len(m.outFolded) && m.outFolded[i] && len(chunk) > 1 {
		chunk = chunk[:1]
	}
	return chunk
}

// shownLines is the main pane with folded outputs reduced to their first line.
func (m Model) shownLines() []string {
	if len(m.outEnds) == 0 {
		return m.main
	}
	lines := make([]string, 0, len(m.main))
	start := 0
	for i, end := range m.outEnds {
		lines = append(lines, m.outputChunk(i, start, end)...)
		if end > start {
			start = end
		}
	}
	if start < len(m.main) {
		lines = append(lines, m.main[start:]...)
	}
	return lines
}

// trimOutputs drops outputs that scrolled off the front and shifts the rest.
func trimOutputs(ends []int, folded []bool, extra int) ([]int, []bool) {
	keptEnds := make([]int, 0, len(ends))
	keptFolded := make([]bool, 0, len(folded))
	for i, end := range ends {
		if end <= extra {
			continue
		}
		keptEnds = append(keptEnds, end-extra)
		fold := false
		if i < len(folded) {
			fold = folded[i]
		}
		keptFolded = append(keptFolded, fold)
	}
	return keptEnds, keptFolded
}

// copyVisible stores the main pane's visible rows for Ctrl+Y. The rows become
// one line. An empty pane leaves the yank buffer alone, and the prompt stays.
func (m Model) copyVisible() Model {
	text := strings.Join(strings.Fields(m.visibleMain()), " ")
	if text == "" {
		return m
	}
	m.killed = []rune(text)
	m.killing = false
	return m
}

// visibleMain is the main pane text on screen, one row per line.
func (m Model) visibleMain() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	rows := wrapRows(m.shownLines(), layout.InnerWidth(frame.Main.W))
	return window(rows, frame.Main.H-feedBorder, m.mainOff)
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
	if m.overlay != nil || m.cache == nil {
		return m.renderFrame()
	}
	key := m.frameKey()
	if m.cache.valid && m.cache.key == key {
		return m.cache.text
	}
	text := m.renderFrame()
	m.cache.valid, m.cache.key, m.cache.text = true, key, text
	return text
}

// frameKey is the state a frame is drawn from.
func (m Model) frameKey() frameKey {
	feedVer := -1
	if m.Feed != nil {
		feedVer = m.Feed.Version()
	}
	return frameKey{
		width: m.width, height: m.height, mainOff: m.mainOff, feedOff: m.feedOff,
		mainVer: m.mainVer, feedVer: feedVer, feedOpen: m.feedOpen, focusFeed: m.focusFeed,
		pillsOff: m.pillsOff,
		status:   m.Status, prompt: m.promptText(),
		model: m.Model, title: m.Title, pills: m.pillsKey(), profile: m.profile,
	}
}

// renderFrame draws the frame at the current terminal size.
func (m Model) renderFrame() string {
	if m.width == 0 || m.height == 0 {
		return ""
	}
	frame := layout.Compute(m.width, m.height, m.feedOpen)
	mainText := window(wrapRows(m.shownLines(), layout.InnerWidth(frame.Main.W)), frame.Main.H-feedBorder, m.mainOff)
	if m.overlay != nil {
		mainText = m.overlay.View().Content
	}
	feedText := ""
	if m.Feed != nil && frame.Feed.W > 0 {
		// The side panel has a bold title row, as the reference sidebar does.
		title := styled(tui.CurrentPickerColors().Accent).Render("activity")
		window := m.Feed.RenderWindow(layout.InnerWidth(frame.Feed.W), frame.Feed.H-feedBorder-1, m.feedOff)
		feedText = title + "\n" + window
	}
	caret := styled(tui.CurrentPickerColors().Accent).Render(">")
	return layout.Render(frame, m.headerText(), mainText, feedText, m.statusText(), caret+" "+m.promptText())
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

// statusKeys is the key help shown after the counters in the status row.
const statusKeys = "? help   tab complete   ctrl+f feed   ctrl+c quit"

// notFollowing is the status hint while the main pane is off the newest row.
const notFollowing = "not following"

// statusText is the status row: the counters, then the key help, dimmed.
// A pane scrolled into older rows says so, until End or Down returns to the tail.
func (m Model) statusText() string {
	keys := lipgloss.NewStyle().Faint(true).Render(statusKeys)
	if m.mainOff > 0 {
		keys = lipgloss.NewStyle().Faint(true).Render(notFollowing) + "   " + keys
	}
	if m.Status == "" {
		return keys
	}
	return m.Status + "   " + keys
}

// headerText is the header row: the product name, a dim diagonal rule, and the title.
func (m Model) headerText() string {
	name := styled(tui.CurrentPickerColors().Accent).Render("kdeps")
	rule := lipgloss.NewStyle().Faint(true).Render(headerRule)
	title := lipgloss.NewStyle().Faint(true).Render(m.Title)
	chip := ""
	if m.Model != "" {
		chip = lipgloss.NewStyle().Reverse(true).Render(" "+m.Model+" ") + " " + rule + " "
	}
	return " " + name + " " + rule + " " + chip + title + m.pillsText()
}

// withPill returns pills with name set to value, copying the map.
func withPill(pills map[string]string, name, value string) map[string]string {
	out := make(map[string]string, len(pills)+1)
	for k, v := range pills {
		out[k] = v
	}
	out[name] = value
	return out
}

// pillsKey is the pill values in display order, so the frame key stays stable.
func (m Model) pillsKey() string {
	var parts []string
	for _, name := range []string{"backend", "session", "tool"} {
		if v := m.pills[name]; v != "" {
			parts = append(parts, name+"="+v)
		}
	}
	return strings.Join(parts, "\n")
}

// pillKey hides or shows the header pills. t still types.
func (m Model) pillKey(v tea.KeyPressMsg) (Model, bool) {
	if !v.Mod.Contains(tea.ModCtrl) || v.Mod.Contains(tea.ModAlt) || v.Mod.Contains(tea.ModShift) {
		return m, false
	}
	if v.Code != 't' && v.String() != "ctrl+t" {
		return m, false
	}
	m.pillsOff = !m.pillsOff
	return m, true
}

// pillsText is the pills in one chip: the values that are set, in the order
// backend, session, tool, joined by a dot. A hidden chip keeps its values.
func (m Model) pillsText() string {
	if m.pillsOff {
		return ""
	}
	var parts []string
	for _, name := range []string{"backend", "session", "tool"} {
		if v := m.pills[name]; v != "" {
			parts = append(parts, v)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "   " + lipgloss.NewStyle().Reverse(true).Render(" "+strings.Join(parts, " · ")+" ")
}

// styled is a bold text style in a palette colour.
func styled(hex string) lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(hex))
}

// userLine is a submitted prompt as the message area shows it: a cyan caret
// and the text in bold, so it reads apart from the agent's reply below it.
func userLine(text string) string {
	return styled(tui.CurrentPickerColors().Accent).Render("› " + text)
}
