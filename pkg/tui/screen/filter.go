package screen

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// inputSampleGap is the shortest gap between two mouse samples the view takes:
// about one 60 Hz frame. A trackpad sends wheel and motion events far faster
// than a frame, and each one queued ahead of a key press delays that key.
const inputSampleGap = 16 * time.Millisecond

// WheelMsg is several wheel notches over one pane, folded into one event by
// InputFilter. Steps is positive for up (into history), negative for down.
type WheelMsg struct {
	X, Y  int
	Steps int
}

// InputFilter thins mouse input before it reaches the update loop: wheel
// notches inside one sample gap are summed into one WheelMsg, and motion
// events inside one gap are dropped. Pass Filter to tea.WithFilter.
type InputFilter struct {
	now        func() time.Time
	lastWheel  time.Time
	lastMotion time.Time
	notches    int
}

// NewInputFilter returns a filter that reads the wall clock.
func NewInputFilter() *InputFilter { return &InputFilter{now: time.Now} }

// Filter returns msg, a folded WheelMsg, or nil to drop a sample.
func (f *InputFilter) Filter(_ tea.Model, msg tea.Msg) tea.Msg {
	switch v := msg.(type) {
	case tea.MouseWheelMsg:
		var step int
		switch v.Button {
		case tea.MouseWheelUp:
			step = 1
		case tea.MouseWheelDown:
			step = -1
		default:
			return msg
		}
		if f.notches*step < 0 {
			f.notches = 0 // the direction turned: drop the other way's notches
		}
		f.notches += step
		if !f.due(&f.lastWheel) {
			return nil
		}
		out := WheelMsg{X: v.X, Y: v.Y, Steps: f.notches}
		f.notches = 0
		return out
	case tea.MouseMotionMsg:
		if !f.due(&f.lastMotion) {
			return nil
		}
	}
	return msg
}

// due reports whether a sample gap has passed since *last, and if so moves
// *last to now.
func (f *InputFilter) due(last *time.Time) bool {
	now := f.now()
	if !last.IsZero() && now.Sub(*last) < inputSampleGap {
		return false
	}
	*last = now
	return true
}
