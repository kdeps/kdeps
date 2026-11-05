package agent

import (
	"io"
	"os"
	"sync"

	"golang.org/x/term"

	"github.com/kdeps/kdeps/v2/pkg/tui/statuswindow"
)

// tuiEnv forces the full-screen view on ("1") or off ("0"). Unset, the view is
// on when stdin and stdout are both terminals, so pipes and scripts keep the
// line REPL.
const tuiEnv = "KDEPS_TUI"

// stdoutIsTerminal reports whether stdout is a terminal (a var for tests).
//
//nolint:gochecknoglobals // swapped in tests
var stdoutIsTerminal = func() bool { return term.IsTerminal(int(os.Stdout.Fd())) }

// tuiStatus holds the running status window, if any.
//
//nolint:gochecknoglobals // one status window per process
var tuiStatus struct {
	sync.Mutex
	run *statuswindow.Runner
}

// tuiEnabled reports whether the full-screen view runs: KDEPS_TUI=1 or 0
// decides; otherwise it runs when stdin and stdout are terminals.
func tuiEnabled() bool {
	switch os.Getenv(tuiEnv) {
	case "1":
		return true
	case "0":
		return false
	}
	return stdinIsTerminal() && stdoutIsTerminal()
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
