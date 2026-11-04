package screen

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
)

func keyPress(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code} }

func ctrl(code rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: code, Mod: tea.ModCtrl} }

func alt(code rune) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: code, Mod: tea.ModAlt, Text: string(code)}
}

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
	assert.Contains(t, ansi.Strip(m.render()), "> aX|bc")
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

func TestPrompt_WordMotion(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "one two  three")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModCtrl})
	assert.Equal(t, len("one two  "), m.cursor, "ctrl+left lands on the start of the last word")
	m, _ = typed(t, m, alt('b'))
	assert.Equal(t, len("one "), m.cursor, "alt+b lands on the previous word")
	assert.Equal(t, "one two  three", m.Prompt(), "alt+b does not type the letter")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModCtrl})
	assert.Equal(t, len("one two"), m.cursor, "ctrl+right lands at the end of the word")
	m, _ = typed(t, m, alt('f'))
	assert.Equal(t, len("one two  three"), m.cursor, "alt+f skips the spaces and the next word")

	m.cursor = len("one two")
	m.anchor = m.cursor
	m, _ = typed(t, m, alt('d'))
	assert.Equal(t, "one two", m.Prompt(), "alt+d deletes the spaces and the next word")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyBackspace, Mod: tea.ModAlt})
	assert.Equal(t, "one ", m.Prompt(), "alt+backspace deletes the word before the cursor")
}

func TestPrompt_YankRestoresKilledText(t *testing.T) {
	yank := tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl, Text: "y"}

	m := typeText(t, newPromptScreen(t), "hello")
	m, _ = typed(t, m, ctrl('u'))
	assert.Equal(t, "", m.Prompt())
	m, _ = typed(t, m, yank)
	assert.Equal(t, "hello", m.Prompt(), "ctrl+y restores the line and does not type y")

	m = m.setPrompt("hello")
	m.cursor = 2
	m.anchor = m.cursor
	m, _ = typed(t, m, ctrl('k'))
	assert.Equal(t, "he", m.Prompt())
	m, _ = typed(t, m, yank)
	assert.Equal(t, "hello", m.Prompt(), "ctrl+k yank restores the tail")

	m = m.setPrompt("abc def ghi")
	m.cursor = 0
	m.anchor = m.cursor
	m, _ = typed(t, m, alt('d'))
	m, _ = typed(t, m, alt('d'))
	assert.Equal(t, " ghi", m.Prompt())
	m, _ = typed(t, m, yank)
	assert.Equal(t, "abc def ghi", m.Prompt(), "two forward kills join into one yank")

	m = m.setPrompt("abc def")
	m, _ = typed(t, m, ctrl('w'))
	m, _ = typed(t, m, ctrl('w'))
	assert.Equal(t, "", m.Prompt())
	m, _ = typed(t, m, yank)
	assert.Equal(t, "abc def", m.Prompt(), "two backward kills prepend into one yank")

	m = m.setPrompt("abcdef")
	m.cursor = 3
	m.anchor = m.cursor
	m, _ = typed(t, m, ctrl('u'))
	m, _ = typed(t, m, ctrl('k'))
	m, _ = typed(t, m, yank)
	assert.Equal(t, "abcdef", m.Prompt(), "a backward kill and a forward kill join")

	m = m.setPrompt("hello")
	m, _ = typed(t, m, ctrl('u'))
	m = typeText(t, m, "x")
	m, _ = typed(t, m, ctrl('u'))
	m, _ = typed(t, m, yank)
	assert.Equal(t, "x", m.Prompt(), "typing between kills replaces the buffer")

	m = typeText(t, newPromptScreen(t), "ab")
	m, _ = typed(t, m, keyPress(tea.KeyBackspace))
	m, _ = typed(t, m, yank)
	assert.Equal(t, "a", m.Prompt(), "backspace does not fill the yank buffer")

	m = typeText(t, newPromptScreen(t), "hi")
	m, _ = typed(t, m, yank)
	assert.Equal(t, "hi", m.Prompt(), "ctrl+y with nothing killed leaves the prompt")

	m = m.setPrompt("hello world")
	m, _ = typed(t, m, ctrl('w'))
	m, _ = typed(t, m, ctrl('k'))
	m, _ = typed(t, m, ctrl('u'))
	m, _ = typed(t, m, yank)
	assert.Equal(t, "hello world", m.Prompt(), "a kill of nothing does not wipe the buffer")

	m = m.setPrompt("abcdef")
	m.cursor = 3
	m.anchor = m.cursor
	m, _ = typed(t, m, ctrl('k'))
	m, _ = typed(t, m, keyPress(tea.KeyLeft))
	m, _ = typed(t, m, ctrl('k'))
	m, _ = typed(t, m, yank)
	assert.Equal(t, "abc", m.Prompt(), "moving the cursor starts a new kill")
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
	assert.Contains(t, ansi.Strip(m.render()), "ctrl+f")

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

func TestCtrlD_DeletesUnderTheCursorAndQuitsWhenEmpty(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "ab")
	m, _ = typed(t, m, keyPress(tea.KeyLeft))
	m, cmd := typed(t, m, ctrl('d'))
	assert.Equal(t, "a", m.Prompt(), "ctrl+d deletes the character under the cursor")
	assert.Nil(t, cmd, "a non-empty prompt does not quit")

	m, cmd = typed(t, m, ctrl('d'))
	assert.Equal(t, "a", m.Prompt(), "ctrl+d at the end of the line leaves it")
	assert.Nil(t, cmd)

	m = typeText(t, m, "d")
	assert.Equal(t, "ad", m.Prompt(), "d still types")

	m, _ = typed(t, m, ctrl('u'))
	_, cmd = typed(t, m, ctrl('d'))
	assert.NotNil(t, cmd, "ctrl+d on an empty prompt quits")
}

func TestCtrlD_EndsAKillAndDropsASelection(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "helloXY")
	m, _ = typed(t, m, keyPress(tea.KeyLeft))
	m, _ = typed(t, m, keyPress(tea.KeyLeft))
	m, _ = typed(t, m, ctrl('u'))
	assert.Equal(t, "XY", m.Prompt())
	m, _ = typed(t, m, ctrl('d'))
	assert.Equal(t, "Y", m.Prompt())
	m, _ = typed(t, m, ctrl('k'))
	m, _ = typed(t, m, ctrl('y'))
	assert.Equal(t, "Y", m.Prompt(), "ctrl+d ends a kill, so the next kill does not join")

	m = typeText(t, newPromptScreen(t), "ab")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: tea.KeyLeft, Mod: tea.ModShift})
	m, cmd := typed(t, m, ctrl('d'))
	assert.Equal(t, "a", m.Prompt(), "ctrl+d removes a selection")
	assert.Nil(t, cmd)
	m, _ = typed(t, m, ctrl('y'))
	assert.Equal(t, "a", m.Prompt(), "the removed selection is not yanked")
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

func TestComplete_AtTokenFillsAndKeepsTheTextBeforeIt(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("@main.go", "@map.txt")
	m = typeText(t, m, "see @m")
	m, _ = typed(t, m, keyPress(tea.KeyTab))
	assert.Equal(t, "see @ma", m.Prompt(), "a shared prefix extends the token")
	assert.False(t, m.focusFeed, "completing an @ token does not switch panes")

	m, _ = typed(t, m, keyPress(tea.KeyTab))
	assert.Equal(t, "see @ma", m.Prompt(), "a shared prefix that is already typed stays")
	assert.Contains(t, strings.Join(m.main, "\n"), "@main.go  @map.txt")

	m.Complete = completeFrom("@main.go")
	m, _ = typed(t, m, keyPress(tea.KeyTab))
	assert.Equal(t, "see @main.go ", m.Prompt(), "one match fills the token and leaves the words before it")
	assert.False(t, m.focusFeed, "completing an @ token does not switch panes")
}

func TestComplete_AtTokenWithNoMatchStays(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("@main.go")
	m = typeText(t, m, "see @zzz")
	m, _ = typed(t, m, keyPress(tea.KeyTab))
	assert.Equal(t, "see @zzz", m.Prompt())
	assert.False(t, m.focusFeed, "an @ token with no match still consumes tab")
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
	assert.Contains(t, ansi.Strip(m.render()), "END", "the wrapped tail of a long line is on screen")
}

func TestPrompt_EmptyShowsTheHintAndTypingReplacesIt(t *testing.T) {
	m := newPromptScreen(t)
	assert.Contains(t, ansi.Strip(m.render()), "? help", "the status row names the key help")
	assert.Equal(t, "", m.Prompt(), "the hint is not part of the text")

	m = typeText(t, m, "h")
	assert.Contains(t, ansi.Strip(m.render()), "> h|", "typing puts the text in the prompt")
	assert.Contains(t, ansi.Strip(m.render()), "> h|")
}

func TestPrompt_SelectCopyCutAndReplace(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "hello")
	m, _ = typed(t, m, ctrl('a'))
	right := tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}
	m, _ = typed(t, m, right)
	m, _ = typed(t, m, right)
	lo, hi := m.selBounds()
	assert.Equal(t, "he", string(m.input[lo:hi]))
	assert.Equal(t, "hello", m.Prompt())
	assert.Contains(t, ansi.Strip(m.promptText()), "he|llo")
	assert.NotEqual(t, "he|llo", m.promptText(), "the selection is drawn reversed")

	copyKey := tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl | tea.ModShift, Text: "c"}
	m, _ = typed(t, m, copyKey)
	assert.Equal(t, "hello", m.Prompt(), "copy does not type c or delete")
	assert.Equal(t, "he", string(m.killed))
	m, _ = typed(t, m, ctrl('e'))
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl, Text: "y"})
	assert.Equal(t, "hellohe", m.Prompt(), "yank puts the copy at the cursor")

	m = typeText(t, m.setPrompt(""), "hello")
	m, _ = typed(t, m, ctrl('a'))
	m, _ = typed(t, m, right)
	m, _ = typed(t, m, right)
	m, _ = typed(t, m, keyPress(tea.KeyLeft))
	assert.False(t, m.hasSel(), "left collapses the selection to its start")
	assert.Equal(t, 0, m.cursor)

	m, _ = typed(t, m, right)
	m, _ = typed(t, m, right)
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'x', Mod: tea.ModCtrl | tea.ModShift, Text: "x"})
	assert.Equal(t, "llo", m.Prompt())
	assert.False(t, m.hasSel())
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl, Text: "y"})
	assert.Equal(t, "hello", m.Prompt(), "cut can be yanked back")

	m = typeText(t, newPromptScreen(t), "hello")
	m, _ = typed(t, m, ctrl('a'))
	m, _ = typed(t, m, right)
	m, _ = typed(t, m, right)
	m, _ = typed(t, m, keyPress(tea.KeyBackspace))
	assert.Equal(t, "llo", m.Prompt())
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'y', Mod: tea.ModCtrl, Text: "y"})
	assert.Equal(t, "llo", m.Prompt(), "backspace does not fill the yank buffer")

	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl | tea.ModShift, Text: "a"})
	assert.True(t, m.hasSel())
	m = typeText(t, m, "X")
	assert.Equal(t, "X", m.Prompt(), "typing replaces the selection")
}

func TestPrompt_EscClearsTheSelection(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "hello")
	m, _ = typed(t, m, ctrl('a'))
	right := tea.KeyPressMsg{Code: tea.KeyRight, Mod: tea.ModShift}
	m, _ = typed(t, m, right)
	m, _ = typed(t, m, right)
	assert.True(t, m.hasSel())
	assert.Equal(t, 2, m.cursor)

	m, cmd := typed(t, m, keyPress(tea.KeyEsc))
	assert.Nil(t, cmd, "esc does not quit")
	assert.False(t, m.hasSel(), "esc clears the selection")
	assert.Equal(t, "hello", m.Prompt(), "the text stays")
	assert.Equal(t, 2, m.cursor, "the cursor stays")

	m, _ = typed(t, m, keyPress(tea.KeyEsc))
	assert.Equal(t, "hello", m.Prompt(), "esc with no selection leaves the prompt")
}

func TestHelp_ListsTheSlashCommands(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("/model", "/session")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "/model  /session", "the help lists the slash commands")
	assert.Contains(t, view, "home", "the help lists Home")
	assert.Contains(t, view, "ctrl+y", "the help lists yank")
	assert.Contains(t, view, "shift+left/right", "the help lists selection")
}

func TestEsc_RestoresTheUnsentPromptWhileRecalling(t *testing.T) {
	m := newPromptScreen(t)
	m = m.UseHistory([]string{"older", "newer"}, nil)
	m = typeText(t, m, "draft")
	m, _ = typed(t, m, ctrl('p'))
	m, _ = typed(t, m, ctrl('p'))
	assert.Equal(t, "older", m.Prompt())

	m, _ = typed(t, m, keyPress(tea.KeyEsc))
	assert.Equal(t, "draft", m.Prompt(), "esc puts the unsent text back")

	m, _ = typed(t, m, keyPress(tea.KeyEsc))
	assert.Equal(t, "draft", m.Prompt(), "esc leaves the prompt when not recalling")

	m, _ = typed(t, m, ctrl('p'))
	assert.Equal(t, "newer", m.Prompt(), "recall still starts from the newest line")
}

func TestEdit_OnARecalledLineKeepsTheEdit(t *testing.T) {
	recall := func() Model {
		m := newPromptScreen(t)
		m = m.UseHistory([]string{"older", "newer"}, nil)
		m = typeText(t, m, "draft")
		m, _ = typed(t, m, ctrl('p'))
		return m
	}

	m := typeText(t, recall(), "!")
	m, _ = typed(t, m, ctrl('n'))
	assert.Equal(t, "newer!", m.Prompt(), "an edit leaves history, so ctrl+n does not replace it")

	m = recall()
	m, _ = typed(t, m, keyPress(tea.KeyLeft))
	m, _ = typed(t, m, ctrl('n'))
	assert.Equal(t, "draft", m.Prompt(), "moving the cursor stays in history")

	m = recall()
	m, _ = typed(t, m, keyPress(tea.KeyDelete))
	m, _ = typed(t, m, ctrl('n'))
	assert.Equal(t, "draft", m.Prompt(), "a delete that changes nothing stays in history")
}

func TestPrompt_SharedHistoryRecallsAndRemembers(t *testing.T) {
	m := newPromptScreen(t)
	m.HistoryMax = 1
	var saved []string
	m = m.UseHistory([]string{"  drop  ", "keep", "", "keep"}, func(line string) {
		saved = append(saved, line)
	})
	m, _ = typed(t, m, ctrl('p'))
	assert.Equal(t, "keep", m.Prompt(), "a repeated neighbor is stored once")
	m, _ = typed(t, m, ctrl('p'))
	assert.Equal(t, "keep", m.Prompt(), "the older line was past the limit")

	m = typeText(t, m.setPrompt(""), "fresh")
	m, cmd := typed(t, m, keyPress(tea.KeyEnter))
	require.NotNil(t, cmd)
	cmd()
	assert.Equal(t, []string{"fresh"}, saved)

	m = typeText(t, m, "fresh")
	_, cmd = typed(t, m, keyPress(tea.KeyEnter))
	assert.Nil(t, cmd, "a repeated line is not written again")
	assert.Equal(t, []string{"fresh"}, saved)
}

func TestPrompt_HistoryKeepsTheLimitAndAllWhenUnlimited(t *testing.T) {
	m := newPromptScreen(t)
	m.HistoryMax = 2
	m.OnSubmit = func(string) {}
	for _, line := range []string{"a", "b", "c"} {
		m = typeText(t, m, line)
		m, _ = typed(t, m, keyPress(tea.KeyEnter))
	}
	assert.Equal(t, []string{"b", "c"}, m.history, "only the last two prompts are kept")

	m.HistoryMax = 0
	m = typeText(t, m, "d")
	m, _ = typed(t, m, keyPress(tea.KeyEnter))
	assert.Equal(t, []string{"b", "c", "d"}, m.history, "zero keeps every prompt")
}

func TestPromptOrigin_KeepsTheCursorInTheWindow(t *testing.T) {
	assert.Equal(t, 0, promptOrigin(20, 4, 30), "a short line stays put")
	assert.Equal(t, 0, promptOrigin(200, 10, 40), "a cursor in the first window stays put")
	origin := promptOrigin(200, 0, 40)
	assert.Equal(t, 0, origin, "the start of a long line stays visible")
	origin = promptOrigin(220, 200, 40)
	assert.Greater(t, origin, 0)
	assert.GreaterOrEqual(t, 200, origin)
	assert.Less(t, 200, origin+40, "the cursor bar stays inside the window")
	assert.LessOrEqual(t, origin, 220-40, "the window does not run past the line")
}
