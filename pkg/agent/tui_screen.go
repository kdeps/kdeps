package agent

import (
	"bufio"
	"errors"
	"io"
	"os"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
	"github.com/kdeps/kdeps/v2/pkg/tui/screen"
)

// runFullScreen runs the REPL in the full-screen view: the main pane, the feed,
// the status box and the prompt. Lines the REPL prints to stdout or stderr are
// captured into the main pane. It returns when the loop ends (/exit, EOF or
// Ctrl+C).
func (r *REPL) runFullScreen() error {
	f := &feed.Feed{}
	setTUIFeed(f)
	defer setTUIFeed(nil)

	origOut, origErr := os.Stdout, os.Stderr
	pr, pw, err := os.Pipe()
	if err != nil {
		return err
	}
	// Capturing the REPL's stdout and stderr is the point of this view.
	os.Stdout, os.Stderr = pw, pw                              //nolint:reassign // the full-screen view owns the output
	defer func() { os.Stdout, os.Stderr = origOut, origErr }() //nolint:reassign // restores the captured streams

	m := screen.New(f)
	var program *tea.Program
	setScreenStatusSink(func(text string) { program.Send(screen.StatusTextMsg{Text: text}) })
	defer setScreenStatusSink(nil)
	m.OnSubmit = func(line string) {
		if procErr := r.processInput(line); procErr != nil {
			program.Send(screen.OutputMsg{Text: "error: " + procErr.Error()})
		}
		if r.loopCtx.Err() != nil {
			program.Quit()
		}
	}
	program = tea.NewProgram(m, tea.WithOutput(origOut))

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			program.Send(screen.OutputMsg{Text: scanner.Text()})
		}
	}()

	_, runErr := program.Run()
	_ = pw.Close()
	<-done
	_ = pr.Close()
	if runErr == nil || errors.Is(runErr, io.EOF) {
		return nil
	}
	return runErr
}

// screenStatusSink receives the status text while the full-screen view runs.
//
//nolint:gochecknoglobals // one full-screen view per process
var screenStatusSink struct {
	sync.Mutex
	send func(text string)
}

// setScreenStatusSink sets (or with nil clears) the status box's text sink.
func setScreenStatusSink(send func(text string)) {
	screenStatusSink.Lock()
	defer screenStatusSink.Unlock()
	screenStatusSink.send = send
}

// sendScreenStatus sends the status text to the full-screen status box. It
// reports false when the full-screen view is not running.
func sendScreenStatus(text string) bool {
	screenStatusSink.Lock()
	defer screenStatusSink.Unlock()
	if screenStatusSink.send == nil {
		return false
	}
	screenStatusSink.send(text)
	return true
}
