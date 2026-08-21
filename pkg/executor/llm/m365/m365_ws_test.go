package m365

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeChatServer runs a local SignalR-shaped WebSocket server: it acks the
// handshake, captures the chat payload, then writes a fixed sequence of
// response frames. Tests point chatWSBase at it to exercise the real transport
// (dial, handshake, frame dispatch, streaming) without a live Microsoft backend.
type fakeChatServer struct {
	srv *httptest.Server

	mu       sync.Mutex
	lastChat string
}

func newFakeChatServer(t *testing.T, respFrames []string) *fakeChatServer {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	f := &fakeChatServer{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		if _, _, hsErr := conn.ReadMessage(); hsErr != nil { // handshake request
			return
		}
		if ackErr := conn.WriteMessage(websocket.TextMessage, []byte("{}"+rs)); ackErr != nil {
			return
		}

		_, data, chatErr := conn.ReadMessage() // chat + metrics payload
		if chatErr == nil {
			for frame := range strings.SplitSeq(string(data), rs) {
				if strings.Contains(frame, `"target":"chat"`) {
					f.mu.Lock()
					f.lastChat = frame
					f.mu.Unlock()
				}
			}
		}

		for _, frame := range respFrames {
			if sendErr := conn.WriteMessage(websocket.TextMessage, []byte(frame+rs)); sendErr != nil {
				return
			}
		}
		// Give the client time to read before the deferred Close tears the socket
		// down; the client treats an immediate close as end-of-turn either way.
		time.Sleep(20 * time.Millisecond)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeChatServer) wsURL() string {
	return "ws" + strings.TrimPrefix(f.srv.URL, "http") + "/Chathub"
}

func (f *fakeChatServer) chatFrame() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.lastChat
}

func withChatWSBase(t *testing.T, base string) {
	t.Helper()
	old := chatWSBase
	chatWSBase = base
	t.Cleanup(func() { chatWSBase = old })
}

func testJWT(t *testing.T) string {
	t.Helper()
	return makeJWT(t, JWTClaims{OID: "oid-1", TID: "tid-1"})
}

// drainDeltas consumes a stream's deltas without caring about their content,
// for tests that only assert on terminal metadata (Err, MessageType, etc.).
func drainDeltas(stream *CopilotStream) {
	for range stream.Deltas() { //nolint:revive // draining, not iterating
	}
}

// --- session.go / stream.go: real transport, fake server ---

func TestCopilotSessionChatStreamsDeltasAndMetadata(t *testing.T) {
	// writeAtCursor deltas are INCREMENTS appended at the current cursor (the
	// server mixes these with occasional full-text snapshots); handleUpdate
	// concatenates FullText()+delta before folding, so each frame here carries
	// only the newly-typed suffix, not the running total.
	srv := newFakeChatServer(t, []string{
		`{"type":1,"target":"update","arguments":[{"writeAtCursor":"hel"}]}`,
		`{"type":1,"target":"update","arguments":[{"writeAtCursor":"lo"}]}`,
		`{"type":2,"item":{"messages":[{"author":"bot","text":"hello","messageId":"m1","messageType":"Chat","contentOrigin":"DeepLeo","turnState":"Completed","scores":[{"component":"dea_violation","score":0.0005},{"component":"BotOffense","score":0.0001}]}],"throttling":{"maxNumUserMessagesInConversation":600,"numUserMessagesInConversation":5},"turnState":"Completed"}}`,
	})
	withChatWSBase(t, srv.wsURL())

	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi there", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}

	var got strings.Builder
	for d := range stream.Deltas() {
		got.WriteString(d)
	}
	if got.String() != "hello" {
		t.Errorf("deltas = %q, want %q", got.String(), "hello")
	}
	if serr := stream.Err(); serr != nil {
		t.Fatalf("stream error: %v", serr)
	}
	if !stream.HasContent() || stream.FullText() != "hello" {
		t.Errorf("FullText = %q", stream.FullText())
	}
	if stream.MessageID() != "m1" || stream.MessageType() != "Chat" ||
		stream.ContentOrigin() != "DeepLeo" {
		t.Errorf(
			"metadata: id=%q type=%q origin=%q",
			stream.MessageID(),
			stream.MessageType(),
			stream.ContentOrigin(),
		)
	}
	if stream.TurnState() != "Completed" {
		t.Errorf("turnState = %q", stream.TurnState())
	}
	th := stream.Throttle()
	if th == nil || th.Current != 5 || th.Max != 600 {
		t.Errorf("throttle = %+v", th)
	}
	scores := stream.Scores()
	if scores["dea_violation"] != 0.0005 {
		t.Errorf("scores = %v", scores)
	}
	if sess.TurnCount() != 1 {
		t.Errorf("turn count = %d", sess.TurnCount())
	}

	// The outgoing chat frame carried the prompt text.
	var env map[string]any
	if uerr := json.Unmarshal([]byte(srv.chatFrame()), &env); uerr != nil {
		t.Fatal(uerr)
	}
	args := env["arguments"].([]any)[0].(map[string]any)
	msg := args["message"].(map[string]any)
	if msg["text"] != "hi there" {
		t.Errorf("sent text = %v", msg["text"])
	}
}

func TestCopilotSessionChatMessageUpdateAndThrottleUpdate(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		// A full-text snapshot via a "messages" update (not writeAtCursor).
		`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"full answer"}]}]}`,
		// A standalone throttle update.
		`{"type":1,"target":"update","arguments":[{"throttling":{"maxNumUserMessagesInConversation":10,"numUserMessagesInConversation":9}}]}`,
		`{"type":3}`, // plain completion, no error
	})
	withChatWSBase(t, srv.wsURL())

	sess := NewCopilotSession(CopilotSessionOptions{AgentID: "T1.b1.gpt.default"})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}
	var got strings.Builder
	for d := range stream.Deltas() {
		got.WriteString(d)
	}
	if got.String() != "full answer" {
		t.Errorf("deltas = %q", got.String())
	}
	th := stream.Throttle()
	if th == nil || th.Current != 9 || th.Max != 10 {
		t.Errorf("throttle = %+v", th)
	}
	if serr := stream.Err(); serr != nil {
		t.Fatalf("unexpected error: %v", serr)
	}
}

// A turn whose answer arrives only in the final authoritative "type:2" item
// (no incremental writeAtCursor or type:1 messages update beforehand) must
// still surface as content -- confirmed live as the m365-copilot "auto" tone
// bug: completion_tokens=0 despite the service actually answering, because
// handleStreamItem previously mined metadata but never extracted the text.
func TestCopilotSessionChatFinalItemOnlyAnswer_EmptyMessageType(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":2,"item":{"messages":[{"author":"bot","text":"final only","messageId":"m1"}],"turnState":"Completed"}}`,
	})
	withChatWSBase(t, srv.wsURL())

	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}
	drainDeltas(stream)
	if serr := stream.Err(); serr != nil {
		t.Fatalf("stream error: %v", serr)
	}
	if !stream.HasContent() || stream.FullText() != "final only" {
		t.Errorf("FullText = %q, HasContent = %v", stream.FullText(), stream.HasContent())
	}
}

// Same as above, but the answer is tagged messageType "Chat" -- one of the
// types kdeps' own allowedMessageTypes list asks the server for, previously
// silently dropped because only an exactly-empty messageType counted.
func TestCopilotSessionChatFinalItemOnlyAnswer_ChatMessageType(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":2,"item":{"messages":[{"author":"bot","text":"chat typed","messageId":"m1","messageType":"Chat"}],"turnState":"Completed"}}`,
	})
	withChatWSBase(t, srv.wsURL())

	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}
	drainDeltas(stream)
	if !stream.HasContent() || stream.FullText() != "chat typed" {
		t.Errorf("FullText = %q, HasContent = %v", stream.FullText(), stream.HasContent())
	}
}

// A type:1 full-text "messages" snapshot (not writeAtCursor) tagged "Chat"
// must also be surfaced -- the same gap as the final-item case, but on the
// incremental-update path.
func TestCopilotSessionChatMessageUpdate_ChatMessageType(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"update chat typed","messageType":"Chat"}]}]}`,
	})
	withChatWSBase(t, srv.wsURL())

	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}
	var got strings.Builder
	for d := range stream.Deltas() {
		got.WriteString(d)
	}
	if got.String() != "update chat typed" {
		t.Errorf("deltas = %q", got.String())
	}
}

// A message tagged with a structural/metadata type (not the plain answer)
// must still be excluded from visible content -- guards against a filter
// broadened so far it treats every message as the answer.
func TestCopilotSessionChatMessageUpdate_NonAnswerTypeExcluded(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"searching...","messageType":"InternalSearchQuery"}]}]}`,
		`{"type":3}`,
	})
	withChatWSBase(t, srv.wsURL())

	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}
	drainDeltas(stream)
	if stream.HasContent() || stream.FullText() != "" {
		t.Errorf(
			"non-answer messageType must not surface as content, got FullText = %q",
			stream.FullText(),
		)
	}
}

// A ChainOfThoughtSummary progress message must surface on ThinkingDeltas,
// not on the answer Deltas channel, and repeats of the same MessageID (the
// service resends messages with growing metadata across frames) must not
// duplicate it.
func TestCopilotSessionChatThinkingDeltas(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"**thinking one**","messageId":"t1","contentOrigin":"ChainOfThoughtSummary","messageType":"Progress"}]}]}`,
		`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"**thinking one**","messageId":"t1","contentOrigin":"ChainOfThoughtSummary","messageType":"Progress"}]}]}`,
		`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"the answer","messageType":"Chat"}]}]}`,
	})
	withChatWSBase(t, srv.wsURL())

	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}

	var thinking []string
	thinkDone := make(chan struct{})
	go func() {
		defer close(thinkDone)
		for t := range stream.ThinkingDeltas() {
			thinking = append(thinking, t)
		}
	}()
	drainDeltas(stream)
	<-thinkDone

	if len(thinking) != 1 || thinking[0] != "**thinking one**" {
		t.Errorf("thinking deltas = %v, want exactly one deduped message", thinking)
	}
	if stream.FullText() != "the answer" {
		t.Errorf("FullText = %q, thinking text must not leak into the answer", stream.FullText())
	}
}

func TestCopilotSessionChatPingAndCompletionError(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":6}`, // keep-alive; client echoes and continues
		`{"type":3,"error":"something went wrong"}`, // completion with an error
	})
	withChatWSBase(t, srv.wsURL())

	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}
	drainDeltas(stream)
	if serr := stream.Err(); serr == nil ||
		!strings.Contains(serr.Error(), "something went wrong") {
		t.Errorf("err = %v", serr)
	}
}

func TestCopilotSessionChatCloseFrameError(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":7,"error":"server closing"}`,
	})
	withChatWSBase(t, srv.wsURL())

	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}
	drainDeltas(stream)
	if serr := stream.Err(); serr == nil || !strings.Contains(serr.Error(), "server closing") {
		t.Errorf("err = %v", serr)
	}
}

func TestCopilotSessionChatHandshakeError(t *testing.T) {
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, hsErr := conn.ReadMessage(); hsErr != nil {
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"bad protocol version"}`+rs))
	}))
	t.Cleanup(srv.Close)
	withChatWSBase(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/Chathub")

	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}
	drainDeltas(stream)
	if serr := stream.Err(); serr == nil || !strings.Contains(serr.Error(), "handshake error") {
		t.Errorf("err = %v", serr)
	}
}

func TestCopilotSessionChatDialFailure(t *testing.T) {
	withChatWSBase(t, "ws://127.0.0.1:1/Chathub") // nothing listens on port 1
	sess := NewCopilotSession(CopilotSessionOptions{})
	_, err := sess.Chat(context.Background(), testJWT(t), "hi", "m365-copilot")
	if err == nil {
		t.Fatal("want dial error")
	}
}

func TestCopilotSessionChatBadToken(t *testing.T) {
	sess := NewCopilotSession(CopilotSessionOptions{})
	if _, err := sess.Chat(context.Background(), "not-a-jwt", "hi", "m365-copilot"); err == nil {
		t.Fatal("want decode error")
	}
}

func TestCopilotStreamStopOnContextCancel(t *testing.T) {
	// Server acks the handshake, reads the chat payload, then just waits — the
	// client's context cancellation should trigger a stop frame or a close,
	// ending the turn without hanging.
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	handshakeDone := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if _, _, hsErr := conn.ReadMessage(); hsErr != nil {
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte("{}"+rs))
		if _, _, chatErr := conn.ReadMessage(); chatErr != nil {
			return
		}
		close(handshakeDone)
		// Wait for the client's stop frame (sent on context cancellation) and ack
		// it with a completion, mirroring the real service's stop-button contract.
		if _, _, stopErr := conn.ReadMessage(); stopErr != nil {
			return
		}
		_ = conn.WriteMessage(websocket.TextMessage, []byte(`{"type":3}`+rs))
	}))
	t.Cleanup(srv.Close)
	withChatWSBase(t, "ws"+strings.TrimPrefix(srv.URL, "http")+"/Chathub")

	ctx, cancel := context.WithCancel(context.Background())
	sess := NewCopilotSession(CopilotSessionOptions{})
	stream, err := sess.Chat(ctx, testJWT(t), "hi", "m365-copilot")
	if err != nil {
		t.Fatal(err)
	}

	select {
	case <-handshakeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("handshake never completed")
	}
	cancel()

	done := make(chan struct{})
	go func() {
		drainDeltas(stream)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("turn did not end after context cancellation")
	}
}

// --- model.go: full pipeline over the fake transport ---

func TestModelSessionRunAgentless(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		// A messages-update with no messageType streams as content (advance);
		// the trailing type:2 item then adds metadata only.
		`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"hi back"}]}]}`,
		`{"type":2,"item":{"messages":[{"author":"bot","text":"hi back","messageType":"Chat"}],"turnState":"Completed"}}`,
	})
	withChatWSBase(t, srv.wsURL())

	m := NewModelSession(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return testJWT(t), nil },
	})
	stream, err := m.Run(context.Background(), "hello", "m365-copilot", false, false)
	if err != nil {
		t.Fatal(err)
	}
	drainDeltas(stream)
	if stream.FullText() != "hi back" {
		t.Errorf("FullText = %q", stream.FullText())
	}
	if m.TurnCount() != 1 {
		t.Errorf("turn count = %d", m.TurnCount())
	}
}

// A turn that wants agent tool-calling but whose agent fails to provision
// (getOrCreateAgent errors, a non-fatal fallback to agentless) must still
// keep the server's own code interpreter off on the wire -- otherwise the
// model silently answers from Microsoft's sandbox instead of the caller's
// real tools. Confirmed live as the root cause of a think-deeper turn
// reporting "no project found" for a real, populated local repository.
func TestModelSessionRunAgentWantedButUnresolved_NoCodeInterpreterOnWire(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"ok"}]}]}`,
		`{"type":2,"item":{"messages":[{"author":"bot","text":"ok","messageType":"Chat"}],"turnState":"Completed"}}`,
	})
	withChatWSBase(t, srv.wsURL())

	m := NewModelSession(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return testJWT(t), nil },
		GetAgent: func(context.Context, bool) (string, error) { return "", errBoom },
	})
	stream, err := m.Run(context.Background(), "hello", "m365-copilot", true, true)
	if err != nil {
		t.Fatal(err)
	}
	drainDeltas(stream)

	var env map[string]any
	if uerr := json.Unmarshal([]byte(srv.chatFrame()), &env); uerr != nil {
		t.Fatal(uerr)
	}
	args := env["arguments"].([]any)[0].(map[string]any)
	optsRaw, _ := args["optionsSets"].([]any)
	for _, o := range optsRaw {
		if o == "cwc_code_interpreter" {
			t.Errorf(
				"code interpreter must not be enabled on the wire when the agent was wanted but unresolved: %v",
				optsRaw,
			)
		}
	}
}

func TestModelSessionRunWithAgent(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"agent reply"}]}]}`,
		`{"type":2,"item":{"messages":[{"author":"bot","text":"agent reply","messageType":"Chat"}],"turnState":"Completed"}}`,
	})
	withChatWSBase(t, srv.wsURL())

	m := NewModelSession(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return testJWT(t), nil },
		GetAgent: func(context.Context, bool) (string, error) { return "T1.b1.gpt.default", nil },
	})
	stream, err := m.Run(context.Background(), "hello", "m365-copilot", true, true)
	if err != nil {
		t.Fatal(err)
	}
	drainDeltas(stream)
	if stream.FullText() != "agent reply" {
		t.Errorf("FullText = %q", stream.FullText())
	}

	var env map[string]any
	if uerr := json.Unmarshal([]byte(srv.chatFrame()), &env); uerr != nil {
		t.Fatal(uerr)
	}
	args := env["arguments"].([]any)[0].(map[string]any)
	if args["threadLevelGptId"] != "T1.b1.gpt.default" {
		t.Errorf("threadLevelGptId = %v", args["threadLevelGptId"])
	}
}

func TestModelSessionRunTokenError(t *testing.T) {
	m := NewModelSession(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return "", errBoom },
	})
	if _, err := m.Run(context.Background(), "hi", "m365-copilot", false, false); err == nil {
		t.Fatal("want token error")
	}
}

func TestModelSessionRunReconnectsOnChatError(t *testing.T) {
	// First dial succeeds via chatWSBase pointing nowhere reachable after the
	// session is built once; simplest reliable way to force session.Chat to
	// fail once is to point at a closed listener for the FIRST call only. Since
	// ModelSession dials fresh per Run, simulate by using a bad base throughout
	// and asserting Run still returns the (final) dial error rather than
	// panicking or hanging.
	withChatWSBase(t, "ws://127.0.0.1:1/Chathub")
	m := NewModelSession(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return testJWT(t), nil },
	})
	if _, err := m.Run(context.Background(), "hi", "m365-copilot", false, false); err == nil {
		t.Fatal("want dial error surfaced after retry")
	}
}

var errBoom = &boomError{}

type boomError struct{}

func (*boomError) Error() string { return "boom" }

// A genuine completion error from the service (frame type:3 with a non-empty
// error) previously vanished into a streaming SSE response with no content
// and no finish_reason chunk, because runStream's producedError case only
// ever wrote a non-standard "error" object no OpenAI-compatible client looks
// for. Confirmed live as m365-copilot's "auto" tone completing with
// completion_tokens=0 on a real upstream error that never reached the user.
func TestServeHTTPChatCompletions_StreamingUpstreamErrorIsVisible(t *testing.T) {
	srv := newFakeChatServer(t, []string{
		`{"type":3,"invocationId":"0","error":"Failed to invoke 'Chat' due to an error on the server."}`,
	})
	withChatWSBase(t, srv.wsURL())

	handler := NewServer(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return testJWT(t), nil },
	})

	reqBody, err := json.Marshal(map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "status"}},
		"stream":   true,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(string(reqBody)),
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"status = %d (SSE headers are already committed by the time the error is known)",
			rec.Code,
		)
	}

	body := rec.Body.String()
	var sawContent, sawFinish bool
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimPrefix(line, "data: ")
		if line == "" || strings.HasPrefix(line, "[DONE]") {
			continue
		}
		var chunkObj map[string]any
		if json.Unmarshal([]byte(line), &chunkObj) != nil {
			continue
		}
		choices, _ := chunkObj["choices"].([]any)
		if len(choices) == 0 {
			continue
		}
		choice, _ := choices[0].(map[string]any)
		if delta, ok := choice["delta"].(map[string]any); ok {
			if content, _ := delta["content"].(string); strings.Contains(
				content,
				"Failed to invoke 'Chat'",
			) {
				sawContent = true
			}
		}
		if fr, _ := choice["finish_reason"].(string); fr != "" {
			sawFinish = true
		}
	}
	if !sawContent {
		t.Errorf("no delta.content chunk carried the error message -- "+
			"a generic OpenAI-compatible client would show nothing:\n%s", body)
	}
	if !sawFinish {
		t.Errorf(
			"no finish_reason chunk -- a generic client has no signal the turn ended:\n%s",
			body,
		)
	}
}

// newSequencedChatServer serves a different fixed frame sequence per
// successive WebSocket connection (connFrames[0] for the first connection,
// connFrames[1] for the second, ...). Used to simulate a completion error on
// the first attempt recovering on a retry against a fresh connection (e.g.
// after RefreshAgent), which the single-sequence fakeChatServer can't express
// since every connection there gets the same frames.
type sequencedChatServer struct {
	srv *httptest.Server
	mu  sync.Mutex
	n   int
}

func newSequencedChatServer(t *testing.T, connFrames [][]string) *sequencedChatServer {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	f := &sequencedChatServer{}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		idx := f.n
		f.n++
		f.mu.Unlock()
		frames := []string{}
		if idx < len(connFrames) {
			frames = connFrames[idx]
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		if _, _, hsErr := conn.ReadMessage(); hsErr != nil {
			return
		}
		if ackErr := conn.WriteMessage(websocket.TextMessage, []byte("{}"+rs)); ackErr != nil {
			return
		}
		if _, _, chatErr := conn.ReadMessage(); chatErr != nil {
			return
		}
		for _, frame := range frames {
			if sendErr := conn.WriteMessage(websocket.TextMessage, []byte(frame+rs)); sendErr != nil {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *sequencedChatServer) wsURL() string {
	return "ws" + strings.TrimPrefix(f.srv.URL, "http") + "/Chathub"
}

// A completion error on the tool-agent path recovers via the same
// RefreshAgent retry the empty-reply branch already had, instead of failing
// the turn immediately -- confirmed live as the root cause of m365-copilot's
// "auto" tone consistently failing with "Failed to invoke 'Chat' due to an
// error on the server." while non-agent (Claude) tones succeeded.
func TestServeHTTPChatCompletions_AgentErrorRecoversOnRefresh(t *testing.T) {
	srv := newSequencedChatServer(t, [][]string{
		{
			`{"type":3,"invocationId":"0","error":"Failed to invoke 'Chat' due to an error on the server."}`,
		},
		{
			`{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":"recovered","messageType":"Chat"}]}]}`,
			`{"type":3}`,
		},
	})
	withChatWSBase(t, srv.wsURL())

	agentCalls := 0
	handler := NewServer(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return testJWT(t), nil },
		GetAgent: func(_ context.Context, forceRefresh bool) (string, error) {
			agentCalls++
			if forceRefresh {
				return "agent-v2", nil
			}
			return "agent-v1", nil
		},
	})

	reqBody, err := json.Marshal(map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "status"}},
		"tools": []map[string]any{
			{"type": "function", "function": map[string]any{"name": "noop"}},
		},
		"stream": true,
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader(string(reqBody)),
	)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if !strings.Contains(body, "recovered") {
		t.Errorf("did not recover onto the successful retry after the agent refresh:\n%s", body)
	}
	if strings.Contains(body, "Failed to invoke") {
		t.Errorf("the transient error leaked into the final response despite recovering:\n%s", body)
	}
	if agentCalls < 2 {
		t.Errorf(
			"expected a forceRefresh agent resolution after the completion error, got %d agent calls",
			agentCalls,
		)
	}
}
