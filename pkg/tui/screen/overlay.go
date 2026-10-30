package screen

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Overlay is a view that takes the keys and the main area until it is done,
// for example the session selector or the model picker. It is one program
// with the screen: the screen forwards messages to it and draws its View in
// place of the main pane.
type Overlay interface {
	tea.Model
	// Done reports that the overlay has a result and should close.
	Done() bool
}

// WithOverlay opens o over the main pane. When o reports Done, the screen
// closes it and calls done with the finished overlay, so the caller can read
// its result.
func (m Model) WithOverlay(o Overlay, done func(Overlay)) Model {
	m.overlay = o
	m.overlayDone = done
	return m
}

// HasOverlay reports whether an overlay is open.
func (m Model) HasOverlay() bool { return m.overlay != nil }

// updateOverlay forwards msg to the open overlay. When the overlay is done it
// is closed and its done callback runs.
func (m Model) updateOverlay(msg tea.Msg) (Model, tea.Cmd) {
	next, cmd := m.overlay.Update(msg)
	o, ok := next.(Overlay)
	if !ok {
		return m, cmd
	}
	m.overlay = o
	if !o.Done() {
		return m, cmd
	}
	finished, done := m.overlay, m.overlayDone
	m.overlay, m.overlayDone = nil, nil
	if done != nil {
		done(finished)
	}
	return m, cmd
}
