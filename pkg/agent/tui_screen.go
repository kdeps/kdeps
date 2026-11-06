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

// screenCompletions is what Tab asks for in the full-screen prompt. An @ token
// completes files and memory ids. Anything else is the slash-command list.
func (r *REPL) screenCompletions(prefix string) []string {
	if rest, ok := strings.CutPrefix(prefix, "@"); ok {
		return r.mentionCompletions(rest)
	}
	return commandNames()
}

// mentionCompletions is the full @ tokens for prefix. A file and a memory id
// with the same name are one match, and the file is kept.
func (r *REPL) mentionCompletions(prefix string) []string {
	files, _ := doAtFileCompletion(r.loopCtx, prefix)
	mem := r.memoryKeyCompletions(prefix)
	out := make([]string, 0, len(files)+len(mem))
	seen := make(map[string]bool, len(files)+len(mem))
	add := func(suffix []rune) {
		full := "@" + prefix + string(suffix)
		if seen[full] {
			return
		}
		seen[full] = true
		out = append(out, full)
	}
	for _, s := range files {
		add(s)
	}
	for _, s := range mem {
		add(s)
	}
	return out
}

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
	m.Complete = r.screenCompletions
	// Everything off the UI loop reaches the program through the pump, so a
	// busy frame never blocks the agent loop. Overlays still use Send: their
	// caller waits for the user anyway.
	pump := screen.NewPump()
	var program *tea.Program
	setScreenStatusSink(func(text string) { pump.Latest("status", screen.StatusTextMsg{Text: text}) })
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
	turns := newSubmitQueue()
	defer turns.close()
	m.OnSubmit = turns.push
	// The filter thins trackpad wheel and motion floods before they queue
	// ahead of key presses in the update loop.
	program = tea.NewProgram(m, tea.WithOutput(origOut), tea.WithFilter(screen.NewInputFilter().Filter))
	defer attachScreenModel(pump)()
	go pump.Run(program.Send)
	defer pump.Close()
	go r.runSubmitted(turns, pump, origOut, program.Quit)
	go r.syncScreenPills()

	done := make(chan struct{})
	go func() {
		defer close(done)
		scanner := bufio.NewScanner(pr)
		for scanner.Scan() {
			pump.Output(scanner.Text())
		}
	}()

	// Show the counters before the first turn.
	drawLiveStatus(origOut, "")
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

// screenPillSend delivers pill values to the running full-screen program.
//
//nolint:gochecknoglobals // one full-screen view per process
var screenPillSend struct {
	sync.Mutex
	send func(msg screen.PillMsg)
}

// sessionPillIDLen is how many characters of a session id the session pill shows.
const sessionPillIDLen = 8

// newScreenModel is the full-screen view's starting state: the folder as its
// title, the active model, and the pills to show.
func newScreenModel(f *feed.Feed) screen.Model {
	m := screen.New(f)
	m.HistoryMax = promptHistoryLimit()
	path := historyPath()
	m = m.UseHistory(loadPromptHistory(path), func(line string) {
		appendPromptHistory(path, line)
	})
	m.Model = screenModelLabel()
	if wd, wdErr := os.Getwd(); wdErr == nil {
		m.Title = filepath.Base(wd)
	}
	return m
}

// defaultPromptHistory is the prompt recall depth when the event is missing.
const defaultPromptHistory = 100

// promptHistoryScan is the scanner's starting buffer. promptHistoryMax is
// the longest history line it will accept.
const (
	promptHistoryScan = 64 << 10
	promptHistoryMax  = 1 << 20
)

// loadPromptHistory reads the line editor's history file. Readline stores one
// prompt per line. A missing file is an empty history.
func loadPromptHistory(path string) []string {
	if path == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, promptHistoryScan), promptHistoryMax)
	var lines []string
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	return lines
}

// appendPromptHistory adds one prompt to the line editor's history file.
// A line that contains a newline would split into two readline entries, so
// it is not written.
func appendPromptHistory(path, line string) {
	if path == "" || strings.TrimSpace(line) == "" || strings.ContainsAny(line, "\r\n") {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = io.WriteString(f, line+"\n")
}

// promptHistoryLimit is how many prompts the full-screen view recalls: the
// prompt-history-limit event's items. A disabled event keeps every prompt (0).
func promptHistoryLimit() int {
	n := effectiveItems(eventPromptHistoryLimit, defaultPromptHistory)
	if n == disabledSentinel {
		return 0
	}
	return n
}

// sendScreenPill sets a pill's value in the running view. It does nothing when
// the view is off.
func sendScreenPill(name, value string) {
	screenPillSend.Lock()
	send := screenPillSend.send
	screenPillSend.Unlock()
	if send != nil {
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

// FullScreenEnabled reports whether the full-screen view is on: running now,
// or chosen for this process (see tuiEnabled). The running check comes first
// because the view redirects stdout into a pipe, so tuiEnabled's terminal
// check is false for as long as the view is up.
func FullScreenEnabled() bool { return screenRunning() || tuiEnabled() }

// screenRunning reports whether a full-screen view is up and taking overlays.
func screenRunning() bool {
	screenOverlay.Lock()
	defer screenOverlay.Unlock()
	return screenOverlay.send != nil
}

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

// attachScreenModel makes setScreenModel, feed redraws and pills reach the
// view through pump. Each kind keeps only its newest value. The returned
// function detaches them again.
func attachScreenModel(pump *screen.Pump) func() {
	screenModelSend.Lock()
	screenModelSend.send = func(msg screen.ModelMsg) { pump.Latest("model", msg) }
	screenModelSend.Unlock()
	tuiFeedSink.Lock()
	tuiFeedSink.redraw = func() { pump.Latest("redraw", screen.RedrawMsg{}) }
	tuiFeedSink.Unlock()
	screenPillSend.Lock()
	screenPillSend.send = func(msg screen.PillMsg) { pump.Latest("pill:"+msg.Name, msg) }
	screenPillSend.Unlock()
	return func() {
		tuiFeedSink.Lock()
		tuiFeedSink.redraw = nil
		tuiFeedSink.Unlock()
		screenModelSend.Lock()
		screenModelSend.send = nil
		screenModelSend.Unlock()
		screenPillSend.Lock()
		screenPillSend.send = nil
		screenPillSend.Unlock()
	}
}

// submitQueue holds submitted lines until the turn before them ends. push
// never blocks, so the UI loop can call it.
type submitQueue struct {
	mu     sync.Mutex
	lines  []string
	closed bool
	wake   chan struct{}
}

func newSubmitQueue() *submitQueue { return &submitQueue{wake: make(chan struct{}, 1)} }

// push queues one line.
func (q *submitQueue) push(line string) {
	q.mu.Lock()
	if !q.closed {
		q.lines = append(q.lines, line)
	}
	q.mu.Unlock()
	q.signal()
}

// close drops queued lines and makes pop return false.
func (q *submitQueue) close() {
	q.mu.Lock()
	q.closed = true
	q.lines = nil
	q.mu.Unlock()
	q.signal()
}

// pop waits for the next line. It returns false after close.
func (q *submitQueue) pop() (string, bool) {
	for {
		q.mu.Lock()
		if q.closed {
			q.mu.Unlock()
			return "", false
		}
		if len(q.lines) > 0 {
			line := q.lines[0]
			q.lines = q.lines[1:]
			q.mu.Unlock()
			return line, true
		}
		q.mu.Unlock()
		<-q.wake
	}
}

func (q *submitQueue) signal() {
	select {
	case q.wake <- struct{}{}:
	default:
	}
}

// runSubmitted runs queued lines one at a time, in the order they were sent,
// so a prompt typed during a turn waits for that turn instead of racing it.
// It quits the view once the loop ends (/exit).
func (r *REPL) runSubmitted(turns *submitQueue, pump *screen.Pump, statusOut io.Writer, quit func()) {
	for {
		line, ok := turns.pop()
		if !ok {
			return
		}
		if err := r.processInput(line); err != nil {
			pump.Output("error: " + err.Error())
		}
		// The status box only refreshes during a turn; show the counters after it too.
		drawLiveStatus(statusOut, "")
		if r.loopCtx.Err() != nil {
			quit()
			return
		}
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
