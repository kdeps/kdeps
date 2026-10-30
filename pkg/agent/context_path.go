// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package agent

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"golang.org/x/term"

	executorLLM "github.com/kdeps/kdeps/v2/pkg/executor/llm"
)

// contextSegment is one labeled, sized contributor to the current turn's
// context (system prompt, memory, conversation history, a tool result, the
// goal directive) -- see contextSegments below.
type contextSegment struct {
	Label  string
	Tokens int
}

// contextSegments/contextPathModel track the active turn's context-path
// trail as package-level state, the same shape llm.TokenInputs/TokenOutputs
// already use for the token counter (tool_monitor.go's compactTokenStatus):
// only one turn is ever in flight per process, and the status line is read
// from a concurrent monitor goroutine while the turn goroutine keeps
// appending, so a mutex guards the slice (unlike a plain int64 counter,
// concurrent append + range over a growing slice is a real data race, not
// just a stale read).
//
//nolint:gochecknoglobals // mirrors llm.TokenInputs/TokenOutputs's existing pattern
var (
	contextSegmentsMu  sync.Mutex
	contextSegments    []contextSegment
	contextPathModel   string
	contextPathBackend string
)

// resetContextSegments clears the context-path trail for a new turn and
// records which model/backend it's being built for (contextPathStatus needs
// this to look up the context window -- a local backend's real window comes
// from executorLLM.LocalContextSize(), not the model-name lookup, which only
// covers cloud providers). Called once at the top of buildChatConfig, which
// every turn-starting path calls before any segment is recorded.
func resetContextSegments(model, backend string) {
	contextSegmentsMu.Lock()
	defer contextSegmentsMu.Unlock()
	contextSegments = nil
	executorLLM.ResetLastCallInput()
	contextPathModel = model
	contextPathBackend = backend
}

// contextWindowForBackend returns the effective context window for the
// active turn: a local backend's actual configured --ctx-size
// (executorLLM.LocalContextSize(), set from the servable model or
// KDEPS_CTX_SIZE) takes priority, since the static ContextWindowForModel
// table only knows cloud provider model names and would otherwise hide the
// context-path line for every local/llamafile/GGUF/Ollama session -- exactly
// the models this project cares most about supporting.
func contextWindowForBackend(model, backend string) int {
	switch backend {
	case executorLLM.BackendFile, executorLLM.BackendGGUF, modelTypeOllama:
		if n := executorLLM.LocalContextSize(); n > 0 {
			return n
		}
	}
	return ContextWindowForModel(model)
}

// recordContextSegment appends a labeled, sized contributor to the current
// turn's context-path trail. A no-op for empty text -- callers pass raw
// content and let this decide whether it's worth showing, so call sites
// never need their own empty-string guard.
func recordContextSegment(label, text string) {
	if text == "" {
		return
	}
	recordContextSegmentTokens(label, EstimateTokenCountFromStrings(text))
}

// recordContextSegmentTokens is recordContextSegment for a caller that
// already has a token count on hand (buildSystemPreamble caches the size of
// its two halves rather than re-measuring the already-built preamble string
// every turn) -- a no-op for a non-positive count.
// contextSegmentTotal is the token sum of the context path recorded so far.
func contextSegmentTotal() int {
	contextSegmentsMu.Lock()
	defer contextSegmentsMu.Unlock()
	n := 0
	for _, seg := range contextSegments {
		n += seg.Tokens
	}
	return n
}

func recordContextSegmentTokens(label string, tokens int) {
	if tokens <= 0 {
		return
	}
	contextSegmentsMu.Lock()
	defer contextSegmentsMu.Unlock()
	contextSegments = append(contextSegments, contextSegment{Label: label, Tokens: tokens})
}

// contextPathSegmentCap bounds how many groups contextPathStatus shows.
// Past this, older groups collapse into a "+N" marker instead of a line
// long enough to wrap. Identical labels are summed first, so twenty calls
// to the same tool are one group, not twenty copies of the same node.
const contextPathSegmentCap = 5

// statusWrapGuard leaves one column unused. A line that fills the width
// makes the terminal wrap, and the next tick then prints a new copy.
const statusWrapGuard = 1

const statusEllipsis = "..."

// stderrFD is the process stderr descriptor. term.GetSize takes the number
// so the status ticker does not call File.Fd, which clears non-blocking mode.
const stderrFD = 2

// contextPathStatus is the line above the session token counter. It is this
// turn's prompt, broken into where the tokens sit, plus how full the model's
// window is:
//
//	[turn 12.4k/200k | sys 8.1k | mem 1.2k | hist 2.4k | bash_exec 700]
//
// Empty when the model's context window is unknown (never a guessed max) or
// no segments have been recorded yet this turn.
func contextPathStatus() string {
	contextSegmentsMu.Lock()
	segs := make([]contextSegment, len(contextSegments))
	copy(segs, contextSegments)
	model, backend := contextPathModel, contextPathBackend
	contextSegmentsMu.Unlock()

	if len(segs) == 0 {
		return ""
	}
	maxWindow := contextWindowForBackend(model, backend)
	if maxWindow <= 0 {
		return ""
	}

	segs = reconcileSegments(segs, executorLLM.LastCallInputTokens())
	var current int
	order := make([]string, 0, len(segs))
	totals := make(map[string]int, len(segs))
	for _, seg := range segs {
		current += seg.Tokens
		label := contextSegmentLabel(seg.Label)
		if _, ok := totals[label]; !ok {
			order = append(order, label)
		}
		totals[label] += seg.Tokens
	}
	hidden := 0
	if len(order) > contextPathSegmentCap {
		hidden = len(order) - contextPathSegmentCap
		order = order[len(order)-contextPathSegmentCap:]
	}

	parts := []string{fmt.Sprintf("turn %s/%s",
		formatCompactCount(int64(current)), formatCompactCount(int64(maxWindow)))}
	if hidden > 0 {
		parts = append(parts, fmt.Sprintf("+%d", hidden))
	}
	for _, label := range order {
		parts = append(parts, fmt.Sprintf("%s %s", label, formatCompactCount(int64(totals[label]))))
	}
	return styleReplDim.Render("[") +
		styleReplMeta.Render(strings.Join(parts, " | ")) +
		styleReplDim.Render("] ")
}

// reconcileSegments makes the breakdown add up to the measured prompt size of
// the latest call. The segments are estimates taken before the call; the
// measurement is what the model actually received, including the user message,
// tool schemas and framing no segment covers. A shortfall becomes an "other"
// segment; an overshoot scales the estimates down. measured <= 0 (no call yet
// this turn) leaves the estimates as they are.
func reconcileSegments(segs []contextSegment, measured int64) []contextSegment {
	if measured <= 0 {
		return segs
	}
	var est int64
	for _, seg := range segs {
		est += int64(seg.Tokens)
	}
	switch {
	case est == measured:
		return segs
	case est < measured:
		return append(segs, contextSegment{Label: "other", Tokens: int(measured - est)})
	}
	out := make([]contextSegment, len(segs))
	var sum int64
	for i, seg := range segs {
		t := int64(seg.Tokens) * measured / est
		out[i] = contextSegment{Label: seg.Label, Tokens: int(t)}
		sum += t
	}
	if len(out) > 0 {
		out[len(out)-1].Tokens += int(measured - sum)
	}
	return out
}

// contextSegmentLabel is the short name shown for one recorded contributor.
// Tool results drop the "tool: " prefix and share a label per tool name, so
// repeated calls add to one number instead of another copy of the same node.
func contextSegmentLabel(label string) string {
	switch label {
	case "system prompt":
		return "sys"
	case "memory":
		return "mem"
	case "history":
		return "hist"
	case "goal":
		return "goal"
	default:
		return strings.TrimPrefix(label, "tool: ")
	}
}

// renderFrame builds the escape sequence for a status-frame redraw: move up
// to the top of whatever the previous frame occupied, erase downward, then
// write top (the turn breakdown, when non-empty) followed by bottom.
// Each line is cut to one row. A wrapped line used to be remembered as one
// row, so each tick left the previous copy on screen.
func renderFrame(prevRows int, top, bottom string, width int) (string, int) {
	top, bottom = fitStatusPair(top, bottom, width)
	rows := 0
	if top != "" {
		rows++
	}
	if bottom != "" {
		rows++
	}
	if rows < 1 {
		rows = 1
	}
	var sb strings.Builder
	sb.WriteString("\r")
	if prevRows > 1 {
		fmt.Fprintf(&sb, "\033[%dA", prevRows-1)
	}
	sb.WriteString("\033[0J")
	if top != "" {
		sb.WriteString(top)
		sb.WriteString("\r\n")
	}
	sb.WriteString(bottom)
	return sb.String(), rows
}

// fitStatusPair cuts both status rows to one terminal row.
func fitStatusPair(top, bottom string, width int) (string, string) {
	cols := width - statusWrapGuard
	if cols < 1 {
		cols = 1
	}
	return fitStatusLine(top, cols), fitStatusLine(bottom, cols)
}

// fitStatusLine cuts s to cols visible columns. ANSI color sequences are
// not columns. A cut line ends with "..." and a reset.
func fitStatusLine(s string, cols int) string {
	if s == "" || cols < 1 {
		return ""
	}
	if statusCols(s) <= cols {
		return s
	}
	budget := cols
	cut := ""
	if cols > len(statusEllipsis) {
		budget = cols - len(statusEllipsis)
		cut = statusEllipsis
	}
	return takeStatusCols(s, budget) + cut + ansiReset
}

func statusCols(s string) int {
	n := 0
	for i := 0; i < len(s); {
		if next := skipANSI(s, i); next != i {
			i = next
			continue
		}
		_, size := utf8.DecodeRuneInString(s[i:])
		n++
		i += size
	}
	return n
}

func takeStatusCols(s string, cols int) string {
	if cols <= 0 {
		return ""
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(s) && n < cols; {
		if next := skipANSI(s, i); next != i {
			b.WriteString(s[i:next])
			i = next
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		b.WriteRune(r)
		n++
		i += size
	}
	return b.String()
}

// skipANSI returns the index just after a CSI sequence at i, or i when
// s[i:] is not an escape.
func skipANSI(s string, i int) int {
	if !strings.HasPrefix(s[i:], "\033") {
		return i
	}
	j := i + 1
	if j >= len(s) {
		return len(s)
	}
	if s[j] != '[' {
		return j + 1
	}
	j++
	for j < len(s) {
		c := s[j]
		j++
		if (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			return j
		}
	}
	return j
}

// statusTermWidth is the real terminal width. terminalWidth() caps at 120
// for markdown; a status line wraps at the actual columns, so the erase
// count has to use those.
func statusTermWidth() int {
	w, _, err := term.GetSize(1)
	if err != nil || w <= 0 {
		return defaultTermWidth
	}
	return w
}

// visualRows is how many terminal rows s occupies at width columns.
// ANSI color codes are not columns. A trailing newline the caller adds
// itself is not part of s.
func visualRows(s string, width int) int {
	if s == "" {
		return 0
	}
	if width < 1 {
		width = 1
	}
	plain := ansiStripRe.ReplaceAllString(s, "")
	plain = strings.ReplaceAll(plain, "\r", "")
	rows := 0
	for _, line := range strings.Split(plain, "\n") {
		n := utf8.RuneCountInString(line)
		if n == 0 {
			rows++
			continue
		}
		rows += (n + width - 1) / width
	}
	return rows
}

// liveStatus is the one on-screen copy of the turn breakdown and the
// sent/generated counter. Spinner, thinking, and the tool monitor all draw
// through it, so a second owner replaces the same rows instead of leaving
// another copy underneath.
//
//nolint:gochecknoglobals // one status frame for the process
var liveStatus = struct {
	mu     sync.Mutex
	w      io.Writer
	rows   int
	logged string // last non-terminal frame, so a spinner tick does not append
}{}

// drawLiveStatus redraws the turn breakdown and the session counter in
// place, then trailer (the spinner glyph, or "bash_exec running").
func drawLiveStatus(w io.Writer, trailer string) {
	liveStatus.mu.Lock()
	defer liveStatus.mu.Unlock()
	drawLiveStatusLocked(w, trailer)
}

func drawLiveStatusLocked(w io.Writer, trailer string) {
	w = unwrapStatusWriter(w)
	width := statusWidth(w)
	glyph := ""
	if statusIsTTY(w) {
		glyph = trailer
	}
	top, bottom := fittedStatus(width, glyph)
	if setTUIStatus(top, bottom, glyph) {
		// The status window owns the rows now.
		return
	}
	if !statusIsTTY(w) {
		writeStatusLog(w, top, bottom)
		return
	}
	seq, rows := renderFrame(liveStatus.rows, top, bottom, width)
	fmt.Fprint(w, seq)
	liveStatus.rows = rows
	liveStatus.w = w
}

// fittedStatus is the turn row and the session row, each one terminal row.
// trailer is the spinner or the tool-monitor suffix, terminal only. The
// trailer is kept: a long session row is cut in front of it, not through it.
func fittedStatus(width int, trailer string) (string, string) {
	cols := width - statusWrapGuard
	if cols < 1 {
		cols = 1
	}
	top := fitStatusLine(contextPathStatus(), cols)
	bottom := compactTokenStatus()
	if trailer == "" {
		return top, fitStatusLine(bottom, cols)
	}
	suffix := "  " + trailer
	room := cols - statusCols(suffix)
	if room < 1 {
		return top, fitStatusLine(suffix, cols)
	}
	return top, fitStatusLine(bottom, room) + suffix
}

// writeStatusLog prints the two rows once. A pipe or a log ignores cursor
// movement, so a later tick with the same numbers writes nothing.
func writeStatusLog(w io.Writer, top, bottom string) {
	text := top + "\n" + bottom
	if text == liveStatus.logged {
		return
	}
	liveStatus.logged = text
	liveStatus.rows = 0
	liveStatus.w = w
	if top != "" {
		fmt.Fprintln(w, top)
	}
	fmt.Fprintln(w, bottom)
}

// unwrapStatusWriter draws on the raw terminal. crlfWriter turns a bare
// carriage return into a newline, which stacks one copy of the frame per tick.
func unwrapStatusWriter(w io.Writer) io.Writer {
	cw, ok := w.(*crlfWriter)
	if !ok || cw == nil || cw.w == nil {
		return w
	}
	return cw.w
}

func statusIsTTY(w io.Writer) bool {
	fd, ok := statusFd(w)
	if !ok {
		return true
	}
	return term.IsTerminal(fd)
}

func statusFd(w io.Writer) (int, bool) {
	switch w {
	case os.Stdout:
		return 1, true
	case os.Stderr:
		return stderrFD, true
	}
	f, ok := w.(*os.File)
	if !ok {
		return 0, false
	}
	return int(f.Fd()), true
}

func statusWidth(w io.Writer) int {
	if fd, ok := statusFd(w); ok {
		if width, _, err := term.GetSize(fd); err == nil && width > 1 {
			return width
		}
	}
	return statusTermWidth()
}

// eraseLiveStatus clears the status frame, if one is on screen.
func eraseLiveStatus(w io.Writer) {
	liveStatus.mu.Lock()
	defer liveStatus.mu.Unlock()
	eraseLiveStatusLocked(w)
}

func eraseLiveStatusLocked(w io.Writer) {
	if tuiRunning() {
		return
	}
	if liveStatus.rows == 0 {
		return
	}
	dest := liveStatus.w
	if dest == nil {
		dest = unwrapStatusWriter(w)
	}
	fmt.Fprint(dest, eraseFrame(liveStatus.rows))
	liveStatus.rows = 0
	liveStatus.logged = ""
}

// statusTail is the turn breakdown plus the session counter, as trailing
// rows of a block that is itself redrawn in place (the thinking pane).
func statusTail() (string, int) {
	top, bottom := fittedStatus(statusTermWidth(), "")
	var b strings.Builder
	rows := 0
	if top != "" {
		b.WriteString(top)
		b.WriteString("\r\n")
		rows++
	}
	b.WriteString(bottom)
	rows++
	return b.String(), rows
}

// eraseFrame builds the escape sequence to erase whatever the last frame
// (prevRows rows) drew, without writing a replacement -- used when a status
// line goroutine stops and self-cleans (see runToolMonitor, drawSpinnerFrames).
func eraseFrame(prevRows int) string {
	var sb strings.Builder
	sb.WriteString("\r")
	if prevRows > 1 {
		fmt.Fprintf(&sb, "\033[%dA", prevRows-1)
	}
	sb.WriteString("\033[0J")
	return sb.String()
}

// turnTokenBase is the session sent/generated total at the start of the current
// user turn, so the display can show what this turn alone consumed.
//
//nolint:gochecknoglobals // per-process turn baseline read by the live status frame
var turnTokenBase struct{ in, out atomic.Int64 }

// beginTurnTokens snapshots the session counters at the start of a user turn.
func beginTurnTokens() {
	turnTokenBase.in.Store(executorLLM.SessionInputTokens())
	turnTokenBase.out.Store(executorLLM.SessionOutputTokens())
}

// turnTokenDelta returns the tokens sent and generated since beginTurnTokens.
// A compaction or fold lowers the session totals mid-turn; the baseline is
// lowered with them so the delta never goes negative.
func turnTokenDelta() (int64, int64) {
	curIn, curOut := executorLLM.SessionInputTokens(), executorLLM.SessionOutputTokens()
	if curIn < turnTokenBase.in.Load() {
		turnTokenBase.in.Store(curIn)
	}
	if curOut < turnTokenBase.out.Load() {
		turnTokenBase.out.Store(curOut)
	}
	return curIn - turnTokenBase.in.Load(), curOut - turnTokenBase.out.Load()
}
