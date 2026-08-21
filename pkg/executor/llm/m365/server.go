package m365

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// This file exposes the M365 client as a local OpenAI-compatible HTTP endpoint
// (/v1/chat/completions and /v1/models). Both kdeps modes talk to LLMs over that
// wire shape, so serving it locally lets M365 plug into the workflow executor and
// the agent loop without either knowing it is really a SignalR WebSocket.

const (
	confabForcePrompt = "The working directory and the files named in the task ARE present on a real filesystem right now. Do NOT ask me to paste anything, and do NOT say commands return no output - you have not run any command yet. Emit ONE ```bash block this turn: run `ls -la` and `cat` the relevant files. Output only the ```bash block, nothing else."

	hallucinationForcePrompt = "You have NOT actually done that - no tool ran this turn, so nothing changed on disk. Do not claim a file was created, replaced, or updated until a <tool_response> confirms it. Emit ONE ```bash block now that performs the change for real (write the file with a `cat > path <<'EOF' ... EOF` heredoc), and nothing else."

	maxRetries      = 2
	shortRetryDelay = 2 * time.Second
	maxIdle         = 30 * time.Minute

	// defaultOutputCeiling flags a reply as likely truncated at/over this length.
	defaultOutputCeiling = 12_000
	// hashShift is the multiplier shift of the conversation-fingerprint hash.
	hashShift = 5
	// percentScale converts a ratio to a 0-100 percentage.
	percentScale = 100
	// chunkExtraKeys is spare capacity for keys added to a copied SSE base map.
	chunkExtraKeys = 2
	// roundHalf rounds a percentage to the nearest integer.
	roundHalf = 0.5
)

func outputCharCeiling() int { return intEnv("M365_OUTPUT_CHAR_CEILING", defaultOutputCeiling) }

// outputFinishReason returns "length" when the answer is at/over the empirical
// output ceiling (the service concludes early rather than truncating), else "stop".
func outputFinishReason(text string) string {
	if c := outputCharCeiling(); c > 0 && len(text) >= c {
		return "length"
	}
	return "stop"
}

// chatRequest is the subset of the OpenAI chat-completions body we consume.
type chatRequest struct {
	Model         string          `json:"model"`
	Messages      []Message       `json:"messages"`
	Tools         []ToolDef       `json:"tools"`
	ToolChoice    json.RawMessage `json:"tool_choice"`
	Stream        bool            `json:"stream"`
	StreamOptions *struct {
		IncludeUsage bool `json:"include_usage"`
	} `json:"stream_options"`
}

func (r *chatRequest) parsedToolChoice() *ToolChoice {
	if len(r.ToolChoice) == 0 {
		return nil
	}
	var s string
	if json.Unmarshal(r.ToolChoice, &s) == nil {
		return &ToolChoice{Mode: s}
	}
	var obj struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(r.ToolChoice, &obj) == nil && obj.Function.Name != "" {
		return &ToolChoice{Mode: "function", FunctionName: obj.Function.Name}
	}
	return nil
}

// conversationState is one client conversation mapped to an M365 session.
type conversationState struct {
	session          *ModelSession
	sentMessageCount int
	lastAccessed     time.Time
}

// SessionPool routes each client conversation (keyed by its first user message)
// to a persistent ModelSession.
type SessionPool struct {
	mu            sync.Mutex
	conversations map[string]*conversationState
	opts          ModelSessionOptions
}

// NewSessionPool creates a pool that builds sessions with the given options.
func NewSessionPool(opts ModelSessionOptions) *SessionPool {
	return &SessionPool{conversations: map[string]*conversationState{}, opts: opts}
}

func (p *SessionPool) resolve(messages []Message) *conversationState {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for k, st := range p.conversations {
		if now.Sub(st.lastAccessed) > maxIdle {
			delete(p.conversations, k)
		}
	}

	fp := fingerprint(messages)
	if st, ok := p.conversations[fp]; ok {
		if len(messages) < st.sentMessageCount {
			st.session.reset()
			st.sentMessageCount = 0
		}
		st.lastAccessed = now
		return st
	}
	st := &conversationState{
		session:      NewModelSession(p.opts),
		lastAccessed: now,
	}
	p.conversations[fp] = st
	return st
}

func fingerprint(messages []Message) string {
	text := ""
	for _, m := range messages {
		if m.Role == "user" {
			text = GetMessageContent(m)
			break
		}
	}
	var hash int32
	for _, c := range text {
		hash = (hash << hashShift) - hash + c
	}
	return strconv.Itoa(int(hash))
}

// formatDeltaMessages renders only the new messages of a follow-up turn (the
// service is stateful and already holds the earlier turns).
func formatDeltaMessages(messages []Message) string {
	var parts []string
	for _, m := range messages {
		switch m.Role {
		case "assistant", "system":
			continue // already server-side / not resent on follow-ups
		case "tool":
			name := m.Name
			if name == "" {
				name = "unknown"
			}
			callID := m.ToolCallID
			if callID == "" {
				callID = "?"
			}
			parts = append(
				parts,
				"<tool_response name=\""+name+"\" call_id=\""+callID+"\">\n"+GetMessageContent(m)+"\n</tool_response>",
			)
		default:
			parts = append(parts, "<"+m.Role+">\n"+GetMessageContent(m)+"\n</"+m.Role+">")
		}
	}
	return strings.Join(parts, "\n\n")
}

// Server is the OpenAI-compatible HTTP front end for M365.
type Server struct {
	pool *SessionPool
}

// NewServer creates a server with a fresh session pool.
func NewServer(opts ModelSessionOptions) *Server {
	return &Server{pool: NewSessionPool(opts)}
}

// ServeHTTP routes the two supported endpoints.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/v1/models" && r.Method == http.MethodGet:
		s.handleModels(w)
	case strings.HasSuffix(r.URL.Path, "/chat/completions") && r.Method == http.MethodPost:
		s.handleChat(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleModels(w http.ResponseWriter) {
	models := AvailableModels()
	data := make([]map[string]any, 0, len(models))
	for _, m := range models {
		entry := map[string]any{"id": m, "object": "model", "owned_by": "m365"}
		data = append(data, entry)
	}
	writeJSON(w, http.StatusOK, map[string]any{"object": "list", "data": data})
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	kdeps_debug.Log("enter: Server.handleChat")
	var body chatRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "invalid_request_error")
		return
	}
	newCompletion(s.pool, &body).run(r.Context(), w)
}

// completion carries the per-request state through the retry/produce pipeline.
type completion struct {
	body         *chatRequest
	conv         *conversationState
	session      *ModelSession
	hasTools     bool
	useToolAgent bool
	text         string

	lastThrottle      *Throttle
	lastContentOrigin string
	lastMessageType   string
	lastScores        map[string]float64
	lastTurnCount     *int
}

func newCompletion(pool *SessionPool, body *chatRequest) *completion {
	conv := pool.resolve(body.Messages)
	tc := body.parsedToolChoice()
	hasTools := len(body.Tools) > 0 && (tc == nil || tc.Mode != "none")

	tone := GetToneForModel(body.Model)
	isClaudeTone := strings.HasPrefix(tone, "Claude_")
	useToolAgent := hasTools && (os.Getenv("M365_FORCE_AGENT") == "1" || !isClaudeTone)

	c := &completion{
		body:         body,
		conv:         conv,
		session:      conv.session,
		hasTools:     hasTools,
		useToolAgent: useToolAgent,
	}

	// Full prompt on the first turn; a delta of new messages on follow-ups.
	if c.session.TurnCount() == 0 || conv.sentMessageCount == 0 {
		c.text = FormatMessages(body.Messages, body.Tools, tc, c.session.ConversationID(), "")
	} else {
		newMessages := body.Messages[minInt(conv.sentMessageCount, len(body.Messages)):]
		delta := ""
		if len(newMessages) > 0 {
			delta = formatDeltaMessages(newMessages)
		}
		if delta != "" {
			c.text = delta
		} else {
			c.text = "Please continue."
		}
	}
	return c
}

// producedKind is the outcome of a completed turn.
type producedKind int

const (
	producedError producedKind = iota
	producedText
	producedTools
)

type produced struct {
	kind      producedKind
	text      string
	toolCalls []ParsedToolCall
	errStatus int
	errMsg    string
	errType   string
}

// tryRefreshAgent re-resolves the tool-calling agent once per completion
// (guarded by *agentRefreshed, shared across runBuffered's retry branches).
// If the agent id changed, it resets c.text to originalText and sleeps
// before the caller's retry. Returns true when the caller should retry.
func (c *completion) tryRefreshAgent(ctx context.Context, agentRefreshed *bool, originalText string) bool {
	if *agentRefreshed {
		return false
	}
	*agentRefreshed = true
	changed, _ := c.session.RefreshAgent(ctx)
	if !changed {
		return false
	}
	c.text = originalText
	sleep(ctx, shortRetryDelay)
	return true
}

// runBuffered runs one turn (with retries on an empty reply), forwarding text
// deltas to onDelta as they arrive. onDelta may be nil.
//
//nolint:gocognit // Disengage/throttle/dead-agent retry branches in one loop
func (c *completion) runBuffered(ctx context.Context, onDelta func(string)) (string, *produced) {
	agentRefreshed := false
	disengageRetried := false
	originalText := c.text
	awaitDegradationBackoff(ctx)

	for attempt := 0; attempt <= maxRetries; attempt++ {
		stream, err := c.session.Run(ctx, c.text, c.body.Model, c.useToolAgent)
		if err != nil {
			return "", upstreamError(err.Error())
		}

		var b strings.Builder
		for delta := range stream.Deltas() {
			b.WriteString(delta)
			if onDelta != nil {
				onDelta(delta)
			}
		}
		fullText := b.String()
		if ft := stream.FullText(); len(ft) > len(fullText) {
			fullText = ft
		}
		if serr := stream.Err(); serr != nil {
			// A completion error on the tool-agent path (getOrCreateAgent) can
			// mean the cached agent id went stale server-side -- exactly the
			// failure mode the empty-reply branch below already recovers from
			// via RefreshAgent, just arriving as an error frame instead of a
			// silent empty one. Give it the same one-shot recovery before
			// surfacing the error, instead of failing immediately.
			if c.useToolAgent && attempt < maxRetries &&
				c.tryRefreshAgent(ctx, &agentRefreshed, originalText) {
				continue
			}
			return "", upstreamError(serr.Error())
		}

		c.lastThrottle = stream.Throttle()
		c.lastContentOrigin = stream.ContentOrigin()
		c.lastMessageType = stream.MessageType()
		c.lastScores = stream.Scores()
		c.lastTurnCount = stream.TurnCount()

		if stream.HasContent() || fullText != "" {
			noteRequestOutcome(false, c.session.ConversationID())
			return fullText, nil
		}

		// Disengaged is a safety refusal, not a transient empty. Retry once with the
		// low-override "softened" framing in a fresh conversation; otherwise fail fast
		// to preserve the per-conversation quota.
		if c.lastMessageType == "Disengaged" {
			if c.hasTools && !disengageRetried && os.Getenv("M365_NO_DISENGAGE_RETRY") == "" {
				disengageRetried = true
				c.session.NewConversation()
				c.text = FormatMessages(
					c.body.Messages,
					c.body.Tools,
					c.body.parsedToolChoice(),
					c.session.ConversationID(),
					"softened",
				)
				attempt-- // free retry, bounded by disengageRetried
				continue
			}
			return "", &produced{
				kind:      producedError,
				errStatus: http.StatusBadGateway,
				errType:   "disengaged",
				errMsg: "M365 Copilot disengaged from this request (its safety filter " +
					"declined to answer). Reduce the toolset or use the default model.",
			}
		}

		// Empty reply. An at-limit throttle is a rate limit; otherwise retry a couple
		// of times, re-resolving a possibly-deleted agent once.
		if t := c.lastThrottle; t != nil && t.Current >= t.Max {
			return "", rateLimitError(t)
		}
		if attempt < maxRetries {
			if c.tryRefreshAgent(ctx, &agentRefreshed, originalText) {
				continue
			}
			sleep(ctx, shortRetryDelay)
			c.text = "Please continue."
		} else {
			noteRequestOutcome(true, c.session.ConversationID())
			return "", emptyResponseError(c.lastThrottle)
		}
	}
	noteRequestOutcome(true, c.session.ConversationID())
	return "", emptyResponseError(nil)
}

// produce runs the turn and post-processes it into text or tool calls.
//
//nolint:gocognit // confab retry, prose guard, reply-tool, and multi-tool trimming
func (c *completion) produce(ctx context.Context, onDelta func(string)) *produced {
	if !c.hasTools {
		fullText, perr := c.runBuffered(ctx, onDelta)
		if perr != nil {
			return perr
		}
		c.conv.sentMessageCount = len(c.body.Messages)
		return &produced{kind: producedText, text: fullText}
	}

	fullText, perr := c.runBuffered(ctx, nil)
	if perr != nil {
		return perr
	}
	c.conv.sentMessageCount = len(c.body.Messages)
	parsed := ParseToolCalls(fullText, c.body.Tools)

	// Salvage turn-1 confabulation / hallucinated completion by forcing a retry.
	maxConfab := intEnv("M365_CONFAB_RETRIES", 1)
	if os.Getenv("M365_NO_CONFAB_RETRY") != "" {
		maxConfab = 0
	}
	everActed := false
	for _, m := range c.body.Messages {
		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			everActed = true
			break
		}
	}
	for attempt := 0; attempt < maxConfab && !parsed.HasToolCalls; attempt++ {
		confab := LooksLikeConfabulation(parsed.TextContent)
		halluc := !everActed && LooksLikeHallucinatedCompletion(parsed.TextContent)
		if !confab && !halluc {
			break
		}
		if confab {
			c.text = confabForcePrompt
		} else {
			c.text = hallucinationForcePrompt
		}
		retryText, rerr := c.runBuffered(ctx, nil)
		if rerr != nil {
			return rerr
		}
		c.conv.sentMessageCount = len(c.body.Messages)
		fullText = retryText
		parsed = ParseToolCalls(fullText, c.body.Tools)
	}

	// Document guard: a prose answer full of code fences is not an action turn.
	if IsProseDocument(parsed) {
		parsed = ParseResult{HasToolCalls: false, TextContent: fullText}
	}

	// Fail-closed: strip stray prose emitted alongside tool calls.
	if parsed.HasToolCalls && strings.TrimSpace(parsed.TextContent) != "" {
		parsed.TextContent = ""
	}

	if parsed.HasToolCalls {
		if p := finalizeToolCalls(&parsed, fullText); p != nil {
			return p
		}
	}

	if parsed.HasToolCalls && len(parsed.ToolCalls) > 0 {
		return &produced{kind: producedTools, toolCalls: parsed.ToolCalls}
	}
	return &produced{kind: producedText, text: fullText}
}

// finalizeToolCalls converts a synthetic reply() call back to plain text and
// enforces one call per turn. It returns a non-nil produced only when the reply
// resolves to plain text.
func finalizeToolCalls(parsed *ParseResult, fullText string) *produced {
	var realCalls []ParsedToolCall
	var replyCall *ParsedToolCall
	for i := range parsed.ToolCalls {
		if parsed.ToolCalls[i].Name == "reply" {
			replyCall = &parsed.ToolCalls[i]
		} else {
			realCalls = append(realCalls, parsed.ToolCalls[i])
		}
	}
	if replyCall != nil && len(realCalls) == 0 {
		return &produced{kind: producedText, text: replyTextFrom(replyCall.Arguments, fullText)}
	}
	if len(realCalls) > 0 {
		parsed.ToolCalls = realCalls
	}
	// One call per turn unless opted out: executing a batched plan runs later
	// steps on guessed state.
	if os.Getenv("M365_ALLOW_MULTI_TOOL") == "" && len(parsed.ToolCalls) > 1 {
		parsed.ToolCalls = parsed.ToolCalls[:1]
	}
	return nil
}

func replyTextFrom(arguments, fallback string) string {
	var args map[string]any
	if json.Unmarshal([]byte(arguments), &args) == nil {
		for _, k := range []string{"text", "message", "content"} {
			if v, ok := args[k].(string); ok && v != "" {
				return v
			}
		}
	}
	return fallback
}

// run renders the produced turn as JSON or an early-flushed SSE stream.
func (c *completion) run(ctx context.Context, w http.ResponseWriter) {
	completionID := "chatcmpl-" + uuid.NewString()
	created := time.Now().Unix()
	includeUsage := c.body.StreamOptions != nil && c.body.StreamOptions.IncludeUsage

	if !c.body.Stream {
		p := c.produce(ctx, nil)
		switch p.kind {
		case producedError:
			writeError(w, p.errStatus, p.errMsg, p.errType)
		case producedTools:
			writeJSON(w, http.StatusOK, map[string]any{
				"id": completionID, "object": "chat.completion", "created": created, "model": c.body.Model,
				"choices": []any{map[string]any{
					"index": 0,
					"message": map[string]any{
						"role":       "assistant",
						"content":    nil,
						"tool_calls": toolCallsJSON(p.toolCalls),
					},
					"finish_reason": "tool_calls",
				}},
				"usage": c.usage(),
			})
		case producedText:
			writeJSON(w, http.StatusOK, map[string]any{
				"id": completionID, "object": "chat.completion", "created": created, "model": c.body.Model,
				"choices": []any{map[string]any{
					"index":         0,
					"message":       map[string]any{"role": "assistant", "content": p.text},
					"finish_reason": outputFinishReason(p.text),
				}},
				"usage": c.usage(),
			})
		}
		return
	}

	c.runStream(ctx, w, completionID, created, includeUsage)
}

// runStream flushes HTTP 200 immediately, then forwards live deltas and the
// final render as SSE chunks.
//
//nolint:funlen // early-flush SSE setup plus the three render branches
func (c *completion) runStream(
	ctx context.Context,
	w http.ResponseWriter,
	completionID string,
	created int64,
	includeUsage bool,
) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming unsupported", "server_error")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	base := map[string]any{
		"id":      completionID,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   c.body.Model,
	}
	var mu sync.Mutex
	send := func(obj map[string]any) {
		mu.Lock()
		defer mu.Unlock()
		data, _ := json.Marshal(obj)
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	send(chunk(base, map[string]any{"role": "assistant"}, nil))

	sent := ""
	var liveDelta func(string)
	if !c.hasTools {
		liveDelta = func(d string) {
			if d == "" {
				return
			}
			sent += d
			send(chunk(base, map[string]any{"content": d}, nil))
		}
	}

	p := c.produce(ctx, liveDelta)

	switch p.kind {
	case producedError:
		// A standard OpenAI-compatible SSE client only understands
		// choices[].delta.content and choices[].finish_reason -- it has no
		// notion of the bare errChunk["error"] object below. Without a
		// content delta and a finish_reason chunk, such a client (which is
		// exactly what kdeps' own generic streaming consumer is) sees a
		// stream with nothing recognizable in it at all: zero content, no
		// completion signal. Confirmed live as m365-copilot's "auto" tone
		// completing with completion_tokens=0 on a genuine upstream error
		// ("Failed to invoke 'Chat' due to an error on the server.") that
		// never reached the user. Send the error as visible text first, so
		// any client shows it, then the raw object for one sophisticated
		// enough to use it, then close the turn properly.
		msg := fmt.Sprintf("[m365 error: %s]", p.errMsg)
		send(chunk(base, map[string]any{"content": msg}, nil))
		errChunk := copyMap(base)
		errChunk["error"] = map[string]any{"message": p.errMsg, "type": "upstream_error"}
		send(errChunk)
		final := chunk(base, map[string]any{}, ptrStr("stop"))
		if includeUsage {
			final["usage"] = c.usage()
		}
		send(final)
	case producedTools:
		for i, tc := range p.toolCalls {
			send(chunk(base, map[string]any{"tool_calls": []any{map[string]any{
				"index": i, "id": tc.ID, "type": "function",
				"function": map[string]any{"name": tc.Name, "arguments": tc.Arguments},
			}}}, nil))
		}
		final := chunk(base, map[string]any{}, ptrStr("tool_calls"))
		if includeUsage {
			final["usage"] = c.usage()
		}
		send(final)
	case producedText:
		remainder := ""
		if strings.HasPrefix(p.text, sent) {
			remainder = p.text[len(sent):]
		}
		if remainder != "" {
			send(chunk(base, map[string]any{"content": remainder}, nil))
		}
		final := chunk(base, map[string]any{}, ptrStr(outputFinishReason(p.text)))
		if includeUsage {
			final["usage"] = c.usage()
		}
		send(final)
	}

	mu.Lock()
	fmt.Fprint(w, "data: [DONE]\n\n")
	flusher.Flush()
	mu.Unlock()
}

// usage builds the OpenAI usage block plus M365 conversation-quota extensions.
func (c *completion) usage() map[string]any {
	u := map[string]any{"prompt_tokens": 0, "completion_tokens": 0, "total_tokens": 0}
	if t := c.lastThrottle; t != nil {
		u["x_m365_conversation_messages"] = t.Current
		u["x_m365_conversation_max"] = t.Max
		pct := 0
		if t.Max > 0 {
			pct = minInt(percentScale, int(float64(t.Current)/float64(t.Max)*percentScale+roundHalf))
		}
		u["x_m365_conversation_pct"] = pct
		u["x_m365_conversation_remaining"] = maxInt(0, t.Max-t.Current)
	}
	if c.lastContentOrigin != "" {
		u["x_m365_content_origin"] = c.lastContentOrigin
	}
	if c.lastMessageType != "" {
		u["x_m365_message_type"] = c.lastMessageType
	}
	if c.lastTurnCount != nil {
		u["x_m365_turn_count"] = *c.lastTurnCount
	}
	if len(c.lastScores) > 0 {
		u["x_m365_classifier_scores"] = c.lastScores
		if v, ok := c.lastScores["dea_violation"]; ok {
			u["x_m365_dea_score"] = v
		}
		if v, ok := c.lastScores["BotOffense"]; ok {
			u["x_m365_offense_score"] = v
		}
	}
	return u
}

// --- small helpers ---

func chunk(base, delta map[string]any, finishReason *string) map[string]any {
	m := copyMap(base)
	choice := map[string]any{"index": 0, "delta": delta}
	if finishReason != nil {
		choice["finish_reason"] = *finishReason
	} else {
		choice["finish_reason"] = nil
	}
	m["choices"] = []any{choice}
	return m
}

func copyMap(src map[string]any) map[string]any {
	dst := make(map[string]any, len(src)+chunkExtraKeys)
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func toolCallsJSON(calls []ParsedToolCall) []any {
	out := make([]any, 0, len(calls))
	for _, tc := range calls {
		out = append(out, map[string]any{
			"id": tc.ID, "type": "function",
			"function": map[string]any{"name": tc.Name, "arguments": tc.Arguments},
		})
	}
	return out
}

func upstreamError(msg string) *produced {
	return &produced{kind: producedError, errStatus: http.StatusBadGateway, errMsg: msg, errType: "upstream_error"}
}

func rateLimitError(t *Throttle) *produced {
	msg := "M365 Copilot returned an empty response. You may be rate limited. Please wait and try again."
	if t != nil {
		msg = fmt.Sprintf(
			"M365 Copilot rate limited (%d/%d messages used). Please wait and try again.",
			t.Current,
			t.Max,
		)
	}
	return &produced{
		kind:      producedError,
		errStatus: http.StatusTooManyRequests,
		errMsg:    msg,
		errType:   "rate_limit_error",
	}
}

func emptyResponseError(t *Throttle) *produced {
	detail := ""
	if t != nil {
		detail = fmt.Sprintf(" (throttle %d/%d)", t.Current, t.Max)
	}
	return &produced{
		kind:      producedError,
		errStatus: http.StatusBadGateway,
		errType:   "upstream_empty_response",
		errMsg: "M365 Copilot returned an empty response" + detail +
			" - likely a content filter, an invalid agent/session, or a transient upstream error.",
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message, errType string) {
	writeJSON(w, status, map[string]any{"error": map[string]any{"message": message, "type": errType}})
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}

func ptrStr(s string) *string { return &s }

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
