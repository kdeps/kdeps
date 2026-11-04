package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// wordKey moves or deletes by word. Alt+letter is matched on the modifier,
// because the key's text is the letter and would otherwise be typed.
func (m Model) wordKey(v tea.KeyPressMsg) (Model, bool) {
	switch {
	case wordDeleteBack(v):
		return m.deleteWordBefore(), true
	case wordDeleteForward(v):
		return m.deleteWordAfter(), true
	case wordBack(v):
		return m.move(wordLeft(m.input, m.cursor), v.Mod.Contains(tea.ModShift)), true
	case wordForward(v):
		return m.move(wordRight(m.input, m.cursor), v.Mod.Contains(tea.ModShift)), true
	default:
		return m, false
	}
}

func wordBack(v tea.KeyPressMsg) bool {
	if v.Code == tea.KeyLeft && (v.Mod.Contains(tea.ModCtrl) || v.Mod.Contains(tea.ModAlt)) {
		return true
	}
	return v.Mod.Contains(tea.ModAlt) && v.Code == 'b'
}

func wordForward(v tea.KeyPressMsg) bool {
	if v.Code == tea.KeyRight && (v.Mod.Contains(tea.ModCtrl) || v.Mod.Contains(tea.ModAlt)) {
		return true
	}
	return v.Mod.Contains(tea.ModAlt) && v.Code == 'f'
}

func wordDeleteBack(v tea.KeyPressMsg) bool {
	return v.Code == tea.KeyBackspace && (v.Mod.Contains(tea.ModAlt) || v.Mod.Contains(tea.ModCtrl))
}

func wordDeleteForward(v tea.KeyPressMsg) bool {
	if v.Code == tea.KeyDelete && v.Mod.Contains(tea.ModCtrl) {
		return true
	}
	return v.Mod.Contains(tea.ModAlt) && v.Code == 'd'
}

// wordLeft is the cursor after moving to the start of the previous word.
// A word is a run of non-spaces, the same split Ctrl+W uses.
func wordLeft(input []rune, cursor int) int {
	i := cursor
	for i > 0 && input[i-1] == ' ' {
		i--
	}
	for i > 0 && input[i-1] != ' ' {
		i--
	}
	return i
}

// wordRight is the cursor after moving to the end of the next word.
func wordRight(input []rune, cursor int) int {
	i := cursor
	for i < len(input) && input[i] == ' ' {
		i++
	}
	for i < len(input) && input[i] != ' ' {
		i++
	}
	return i
}

// editKey applies a prompt editing key (cursor, delete, kill, history). It
// reports false for keys that are not editing keys.
func (m Model) editKey(name string) (Model, bool) {
	switch name {
	case "ctrl+u":
		if m.hasSel() {
			return m.deleteSel(true), true
		}
		m = m.killText(m.input[:m.cursor], true)
		m.input = m.input[m.cursor:]
		m.cursor = 0
		m.anchor = 0
		return m, true
	case "ctrl+k":
		if m.hasSel() {
			return m.deleteSel(true), true
		}
		m = m.killText(m.input[m.cursor:], false)
		m.input = m.input[:m.cursor]
		m.anchor = m.cursor
		return m, true
	case "ctrl+w":
		return m.deleteWordBefore(), true
	case "left":
		if m.hasSel() {
			lo, _ := m.selBounds()
			m = m.move(lo, false)
			break
		}
		m = m.move(m.cursor-1, false)
	case "right":
		if m.hasSel() {
			_, hi := m.selBounds()
			m = m.move(hi, false)
			break
		}
		m = m.move(m.cursor+1, false)
	case "ctrl+a":
		m = m.move(0, false)
	case "ctrl+e":
		m = m.move(len(m.input), false)
	case "backspace":
		m = m.deleteBefore()
	case "delete":
		m = m.deleteAfter()
	case "ctrl+p":
		m = m.historyBack()
	case "ctrl+n":
		m = m.historyForward()
	case "tab":
		if next, ok := m.completeMention(); ok {
			return next, true
		}
		return m.completeCommand()
	default:
		return m, false
	}
	m.killing = false
	return m, true
}

// commandList is every slash command the completion source knows, or nil.
func (m Model) commandList() []string {
	if m.Complete == nil {
		return nil
	}
	return m.Complete("/")
}

// completeMention completes the @ token at the cursor. A prompt with no @
// token is not handled, so tab can still move focus.
func (m Model) completeMention() (Model, bool) {
	if m.Complete == nil {
		return m, false
	}
	token, start := tokenBeforeCursor(m.input, m.cursor)
	if !strings.HasPrefix(token, "@") {
		return m, false
	}
	return m.completeToken(token, start), true
}

// tokenBeforeCursor is the word ending at the cursor, and where it starts.
func tokenBeforeCursor(input []rune, cursor int) (string, int) {
	if cursor < 0 {
		cursor = 0
	}
	if cursor > len(input) {
		cursor = len(input)
	}
	start := cursor
	for start > 0 && input[start-1] != ' ' && input[start-1] != '\t' {
		start--
	}
	return string(input[start:cursor]), start
}

// completeToken replaces the token with the completion, the same way a slash
// command completes: one match fills it, a shared prefix extends it, and
// anything else is listed.
func (m Model) completeToken(token string, start int) Model {
	m.killing = false
	var matches []string
	for _, c := range m.Complete(token) {
		if strings.HasPrefix(c, token) {
			matches = append(matches, c)
		}
	}
	switch {
	case len(matches) == 0:
	case len(matches) == 1:
		m = m.replaceSpan(start, matches[0]+" ")
	default:
		if common := longestCommonPrefix(matches); len([]rune(common)) > len([]rune(token)) {
			m = m.replaceSpan(start, common)
		} else {
			m.mainOff = 0
			m = m.appendOutput(strings.Join(matches, "  "))
		}
	}
	return m
}

// replaceSpan replaces the token that ends at the cursor. Text before it stays.
func (m Model) replaceSpan(start int, text string) Model {
	if start < 0 || start > m.cursor || m.cursor > len(m.input) {
		return m
	}
	rest := append([]rune(nil), m.input[m.cursor:]...)
	head := append([]rune(nil), m.input[:start]...)
	mid := []rune(text)
	m.input = append(head, append(mid, rest...)...)
	m.cursor = start + len(mid)
	m.anchor = m.cursor
	m.killing = false
	return m
}

// completeCommand completes the slash command before the cursor. A prompt
// that does not start with "/" is not handled, so tab keeps moving focus.
func (m Model) completeCommand() (Model, bool) {
	if m.Complete == nil || len(m.input) == 0 || m.input[0] != '/' {
		return m, false
	}
	m.killing = false
	word := string(m.input[:m.cursor])
	if strings.ContainsRune(word, ' ') {
		return m, true
	}
	var matches []string
	for _, c := range m.Complete(word) {
		if strings.HasPrefix(c, word) {
			matches = append(matches, c)
		}
	}
	switch {
	case len(matches) == 0:
	case len(matches) == 1:
		m = m.replaceWord(matches[0] + " ")
	default:
		if common := longestCommonPrefix(matches); len(common) > len(word) {
			m = m.replaceWord(common)
		} else {
			m.mainOff = 0
			m = m.appendOutput(strings.Join(matches, "  "))
		}
	}
	return m, true
}

// replaceWord replaces the text before the cursor with text.
func (m Model) replaceWord(text string) Model {
	if m.hasSel() {
		m = m.deleteSel(false)
	}
	rest := append([]rune(nil), m.input[m.cursor:]...)
	m.input = append([]rune(text), rest...)
	m.cursor = len([]rune(text))
	m.anchor = m.cursor
	m.killing = false
	return m
}

// selectKey handles selection keys. Shift+Left and Shift+Right extend the
// selection. Ctrl+Shift+A selects the line, Ctrl+Shift+C copies it, and
// Ctrl+Shift+X cuts it into the yank buffer. With no selection, Ctrl+Shift+C
// copies the visible main pane instead.
func (m Model) selectKey(v tea.KeyPressMsg) (Model, bool) {
	if v.Mod.Contains(tea.ModCtrl) && v.Mod.Contains(tea.ModShift) {
		switch {
		case v.Code == 'a' || v.String() == "ctrl+shift+a":
			return m.selectAll(), true
		case v.Code == 'c' || v.String() == "ctrl+shift+c":
			if m.hasSel() {
				return m.copySel(), true
			}
			return m.copyVisible(), true
		case v.Code == 'x' || v.String() == "ctrl+shift+x":
			return m.cutSel(), true
		}
	}
	if !v.Mod.Contains(tea.ModShift) || v.Mod.Contains(tea.ModCtrl) || v.Mod.Contains(tea.ModAlt) {
		return m, false
	}
	switch v.Code {
	case tea.KeyLeft:
		return m.move(m.cursor-1, true), true
	case tea.KeyRight:
		return m.move(m.cursor+1, true), true
	default:
		return m, false
	}
}

// move puts the cursor at to. extend keeps the selection anchor.
func (m Model) move(to int, extend bool) Model {
	if to < 0 {
		to = 0
	}
	if to > len(m.input) {
		to = len(m.input)
	}
	m.cursor = to
	if !extend {
		m.anchor = to
	}
	m.killing = false
	return m
}

// selBounds is the selection in order, clamped to the prompt.
func (m Model) selBounds() (int, int) {
	lo, hi := m.anchor, m.cursor
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo < 0 {
		lo = 0
	}
	if hi > len(m.input) {
		hi = len(m.input)
	}
	if lo > hi {
		lo = hi
	}
	return lo, hi
}

// hasSel reports whether a range of the prompt is selected.
func (m Model) hasSel() bool {
	lo, hi := m.selBounds()
	return lo != hi
}

// selectAll selects the whole prompt.
func (m Model) selectAll() Model {
	if len(m.input) == 0 {
		return m
	}
	m.anchor = 0
	m.cursor = len(m.input)
	m.killing = false
	return m
}

// copySel stores the selection for Ctrl+Y and leaves the prompt in place.
func (m Model) copySel() Model {
	if !m.hasSel() {
		return m
	}
	lo, hi := m.selBounds()
	m.killed = append([]rune(nil), m.input[lo:hi]...)
	m.killing = false
	return m
}

// cutSel removes the selection and stores it for Ctrl+Y.
func (m Model) cutSel() Model {
	return m.deleteSel(true)
}

// deleteSel removes the selection. kill stores it for Ctrl+Y. Backspace does not.
func (m Model) deleteSel(kill bool) Model {
	if !m.hasSel() {
		return m
	}
	lo, hi := m.selBounds()
	if kill {
		m.killing = false
		m = m.killText(m.input[lo:hi], false)
	} else {
		m.killing = false
	}
	m.input = append(m.input[:lo], m.input[hi:]...)
	m.cursor = lo
	m.anchor = lo
	return m
}

// longestCommonPrefix is the longest prefix every string shares.
func longestCommonPrefix(items []string) string {
	prefix := items[0]
	for _, s := range items[1:] {
		for !strings.HasPrefix(s, prefix) {
			prefix = prefix[:len(prefix)-1]
		}
	}
	return prefix
}

// promptText is the prompt with the cursor drawn as a bar. A selection is reversed.
func (m Model) promptText() string {
	lo, hi := m.selBounds()
	before := string(m.input[:lo])
	mid := string(m.input[lo:hi])
	after := string(m.input[hi:])
	if mid != "" {
		mid = lipgloss.NewStyle().Reverse(true).Render(mid)
	}
	if m.cursor == lo && lo != hi {
		return before + "|" + mid + after
	}
	return before + mid + "|" + after
}

// insertText puts text at the cursor and moves the cursor past it.
// A selection is replaced, and that removal is not a kill.
func (m Model) insertText(text string) Model {
	if m.hasSel() {
		m = m.deleteSel(false)
	}
	runes := []rune(text)
	m.input = append(m.input[:m.cursor], append(runes, m.input[m.cursor:]...)...)
	m.cursor += len(runes)
	m.anchor = m.cursor
	m.killing = false
	return m
}

// killText stores chunk so Ctrl+Y can put it back. A kill that follows
// another kill joins that text: backward text is prepended, forward text
// appended. An empty chunk leaves the buffer as it is.
func (m Model) killText(chunk []rune, backward bool) Model {
	if len(chunk) == 0 {
		return m
	}
	got := append([]rune(nil), chunk...)
	if m.killing {
		if backward {
			m.killed = append(got, m.killed...)
		} else {
			m.killed = append(m.killed, got...)
		}
	} else {
		m.killed = got
	}
	m.killing = true
	return m
}

// yank inserts the last killed text at the cursor.
func (m Model) yank() Model {
	if len(m.killed) == 0 {
		return m
	}
	return m.insertText(string(m.killed))
}

// deleteBefore removes the rune before the cursor.
func (m Model) deleteBefore() Model {
	if m.hasSel() {
		return m.deleteSel(false)
	}
	if m.cursor > 0 {
		m.input = append(m.input[:m.cursor-1], m.input[m.cursor:]...)
		m.cursor--
		m.anchor = m.cursor
	}
	return m
}

// deleteAfter removes the rune under the cursor.
func (m Model) deleteAfter() Model {
	if m.hasSel() {
		return m.deleteSel(false)
	}
	if m.cursor < len(m.input) {
		m.input = append(m.input[:m.cursor], m.input[m.cursor+1:]...)
		m.anchor = m.cursor
	}
	return m
}

// deleteWordBefore removes the word before the cursor, and the spaces before it.
func (m Model) deleteWordBefore() Model {
	if m.hasSel() {
		return m.deleteSel(true)
	}
	start := wordLeft(m.input, m.cursor)
	m = m.killText(m.input[start:m.cursor], true)
	m.input = append(m.input[:start], m.input[m.cursor:]...)
	m.cursor = start
	m.anchor = m.cursor
	return m
}

// deleteWordAfter removes the word after the cursor, and the spaces before it.
func (m Model) deleteWordAfter() Model {
	if m.hasSel() {
		return m.deleteSel(true)
	}
	end := wordRight(m.input, m.cursor)
	m = m.killText(m.input[m.cursor:end], false)
	m.input = append(m.input[:m.cursor], m.input[end:]...)
	m.anchor = m.cursor
	return m
}

// UseHistory installs earlier prompts, oldest first, and remembers each new
// submitted line. HistoryMax drops the oldest (0 keeps all). Blank lines and
// a line equal to the one before it are left out.
func (m Model) UseHistory(lines []string, remember func(string)) Model {
	kept := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if n := len(kept); n > 0 && kept[n-1] == line {
			continue
		}
		kept = append(kept, line)
	}
	if m.HistoryMax > 0 && len(kept) > m.HistoryMax {
		kept = kept[len(kept)-m.HistoryMax:]
	}
	m.history = kept
	m.histAt = len(m.history)
	m.remember = remember
	return m
}

// recordPrompt adds a submitted line to the history. A line equal to the
// previous one is not repeated. The bool reports whether the line was added.
func (m Model) recordPrompt(line string) (Model, bool) {
	if n := len(m.history); n > 0 && m.history[n-1] == line {
		m.histAt = n
		return m, false
	}
	m.history = append(m.history, line)
	if extra := len(m.history) - m.HistoryMax; m.HistoryMax > 0 && extra > 0 {
		m.history = m.history[extra:]
	}
	m.histAt = len(m.history)
	return m, true
}

// historyBack recalls the previous submitted line. The first step back keeps
// the unsent text, so historyForward can restore it.
func (m Model) historyBack() Model {
	if len(m.history) == 0 || m.histAt == 0 {
		return m
	}
	if m.histAt == len(m.history) {
		m.histDraft = string(m.input)
	}
	m.histAt--
	return m.setPrompt(m.history[m.histAt])
}

// historyForward recalls the next submitted line, or the unsent text.
func (m Model) historyForward() Model {
	if m.histAt == len(m.history) {
		return m
	}
	m.histAt++
	if m.histAt == len(m.history) {
		return m.setPrompt(m.histDraft)
	}
	return m.setPrompt(m.history[m.histAt])
}

// setPrompt replaces the prompt text and puts the cursor at its end.
func (m Model) setPrompt(text string) Model {
	m.input = []rune(text)
	m.cursor = len(m.input)
	m.anchor = m.cursor
	m.killing = false
	return m
}

// submitPrompt returns the trimmed prompt and clears the editor. The bool
// reports that the line was added to the history.
func (m Model) submitPrompt() (Model, string, bool) {
	line := strings.TrimSpace(string(m.input))
	m = m.setPrompt("")
	if line == "" {
		return m, "", false
	}
	m, recorded := m.recordPrompt(line)
	return m, line, recorded
}
