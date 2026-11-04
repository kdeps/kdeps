package screen

import (
	"os"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// editorResultMsg is the text an external editor wrote back. ok is false when
// the editor failed or the file could not be read, and the prompt stays.
type editorResultMsg struct {
	text string
	ok   bool
}

// promptEditor is the editor binary: $KDEPS_EDITOR, then $VISUAL, then $EDITOR, then vi.
func promptEditor() string {
	for _, key := range []string{"KDEPS_EDITOR", "VISUAL", "EDITOR"} {
		if v := os.Getenv(key); v != "" {
			return v
		}
	}
	return "vi"
}

// editorCommand is the process that edits path. The program attaches the
// terminal when it runs; the command's own streams stay unset until then.
// The binary is KDEPS_EDITOR, VISUAL, EDITOR, or vi, not a shell line.
func editorCommand(path string) *exec.Cmd {
	return exec.Command(promptEditor(), path) //nolint:gosec,noctx // user editor; tea owns the process
}

// writePromptFile seeds a temp file with the current prompt.
func writePromptFile(text string) (string, error) {
	f, err := os.CreateTemp("", "kdeps-prompt-*.md")
	if err != nil {
		return "", err
	}
	path := f.Name()
	_, err = f.WriteString(text)
	if err != nil {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}
	err = f.Close()
	if err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

// flattenPrompt makes editor text fit the one-line prompt. A newline becomes
// a space. Blank text is empty, which clears the prompt.
func flattenPrompt(raw string) string {
	raw = strings.TrimSpace(raw)
	return strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(raw)
}

// editorResult reads the file the editor wrote and removes it. A failed run
// or a missing file does not replace the prompt.
func editorResult(path string, runErr error) editorResultMsg {
	defer os.Remove(path)
	if runErr != nil {
		return editorResultMsg{}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return editorResultMsg{}
	}
	return editorResultMsg{text: flattenPrompt(string(data)), ok: true}
}

// defaultLaunchPromptEditor opens the prompt in the user's editor. Saving
// replaces the prompt; the line is not sent.
func defaultLaunchPromptEditor(text string) tea.Cmd {
	path, err := writePromptFile(text)
	if err != nil {
		return nil
	}
	return tea.ExecProcess(editorCommand(path), func(runErr error) tea.Msg {
		return editorResult(path, runErr)
	})
}

// openEditor starts the external editor on the current prompt.
func (m Model) openEditor() tea.Cmd {
	launch := m.launchEditor
	if launch == nil {
		launch = defaultLaunchPromptEditor
	}
	return launch(string(m.input))
}
