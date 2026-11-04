package screen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptEditor_EnvOrder(t *testing.T) {
	t.Setenv("KDEPS_EDITOR", "ked")
	t.Setenv("VISUAL", "vis")
	t.Setenv("EDITOR", "ed")
	assert.Equal(t, "ked", promptEditor())
	assert.Equal(t, []string{"ked", "/tmp/prompt.md"}, editorCommand("/tmp/prompt.md").Args)

	t.Setenv("KDEPS_EDITOR", "")
	assert.Equal(t, "vis", promptEditor())
	t.Setenv("VISUAL", "")
	assert.Equal(t, "ed", promptEditor())
	t.Setenv("EDITOR", "")
	assert.Equal(t, "vi", promptEditor())
}

func TestEditorResult_ReplacesWithOneLineAndRemovesTheFile(t *testing.T) {
	path, err := writePromptFile("keep")
	require.NoError(t, err)
	body, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "keep", string(body))

	require.NoError(t, os.WriteFile(path, []byte("  line one\r\nline two\n"), 0o644))
	msg := editorResult(path, nil)
	assert.True(t, msg.ok)
	assert.Equal(t, "line one line two", msg.text)
	_, statErr := os.Stat(path)
	assert.True(t, os.IsNotExist(statErr), "the temp file is removed after a read")
}

func TestEditorResult_BlankClearsAndFailureKeepsThePrompt(t *testing.T) {
	blank, err := writePromptFile("x")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(blank, []byte("\n \n\t\n"), 0o644))
	msg := editorResult(blank, nil)
	assert.True(t, msg.ok)
	assert.Equal(t, "", msg.text)

	failed, err := writePromptFile("stay")
	require.NoError(t, err)
	msg = editorResult(failed, errors.New("editor exited"))
	assert.False(t, msg.ok)
	_, statErr := os.Stat(failed)
	assert.True(t, os.IsNotExist(statErr))

	msg = editorResult(filepath.Join(t.TempDir(), "missing.md"), nil)
	assert.False(t, msg.ok)
}

func TestPrompt_CtrlOReplacesTheLineAndDoesNotSend(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "hi")
	sent := false
	m.OnSubmit = func(string) { sent = true }
	m.launchEditor = func(text string) tea.Cmd {
		assert.Equal(t, "hi", text)
		return func() tea.Msg { return editorResultMsg{text: "from the editor", ok: true} }
	}

	next, cmd := typed(t, m, tea.KeyPressMsg{Code: 'o', Mod: tea.ModCtrl, Text: "o"})
	assert.Equal(t, "hi", next.Prompt(), "ctrl+o does not type o")
	require.NotNil(t, cmd)

	applied, _ := next.Update(cmd())
	got := applied.(Model)
	assert.Equal(t, "from the editor", got.Prompt())
	assert.Contains(t, ansi.Strip(got.render()), "> from the editor|")
	assert.False(t, sent, "the editor does not send the prompt")

	kept, _ := got.Update(editorResultMsg{})
	assert.Equal(t, "from the editor", kept.(Model).Prompt(), "a failed editor leaves the prompt")

	cleared, _ := got.Update(editorResultMsg{ok: true})
	assert.Equal(t, "", cleared.(Model).Prompt(), "a blank file clears the prompt")
}

func TestHelp_ListsTheEditorKey(t *testing.T) {
	m := newPromptScreen(t)
	m.Complete = completeFrom("/model", "/session")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: '?', Text: "?"})
	view := ansi.Strip(m.render())
	assert.Contains(t, view, "ctrl+o", "the help lists the editor key")
	assert.Contains(t, view, "/model  /session", "the command list still fits")
}
