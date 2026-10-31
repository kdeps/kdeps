package screen

import (
	"strings"
)

// promptHistoryMax bounds the submitted lines ctrl+p and ctrl+n recall.
const promptHistoryMax = 100

// editKey applies a prompt editing key (cursor, delete, kill, history). It
// reports false for keys that are not editing keys.
func (m Model) editKey(name string) (Model, bool) {
	switch name {
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
	case "ctrl+u":
		m.input = m.input[m.cursor:]
		m.cursor = 0
	case "ctrl+k":
		m.input = m.input[:m.cursor]
	case "ctrl+w":
		m = m.deleteWordBefore()
	case "ctrl+p":
		m = m.historyBack()
	case "ctrl+n":
		m = m.historyForward()
	case "tab":
		return m.completeCommand()
	default:
		return m, false
	}
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
	return m
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
	start := m.cursor
	for start > 0 && m.input[start-1] == ' ' {
		start--
	}
	for start > 0 && m.input[start-1] != ' ' {
		start--
	}
	m.input = append(m.input[:start], m.input[m.cursor:]...)
	m.cursor = start
	return m
}

// recordPrompt adds a submitted line to the history. A line equal to the
// previous one is not repeated.
func (m Model) recordPrompt(line string) Model {
	if n := len(m.history); n == 0 || m.history[n-1] != line {
		m.history = append(m.history, line)
		if extra := len(m.history) - promptHistoryMax; extra > 0 {
			m.history = m.history[extra:]
		}
	}
	m.histAt = len(m.history)
	return m
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
	return m
}

// submitPrompt returns the trimmed prompt and clears the editor. The line is
// recorded in the history when it is not empty.
func (m Model) submitPrompt() (Model, string) {
	line := strings.TrimSpace(string(m.input))
	m = m.setPrompt("")
	if line != "" {
		m = m.recordPrompt(line)
	}
	return m, line
}
