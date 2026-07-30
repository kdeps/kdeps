package m365

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// --- copilot.go ---

func TestGetToneForModel(t *testing.T) {
	cases := map[string]string{
		"m365-copilot":       "auto",
		"quick":              "Gpt_Quick",
		"claude-opus":        "Claude_Opus",
		"claude-sonnet-4.5":  "Claude_Sonnet",
		"claude-anything-xy": "Claude_Sonnet", // unmapped claude-* falls back to Claude_Sonnet
		"totally-unknown":    "auto",          // falls back to default tone
	}
	for model, want := range cases {
		if got := GetToneForModel(model); got != want {
			t.Errorf("GetToneForModel(%q) = %q, want %q", model, got, want)
		}
	}
}

func TestAvailableModels(t *testing.T) {
	models := AvailableModels()
	if len(models) == 0 {
		t.Fatal("AvailableModels returned none")
	}
	found := false
	for _, m := range models {
		if m == "m365-copilot" {
			found = true
		}
	}
	if !found {
		t.Error("expected m365-copilot in AvailableModels")
	}
}

func makeJWT(t *testing.T, claims JWTClaims) string {
	t.Helper()
	payload, err := json.Marshal(claims)
	if err != nil {
		t.Fatal(err)
	}
	seg := base64.RawURLEncoding.EncodeToString(payload)
	return "hdr." + seg + ".sig"
}

func TestDecodeJWT(t *testing.T) {
	tok := makeJWT(t, JWTClaims{OID: "oid-1", TID: "tid-1"})
	claims, err := DecodeJWT(tok)
	if err != nil {
		t.Fatal(err)
	}
	if claims.OID != "oid-1" || claims.TID != "tid-1" {
		t.Errorf("got %+v", claims)
	}
}

func TestDecodeJWTErrors(t *testing.T) {
	if _, err := DecodeJWT("onlyonesegment"); err == nil {
		t.Error("want error for single segment")
	}
	if _, err := DecodeJWT(makeJWT(t, JWTClaims{OID: "x"})); err == nil {
		t.Error("want error for missing tid")
	}
}

// --- fenced.go ---

func shellToolDef() ToolDef {
	return ToolDef{Function: ToolFunction{
		Name: "bash",
		Parameters: &ToolParameters{
			Properties: map[string]json.RawMessage{"command": json.RawMessage(`{"type":"string"}`)},
		},
	}}
}

func writeToolDef() ToolDef {
	return ToolDef{Function: ToolFunction{
		Name: "write_file",
		Parameters: &ToolParameters{Properties: map[string]json.RawMessage{
			"path":    json.RawMessage(`{"type":"string"}`),
			"content": json.RawMessage(`{"type":"string"}`),
		}},
	}}
}

func editToolDef() ToolDef {
	return ToolDef{Function: ToolFunction{
		Name: "edit_file",
		Parameters: &ToolParameters{Properties: map[string]json.RawMessage{
			"path": json.RawMessage(`{"type":"string"}`),
			"old":  json.RawMessage(`{"type":"string"}`),
			"new":  json.RawMessage(`{"type":"string"}`),
		}},
	}}
}

func TestDeriveFencedSpec(t *testing.T) {
	spec := DeriveFencedSpec(writeToolDef())
	if spec.BodyParam != "content" {
		t.Errorf("body param = %q, want content", spec.BodyParam)
	}
	if len(spec.HeaderParams) != 1 || spec.HeaderParams[0] != "path" {
		t.Errorf("header params = %v", spec.HeaderParams)
	}

	edit := DeriveFencedSpec(editToolDef())
	if edit.EditPair == nil || edit.EditPair.search != "old" || edit.EditPair.replace != "new" {
		t.Errorf("edit pair = %+v", edit.EditPair)
	}
}

func TestBuildSpecMapShellAlias(t *testing.T) {
	m := BuildSpecMap([]ToolDef{shellToolDef()})
	for _, lang := range []string{"bash", "sh", "shell", "zsh"} {
		if _, ok := m[lang]; !ok {
			t.Errorf("missing shell alias %q", lang)
		}
	}
}

func TestRenderAndParseFencedRoundTrip(t *testing.T) {
	tools := []ToolDef{writeToolDef()}
	specs := BuildSpecMap(tools)
	rendered := RenderFencedCall(specs["write_file"], map[string]any{
		"path": "main.go", "content": "package main",
	})
	calls, leftover := ParseFencedToolCalls(rendered, specs)
	if len(calls) != 1 {
		t.Fatalf("got %d calls, leftover=%q", len(calls), leftover)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != "main.go" || args["content"] != "package main" {
		t.Errorf("args = %v", args)
	}
}

func TestParseFencedShellRouting(t *testing.T) {
	specs := BuildSpecMap([]ToolDef{shellToolDef()})
	calls, _ := ParseFencedToolCalls("```bash\nls -la\n```", specs)
	if len(calls) != 1 || calls[0].Name != "bash" {
		t.Fatalf("calls = %+v", calls)
	}
}

func TestFormatFencedToolDefinitionsVariants(t *testing.T) {
	tools := []ToolDef{shellToolDef()}
	for _, v := range []string{"baseline", "minimal", "softened", "unknown-falls-to-baseline", ""} {
		out := FormatFencedToolDefinitions(tools, v)
		if !strings.Contains(out, "<tools>") {
			t.Errorf("variant %q missing <tools> block", v)
		}
	}
}

// --- tools.go ---

func TestGetMessageContent(t *testing.T) {
	if GetMessageContent(Message{Content: "hi"}) != "hi" {
		t.Error("string content")
	}
	parts := []any{map[string]any{"type": "text", "text": "a"}, map[string]any{"type": "text", "text": "b"}}
	if GetMessageContent(Message{Content: parts}) != "ab" {
		t.Error("array content")
	}
	if GetMessageContent(Message{}) != "" {
		t.Error("nil content")
	}
}

func TestParseToolCallsFenced(t *testing.T) {
	tools := []ToolDef{shellToolDef()}
	res := ParseToolCalls("```bash\necho hi\n```", tools)
	if !res.HasToolCalls || len(res.ToolCalls) != 1 {
		t.Fatalf("res = %+v", res)
	}
}

func TestParseToolCallsJSONFallback(t *testing.T) {
	res := ParseToolCalls(`prefix {"tool":"foo","arguments":{"x":1}} suffix`, nil)
	if !res.HasToolCalls || res.ToolCalls[0].Name != "foo" {
		t.Fatalf("res = %+v", res)
	}
}

func TestParseToolCallsPlainText(t *testing.T) {
	res := ParseToolCalls("just an answer", []ToolDef{shellToolDef()})
	if res.HasToolCalls || res.TextContent != "just an answer" {
		t.Fatalf("res = %+v", res)
	}
}

func TestCleanLooseText(t *testing.T) {
	if got := cleanLooseText(`{"final":"done"}`); got != "done" {
		t.Errorf("final unwrap = %q", got)
	}
	if got := cleanLooseText(`answer {"confidence":0.5}`); got != "answer" {
		t.Errorf("confidence strip = %q", got)
	}
}

func TestLooksLikeConfabulation(t *testing.T) {
	if !LooksLikeConfabulation("I cannot access the files, please paste them") {
		t.Error("should detect confabulation")
	}
	if LooksLikeConfabulation("Here is the answer to your question.") {
		t.Error("false positive")
	}
	if LooksLikeConfabulation("short") {
		t.Error("too short should be false")
	}
}

func TestLooksLikeHallucinatedCompletion(t *testing.T) {
	if !LooksLikeHallucinatedCompletion("I've created the file and updated the README") {
		t.Error("should detect hallucination")
	}
	if LooksLikeHallucinatedCompletion("What would you like to do?") {
		t.Error("false positive")
	}
}

func TestIsProseDocument(t *testing.T) {
	doc := ParseResult{
		HasToolCalls: true,
		ToolCalls:    []ParsedToolCall{{Name: "bash"}, {Name: "bash"}},
		TextContent:  "# Heading\n\nlots of prose describing the document",
	}
	if !IsProseDocument(doc) {
		t.Error("markdown-header doc should be flagged")
	}
	action := ParseResult{HasToolCalls: true, ToolCalls: []ParsedToolCall{{Name: "bash"}}}
	if IsProseDocument(action) {
		t.Error("single action must not be flagged")
	}
}

func TestFormatMessagesInjectsTools(t *testing.T) {
	msgs := []Message{{Role: "user", Content: "do it"}}
	out := FormatMessages(msgs, []ToolDef{shellToolDef()}, nil, "conv-1", "baseline")
	if !strings.Contains(out, "<conversation_id>conv-1</conversation_id>") {
		t.Error("missing conversation id")
	}
	if !strings.Contains(out, "<system>") || !strings.Contains(out, "<tools>") {
		t.Error("missing tool system block")
	}
	if !strings.Contains(out, "<user>\ndo it\n</user>") {
		t.Error("missing user turn")
	}
}

func TestFormatMessagesToolResult(t *testing.T) {
	msgs := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{{
			ID: "c1", Function: ToolCallFunction{Name: "bash", Arguments: `{"command":"ls"}`},
		}}},
		{Role: "tool", ToolCallID: "c1", Name: "bash", Content: "file.txt"},
	}
	out := FormatMessages(msgs, []ToolDef{shellToolDef()}, nil, "", "")
	if !strings.Contains(out, `<tool_response tool="bash" command="ls">`) {
		t.Errorf("missing labelled tool response:\n%s", out)
	}
}

// --- totp.go (RFC 6238 SHA-1 test vector) ---

func TestTOTPRFC6238(t *testing.T) {
	// RFC 6238 seed "12345678901234567890" is base32 "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ".
	const secret = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"
	key, err := decodeBase32Secret(secret)
	if err != nil {
		t.Fatal(err)
	}
	// At counter 1 (T=59s), RFC 6238 SHA-1 expects 94287082 -> last 6 = 287082.
	if got := hotp(key, 1); got != "287082" {
		t.Errorf("hotp = %q, want 287082", got)
	}
	if _, terr := totpNow(secret); terr != nil {
		t.Errorf("totpNow: %v", terr)
	}
	if _, terr := totpNow("!!!not base32!!!"); terr == nil {
		t.Error("want error for invalid secret")
	}
}

// --- session.go ---

func TestFoldStreamText(t *testing.T) {
	ans, emit, ok := foldStreamText("hel", "hello")
	if ans != "hello" || emit != "lo" || !ok {
		t.Errorf("got %q %q %v", ans, emit, ok)
	}
	if _, _, ok2 := foldStreamText("hello", "hel"); ok2 {
		t.Error("shrink should not emit")
	}
	// A longer, non-prefix snapshot is adopted as the buffer but emits nothing.
	if a, _, ok2 := foldStreamText("abc", "xyzw"); a != "xyzw" || ok2 {
		t.Errorf("divergent adopt without emit: %q %v", a, ok2)
	}
}

func TestBuildChatArgsAgentSplit(t *testing.T) {
	agentless := NewCopilotSession(CopilotSessionOptions{})
	args := agentless.buildChatArgs("req", "hi", "m365-copilot", true)
	if _, ok := args["plugins"]; !ok {
		t.Error("agent-less should carry plugins")
	}
	if args["threadLevelGptId"] != nil {
		t.Error("agent-less threadLevelGptId must be nil")
	}

	withAgent := NewCopilotSession(CopilotSessionOptions{AgentID: "T1.b1.gpt.default"})
	aargs := withAgent.buildChatArgs("req", "hi", "m365-copilot", true)
	if _, ok := aargs["gpts"]; !ok {
		t.Error("agent path should carry gpts")
	}
	if aargs["threadLevelGptId"] != "T1.b1.gpt.default" {
		t.Error("agent path threadLevelGptId mismatch")
	}
}

func TestBuildSendPayload(t *testing.T) {
	payload, err := buildSendPayload(map[string]any{"x": 1})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(payload, rs) {
		t.Error("payload must end with record separator")
	}
	if strings.Count(payload, rs) != 2 {
		t.Errorf("expected chat+metrics frames, rs count = %d", strings.Count(payload, rs))
	}
}

func TestEmptyToNil(t *testing.T) {
	if emptyToNil("") != nil {
		t.Error("empty should be nil")
	}
	if emptyToNil("x") != "x" {
		t.Error("non-empty passthrough")
	}
}

// --- backoff.go (the pacing controller itself is tested in toolguard) ---

func TestBackoffDisabledEnv(t *testing.T) {
	t.Setenv("M365_NO_BACKOFF", "1")
	if !backoffDisabled() {
		t.Error("should be disabled")
	}
}

// --- model.go ---

func TestModelSessionAgentToggle(t *testing.T) {
	m := NewModelSession(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return "tok", nil },
		GetAgent: func(context.Context, bool) (string, error) { return "agent-1", nil },
	})
	if m.ConversationID() == "" || m.TurnCount() != 0 {
		t.Error("fresh session state")
	}
	first := m.ConversationID()
	m.NewConversation()
	if m.ConversationID() == first {
		t.Error("NewConversation should rotate the id")
	}
}

func TestModelSessionRefreshAgent(t *testing.T) {
	m := NewModelSession(ModelSessionOptions{
		GetToken: func(context.Context) (string, error) { return "tok", nil },
		GetAgent: func(context.Context, bool) (string, error) { return "agent-new", nil },
	})
	m.cachedAgentID = "agent-old"
	changed, err := m.RefreshAgent(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !changed || m.cachedAgentID != "agent-new" {
		t.Errorf("refresh: changed=%v id=%q", changed, m.cachedAgentID)
	}

	// UseAgent=false disables refresh.
	off := NewModelSession(ModelSessionOptions{UseAgentSet: true, UseAgent: false})
	if offChanged, _ := off.RefreshAgent(context.Background()); offChanged {
		t.Error("agent-less session must not refresh")
	}
}

// --- agent.go ---

func TestAgentNamingAndCache(t *testing.T) {
	if len(getInstructionsHash()) != 8 {
		t.Errorf("hash length = %d", len(getInstructionsHash()))
	}
	if !strings.HasPrefix(getAgentName(), agentBaseName+"-") {
		t.Errorf("agent name = %q", getAgentName())
	}

	dir := t.TempDir()
	t.Setenv("M365_AGENT_CACHE_FILE", filepath.Join(dir, "agent-id.json"))
	saveCachedAgent(cachedAgent{AgentID: "A.b.gpt.default", BotID: "b", InstructionsHash: "hash1234"})
	got := loadCachedAgent()
	if got == nil || got.AgentID != "A.b.gpt.default" {
		t.Errorf("round-trip cache = %+v", got)
	}
}

func TestStripEnvName(t *testing.T) {
	if got := stripEnvName("Default-FA7F-56d8"); got != "fa7f56d8" {
		t.Errorf("stripEnvName = %q", got)
	}
}

// --- auth.go ---

func TestPKCE(t *testing.T) {
	v, c := pkce()
	if v == "" || c == "" || v == c {
		t.Errorf("pkce v=%q c=%q", v, c)
	}
	sum := base64.RawURLEncoding.EncodeToString([]byte(v))
	_ = sum // verifier is opaque; just assert challenge is url-safe base64
	if _, err := base64.RawURLEncoding.DecodeString(c); err != nil {
		t.Errorf("challenge not raw-url-base64: %v", err)
	}
}

func TestBuildAuthURL(t *testing.T) {
	u, verifier := buildAuthURL(chatScopes)
	if verifier == "" || !strings.Contains(u, "code_challenge=") {
		t.Errorf("auth url = %q", u)
	}
	if !strings.Contains(u, "code_challenge_method=S256") {
		t.Error("missing S256")
	}
}

func TestScopeKeyStable(t *testing.T) {
	a := scopeKey([]string{"b", "a"})
	b := scopeKey([]string{"a", "b"})
	if a != b {
		t.Errorf("scope key not order-stable: %q vs %q", a, b)
	}
}

func TestLoadSecrets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.json")
	t.Setenv("M365_SECRETS_FILE", path)

	if loadSecrets() != nil {
		t.Error("missing file should be nil")
	}
	_ = os.WriteFile(path, []byte(`{"email":"e","password":"p","mfaSecret":"m"}`), 0o600)
	s := loadSecrets()
	if s == nil || s.Email != "e" || s.MFASecret != "m" {
		t.Errorf("secrets = %+v", s)
	}
}

func TestForceReauthNoCredentials(t *testing.T) {
	t.Setenv("M365_SECRETS_FILE", filepath.Join(t.TempDir(), "none.json"))
	// No refresh token in cache and no secrets file -> cannot reauth.
	if forceReauth(context.Background()) {
		t.Error("forceReauth should fail without cache or secrets")
	}
}

// --- server.go ---

func TestParsedToolChoice(t *testing.T) {
	none := &chatRequest{ToolChoice: json.RawMessage(`"none"`)}
	if tc := none.parsedToolChoice(); tc == nil || tc.Mode != "none" {
		t.Errorf("string tool_choice = %+v", tc)
	}
	fn := &chatRequest{ToolChoice: json.RawMessage(`{"type":"function","function":{"name":"bash"}}`)}
	if tc := fn.parsedToolChoice(); tc == nil || tc.Mode != "function" || tc.FunctionName != "bash" {
		t.Errorf("function tool_choice = %+v", tc)
	}
	empty := &chatRequest{}
	if empty.parsedToolChoice() != nil {
		t.Error("empty tool_choice should be nil")
	}
}

func TestFingerprintStable(t *testing.T) {
	a := fingerprint([]Message{{Role: "user", Content: "hello"}})
	b := fingerprint([]Message{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "hi"}})
	if a != b {
		t.Error("fingerprint should key on first user message only")
	}
}

func TestOutputFinishReason(t *testing.T) {
	t.Setenv("M365_OUTPUT_CHAR_CEILING", "10")
	if outputFinishReason("short") != "stop" {
		t.Error("short -> stop")
	}
	if outputFinishReason(strings.Repeat("x", 20)) != "length" {
		t.Error("long -> length")
	}
}

func TestFormatDeltaMessages(t *testing.T) {
	out := formatDeltaMessages([]Message{
		{Role: "assistant", Content: "skip me"},
		{Role: "tool", Name: "bash", ToolCallID: "c1", Content: "result"},
		{Role: "user", Content: "next"},
	})
	if strings.Contains(out, "skip me") {
		t.Error("assistant messages must be dropped on follow-ups")
	}
	if !strings.Contains(out, `<tool_response name="bash" call_id="c1">`) {
		t.Error("missing tool response")
	}
	if !strings.Contains(out, "<user>\nnext\n</user>") {
		t.Error("missing user delta")
	}
}

func TestServeHTTPModels(t *testing.T) {
	srv := NewServer(ModelSessionOptions{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v1/models", nil)
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	var body struct {
		Object string           `json:"object"`
		Data   []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Object != "list" || len(body.Data) == 0 {
		t.Errorf("models body = %+v", body)
	}
}

func TestServeHTTPNotFound(t *testing.T) {
	srv := NewServer(ModelSessionOptions{})
	rec := httptest.NewRecorder()
	srv.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestServeHTTPBadRequest(t *testing.T) {
	srv := NewServer(ModelSessionOptions{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader("{bad json"))
	srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestCompletionUsageThrottle(t *testing.T) {
	c := &completion{lastThrottle: &Throttle{Current: 30, Max: 60}}
	u := c.usage()
	if u["x_m365_conversation_pct"] != 50 {
		t.Errorf("pct = %v", u["x_m365_conversation_pct"])
	}
	if u["x_m365_conversation_remaining"] != 30 {
		t.Errorf("remaining = %v", u["x_m365_conversation_remaining"])
	}
}

// --- additional reachable-branch coverage ---

func TestSessionPoolResolve(t *testing.T) {
	p := NewSessionPool(ModelSessionOptions{})
	msgs := []Message{{Role: "user", Content: "hello"}}
	first := p.resolve(msgs)
	first.sentMessageCount = 2
	again := p.resolve(msgs)
	if first != again {
		t.Error("same first user message should reuse the conversation")
	}
	// A shorter transcript than sent means the client restarted -> reset.
	again.sentMessageCount = 5
	shrunk := p.resolve([]Message{{Role: "user", Content: "hello"}})
	if shrunk.sentMessageCount != 0 {
		t.Error("shrunk transcript should reset sentMessageCount")
	}
}

func TestNewCompletionToneAgent(t *testing.T) {
	pool := NewSessionPool(ModelSessionOptions{})
	// Claude tone with tools -> stay agent-less.
	claude := newCompletion(pool, &chatRequest{
		Model:    "claude-sonnet",
		Messages: []Message{{Role: "user", Content: "hi"}},
		Tools:    []ToolDef{shellToolDef()},
	})
	if !claude.hasTools || claude.useToolAgent {
		t.Errorf("claude tone should be agent-less: hasTools=%v useAgent=%v", claude.hasTools, claude.useToolAgent)
	}
	if !strings.Contains(claude.text, "<tools>") {
		t.Error("first turn should contain the full tool prompt")
	}
	// GPT tone with tools -> use the tool agent.
	gpt := newCompletion(NewSessionPool(ModelSessionOptions{}), &chatRequest{
		Model:    "m365-copilot",
		Messages: []Message{{Role: "user", Content: "hi"}},
		Tools:    []ToolDef{shellToolDef()},
	})
	if !gpt.useToolAgent {
		t.Error("non-claude tone with tools should use the agent")
	}
}

func TestFinalizeToolCalls(t *testing.T) {
	// A lone reply() becomes plain text.
	replyOnly := &ParseResult{
		HasToolCalls: true,
		ToolCalls:    []ParsedToolCall{{Name: "reply", Arguments: `{"text":"the answer"}`}},
	}
	if p := finalizeToolCalls(replyOnly, "fallback"); p == nil || p.kind != producedText || p.text != "the answer" {
		t.Errorf("reply-only should resolve to text: %+v", p)
	}
	// Batched calls are trimmed to one.
	multi := &ParseResult{
		HasToolCalls: true,
		ToolCalls:    []ParsedToolCall{{Name: "bash"}, {Name: "bash"}},
	}
	if p := finalizeToolCalls(multi, ""); p != nil {
		t.Errorf("multi tool should not short-circuit: %+v", p)
	}
	if len(multi.ToolCalls) != 1 {
		t.Errorf("batched calls should be trimmed to one, got %d", len(multi.ToolCalls))
	}
}

func TestReplyTextFrom(t *testing.T) {
	if got := replyTextFrom(`{"message":"hi"}`, "fb"); got != "hi" {
		t.Errorf("message key = %q", got)
	}
	if got := replyTextFrom(`not json`, "fb"); got != "fb" {
		t.Errorf("fallback = %q", got)
	}
}

func TestChunkAndCopyMap(t *testing.T) {
	base := map[string]any{"id": "x"}
	c := chunk(base, map[string]any{"content": "hi"}, ptrStr("stop"))
	choices, ok := c["choices"].([]any)
	if !ok || len(choices) != 1 {
		t.Fatalf("choices = %v", c["choices"])
	}
	if _, exists := base["choices"]; exists {
		t.Error("chunk must not mutate the base map")
	}
	tc := toolCallsJSON([]ParsedToolCall{{ID: "c1", Name: "bash", Arguments: "{}"}})
	if len(tc) != 1 {
		t.Errorf("toolCallsJSON len = %d", len(tc))
	}
}

func TestErrorBuilders(t *testing.T) {
	if upstreamError("boom").errStatus != http.StatusBadGateway {
		t.Error("upstream status")
	}
	if rateLimitError(&Throttle{Current: 60, Max: 60}).errStatus != http.StatusTooManyRequests {
		t.Error("rate limit status")
	}
	if emptyResponseError(nil).errType != "upstream_empty_response" {
		t.Error("empty response type")
	}
}

func TestMinMaxInt(t *testing.T) {
	if minInt(2, 5) != 2 || maxInt(2, 5) != 5 {
		t.Error("min/max int")
	}
}

func TestBackoffDisabled(t *testing.T) {
	t.Setenv("M365_NO_BACKOFF", "1")
	if !backoffDisabled() {
		t.Error("should be disabled")
	}
	// Disabled path is a no-op and must not block.
	noteRequestOutcome(true, "c")
	awaitDegradationBackoff(context.Background())
}

func TestGetTokenNoCredentials(t *testing.T) {
	t.Setenv("M365_SECRETS_FILE", filepath.Join(t.TempDir(), "none.json"))
	if _, err := getToken(context.Background()); err == nil {
		t.Error("getToken should fail without cache or secrets")
	}
	// getTokenForScope returns ("", nil) so the agent path can fall back.
	tok, err := getTokenForScope(context.Background(), []string{"scope"})
	if err != nil || tok != "" {
		t.Errorf("getTokenForScope = %q, %v", tok, err)
	}
}

func TestAcquireSilentCacheHit(t *testing.T) {
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "cache.json"))
	cacheMu.Lock()
	cacheState = &tokenCache{Access: map[string]cachedToken{
		scopeKey([]string{"s"}): {AccessToken: "cached-tok", ExpiresAt: time.Now().Add(time.Hour).Unix()},
	}}
	cacheMu.Unlock()
	defer func() {
		cacheMu.Lock()
		cacheState = nil
		cacheMu.Unlock()
	}()
	tok, err := acquireSilent(context.Background(), []string{"s"})
	if err != nil || tok != "cached-tok" {
		t.Errorf("acquireSilent = %q, %v", tok, err)
	}
}

// --- HTTP-flow integration coverage (agent provisioning + token endpoint) ---

func resetCache() {
	cacheMu.Lock()
	cacheState = nil
	cacheMu.Unlock()
}

func seedAccessToken(scopes []string, tok string) {
	cacheMu.Lock()
	if cacheState == nil {
		cacheState = &tokenCache{Access: map[string]cachedToken{}}
	}
	cacheState.Access[scopeKey(scopes)] = cachedToken{AccessToken: tok, ExpiresAt: time.Now().Add(time.Hour).Unix()}
	cacheMu.Unlock()
}

func TestGetOrCreateAgentFlow(t *testing.T) {
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	t.Setenv("M365_AGENT_CACHE_FILE", filepath.Join(t.TempDir(), "agent.json"))
	resetCache()
	defer resetCache()
	seedAccessToken([]string{bapScope}, "bap-tok")
	seedAccessToken([]string{powerPlatformScope}, "pp-tok")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "minimalBots"):
			_, _ = w.Write([]byte(`[]`)) // no existing bot -> create
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/publish"):
			_, _ = w.Write([]byte(`{"TitleId":"T1"}`))
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "minimalBots"):
			_, _ = w.Write([]byte(`{"bot":{"schemaName":"botX"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	oldEnv, oldBap := envURLOverride, bapAPI
	envURLOverride, bapAPI = srv.URL, srv.URL
	defer func() { envURLOverride, bapAPI = oldEnv, oldBap }()

	id, err := getOrCreateAgent(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if id != "T1.botX.gpt.default" {
		t.Errorf("agent id = %q", id)
	}
}

func TestGetOrCreateAgentCacheFastPath(t *testing.T) {
	t.Setenv("M365_AGENT_CACHE_FILE", filepath.Join(t.TempDir(), "agent.json"))
	saveCachedAgent(cachedAgent{AgentID: "cached.agent", InstructionsHash: getInstructionsHash()})
	id, err := getOrCreateAgent(context.Background(), false)
	if err != nil || id != "cached.agent" {
		t.Errorf("cache fast-path = %q, %v", id, err)
	}
}

func TestGetEnvironmentURLBAP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"name":"Default-abcd1234"}`))
	}))
	defer srv.Close()
	oldEnv, oldBap := envURLOverride, bapAPI
	envURLOverride, bapAPI = "", srv.URL
	defer func() { envURLOverride, bapAPI = oldEnv, oldBap }()

	// The env-API candidates use real hostnames that will not resolve, so the
	// function falls back to the first candidate derived from the BAP response.
	url, err := getEnvironmentURL(context.Background(), "tok")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "https://defaultabcd1234") {
		t.Errorf("env url = %q", url)
	}
}

func TestExchangeCodeAndRefresh(t *testing.T) {
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	resetCache()
	defer resetCache()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		_, _ = w.Write([]byte(`{"access_token":"AT","refresh_token":"RT","expires_in":3600}`))
	}))
	defer srv.Close()
	old := authority
	authority = srv.URL
	defer func() { authority = old }()

	tok, err := exchangeCode(context.Background(), "code", "verifier", []string{"s"})
	if err != nil || tok != "AT" {
		t.Fatalf("exchangeCode = %q, %v", tok, err)
	}
	// The refresh token was stored; a silent acquire for a new scope refreshes.
	tok2, err := acquireSilent(context.Background(), []string{"other"})
	if err != nil || tok2 != "AT" {
		t.Fatalf("acquireSilent refresh = %q, %v", tok2, err)
	}
}

func TestPostTokenError(t *testing.T) {
	resetCache()
	defer resetCache()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"error":"invalid_grant","error_description":"nope"}`))
	}))
	defer srv.Close()
	old := authority
	authority = srv.URL
	defer func() { authority = old }()

	if _, err := exchangeCode(context.Background(), "c", "v", []string{"s"}); err == nil {
		t.Error("want error from token endpoint")
	}
}
