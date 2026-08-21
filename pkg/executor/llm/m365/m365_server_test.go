package m365

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// readSSEBody reads the whole streamed response until the server closes the
// connection (after its trailing "data: [DONE]\n\n"), so tests observe every
// chunk rather than whatever happened to be flushed by one Read call.
func readSSEBody(t *testing.T, body io.Reader) string {
	t.Helper()
	data, err := io.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// scriptedChatServer serves one scripted frame sequence per incoming WebSocket
// connection (i.e. per chat turn), in order. It lets a test drive the full
// server.go pipeline (runBuffered/produce/run/runStream) through repeated
// retries, each turn getting a different scripted upstream response.
type scriptedChatServer struct {
	srv *httptest.Server

	mu      sync.Mutex
	scripts [][]string
	next    int
	chats   []string // captured chat frame per connection, in order
}

// nextScript pops the next scripted frame sequence, reusing the last one once
// the list is exhausted.
func (f *scriptedChatServer) nextScript() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.next
	f.next++
	switch {
	case idx < len(f.scripts):
		return f.scripts[idx]
	case len(f.scripts) > 0:
		return f.scripts[len(f.scripts)-1]
	default:
		return nil
	}
}

// recordChatFrame stores the "target":"chat" frame out of a raw multi-frame
// payload, for later inspection by tests.
func (f *scriptedChatServer) recordChatFrame(payload string) {
	for frame := range strings.SplitSeq(payload, rs) {
		if strings.Contains(frame, `"target":"chat"`) {
			f.mu.Lock()
			f.chats = append(f.chats, frame)
			f.mu.Unlock()
		}
	}
}

func newScriptedChatServer(t *testing.T, scripts [][]string) *scriptedChatServer {
	t.Helper()
	upgrader := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
	f := &scriptedChatServer{scripts: scripts}
	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, upErr := upgrader.Upgrade(w, r, nil)
		if upErr != nil {
			return
		}
		defer conn.Close()

		if _, _, hsErr := conn.ReadMessage(); hsErr != nil {
			return
		}
		if ackErr := conn.WriteMessage(websocket.TextMessage, []byte("{}"+rs)); ackErr != nil {
			return
		}

		_, data, chatErr := conn.ReadMessage()
		if chatErr == nil {
			f.recordChatFrame(string(data))
		}

		for _, frame := range f.nextScript() {
			if sendErr := conn.WriteMessage(websocket.TextMessage, []byte(frame+rs)); sendErr != nil {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *scriptedChatServer) wsURL() string {
	return "ws" + strings.TrimPrefix(f.srv.URL, "http") + "/Chathub"
}

func (f *scriptedChatServer) turnCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.next
}

func newTestServer(t *testing.T, scripts [][]string) (*httptest.Server, *scriptedChatServer) {
	t.Helper()
	wsSrv := newScriptedChatServer(t, scripts)
	withChatWSBase(t, wsSrv.wsURL())
	m365Srv := NewServer(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return testJWT(t), nil },
	})
	httpSrv := httptest.NewServer(m365Srv)
	t.Cleanup(httpSrv.Close)
	return httpSrv, wsSrv
}

func postChat(t *testing.T, base string, body map[string]any) (*http.Response, map[string]any) {
	t.Helper()
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.Post(base+"/v1/chat/completions", "application/json", strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var parsed map[string]any
	_ = json.NewDecoder(res.Body).Decode(&parsed)
	return res, parsed
}

func successFrame(text string) []string {
	return []string{fmtDelta(text), `{"type":2,"item":{"turnState":"Completed"}}`}
}

// fmtDelta builds a type:1 target:update frame carrying text as a bot message.
// text is JSON-encoded properly so newlines/backticks/quotes in fenced tool-call
// bodies don't corrupt the frame.
func fmtDelta(text string) string {
	encoded, _ := json.Marshal(text)
	return `{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":` + string(encoded) + `}]}]}`
}

// --- non-streaming, no tools ---

func TestServerChatCompletionNonStreaming(t *testing.T) {
	srv, _ := newTestServer(t, [][]string{successFrame("hello there")})
	base := srv.URL

	_, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
	})
	choices, _ := body["choices"].([]any)
	if len(choices) != 1 {
		t.Fatalf("body = %+v", body)
	}
	choice := choices[0].(map[string]any)
	msg := choice["message"].(map[string]any)
	if msg["content"] != "hello there" {
		t.Errorf("content = %v", msg["content"])
	}
	if choice["finish_reason"] != "stop" {
		t.Errorf("finish_reason = %v", choice["finish_reason"])
	}
}

func TestServerChatCompletionStreaming(t *testing.T) {
	srv, _ := newTestServer(t, [][]string{successFrame("streamed text")})
	base := srv.URL

	data, _ := json.Marshal(map[string]any{
		"model":    "m365-copilot",
		"stream":   true,
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
	})
	res, err := http.Post(base+"/v1/chat/completions", "application/json", strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out := readSSEBody(t, res.Body)
	if !strings.Contains(out, "data:") {
		t.Errorf("no SSE data in stream output: %q", out)
	}
	if !strings.Contains(out, `"role":"assistant"`) {
		t.Errorf("missing role chunk: %q", out)
	}
}

// --- tool calling ---

func TestServerChatCompletionToolCall(t *testing.T) {
	srv, _ := newTestServer(t, [][]string{successFrame("```bash\nls -la\n```")})
	base := srv.URL

	_, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "list files"}},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name": "bash",
				"parameters": map[string]any{
					"properties": map[string]any{"command": map[string]any{"type": "string"}},
				},
			},
		}},
	})
	choices, _ := body["choices"].([]any)
	if len(choices) != 1 {
		t.Fatalf("body = %+v", body)
	}
	choice := choices[0].(map[string]any)
	if choice["finish_reason"] != "tool_calls" {
		t.Fatalf("finish_reason = %v, body=%+v", choice["finish_reason"], body)
	}
	msg := choice["message"].(map[string]any)
	calls, _ := msg["tool_calls"].([]any)
	if len(calls) != 1 {
		t.Fatalf("tool_calls = %+v", calls)
	}
}

func TestServerChatCompletionReplyToolBecomesText(t *testing.T) {
	// "text" is the reply tool's sole (body) param, so it has no header line -
	// the fence body IS the text argument verbatim.
	srv, _ := newTestServer(t, [][]string{successFrame("```reply\nthe answer\n```")})
	base := srv.URL
	_, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name":       "reply",
				"parameters": map[string]any{"properties": map[string]any{"text": map[string]any{"type": "string"}}},
			},
		}},
	})
	choices, _ := body["choices"].([]any)
	choice := choices[0].(map[string]any)
	if choice["finish_reason"] != "stop" {
		t.Fatalf("finish_reason = %v, body=%+v", choice["finish_reason"], body)
	}
	msg := choice["message"].(map[string]any)
	if msg["content"] != "the answer" {
		t.Errorf("content = %v", msg["content"])
	}
}

// --- confabulation / hallucination forced retry ---

func TestServerChatCompletionConfabulationRetry(t *testing.T) {
	frames := [][]string{
		successFrame("I cannot access the files, please paste them here"),
		successFrame("```bash\nls -la\n```"),
	}
	srv, wsSrv := newTestServer(t, frames)
	base := srv.URL

	_, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "list files"}},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name": "bash",
				"parameters": map[string]any{
					"properties": map[string]any{"command": map[string]any{"type": "string"}},
				},
			},
		}},
	})
	choices, _ := body["choices"].([]any)
	if len(choices) != 1 {
		t.Fatalf("body = %+v", body)
	}
	choice := choices[0].(map[string]any)
	if choice["finish_reason"] != "tool_calls" {
		t.Fatalf("confabulation should have forced a retry that produced a tool call: %+v", body)
	}
	if wsSrv.turnCount() != 2 {
		t.Errorf("expected exactly one forced retry (2 WS turns), got %d", wsSrv.turnCount())
	}
}

// A session with no shell tool must not be told to write a ```bash block on
// retry -- BuildSpecMap only aliases that fence language onto a real tool
// when a shell tool is registered, so asking for it here would never parse
// into a call. The retry prompt must instead point at a real registered tool
// (list_files), and the model complying with that should succeed.
func TestServerChatCompletionConfabulationRetryNoShellTool(t *testing.T) {
	frames := [][]string{
		successFrame("I cannot access the files, please paste them here"),
		successFrame("```list_files\n/tmp\n```"),
	}
	srv, wsSrv := newTestServer(t, frames)
	base := srv.URL

	_, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "list files"}},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name": "list_files",
				"parameters": map[string]any{
					"properties": map[string]any{"path": map[string]any{"type": "string"}},
				},
			},
		}},
	})
	choices, _ := body["choices"].([]any)
	if len(choices) != 1 {
		t.Fatalf("body = %+v", body)
	}
	choice := choices[0].(map[string]any)
	if choice["finish_reason"] != "tool_calls" {
		t.Fatalf("retry should have produced a real tool call: %+v", body)
	}
	if wsSrv.turnCount() != 2 {
		t.Errorf("expected exactly one forced retry (2 WS turns), got %d", wsSrv.turnCount())
	}
}

// --- Disengaged recovery ---

func TestServerChatCompletionDisengageRetry(t *testing.T) {
	frames := [][]string{
		{`{"type":2,"item":{"messages":[{"author":"bot","messageType":"Disengaged"}],"turnState":"Completed"}}`},
		successFrame("recovered answer"),
	}
	srv, _ := newTestServer(t, frames)
	base := srv.URL

	_, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name": "bash",
				"parameters": map[string]any{
					"properties": map[string]any{"command": map[string]any{"type": "string"}},
				},
			},
		}},
	})
	choices, _ := body["choices"].([]any)
	if len(choices) != 1 {
		t.Fatalf("body = %+v", body)
	}
	choice := choices[0].(map[string]any)
	msg := choice["message"].(map[string]any)
	if msg["content"] != "recovered answer" {
		t.Errorf("disengage-retry should have recovered: %+v", body)
	}
}

func TestServerChatCompletionDisengageNoRetryEnv(t *testing.T) {
	t.Setenv("M365_NO_DISENGAGE_RETRY", "1")
	frames := [][]string{
		{`{"type":2,"item":{"messages":[{"author":"bot","messageType":"Disengaged"}],"turnState":"Completed"}}`},
	}
	srv, _ := newTestServer(t, frames)
	base := srv.URL

	res, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name": "bash",
				"parameters": map[string]any{
					"properties": map[string]any{"command": map[string]any{"type": "string"}},
				},
			},
		}},
	})
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, body=%+v", res.StatusCode, body)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["type"] != "disengaged" {
		t.Errorf("error type = %v", errObj["type"])
	}
}

// --- empty response handling ---

func TestServerChatCompletionEmptyThenRecovers(t *testing.T) {
	frames := [][]string{
		{`{"type":3}`}, // empty turn
		successFrame("finally an answer"),
	}
	srv, _ := newTestServer(t, frames)
	base := srv.URL

	_, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
	})
	choices, _ := body["choices"].([]any)
	if len(choices) != 1 {
		t.Fatalf("body = %+v", body)
	}
	choice := choices[0].(map[string]any)
	msg := choice["message"].(map[string]any)
	if msg["content"] != "finally an answer" {
		t.Errorf("content = %v, body=%+v", msg["content"], body)
	}
}

func TestServerChatCompletionAlwaysEmpty(t *testing.T) {
	frames := [][]string{{`{"type":3}`}}
	srv, _ := newTestServer(t, frames)
	base := srv.URL

	res, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
	})
	if res.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, body=%+v", res.StatusCode, body)
	}
	errObj, _ := body["error"].(map[string]any)
	if errObj["type"] != "upstream_empty_response" {
		t.Errorf("error type = %v", errObj["type"])
	}
}

func TestServerChatCompletionAtLimitThrottle(t *testing.T) {
	frames := [][]string{
		{
			`{"type":2,"item":{"throttling":{"maxNumUserMessagesInConversation":5,"numUserMessagesInConversation":5},"turnState":"Completed"}}`,
		},
	}
	srv, _ := newTestServer(t, frames)
	base := srv.URL

	res, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
	})
	if res.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, body=%+v", res.StatusCode, body)
	}
}

// --- multi-turn / follow-up delta path ---

func TestServerChatCompletionFollowUpDelta(t *testing.T) {
	frames := [][]string{successFrame("first"), successFrame("second")}
	srv, _ := newTestServer(t, frames)
	base := srv.URL

	// Same first user message => same pooled conversation.
	_, first := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "same-conversation-key"}},
	})
	firstChoice := first["choices"].([]any)[0].(map[string]any)
	firstMsg := firstChoice["message"].(map[string]any)
	if firstMsg["content"] != "first" {
		t.Fatalf("first turn content = %v", firstMsg["content"])
	}

	_, second := postChat(t, base, map[string]any{
		"model": "m365-copilot",
		"messages": []map[string]any{
			{"role": "user", "content": "same-conversation-key"},
			{"role": "assistant", "content": "first"},
			{"role": "user", "content": "and then?"},
		},
	})
	secondChoice := second["choices"].([]any)[0].(map[string]any)
	secondMsg := secondChoice["message"].(map[string]any)
	if secondMsg["content"] != "second" {
		t.Fatalf("second turn content = %v", secondMsg["content"])
	}
}

// --- models endpoint smoke (already covered elsewhere, kept for the httptest path) ---

func TestServerModelsEndpointOverHTTP(t *testing.T) {
	srv, _ := newTestServer(t, nil)
	base := srv.URL
	res, err := http.Get(base + "/v1/models")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
}

// --- coverage: runStream tool-call branch, usage, error builders ---

func TestServerChatCompletionStreamingToolCalls(t *testing.T) {
	srv, _ := newTestServer(t, [][]string{successFrame("```bash\nls -la\n```")})
	base := srv.URL

	data, _ := json.Marshal(map[string]any{
		"model":  "m365-copilot",
		"stream": true,
		"stream_options": map[string]any{
			"include_usage": true,
		},
		"messages": []map[string]any{{"role": "user", "content": "list files"}},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name": "bash",
				"parameters": map[string]any{
					"properties": map[string]any{"command": map[string]any{"type": "string"}},
				},
			},
		}},
	})
	res, err := http.Post(base+"/v1/chat/completions", "application/json", strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out := readSSEBody(t, res.Body)
	if !strings.Contains(out, `"tool_calls"`) {
		t.Errorf("expected tool_calls in stream output: %q", out)
	}
	if !strings.Contains(out, `"finish_reason":"tool_calls"`) {
		t.Errorf("expected tool_calls finish reason: %q", out)
	}
}

func TestServerChatCompletionStreamingError(t *testing.T) {
	withChatWSBase(t, "ws://127.0.0.1:1/Chathub") // unreachable -> dial error
	m365Srv := NewServer(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return testJWT(t), nil },
	})
	httpSrv := httptest.NewServer(m365Srv)
	t.Cleanup(httpSrv.Close)

	data, _ := json.Marshal(map[string]any{
		"model":    "m365-copilot",
		"stream":   true,
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
	})
	res, err := http.Post(httpSrv.URL+"/v1/chat/completions", "application/json", strings.NewReader(string(data)))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	out := readSSEBody(t, res.Body)
	if !strings.Contains(out, `"error"`) {
		t.Errorf("expected an in-stream error chunk: %q", out)
	}
}

func TestUsageFullBlock(t *testing.T) {
	c := &completion{
		lastThrottle:      &Throttle{Current: 3, Max: 10},
		lastContentOrigin: "DeepLeo",
		lastMessageType:   "Chat",
		lastTurnCount:     intPtr(2),
		lastScores:        map[string]float64{"dea_violation": 0.001, "BotOffense": 0.002, "other": 0.5},
	}
	u := c.usage()
	if u["x_m365_content_origin"] != "DeepLeo" || u["x_m365_message_type"] != "Chat" {
		t.Errorf("usage = %+v", u)
	}
	if u["x_m365_turn_count"] != 2 {
		t.Errorf("turn count = %v", u["x_m365_turn_count"])
	}
	if u["x_m365_dea_score"] != 0.001 || u["x_m365_offense_score"] != 0.002 {
		t.Errorf("scores = %+v", u)
	}
}

func intPtr(i int) *int { return &i }

func TestEmptyResponseErrorWithThrottle(t *testing.T) {
	p := emptyResponseError(&Throttle{Current: 1, Max: 5})
	if !strings.Contains(p.errMsg, "throttle 1/5") {
		t.Errorf("errMsg = %q", p.errMsg)
	}
}

func TestMaxIntHelper(t *testing.T) {
	if maxInt(2, 5) != 5 || maxInt(5, 2) != 5 {
		t.Error("maxInt")
	}
}
