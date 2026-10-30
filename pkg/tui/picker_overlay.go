package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// PickerOverlay is the list picker as an overlay for the full-screen view: it
// takes the keys until the user picks or cancels, then reports Done.
type PickerOverlay struct {
	m listPickerModel
}

// NewPickerOverlay builds the picker for the full-screen view.
func NewPickerOverlay(title string, items []ListItem, onDelete func(id string) (bool, error)) PickerOverlay {
	m := newListPickerModel(title, items, onDelete)
	m.embedded = true
	return PickerOverlay{m: m}
}

// Init has no startup command.
func (p PickerOverlay) Init() tea.Cmd { return nil }

// Update forwards the message to the picker.
func (p PickerOverlay) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := p.m.Update(msg)
	if m, ok := next.(listPickerModel); ok {
		p.m = m
	}
	return p, cmd
}

// View draws the picker.
func (p PickerOverlay) View() string { return p.m.View() }

// Done reports that the user picked a row or cancelled.
func (p PickerOverlay) Done() bool { return p.m.quitted || p.m.cancelled }

// Result returns the chosen item's id, and false when the user cancelled or
// nothing was chosen.
func (p PickerOverlay) Result() (string, bool) {
	if p.m.cancelled || !p.m.quitted {
		return "", false
	}
	flat := p.m.filtered()
	if p.m.cursor >= 0 && p.m.cursor < len(flat) {
		return flat[p.m.cursor].ID, true
	}
	return "", false
}
