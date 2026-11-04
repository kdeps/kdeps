package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
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
		m.cursor = wordLeft(m.input, m.cursor)
	case wordForward(v):
		m.cursor = wordRight(m.input, m.cursor)
	default:
		return m, false
	}
	m.killing = false
	return m, true
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
		m = m.killText(m.input[:m.cursor], true)
		m.input = m.input[m.cursor:]
		m.cursor = 0
		return m, true
	case "ctrl+k":
		m = m.killText(m.input[m.cursor:], false)
		m.input = m.input[:m.cursor]
		return m, true
	case "ctrl+w":
		return m.deleteWordBefore(), true
	case "left":
		m.cursor = max(m.cursor-1, 0)
	case "right":
		m.cursor = min(m.cursor+1, len(m.input))
	case "ctrl+a":
		m.cursor = 0
	case "ctrl+e":
		m.cursor = len(m.input)
	case "backspace":
		m = m.deleteBefore()
	case "delete":
		m = m.deleteAfter()
	case "ctrl+p":
		m = m.historyBack()
	case "ctrl+n":
		m = m.historyForward()
	case "tab":
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
			m = m.appendOutput(strings.Join(matches, "  "))
		}
	}
	return m, true
}

// replaceWord replaces the text before the cursor with text.
func (m Model) replaceWord(text string) Model {
	rest := append([]rune(nil), m.input[m.cursor:]...)
	m.input = append([]rune(text), rest...)
	m.cursor = len([]rune(text))
	m.killing = false
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

// promptText is the prompt with the cursor drawn as a bar at its position.
func (m Model) promptText() string {
	return string(m.input[:m.cursor]) + "|" + string(m.input[m.cursor:])
}

// insertText puts text at the cursor and moves the cursor past it.
func (m Model) insertText(text string) Model {
	runes := []rune(text)
	m.input = append(m.input[:m.cursor], append(runes, m.input[m.cursor:]...)...)
	m.cursor += len(runes)
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
	if m.cursor > 0 {
		m.input = append(m.input[:m.cursor-1], m.input[m.cursor:]...)
		m.cursor--
	}
	return m
}

// deleteAfter removes the rune under the cursor.
func (m Model) deleteAfter() Model {
	if m.cursor < len(m.input) {
		m.input = append(m.input[:m.cursor], m.input[m.cursor+1:]...)
	}
	return m
}

// deleteWordBefore removes the word before the cursor, and the spaces before it.
func (m Model) deleteWordBefore() Model {
	start := wordLeft(m.input, m.cursor)
	m = m.killText(m.input[start:m.cursor], true)
	m.input = append(m.input[:start], m.input[m.cursor:]...)
	m.cursor = start
	return m
}

// deleteWordAfter removes the word after the cursor, and the spaces before it.
func (m Model) deleteWordAfter() Model {
	end := wordRight(m.input, m.cursor)
	m = m.killText(m.input[m.cursor:end], false)
	m.input = append(m.input[:m.cursor], m.input[end:]...)
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
