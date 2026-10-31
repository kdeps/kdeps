package screen

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

func keyPress(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func ctrl(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl} }

func typeText(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m, _ = typed(t, m, tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func newPromptScreen(t *testing.T) Model {
	t.Helper()
	return resized(t, New(&feed.Feed{}))
}

func TestPrompt_InsertsAtTheCursor(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "abc")
	m, _ = typed(t, m, keyPress(tea.KeyLeft))
	m, _ = typed(t, m, keyPress(tea.KeyLeft))
	m = typeText(t, m, "X")
	assert.Equal(t, "aXbc", m.Prompt())
	assert.Contains(t, m.render(), "> aX|bc")
}

func TestPrompt_HomeEndAndDelete(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "abc")
	m, _ = typed(t, m, ctrl('a'))
	m, _ = typed(t, m, keyPress(tea.KeyDelete))
	assert.Equal(t, "bc", m.Prompt())
	m, _ = typed(t, m, ctrl('e'))
	m = typeText(t, m, "d")
	assert.Equal(t, "bcd", m.Prompt())
}

func TestPrompt_KillLineAndWord(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "one two  three")
	m, _ = typed(t, m, ctrl('w'))
	assert.Equal(t, "one two  ", m.Prompt(), "ctrl+w removes the last word")
	m, _ = typed(t, m, ctrl('w'))
	assert.Equal(t, "one ", m.Prompt())

	m = typeText(t, m, "x y")
	m, _ = typed(t, m, keyPress(tea.KeyLeft))
	m, _ = typed(t, m, ctrl('u'))
	assert.Equal(t, "y", m.Prompt(), "ctrl+u removes everything before the cursor")
	m, _ = typed(t, m, ctrl('k'))
	assert.Equal(t, "", m.Prompt(), "ctrl+k removes everything after the cursor")
}

func TestPrompt_HistoryRecallsAndRestoresTheDraft(t *testing.T) {
	var sent []string
	m := newPromptScreen(t)
	m.OnSubmit = func(line string) { sent = append(sent, line) }
	m = typeText(t, m, "first")
	m, _ = typed(t, m, keyPress(tea.KeyEnter))
	m = typeText(t, m, "second")
	m, _ = typed(t, m, keyPress(tea.KeyEnter))
	m = typeText(t, m, "draft")

	m, _ = typed(t, m, ctrl('p'))
	assert.Equal(t, "second", m.Prompt())
	m, _ = typed(t, m, ctrl('p'))
	assert.Equal(t, "first", m.Prompt())
	m, _ = typed(t, m, ctrl('p'))
	assert.Equal(t, "first", m.Prompt(), "the oldest line stays put")
	m, _ = typed(t, m, ctrl('n'))
	m, _ = typed(t, m, ctrl('n'))
	assert.Equal(t, "draft", m.Prompt(), "stepping forward restores the unsent text")
}

func TestPrompt_HistorySkipsRepeatsAndEmptyLines(t *testing.T) {
	m := newPromptScreen(t)
	m.OnSubmit = func(string) {}
	for _, line := range []string{"same", "same", "   "} {
		m = typeText(t, m, line)
		m, _ = typed(t, m, keyPress(tea.KeyEnter))
	}
	assert.Equal(t, []string{"same"}, m.history)
}

func TestPrompt_UpAndDownStillScrollTheMain(t *testing.T) {
	m := resized(t, New(&feed.Feed{}))
	m, _ = typed(t, m, OutputMsg{Text: manyLines(100)})
	m, _ = typed(t, m, keyPress(tea.KeyUp))
	assert.Equal(t, 1, m.mainOff, "up scrolls the main pane, not the prompt history")
}

func TestHelp_QuestionMarkOpensTheKeysAndEscCloses(t *testing.T) {
	m := newPromptScreen(t)
	m, _ = typed(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	assert.True(t, m.HasOverlay(), "? in an empty prompt opens the help")
	assert.Contains(t, m.render(), "ctrl+f")

	m, _ = typed(t, m, keyPress(tea.KeyEsc))
	assert.False(t, m.HasOverlay(), "esc closes the help")
	assert.Equal(t, "", m.Prompt(), "the ? did not reach the prompt")
}

func TestHelp_QuestionMarkInTextIsTyped(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "why")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	assert.False(t, m.HasOverlay())
	assert.Equal(t, "why?", m.Prompt())
}

func TestCtrlC_ClearsThePromptBeforeQuitting(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "draft")
	m, cmd := typed(t, m, ctrl('c'))
	assert.Equal(t, "", m.Prompt(), "the first Ctrl+C clears the prompt")
	assert.Nil(t, cmd, "and does not quit")

	_, cmd = typed(t, m, ctrl('c'))
	assert.NotNil(t, cmd, "Ctrl+C on an empty prompt quits")
}

func completeFrom(items ...string) func(string) []string {
	return func(string) []string { return items }
}

func TestComplete_UniqueMatchFillsTheCommand(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("/model", "/help")
	m = typeText(t, m, "/mo")
	m, _ = typed(t, m, keyPress(tea.KeyTab))
	assert.Equal(t, "/model ", m.Prompt())
}

func TestComplete_SharedPrefixExtendsThePrompt(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("/session", "/settings")
	m = typeText(t, m, "/s")
	m, _ = typed(t, m, keyPress(tea.KeyTab))
	assert.Equal(t, "/se", m.Prompt(), "both commands start with /se")
	m = typeText(t, m, "s")
	m, _ = typed(t, m, keyPress(tea.KeyTab))
	assert.Equal(t, "/session ", m.Prompt(), "one match left fills the command")
}

func TestComplete_AmbiguousListsTheMatches(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("/model", "/mesh")
	m = typeText(t, m, "/m")
	m, _ = typed(t, m, keyPress(tea.KeyTab))
	assert.Equal(t, "/m", m.Prompt(), "the shared prefix is the prompt already: the prompt stays")
	assert.Contains(t, strings.Join(m.main, "\n"), "/model  /mesh")
}

func TestComplete_NotASlashPromptKeepsTabForFocus(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("/model")
	m = typeText(t, m, "mo")
	m, _ = typed(t, m, keyPress(tea.KeyTab))
	assert.True(t, m.focusFeed, "tab toggles focus when the prompt is not a command")
	assert.Equal(t, "mo", m.Prompt())
}

func TestPrompt_PasteInsertsOneLine(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "ab")
	m, _ = typed(t, m, tea.PasteMsg{Content: "x\ny\r\nz"})
	assert.Equal(t, "abx y z", m.Prompt(), "a paste keeps its text on one line and does not submit")
}

func TestMain_LongLineWrapsAndTheTailShows(t *testing.T) {
	m := newPromptScreen(t)
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'f', Mod: tea.ModCtrl}) // feed closed: the main pane is full width
	m, _ = typed(t, m, OutputMsg{Text: strings.Repeat("a", 150) + "END"})
	assert.Contains(t, m.render(), "END", "the wrapped tail of a long line is on screen")
}

func TestPrompt_EmptyShowsTheHintAndTypingReplacesIt(t *testing.T) {
	m := newPromptScreen(t)
	assert.Contains(t, m.render(), "/ for commands", "an empty prompt says what to type")
	assert.Equal(t, "", m.Prompt(), "the hint is not part of the text")

	m = typeText(t, m, "h")
	assert.NotContains(t, m.render(), "/ for commands", "typing replaces the hint")
	assert.Contains(t, m.render(), "> h|")
}
