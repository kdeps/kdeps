package screen

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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
	assert.Contains(t, view, "ctrl+shift+v", "the help lists paste")
	assert.Contains(t, view, "/model  /session", "the command list still fits")
}

func TestPrompt_CtrlShiftVPastesOneLine(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "ab")
	m.readClipboard = func() (string, bool) { return "x\ny\n", true }
	chord := tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl | tea.ModShift, Text: "v"}

	next, cmd := typed(t, m, chord)
	assert.Equal(t, "ab", next.Prompt(), "ctrl+shift+v does not type v")
	require.NotNil(t, cmd)
	applied, _ := next.Update(cmd())
	assert.Equal(t, "abx y", applied.(Model).Prompt(), "a newline becomes a space and a trailing newline is dropped")

	m.readClipboard = func() (string, bool) { return "", false }
	next, cmd = typed(t, m, chord)
	require.NotNil(t, cmd)
	assert.Nil(t, cmd(), "an empty clipboard has nothing to paste")
	assert.Equal(t, "ab", next.Prompt())

	m.readClipboard = func() (string, bool) { return "\n\n", true }
	next, cmd = typed(t, m, chord)
	assert.Nil(t, cmd(), "a clipboard of only newlines leaves the prompt")
	assert.Equal(t, "ab", next.Prompt())
}

func TestPrompt_CtrlShiftVReplacesASelection(t *testing.T) {
	m := typeText(t, newPromptScreen(t), "hello")
	m, _ = typed(t, m, tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl | tea.ModShift, Text: "a"})
	m.readClipboard = func() (string, bool) { return "z", true }
	next, cmd := typed(t, m, tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl | tea.ModShift, Text: "v"})
	require.NotNil(t, cmd)
	applied, _ := next.Update(cmd())
	assert.Equal(t, "z", applied.(Model).Prompt())
}

func TestPrompt_VStillTypes(t *testing.T) {
	m := newPromptScreen(t)
	called := false
	m.readClipboard = func() (string, bool) {
		called = true
		return "no", true
	}
	m, cmd := typed(t, m, tea.KeyPressMsg{Code: 'v', Text: "v"})
	assert.Equal(t, "v", m.Prompt())
	assert.Nil(t, cmd)
	assert.False(t, called)

	m.readImage = func() (string, bool) { return "", false }
	_, cmd = typed(t, m, tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl, Text: "v"})
	require.NotNil(t, cmd, "ctrl+v pastes an image")
	assert.Nil(t, cmd())
	assert.False(t, called, "ctrl+v does not read clipboard text")
}

func TestReadClipboardText_SkipsAMissingTool(t *testing.T) {
	dir := t.TempDir()
	writeTool(t, dir, "xclip", "#!/bin/sh\nprintf 'from xclip'\n")
	t.Setenv("PATH", dir)
	text, ok := readClipboardTextWithin(fakeToolLimit)
	assert.True(t, ok)
	assert.Equal(t, "from xclip", text)
}

func TestReadClipboardText_EmptyStops(t *testing.T) {
	dir := t.TempDir()
	writeTool(t, dir, "pbpaste", "#!/bin/sh\nexit 0\n")
	ran := filepath.Join(dir, "ran")
	writeTool(t, dir, "xclip", "#!/bin/sh\ntouch "+ran+"\nprintf no\n")
	t.Setenv("PATH", dir)
	text, ok := readClipboardTextWithin(fakeToolLimit)
	assert.False(t, ok)
	assert.Equal(t, "", text)
	_, err := os.Stat(ran)
	assert.True(t, os.IsNotExist(err), "an empty clipboard does not try the next tool")
}

func TestCommandOutput_KillsASlowTool(t *testing.T) {
	dir := t.TempDir()
	writeTool(t, dir, "slow", "#!/bin/sh\nsleep 5\n")
	_, err := commandOutput([]string{filepath.Join(dir, "slow")}, 100*time.Millisecond)
	assert.Error(t, err)
}

// fakeToolLimit gives a freshly written fake tool time for its first run,
// which a loaded macOS machine can stretch past clipboardReadLimit.
const fakeToolLimit = 30 * time.Second

func writeTool(t *testing.T, dir, name, body string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("fake clipboard tool is a #!/bin/sh script; not executable on Windows")
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o755))
}
