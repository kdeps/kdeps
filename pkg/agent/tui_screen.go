package agent

import (
	"bufio"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	tea "charm.land/bubbletea/v2"

	"github.com/kdeps/kdeps/v2/pkg/tui"
	"github.com/kdeps/kdeps/v2/pkg/tui/feed"
	"github.com/kdeps/kdeps/v2/pkg/tui/screen"
)

// runFullScreen runs the REPL in the full-screen view: the main pane, the feed,
// the status box and the prompt. Lines the REPL prints to stdout or stderr are
// captured into the main pane. It returns when the loop ends (/exit, EOF or
// Ctrl+C).
func (r *REPL) runFullScreen() error {
	f := &feed.Feed{}
	setScreenModel(r.loop.config.Model)
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

	m := newScreenModel(f)
	m.Complete = func(string) []string { return commandNames() }
	var program *tea.Program
	setScreenStatusSink(func(text string) { program.Send(screen.StatusTextMsg{Text: text}) })
	defer setScreenStatusSink(nil)
	closed := make(chan struct{})
	setScreenOverlaySink(func(msg screen.OpenOverlayMsg) { program.Send(msg) }, closed)
	defer func() {
		close(closed)
		setScreenOverlaySink(nil, nil)
	}()
	if r.loop.config.OfferSessionPicker {
		m = r.withSessionPicker(m)
	}
	m.OnSubmit = func(line string) {
		if procErr := r.processInput(line); procErr != nil {
			program.Send(screen.OutputMsg{Text: "error: " + procErr.Error()})
		}
		// The status box only refreshes during a turn; show the counters after it too.
		drawLiveStatus(origOut, "")
		if r.loopCtx.Err() != nil {
			program.Quit()
		}
	}
	program = tea.NewProgram(m, tea.WithOutput(origOut))
	defer attachScreenModel(program)()
	go r.syncScreenPills()

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			program.Send(screen.OutputMsg{Text: scanner.Text()})
		}
	}()

	// Show the counters before the first turn. Send blocks until the program
	// runs, so it goes from a goroutine.
	go drawLiveStatus(origOut, "")
	_, runErr := program.Run()
	// Other goroutines may still hold the write end, so closing ours does not
	// end the scanner. Restore the streams and close the read end to unblock it.
	os.Stdout, os.Stderr = origOut, origErr //nolint:reassign // the view has ended
	_ = pw.Close()
	_ = pr.Close()
	<-done
	if runErr == nil || errors.Is(runErr, io.EOF) {
		return nil
	}
	return runErr
}

// screenOverlay opens overlays in the running full-screen view. closed is
// closed when the view ends, so a caller waiting on an overlay stops waiting.
//
//nolint:gochecknoglobals // one full-screen view per process
var screenOverlay struct {
	sync.Mutex
	send   func(msg screen.OpenOverlayMsg)
	closed chan struct{}
}

// setScreenOverlaySink sets (or with nil clears) how overlays reach the view.
func setScreenOverlaySink(send func(msg screen.OpenOverlayMsg), closed chan struct{}) {
	screenOverlay.Lock()
	defer screenOverlay.Unlock()
	screenOverlay.send = send
	screenOverlay.closed = closed
}

// screenModel is the model name the full-screen status box shows. It is set
// from the goroutines that change the model, so reads and writes are locked.
//
//nolint:gochecknoglobals // one full-screen view per process
var screenModel struct {
	sync.Mutex
	name string
}

// screenModelSend delivers model names to the running full-screen program.
//
//nolint:gochecknoglobals // one full-screen view per process
var screenModelSend struct {
	sync.Mutex
	send func(msg screen.ModelMsg)
}

// screenPillSend delivers pill values to the running full-screen program. Only
// the pills named in KDEPS_TUI_PILLS are sent.
//
//nolint:gochecknoglobals // one full-screen view per process
var screenPillSend struct {
	sync.Mutex
	send    func(msg screen.PillMsg)
	enabled map[string]bool
}

// screenPillKinds are the pills the full-screen view can show.
func screenPillKinds() []string { return []string{"backend", "session", "tool"} }

// sessionPillIDLen is how many characters of a session id the session pill shows.
const sessionPillIDLen = 8

// newScreenModel is the full-screen view's starting state: the folder as its
// title, the active model, and the pills to show.
func newScreenModel(f *feed.Feed) screen.Model {
	m := screen.New(f)
	m.Model = screenModelLabel()
	m.PillNames = screenPillNames()
	if wd, wdErr := os.Getwd(); wdErr == nil {
		m.Title = filepath.Base(wd)
	}
	return m
}

// screenPillNames is the pills to show, from KDEPS_TUI_PILLS: a comma-separated
// list of backend, session and tool, in that order. Unset shows all three; an
// empty value shows none. Unknown names are ignored.
func screenPillNames() []string {
	raw, set := os.LookupEnv("KDEPS_TUI_PILLS")
	kinds := screenPillKinds()
	if !set {
		return kinds
	}
	want := map[string]bool{}
	for _, name := range strings.Split(raw, ",") {
		want[strings.TrimSpace(name)] = true
	}
	var out []string
	for _, kind := range kinds {
		if want[kind] {
			out = append(out, kind)
		}
	}
	return out
}

// sendScreenPill sets a pill's value in the running view. It does nothing when
// the view is off or the pill is not enabled.
func sendScreenPill(name, value string) {
	screenPillSend.Lock()
	send, on := screenPillSend.send, screenPillSend.enabled[name]
	screenPillSend.Unlock()
	if send != nil && on {
		send(screen.PillMsg{Name: name, Value: value})
	}
}

// setScreenModel records the active model for the full-screen status box and
// sends it to the running view, when there is one.
func setScreenModel(name string) {
	screenModel.Lock()
	screenModel.name = name
	screenModel.Unlock()
	screenModelSend.Lock()
	send := screenModelSend.send
	screenModelSend.Unlock()
	if send != nil {
		send(screen.ModelMsg{Name: name})
	}
}

// screenModelLabel is the active model's name, or "" before one is set.
func screenModelLabel() string {
	screenModel.Lock()
	defer screenModel.Unlock()
	return screenModel.name
}

// FullScreenEnabled reports whether the full-screen view is on (KDEPS_TUI=1).
func FullScreenEnabled() bool { return tuiEnabled() }

// RunInScreen opens o over the full-screen view and blocks until the user
// closes it. It returns the finished overlay, or false when no full-screen view
// is running. Call it from the REPL's command goroutine, never from Update.
func RunInScreen(o screen.Overlay) (screen.Overlay, bool) {
	screenOverlay.Lock()
	send, closed := screenOverlay.send, screenOverlay.closed
	screenOverlay.Unlock()
	if send == nil {
		return nil, false
	}
	finished := make(chan screen.Overlay, 1)
	send(screen.OpenOverlayMsg{Overlay: o, Done: func(f screen.Overlay) tea.Cmd {
		finished <- f
		return nil
	}})
	select {
	case f := <-finished:
		return f, true
	case <-closed:
		return nil, false
	}
}

// withSessionPicker opens the resume picker over the main pane when this
// folder has saved sessions.
func (r *REPL) withSessionPicker(m screen.Model) screen.Model {
	store := r.loop.Store()
	if store == nil {
		return m
	}
	items := sessionPickerItems(store)
	if items == nil {
		return m
	}
	picker := tui.NewPickerOverlay("Resume a session in this folder?", items, sessionPickerOnDelete(store))
	return m.WithOverlay(picker, r.resumeChosenSession(store))
}

// resumeChosenSession loads the session the user picked. Starting fresh and
// cancelling load nothing; a load error is printed to the main pane.
func (r *REPL) resumeChosenSession(store *SessionStore) func(screen.Overlay) tea.Cmd {
	return func(o screen.Overlay) tea.Cmd {
		picker, ok := o.(tui.PickerOverlay)
		if !ok {
			return nil
		}
		id, chosen := picker.Result()
		if !chosen || id == newSessionSentinel {
			return nil
		}
		return func() tea.Msg {
			if err := r.loadSession(store, id); err != nil {
				return screen.OutputMsg{Text: "error: " + err.Error()}
			}
			return nil
		}
	}
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

// attachScreenModel makes setScreenModel send names to program. The returned
// function detaches it again.
func attachScreenModel(program *tea.Program) func() {
	screenModelSend.Lock()
	screenModelSend.send = func(msg screen.ModelMsg) { program.Send(msg) }
	screenModelSend.Unlock()
	screenPillSend.Lock()
	screenPillSend.send = func(msg screen.PillMsg) { program.Send(msg) }
	screenPillSend.enabled = map[string]bool{}
	for _, name := range screenPillNames() {
		screenPillSend.enabled[name] = true
	}
	screenPillSend.Unlock()
	return func() {
		screenModelSend.Lock()
		screenModelSend.send = nil
		screenModelSend.Unlock()
		screenPillSend.Lock()
		screenPillSend.send = nil
		screenPillSend.Unlock()
	}
}

// sessionPillValue is the session pill: "new" before a session is saved, else
// the first sessionPillIDLen characters of its id.
func sessionPillValue(id string) string {
	if id == "" {
		return "new"
	}
	if len(id) > sessionPillIDLen {
		return id[:sessionPillIDLen]
	}
	return id
}

// syncScreenPills sends the backend and session pills from the REPL goroutine,
// which is the one that changes the loop's config.
func (r *REPL) syncScreenPills() {
	sendScreenPill("backend", r.loop.config.Backend)
	sendScreenPill("session", sessionPillValue(r.loop.SessionID()))
}
