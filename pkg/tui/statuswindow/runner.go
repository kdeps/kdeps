package statuswindow

import (
	"io"

	tea "charm.land/bubbletea/v2"
)

const (
	statusWidth  = 200
	statusHeight = 3
)

// Runner drives a status window program on a writer. It takes no keyboard
// input, so the REPL keeps reading its own input.
type Runner struct {
	program *tea.Program
	done    chan struct{}
}

// Start runs the status window on out until Stop.
func Start(out io.Writer) *Runner {
	// A non-terminal writer reports no size, and v2 draws nothing without one.
	// The status rows are two lines, so a fixed width only bounds the text.
	p := tea.NewProgram(New(), tea.WithOutput(out), tea.WithInput(nil), tea.WithoutSignalHandler(),
		tea.WithWindowSize(statusWidth, statusHeight))
	r := &Runner{program: p, done: make(chan struct{})}
	go func() {
		_, _ = p.Run()
		close(r.done)
	}()
	return r
}

// Set replaces the two status rows and the spinner.
func (r *Runner) Set(turn, session, spinner string) {
	r.program.Send(StatusMsg{Turn: turn, Session: session, Spinner: spinner})
}

// Log prints one line above the status rows.
func (r *Runner) Log(line string) {
	r.program.Send(LogMsg{Line: line})
}

// Stop quits the program and waits for its final frame.
func (r *Runner) Stop() {
	r.program.Quit()
	<-r.done
}
