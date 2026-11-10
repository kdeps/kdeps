package screen

import (
	tea "charm.land/bubbletea/v2"

	"github.com/kdeps/kdeps/v3/pkg/tui/layout"
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
	if size, ok := m.overlaySize(); ok {
		// Lay the overlay out for the pane it is drawn in from the first frame.
		if next, _ := o.Update(size); next != nil {
			if sized, isOverlay := next.(Overlay); isOverlay {
				m.overlay = sized
			}
		}
	}
	return m
}

// overlaySize is the main pane's inside, where the overlay is drawn. It is
// false before the first resize.
func (m Model) overlaySize() (tea.WindowSizeMsg, bool) {
	if m.width == 0 || m.height == 0 {
		return tea.WindowSizeMsg{}, false
	}
	f := m.frame()
	return tea.WindowSizeMsg{Width: layout.InnerWidth(f.Main.W), Height: max(f.Main.H-feedBorder, 1)}, true
}

// HasOverlay reports whether an overlay is open.
func (m Model) HasOverlay() bool { return m.overlay != nil }

// toOverlay gives the open overlay every message. A resize sizes the screen
// and reaches the overlay as the main pane's inside, where it is drawn.
func (m Model) toOverlay(msg tea.Msg) (Model, tea.Cmd) {
	if ws, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = ws.Width, ws.Height
		m = m.clampScroll()
		if size, sized := m.overlaySize(); sized {
			msg = size
		}
	}
	return m.updateOverlay(msg)
}

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
