package m365

import (
	"context"
	"encoding/json"
	"fmt"
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

// fmtThinking builds a type:1 target:update frame carrying a
// ChainOfThoughtSummary progress message, the shape the service uses for its
// reasoning/"thinking" text.
func fmtThinking(text, messageID string) string {
	encodedText, _ := json.Marshal(text)
	encodedID, _ := json.Marshal(messageID)
	return `{"type":1,"target":"update","arguments":[{"messages":[{"author":"bot","text":` +
		string(encodedText) + `,"messageId":` + string(encodedID) +
		`,"contentOrigin":"ChainOfThoughtSummary","messageType":"Progress"}]}]}`
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

// ChainOfThoughtSummary progress messages must reach the caller as
// reasoning_content, not be silently dropped -- confirmed live as leaving the
// user with no visibility into what a long or tool-heavy m365 turn was
// actually doing while it ran.
func TestServerChatCompletionNonStreamingReasoningContent(t *testing.T) {
	frames := []string{
		fmtThinking("**Exploring the repo**", "think-1"),
		fmtDelta("final answer"),
		`{"type":2,"item":{"turnState":"Completed"}}`,
	}
	srv, _ := newTestServer(t, [][]string{frames})
	base := srv.URL

	_, body := postChat(t, base, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "hi"}},
	})
	choices, _ := body["choices"].([]any)
	if len(choices) != 1 {
		t.Fatalf("body = %+v", body)
	}
	msg := choices[0].(map[string]any)["message"].(map[string]any)
	if msg["content"] != "final answer" {
		t.Errorf("content = %v", msg["content"])
	}
	if !strings.Contains(fmt.Sprint(msg["reasoning_content"]), "Exploring the repo") {
		t.Errorf("reasoning_content = %v, want it to contain the thinking text", msg["reasoning_content"])
	}
}

// The same thinking text must also stream live as reasoning_content chunks,
// not just appear in the final buffered response.
func TestServerChatCompletionStreamingReasoningContent(t *testing.T) {
	frames := []string{
		fmtThinking("**Checking status**", "think-1"),
		fmtDelta("done"),
		`{"type":2,"item":{"turnState":"Completed"}}`,
	}
	srv, _ := newTestServer(t, [][]string{frames})
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
	if !strings.Contains(out, `"reasoning_content":"**Checking status**"`) {
		t.Errorf("stream missing live reasoning_content chunk:\n%s", out)
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
	srv, _ := newTestServer(t, [][]string{successFrame(
		`<invoke name="bash"><parameter name="command">ls -la</parameter></invoke>`,
	)})
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

// writeFileToolParam is the OpenAI-style tool schema for write_file, shared by
// the end-to-end tests below that drive the real WebSocket-mocked server
// through to a parsed HTTP tool_calls response (not just ParseFencedToolCalls
// called directly on a string), so a fix verified here is confirmed to hold
// through the actual production request/response pipeline, not just at the
// unit level.
func writeFileToolParam() map[string]any {
	return map[string]any{
		"type": "function",
		"function": map[string]any{
			"name": "write_file",
			"parameters": map[string]any{
				"properties": map[string]any{
					"path":    map[string]any{"type": "string"},
					"content": map[string]any{"type": "string"},
				},
			},
		},
	}
}

// TestServerChatCompletionToolCall_ContentWithNestedFence drives the nested-
// content truncation fix (parseFinalParam's greedy body match) through the
// real server pipeline: a scripted upstream frame carrying a write_file call
// whose content contains its own ```bash example block, verified not to
// truncate in the actual HTTP tool_calls response.
func TestServerChatCompletionToolCall_ContentWithNestedFence(t *testing.T) {
	content := "# Plan\n\nExample:\n\n```bash\necho hi\n```\n\nDone."
	raw := `<invoke name="write_file"><parameter name="path">PLAN.md</parameter>` +
		`<parameter name="content">` + content + `</parameter></invoke>`
	srv, _ := newTestServer(t, [][]string{successFrame(raw)})

	_, body := postChat(t, srv.URL, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "write the plan"}},
		"tools":    []map[string]any{writeFileToolParam()},
	})
	choices, _ := body["choices"].([]any)
	if len(choices) != 1 {
		t.Fatalf("body = %+v", body)
	}
	msg := choices[0].(map[string]any)["message"].(map[string]any)
	calls, _ := msg["tool_calls"].([]any)
	if len(calls) != 1 {
		t.Fatalf("tool_calls = %+v", calls)
	}
	fn := calls[0].(map[string]any)["function"].(map[string]any)
	var args map[string]any
	if err := json.Unmarshal([]byte(fn["arguments"].(string)), &args); err != nil {
		t.Fatal(err)
	}
	if args["content"] != content {
		t.Errorf("content truncated through the real pipeline:\ngot:  %q\nwant: %q", args["content"], content)
	}
}

// TestServerChatCompletionToolCall_SelfWrappedContent drives the redundant-
// self-wrap stripping fix through the real server pipeline: a scripted
// upstream frame carrying a write_file call whose content is itself wrapped
// in a ```markdown ... ``` fence, verified stripped in the actual HTTP
// tool_calls response.
func TestServerChatCompletionToolCall_SelfWrappedContent(t *testing.T) {
	intended := "# Plan\n\n## Overview\nDo the thing."
	selfWrapped := "```markdown\n" + intended + "\n```"
	raw := `<invoke name="write_file"><parameter name="path">PLAN.md</parameter>` +
		`<parameter name="content">` + selfWrapped + `</parameter></invoke>`
	srv, _ := newTestServer(t, [][]string{successFrame(raw)})

	_, body := postChat(t, srv.URL, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "write the plan"}},
		"tools":    []map[string]any{writeFileToolParam()},
	})
	choices, _ := body["choices"].([]any)
	if len(choices) != 1 {
		t.Fatalf("body = %+v", body)
	}
	msg := choices[0].(map[string]any)["message"].(map[string]any)
	calls, _ := msg["tool_calls"].([]any)
	if len(calls) != 1 {
		t.Fatalf("tool_calls = %+v", calls)
	}
	fn := calls[0].(map[string]any)["function"].(map[string]any)
	var args map[string]any
	if err := json.Unmarshal([]byte(fn["arguments"].(string)), &args); err != nil {
		t.Fatal(err)
	}
	if args["content"] != intended {
		t.Errorf("self-wrap not stripped through the real pipeline:\ngot:  %q\nwant: %q", args["content"], intended)
	}
}

func TestServerChatCompletionReplyToolBecomesText(t *testing.T) {
	// "text" is the reply tool's sole (body) param, so it has no header
	// parameter - the final <parameter> value IS the text argument verbatim.
	srv, _ := newTestServer(t, [][]string{successFrame(
		`<invoke name="reply"><parameter name="text">the answer</parameter></invoke>`,
	)})
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

// TestServerChatCompletion_IllustrativeCallStaysText drives the issue #701 fix
// through the real server pipeline: a long written answer that quotes
// read_file-style syntax gets parsed into a bogus read_file call (missing the
// required file_path) plus scrap leftover. The response must come back as the
// full text, not a tool call.
func TestServerChatCompletion_IllustrativeCallStaysText(t *testing.T) {
	prose := "## Using design/ on the current plans\n\n" + strings.Repeat(
		"Read the plan file (limit 100 lines) and cross-reference each UI Framework "+
			"component against the 5 open items before editing. ", 6,
	)
	raw := prose + "\n\n<invoke name=\"read_file\"><parameter name=\"limit\">100</parameter></invoke>"
	srv, _ := newTestServer(t, [][]string{successFrame(raw)})

	_, body := postChat(t, srv.URL, map[string]any{
		"model":    "m365-copilot",
		"messages": []map[string]any{{"role": "user", "content": "how can design/ be used"}},
		"tools": []map[string]any{{
			"type": "function",
			"function": map[string]any{
				"name": "read_file",
				"parameters": map[string]any{
					"properties": map[string]any{
						"file_path": map[string]any{"type": "string"},
						"limit":     map[string]any{"type": "integer"},
					},
					"required": []any{"file_path"},
				},
			},
		}},
	})
	choice := body["choices"].([]any)[0].(map[string]any)
	if choice["finish_reason"] != "stop" {
		t.Fatalf("finish_reason = %v, body=%+v", choice["finish_reason"], body)
	}
	msg := choice["message"].(map[string]any)
	if calls, _ := msg["tool_calls"].([]any); len(calls) != 0 {
		t.Fatalf("illustrative call should not execute: %+v", calls)
	}
	if got, _ := msg["content"].(string); !strings.Contains(got, "UI Framework") ||
		!strings.Contains(got, "Using design/ on the current plans") {
		t.Errorf("full prose answer not returned, got %q", got)
	}
}

// --- confabulation / hallucination forced retry ---

func TestServerChatCompletionConfabulationRetry(t *testing.T) {
	frames := [][]string{
		successFrame("I cannot access the files, please paste them here"),
		successFrame(`<invoke name="bash"><parameter name="command">ls -la</parameter></invoke>`),
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

// A reasoning-tone model can ignore the fenced-retry instruction too --
// confirmed live: it fabricated a bash action against M365's own sandbox on
// both the natural turn and the forced retry, never producing a real tool
// call. The third attempt falls back to Claude tone, which must succeed
// where the reasoning tone couldn't.
func TestServerChatCompletionToneFallbackAfterRepeatedHallucination(t *testing.T) {
	frames := [][]string{
		successFrame("I cannot access the files, `/mnt/data` is empty"),
		successFrame("I cannot access the files, `/mnt/data` is still empty"),
		successFrame(`<invoke name="list_files"><parameter name="path">/tmp</parameter></invoke>`),
	}
	srv, wsSrv := newTestServer(t, frames)
	base := srv.URL

	_, body := postChat(t, base, map[string]any{
		"model":    "gpt-5.6-think-deeper",
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
		t.Fatalf("tone fallback should have produced a real tool call: %+v", body)
	}
	if wsSrv.turnCount() != 3 {
		t.Errorf("expected the confab retry plus one tone-fallback retry (3 WS turns), got %d", wsSrv.turnCount())
	}
	// The response must still report the originally requested model, not the
	// internal fallback tone.
	if body["model"] != "gpt-5.6-think-deeper" {
		t.Errorf("response model = %v, want the originally requested model", body["model"])
	}
	// The third (fallback) chat frame must carry the Claude tone.
	frame := wsSrv.chats[2]
	if !strings.Contains(frame, `"tone":"Claude_Sonnet"`) {
		t.Errorf("fallback turn did not use Claude tone: %s", frame)
	}
}

// Once a tool call has already succeeded earlier in the conversation, a
// plain-text final reply with no new tool call is the normal, expected end
// of a tool-use turn (the model synthesizing the result) -- it must NOT
// trigger the Claude-tone fallback. Only the everActed-false (opening-turn)
// case fires unconditionally; this guards the common case doesn't pay for it.
func TestServerChatCompletionNoToneFallbackAfterSuccessfulToolCall(t *testing.T) {
	frames := [][]string{successFrame("Here is a summary of what I found.")}
	srv, wsSrv := newTestServer(t, frames)
	base := srv.URL

	_, body := postChat(t, base, map[string]any{
		"model": "gpt-5.6-think-deeper",
		"messages": []map[string]any{
			{"role": "user", "content": "list files then summarize"},
			{
				"role": "assistant",
				"tool_calls": []map[string]any{{
					"id":       "call_1",
					"function": map[string]any{"name": "list_files", "arguments": `{"path":"/tmp"}`},
				}},
			},
			{"role": "tool", "tool_call_id": "call_1", "content": "a.txt\nb.txt"},
		},
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
	if choice["finish_reason"] != "stop" {
		t.Fatalf("plain synthesis reply should not trigger a fallback retry: %+v", body)
	}
	if wsSrv.turnCount() != 1 {
		t.Errorf("expected exactly one WS turn (no fallback), got %d", wsSrv.turnCount())
	}
}

// A session with no shell tool must not be told to write a bash invoke on
// retry -- BuildSpecMap only aliases that invoke name onto a real tool when
// a shell tool is registered, so asking for it here would never parse into a
// call. The retry prompt must instead point at a real registered tool
// (list_files), and the model complying with that should succeed.
func TestServerChatCompletionConfabulationRetryNoShellTool(t *testing.T) {
	frames := [][]string{
		successFrame("I cannot access the files, please paste them here"),
		successFrame(`<invoke name="list_files"><parameter name="path">/tmp</parameter></invoke>`),
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
	srv, _ := newTestServer(t, [][]string{successFrame(
		`<invoke name="bash"><parameter name="command">ls -la</parameter></invoke>`,
	)})
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
