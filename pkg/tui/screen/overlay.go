package screen

import (
	tea "charm.land/bubbletea/v2"
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

// OpenOverlayMsg opens Overlay in the screen. Done runs once when the overlay
// reports Done, with the finished overlay, and returns a command for any
// follow-up work so the screen never does IO in Update.
type OpenOverlayMsg struct {
	Overlay Overlay
	Done    func(Overlay) tea.Cmd
}

// WithOverlay opens o over the main pane. done runs when o reports Done.
func (m Model) WithOverlay(o Overlay, done func(Overlay) tea.Cmd) Model {
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
		return m, tea.Batch(cmd, done(finished))
	}
	return m, cmd
}
