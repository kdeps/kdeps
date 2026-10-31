package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// helpText lists the full-screen keys, shown by typing ? into an empty prompt.
const helpText = `keys

  enter          send the prompt
  up / down      scroll the main pane
  pgup / pgdown  scroll a page
  end            jump to the newest output
  tab            complete a slash command, else switch focus between panes
  ctrl+f         show or hide the feed
  ctrl+p/ctrl+n  recall earlier prompts
  ctrl+a/ctrl+e  start / end of the prompt
  ctrl+u/ctrl+k  delete to the start / end of the prompt
  ctrl+w         delete the word before the cursor
  ctrl+c         clear the prompt; on an empty prompt, quit

press esc, enter or q to close`

// helpOverlay shows the key list and the slash commands until the user closes it.
type helpOverlay struct {
	commands []string
	done     bool
}

// Init has no startup command.
func (h helpOverlay) Init() tea.Cmd { return nil }

// Update closes the overlay on esc, enter or q.
func (h helpOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "enter", "q":
			h.done = true
		}
	}
	return h, nil
}

// View draws the key list, then the slash commands.
func (h helpOverlay) View() tea.View {
	if len(h.commands) == 0 {
		return tea.NewView(helpText)
	}
	return tea.NewView(helpText + "\n\ncommands\n  " + strings.Join(h.commands, "  "))
}

// Done reports that the user closed the help.
func (h helpOverlay) Done() bool { return h.done }
