package m365

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/websocket"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// CopilotStream is the streamed result of one chat turn. Consume Deltas until it
// closes, then read the metadata accessors and check Err.
type CopilotStream struct {
	deltas    chan string
	thinking  chan string
	thinkOnce map[string]bool

	// writeMu serializes every conn.WriteMessage call. gorilla/websocket
	// requires a single writer at a time; the stop-on-cancel goroutine in
	// run() can otherwise write concurrently with the main read/dispatch
	// loop and panic ("concurrent write to websocket connection").
	writeMu sync.Mutex

	mu            sync.Mutex
	answer        string
	received      bool
	throttle      *Throttle
	contentOrigin string
	messageType   string
	messageID     string
	maxScores     map[string]float64
	turnCount     *int
	turnState     string
	sawAction     bool
	err           error
}

// Deltas returns the channel of streamed answer suffixes. It is closed when the
// turn completes.
func (s *CopilotStream) Deltas() <-chan string { return s.deltas }

// ThinkingDeltas returns the channel of the model's reasoning/chain-of-thought
// summary text, one message at a time as the service sends it. It is closed
// when the turn completes. This is separate from Deltas because reasoning
// text is always safe to show live, even on a tool-calling turn where the
// answer itself is buffered until parsed (it might contain in-progress
// fenced tool-call syntax that shouldn't leak to the user raw).
func (s *CopilotStream) ThinkingDeltas() <-chan string { return s.thinking }

// FullText returns the reconstructed answer accumulated so far.
func (s *CopilotStream) FullText() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.answer
}

// HasContent reports whether any answer text was received.
func (s *CopilotStream) HasContent() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.received
}

// Throttle returns the conversation quota info, or nil if none was reported.
func (s *CopilotStream) Throttle() *Throttle {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.throttle
}

// ContentOrigin returns the server's content-origin label (e.g. DeepLeo).
func (s *CopilotStream) ContentOrigin() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.contentOrigin
}

// MessageType returns the last bot messageType (e.g. Disengaged, EndOfRequest).
func (s *CopilotStream) MessageType() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.messageType
}

// MessageID returns the server-assigned message id.
func (s *CopilotStream) MessageID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.messageID
}

// Scores returns the highest per-component classifier score seen this turn.
func (s *CopilotStream) Scores() map[string]float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.maxScores) == 0 {
		return nil
	}
	out := make(map[string]float64, len(s.maxScores))
	maps.Copy(out, s.maxScores)
	return out
}

// TurnCount returns the server-side conversation turn count, or nil.
func (s *CopilotStream) TurnCount() *int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turnCount
}

// TurnState returns the server-reported turn state (e.g. Completed).
func (s *CopilotStream) TurnState() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turnState
}

// SawAction reports whether the model triggered a native custom action.
func (s *CopilotStream) SawAction() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sawAction
}

// Err returns the terminal error, if the turn failed.
func (s *CopilotStream) Err() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.err
}

// setErr records a terminal error once.
func (s *CopilotStream) setErr(err error) {
	s.mu.Lock()
	if s.err == nil {
		s.err = err
	}
	s.mu.Unlock()
}

// writeMessage sends a text frame, serialized against every other writer on
// conn (every SignalR frame this client sends is text).
func (s *CopilotStream) writeMessage(conn *websocket.Conn, data []byte) error {
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return conn.WriteMessage(websocket.TextMessage, data)
}

// advance folds an incoming text piece and emits any new suffix.
func (s *CopilotStream) advance(next string) {
	s.mu.Lock()
	newAnswer, emit, emitted := foldStreamText(s.answer, next)
	if newAnswer != s.answer {
		s.answer = newAnswer
		s.received = true
	}
	s.mu.Unlock()
	if emitted && emit != "" {
		s.deltas <- emit
	}
}

// isChainOfThoughtSummary reports whether a bot message is one of the
// service's reasoning/"thinking" progress messages, as opposed to the
// user-visible answer. These carry the model's chain-of-thought summary
// prose (e.g. "**Exploring project status tools**...") and are otherwise
// silently dropped by visibleBotText, leaving the caller with no visibility
// into what the model is doing on a long or tool-heavy turn.
func isChainOfThoughtSummary(m botMessage) bool {
	return m.Author == "bot" && m.Text != "" && m.ContentOrigin == "ChainOfThoughtSummary"
}

// advanceThinking emits a reasoning message exactly once per MessageID: the
// service resends the same message (with growing metadata) across multiple
// frames, and each one is a full-text snapshot, not an incremental delta, so
// folding logic like advance's would be wrong here.
func (s *CopilotStream) advanceThinking(m botMessage) {
	s.mu.Lock()
	key := m.MessageID
	if key == "" {
		key = m.Text
	}
	if s.thinkOnce[key] {
		s.mu.Unlock()
		return
	}
	s.thinkOnce[key] = true
	s.mu.Unlock()
	s.thinking <- m.Text
}

// logRawFrame prints the raw SignalR frame bytes when KDEPS_DEBUG is set. The
// call-chain instrumentation (kdeps_debug.Log) only records function names,
// not frame content, so this is the only way to see what the service
// actually sent on the wire -- needed to diagnose content that silently
// fails to surface (an unrecognized messageType, a genuinely empty
// completion under a given tone, etc.) rather than guessing at it.
func logRawFrame(frame string) {
	if !kdeps_debug.Enabled() {
		return
	}
	fmt.Fprintf(os.Stderr, "m365 raw frame: %s\n", frame)
}

// run drives the WebSocket: handshake, send chat, then fold frames until the
// server closes the turn. It always closes the Deltas channel and the socket.
//
//nolint:gocognit // SignalR handshake, stop-on-cancel, and frame dispatch in one loop
func (s *CopilotStream) run(ctx context.Context, conn *websocket.Conn, args map[string]any) {
	kdeps_debug.Log("enter: CopilotStream.run")
	defer close(s.deltas)
	defer close(s.thinking)
	defer conn.Close()

	// Cancel the in-flight turn when the caller's context is done, mirroring the
	// UI's Stop button, then let the server's completion ack close the socket.
	stopCtx, cancelStop := context.WithCancel(ctx)
	defer cancelStop()
	handshakeDone := false
	go func() {
		select {
		case <-ctx.Done():
			s.mu.Lock()
			hd := handshakeDone
			s.mu.Unlock()
			if hd {
				_ = s.writeMessage(conn, []byte(stopFrame))
			} else {
				_ = conn.Close()
			}
		case <-stopCtx.Done():
		}
	}()

	// Send the SignalR handshake.
	if err := s.writeMessage(conn, []byte(`{"protocol":"json","version":1}`+rs)); err != nil {
		s.setErr(fmt.Errorf("m365: send handshake: %w", err))
		return
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			// A normal close after the turn completes is not an error.
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				return
			}
			if s.FullText() == "" {
				s.setErr(fmt.Errorf("m365: read frame: %w", err))
			}
			return
		}

		for frame := range strings.SplitSeq(string(data), rs) {
			if frame == "" {
				continue
			}
			logRawFrame(frame)
			var env signalRFrame
			if jerr := json.Unmarshal([]byte(frame), &env); jerr != nil {
				// Non-JSON before handshake completion means the handshake ack is
				// done; send the chat message.
				if !handshakeDone {
					s.mu.Lock()
					handshakeDone = true
					s.mu.Unlock()
					if !s.sendChat(conn, args) {
						return
					}
				}
				continue
			}

			if !handshakeDone {
				s.mu.Lock()
				handshakeDone = true
				s.mu.Unlock()
				var hs signalRHandshakeResponse
				if json.Unmarshal([]byte(frame), &hs) == nil && hs.Error != "" {
					s.setErr(fmt.Errorf("m365: handshake error: %s", hs.Error))
					return
				}
				if !s.sendChat(conn, args) {
					return
				}
				continue
			}

			if done := s.handleFrame(conn, env, frame); done {
				return
			}
		}
	}
}

// sendChat writes the chat+metrics payload. Returns false on write failure.
func (s *CopilotStream) sendChat(conn *websocket.Conn, args map[string]any) bool {
	payload, err := buildSendPayload(args)
	if err != nil {
		s.setErr(fmt.Errorf("m365: build chat payload: %w", err))
		return false
	}
	if err = s.writeMessage(conn, []byte(payload)); err != nil {
		s.setErr(fmt.Errorf("m365: send chat: %w", err))
		return false
	}
	return true
}

// handleFrame dispatches one parsed frame. It returns true when the turn is
// complete and the read loop should stop.
func (s *CopilotStream) handleFrame(conn *websocket.Conn, env signalRFrame, raw string) bool {
	switch env.Type {
	case framePing: // keep-alive ping — echo it.
		_ = s.writeMessage(conn, []byte(`{"type":6}`+rs))
		return false
	case frameClose: // close
		if env.Error != "" {
			s.setErr(fmt.Errorf("m365: server close: %s", env.Error))
		}
		return true
	case frameCompletion: // completion
		if env.Error != "" {
			s.setErr(fmt.Errorf("m365: completion error: %s", env.Error))
		}
		return true
	case frameStreamItem: // final stream item — authoritative metadata
		s.handleStreamItem(raw)
		return true
	case frameInvocation:
		if env.Target == "update" {
			s.handleUpdate(env.Arguments)
		}
		return false
	default:
		return false
	}
}

// handleStreamItem mines the final "type:2" item for throttle/scores/state,
// and -- since some turns never send their answer as an incremental type:1
// update, only in this authoritative snapshot -- also surfaces its text.
// Without that, such a turn produces zero visible output even though the
// server did answer (confirmed live: m365-copilot's "auto" tone completing
// with completion_tokens=0 despite the service actually responding).
func (s *CopilotStream) handleStreamItem(raw string) {
	var item streamItem
	if json.Unmarshal([]byte(raw), &item) != nil || item.Item == nil {
		return
	}

	var text string
	s.mu.Lock()
	if item.Item.TurnState != "" {
		s.turnState = item.Item.TurnState
	}
	if item.Item.Throttling != nil {
		s.throttle = &Throttle{
			Current: item.Item.Throttling.NumUserMessagesInConversation,
			Max:     item.Item.Throttling.MaxNumUserMessagesInConversation,
		}
	}
	var thinkMsgs []botMessage
	for _, m := range item.Item.Messages {
		if m.Author != "bot" {
			continue
		}
		s.mineMessageLocked(m)
		if isChainOfThoughtSummary(m) {
			thinkMsgs = append(thinkMsgs, m)
			continue
		}
		if visible, ok := visibleBotText(m); ok {
			text = visible
		}
	}
	s.mu.Unlock()
	for _, m := range thinkMsgs {
		s.advanceThinking(m)
	}

	// advance's own fold logic (foldStreamText) is idempotent against text
	// already emitted by an earlier type:1 update, so calling it here
	// unconditionally is safe even when deltas already streamed the answer.
	if text != "" {
		s.advance(text)
	}
}

// visibleBotText reports the text of a bot message that represents the
// user-visible answer, as opposed to a structural/metadata-only message
// (a search step, a card, a suggestion). The service tags the plain answer
// with either an empty messageType or "Chat" -- the latter is explicitly one
// of the types kdeps' own allowedMessageTypes list (buildChatArgs) asks the
// server to send, but was previously never treated as visible content here,
// silently dropping the answer for any turn that arrives tagged that way.
func visibleBotText(m botMessage) (string, bool) {
	if m.Author != "bot" || m.Text == "" {
		return "", false
	}
	if m.MessageType == "" || m.MessageType == "Chat" {
		return m.Text, true
	}
	return "", false
}

// handleUpdate processes "type:1 target:update" argument frames.
//
//nolint:gocognit // three alternative update shapes decoded from one arg list
func (s *CopilotStream) handleUpdate(arguments []json.RawMessage) {
	for _, arg := range arguments {
		var delta deltaUpdate
		if json.Unmarshal(arg, &delta) == nil && delta.WriteAtCursor != "" {
			s.advance(s.FullText() + delta.WriteAtCursor)
			continue
		}

		var upd messageUpdate
		if json.Unmarshal(arg, &upd) == nil && len(upd.Messages) > 0 {
			for _, m := range upd.Messages {
				if m.Author == "bot" {
					s.mu.Lock()
					s.mineMessageLocked(m)
					s.mu.Unlock()
				}
				if isChainOfThoughtSummary(m) {
					s.advanceThinking(m)
					continue
				}
				if text, ok := visibleBotText(m); ok {
					s.advance(text)
				}
			}
			continue
		}

		var thr throttlingUpdate
		if json.Unmarshal(arg, &thr) == nil && thr.Throttling.MaxNumUserMessagesInConversation > 0 {
			s.mu.Lock()
			s.throttle = &Throttle{
				Current: thr.Throttling.NumUserMessagesInConversation,
				Max:     thr.Throttling.MaxNumUserMessagesInConversation,
			}
			s.mu.Unlock()
		}
	}
}

// mineMessageLocked copies per-message metadata into the stream. Caller holds mu.
func (s *CopilotStream) mineMessageLocked(m botMessage) {
	if m.ContentOrigin != "" {
		s.contentOrigin = m.ContentOrigin
	}
	if m.MessageType != "" {
		s.messageType = m.MessageType
	}
	if m.MessageID != "" {
		s.messageID = m.MessageID
	}
	if m.TurnCount != nil {
		s.turnCount = m.TurnCount
	}
	if m.TurnState != "" {
		s.turnState = m.TurnState
	}
	for _, sc := range m.Scores {
		if sc.Component == "" {
			continue
		}
		if cur, ok := s.maxScores[sc.Component]; !ok || sc.Score > cur {
			s.maxScores[sc.Component] = sc.Score
		}
	}
}
