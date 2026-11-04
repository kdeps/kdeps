package screen

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// helpText lists the full-screen keys. ? in an empty prompt opens it, and
// ctrl+g opens it with the prompt left as it is.
const helpText = `keys
  enter/ctrl+g   send the prompt / key help
  up / down      scroll one row   ctrl+space folds
  pgup / pgdown  scroll a page
  end            jump to the newest output
  home           jump to the oldest output
  tab            complete a slash command, or switch pane
  ctrl+f/ctrl+z  feed / suspend
  ctrl+p/ctrl+n  recall earlier prompts
  ctrl+a/ctrl+e  start / end of the prompt
  ctrl+u/ctrl+k  delete to the start / end of the prompt
  ctrl+w         delete the word before the cursor
  ctrl+left/right previous / next word
  ctrl+y         put back the last killed text
  ctrl+c         clear the prompt; on an empty prompt, quit
  click          focus a pane, or move the prompt cursor
  shift+left/right select   ctrl+shift+c/x copy/cut pane
  ctrl+o         edit the prompt in your editor
press esc, enter, q, or ctrl+g to close`

// helpOverlay shows the key list and the slash commands until the user closes it.
type helpOverlay struct {
	commands []string
	done     bool
}

// Init has no startup command.
func (h helpOverlay) Init() tea.Cmd { return nil }

// Update closes the overlay on esc, enter, q, or ctrl+g.
func (h helpOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyPressMsg); ok {
		switch k.String() {
		case "esc", "enter", "q", "ctrl+g":
			h.done = true
		default:
			if k.Code == 'g' && k.Mod.Contains(tea.ModCtrl) && !k.Mod.Contains(tea.ModAlt) && !k.Mod.Contains(tea.ModShift) {
				h.done = true
			}
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
