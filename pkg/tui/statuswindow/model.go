// Package statuswindow is the REPL's bottom status area: two status rows and a
// spinner, drawn by bubbletea. Lines sent as LogMsg print above the rows, so
// output never overwrites them.
package statuswindow

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

// LogMsg prints one line above the status rows.
type LogMsg struct{ Line string }

// StatusMsg replaces the two status rows and the spinner glyph.
type StatusMsg struct {
	Turn    string
	Session string
	Spinner string
}

// Model is the bubbletea model for the status area.
type Model struct {
	turn    string
	session string
	spinner string
}

// New returns a model with empty rows.
func New() Model { return Model{} }

// Init has no startup command.
func (m Model) Init() tea.Cmd { return nil }

// Update applies a LogMsg by printing it above the rows, or a StatusMsg by
// replacing the rows.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch v := msg.(type) {
	case LogMsg:
		return m, tea.Println(v.Line)
	case StatusMsg:
		m.turn, m.session, m.spinner = v.Turn, v.Session, v.Spinner
	}
	return m, nil
}

// View draws the two status rows, the second with the spinner.
func (m Model) render() string {
	second := m.session
	if m.spinner != "" {
		second += "  " + m.spinner
	}
	return strings.Join([]string{m.turn, second}, "\n")
}

// View draws the picker. It is a full-screen view on the alternate screen.
func (m Model) View() tea.View {
	v := tea.NewView(m.render())
	v.AltScreen = false
	return v
}
