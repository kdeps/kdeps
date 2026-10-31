package agent

import (
	"io"
	"os"
	"sync"

	"github.com/kdeps/kdeps/v2/pkg/tui/statuswindow"
)

// tuiEnv turns the bubbletea status window on for the REPL. It is off by
// default, so the hand-drawn status rows stay the default until the window is
// tested in a real terminal.
const tuiEnv = "KDEPS_TUI"

// tuiStatus holds the running status window, if any.
//
//nolint:gochecknoglobals // one status window per process
var tuiStatus struct {
	sync.Mutex
	run *statuswindow.Runner
}

// tuiEnabled reports whether KDEPS_TUI asks for the status window.
func tuiEnabled() bool {
	return os.Getenv(tuiEnv) == "1"
}

// startTUIStatus runs the status window on out and returns the function that
// stops it. When the window is off it returns a no-op.
func startTUIStatus(out io.Writer) func() {
	tuiStatus.Lock()
	tuiStatus.run = statuswindow.Start(out)
	tuiStatus.Unlock()
	return func() {
		tuiStatus.Lock()
		run := tuiStatus.run
		tuiStatus.run = nil
		tuiStatus.Unlock()
		if run != nil {
			run.Stop()
		}
	}
}

// setTUIStatus sends the two rows and the spinner to the window. It reports
// false when no window is running, so the caller draws the rows itself.
func setTUIStatus(top, bottom, glyph string) bool {
	tuiStatus.Lock()
	defer tuiStatus.Unlock()
	if tuiStatus.run == nil {
		return false
	}
	tuiStatus.run.Set(top, bottom, glyph)
	return true
}

// tuiRunning reports whether a status window is up.
func tuiRunning() bool {
	tuiStatus.Lock()
	defer tuiStatus.Unlock()
	return tuiStatus.run != nil
}
