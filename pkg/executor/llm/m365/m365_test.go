package m365

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// --- copilot.go ---

func TestGetToneForModel(t *testing.T) {
	cases := map[string]string{
		"m365-copilot":       "magic",
		"auto":               "magic",
		"quick":              "Gpt_Quick",
		"claude-opus":        "Claude_Opus",
		"claude-sonnet-4.5":  "Claude_Sonnet",
		"claude-anything-xy": "Claude_Sonnet", // unmapped claude-* falls back to Claude_Sonnet
		"totally-unknown":    "magic",         // falls back to default tone
	}
	for model, want := range cases {
		if got := GetToneForModel(model); got != want {
			t.Errorf("GetToneForModel(%q) = %q, want %q", model, got, want)
		}
	}
}

// The service validates tones and rejects an unrecognized one with a
// completion error ("Failed to invoke 'Chat'..."). "auto" reads like a
// plausible tone but is not one the service accepts; the real auto-select
// tone is "magic". This guards against that regressing.
func TestGetToneForModel_NeverReturnsInvalidAutoTone(t *testing.T) {
	models := append(AvailableModels(), "totally-unknown", "claude-something")
	for _, model := range models {
		if tone := GetToneForModel(model); tone == "auto" {
			t.Errorf("GetToneForModel(%q) returned the invalid tone %q", model, tone)
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

// TestRenderAndParseFencedRoundTrip_EditPair covers the SEARCH/REPLACE-style
// tool (edit_file) round-tripping through the XML invoke format: two
// parameters (old, new) instead of the ASCII <<<<<<<SEARCH/=======/>>>>>>>
// REPLACE block the prior Markdown-fence convention used.
func TestRenderAndParseFencedRoundTrip_EditPair(t *testing.T) {
	tools := []ToolDef{editToolDef()}
	specs := BuildSpecMap(tools)
	rendered := RenderFencedCall(specs["edit_file"], map[string]any{
		"path": "app.go", "old": "debug = false", "new": "debug = true",
	})
	calls, leftover := ParseFencedToolCalls(rendered, specs)
	if len(calls) != 1 {
		t.Fatalf("got %d calls, leftover=%q, rendered=%q", len(calls), leftover, rendered)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatal(err)
	}
	if args["path"] != "app.go" || args["old"] != "debug = false" || args["new"] != "debug = true" {
		t.Errorf("args = %v", args)
	}
}

// TestParseFencedToolCalls_ChainedInvokeDoesNotLeakIntoBody is a regression
// test for a live report: the model chained a second real call
// (<invoke name="task_complete">...) directly after a write_file call in the
// same turn, despite the "one invoke per turn" framing rule. With
// invokeRegex greedy to the LAST </invoke> in the whole text, the first
// call's "content" parameter swallowed everything up to the second call's
// own closing tags, and kdeps wrote
// "...actual content</parameter></invoke>\n<invoke name=\"task_complete\">
// <parameter name=\"task_id\">1</parameter>" literally into the file.
// invokeRegex must recover each invoke scoped to its own nearest close tag.
func TestParseFencedToolCalls_ChainedInvokeDoesNotLeakIntoBody(t *testing.T) {
	tools := []ToolDef{writeToolDef(), bashExecToolDef()}
	specs := BuildSpecMap(tools)
	content := "line one\nline two"
	raw := `<invoke name="write_file"><parameter name="path">PLAN.md</parameter>` +
		`<parameter name="content">` + content + `</parameter></invoke>` +
		"\n" +
		`<invoke name="bash_exec"><parameter name="command">echo done</parameter></invoke>`
	calls, _ := ParseFencedToolCalls(raw, specs)
	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2: %+v", len(calls), calls)
	}
	if calls[0].Name != "write_file" {
		t.Fatalf("calls[0] = %+v, want write_file", calls[0])
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatal(err)
	}
	if args["content"] != content {
		t.Errorf("second chained invoke leaked into content:\ngot:  %q\nwant: %q", args["content"], content)
	}
	if calls[1].Name != "bash_exec" {
		t.Fatalf("calls[1] = %+v, want bash_exec", calls[1])
	}
}

// TestParseFencedToolCalls_ContentWithNestedFence is a regression test for a
// live report: a write_file call whose content is a markdown document (e.g.
// PLAN.md) containing its own ```-fenced code block was silently truncated
// at that inner fence under the old Markdown-fence tool-calling convention
// -- kdeps reported the write as "completed" but the file on disk stopped
// right where the nested fence began, even though the model's full raw text
// (visible in the REPL) was complete. The XML <invoke>/<parameter>
// convention this test now exercises is immune to a nested "```" the same
// way the old greedy fenceRegex was made immune to it: the closing
// </parameter> is a far rarer substring than "```" to find inside real file
// content, and parseFinalParam is greedy through to the LAST occurrence.
func TestParseFencedToolCalls_ContentWithNestedFence(t *testing.T) {
	tools := []ToolDef{writeToolDef()}
	specs := BuildSpecMap(tools)
	content := "# Plan\n\nRun this:\n\n```bash\necho hello\n```\n\nThen do the next step."
	rendered := RenderFencedCall(specs["write_file"], map[string]any{
		"path": "PLAN.md", "content": content,
	})
	calls, leftover := ParseFencedToolCalls(rendered, specs)
	if len(calls) != 1 {
		t.Fatalf("got %d calls, leftover=%q", len(calls), leftover)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatal(err)
	}
	if args["content"] != content {
		t.Errorf("content truncated at nested fence:\ngot:  %q\nwant: %q", args["content"], content)
	}
}

// bashExecToolDef mirrors the real agent-loop built-in tool name
// ("bash_exec"), unlike shellToolDef's test-only "bash" alias.
func bashExecToolDef() ToolDef {
	return ToolDef{Function: ToolFunction{
		Name: "bash_exec",
		Parameters: &ToolParameters{
			Properties: map[string]json.RawMessage{"command": json.RawMessage(`{"type":"string"}`)},
		},
	}}
}

// TestParseFencedToolCalls_ContentWithNestedRealToolName covers the exact
// live scenario reported: write_file and bash_exec are both registered
// tools, and the file content itself contains an example ```bash_exec
// fence (e.g. a plan document showing an example command) -- unlike the
// generic ```bash used in TestParseFencedToolCalls_ContentWithNestedFence,
// here the nested fence's info-string is ITSELF a real registered tool
// name. Under the XML convention this content is plain Markdown prose to
// the parser (it contains no <invoke>/<parameter> tags at all), so it can
// never be misparsed as a second call regardless of what tool name appears
// inside it.
func TestParseFencedToolCalls_ContentWithNestedRealToolName(t *testing.T) {
	tools := []ToolDef{writeToolDef(), bashExecToolDef()}
	specs := BuildSpecMap(tools)
	content := "# Plan\n\nExample:\n\n```bash_exec\necho hello\n```\n\nThen do the next step."
	rendered := RenderFencedCall(specs["write_file"], map[string]any{
		"path": "PLAN.md", "content": content,
	})
	calls, leftover := ParseFencedToolCalls(rendered, specs)
	if len(calls) != 1 {
		t.Fatalf("got %d calls, leftover=%q", len(calls), leftover)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatal(err)
	}
	if args["content"] != content {
		t.Errorf("content truncated at nested real-tool-name fence:\ngot:  %q\nwant: %q", args["content"], content)
	}
}

// TestParseFencedToolCalls_ContentWithNestedShellAlias covers the realistic
// default agent-loop registration: write_file AND bash_exec (the real
// shell tool, whose single "command" property makes findShellTool
// recognize it) registered together, so shellLangs aliases like ```bash
// route to bash_exec's spec too -- unlike
// TestParseFencedToolCalls_ContentWithNestedFence, which only registered
// write_file and so never actually exercised a nested fence whose alias
// resolves to a real spec.
func TestParseFencedToolCalls_ContentWithNestedShellAlias(t *testing.T) {
	tools := []ToolDef{writeToolDef(), bashExecToolDef()}
	specs := BuildSpecMap(tools)
	if _, ok := specs["bash"]; !ok {
		t.Fatal("expected \"bash\" aliased to bash_exec's spec -- test setup invalid")
	}
	content := "# Plan\n\nExample:\n\n```bash\necho hello\n```\n\nThen do the next step."
	rendered := RenderFencedCall(specs["write_file"], map[string]any{
		"path": "PLAN.md", "content": content,
	})
	calls, leftover := ParseFencedToolCalls(rendered, specs)
	if len(calls) != 1 {
		t.Fatalf("got %d calls, leftover=%q", len(calls), leftover)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatal(err)
	}
	if args["content"] != content {
		t.Errorf("content truncated at nested shell-alias fence:\ngot:  %q\nwant: %q", args["content"], content)
	}
}

// TestParseFencedToolCalls_ContentWithMultipleNestedFences covers content
// with more than one nested fence, confirming the greedy match reaches the
// real final close, not just the second-to-last stray "```".
func TestParseFencedToolCalls_ContentWithMultipleNestedFences(t *testing.T) {
	tools := []ToolDef{writeToolDef()}
	specs := BuildSpecMap(tools)
	content := "# Plan\n\n```bash\nstep one\n```\n\nmiddle text\n\n```bash\nstep two\n```\n\ndone."
	rendered := RenderFencedCall(specs["write_file"], map[string]any{
		"path": "PLAN.md", "content": content,
	})
	calls, _ := ParseFencedToolCalls(rendered, specs)
	if len(calls) != 1 {
		t.Fatalf("got %d calls", len(calls))
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatal(err)
	}
	if args["content"] != content {
		t.Errorf("content truncated:\ngot:  %q\nwant: %q", args["content"], content)
	}
}

// TestParseFencedToolCalls_StripsRedundantSelfWrappedContent is a
// regression test for a live report: the written PLAN.md's first line was a
// bare ``` -- the model wrapped its entire markdown answer in its own
// ```markdown ... ``` fence (a habit from being asked to "write" content,
// as if displaying it) even though the outer ```write_file ... ``` tool-call
// fence already delimits the body. Before the fix, kdeps wrote that inner
// wrapper's fence markers literally into the file.
func TestParseFencedToolCalls_StripsRedundantSelfWrappedContent(t *testing.T) {
	tools := []ToolDef{writeToolDef()}
	specs := BuildSpecMap(tools)
	intended := "# Plan\n\n## Overview\nDo the thing.\n\n### Data Flow\nA -> B -> C"
	selfWrapped := "```markdown\n" + intended + "\n```"
	rendered := RenderFencedCall(specs["write_file"], map[string]any{
		"path": "PLAN.md", "content": selfWrapped,
	})
	calls, leftover := ParseFencedToolCalls(rendered, specs)
	if len(calls) != 1 {
		t.Fatalf("got %d calls, leftover=%q", len(calls), leftover)
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatal(err)
	}
	if args["content"] != intended {
		t.Errorf("redundant self-wrap not stripped:\ngot:  %q\nwant: %q", args["content"], intended)
	}
}

// TestParseFencedToolCalls_DoesNotStripInternalCodeBlock confirms the strip
// is scoped to a whole-body self-wrap, not any document that merely
// contains a code block partway through -- its first line is prose, not a
// fence, so nothing should be removed.
func TestParseFencedToolCalls_DoesNotStripInternalCodeBlock(t *testing.T) {
	tools := []ToolDef{writeToolDef()}
	specs := BuildSpecMap(tools)
	content := "# Plan\n\nExample:\n\n```bash\necho hi\n```\n\nDone."
	rendered := RenderFencedCall(specs["write_file"], map[string]any{
		"path": "PLAN.md", "content": content,
	})
	calls, _ := ParseFencedToolCalls(rendered, specs)
	if len(calls) != 1 {
		t.Fatalf("got %d calls", len(calls))
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatal(err)
	}
	if args["content"] != content {
		t.Errorf("content should be unchanged:\ngot:  %q\nwant: %q", args["content"], content)
	}
}

// TestFraming_WarnsAgainstCallingToolsAsShellCommands is a regression test
// for a live report: the model tried to invoke other tools (bash_exec,
// memory_search, etc.) by writing their name as a shell command inside a
// ```bash block, producing real shell errors like "memory_search: command
// not found". None of the three framing variants explicitly warned against
// this confusion -- they described "use its own fence" but never named the
// specific anti-pattern that was actually happening.
func TestFraming_WarnsAgainstCallingToolsAsShellCommands(t *testing.T) {
	tools := []ToolDef{shellToolDef(), writeToolDef()}
	for _, variant := range []string{"baseline", "minimal", "softened"} {
		t.Run(variant, func(t *testing.T) {
			out := FormatFencedToolDefinitions(tools, variant)
			if !strings.Contains(out, "command not found") {
				t.Errorf("%s framing missing the tool-as-shell-command warning:\n%s", variant, out)
			}
		})
	}
}

// confabForcePrompt/hallucinationForcePrompt must never tell the model to
// write a ```bash block unless a shell tool is actually registered --
// BuildSpecMap only aliases that fence language onto a real tool in that
// case, so asking for it otherwise is a retry that can never parse into a
// tool call.
func TestForcePrompts_BashOnlyWithShellTool(t *testing.T) {
	const bashInvoke = `<invoke name="bash">`

	withShell := confabForcePrompt([]ToolDef{shellToolDef()})
	if !strings.Contains(withShell, bashInvoke) {
		t.Errorf("confabForcePrompt with a shell tool should ask for %s, got %q", bashInvoke, withShell)
	}
	withShellHalluc := hallucinationForcePrompt([]ToolDef{shellToolDef()})
	if !strings.Contains(withShellHalluc, bashInvoke) {
		t.Errorf(
			"hallucinationForcePrompt with a shell tool should ask for %s, got %q",
			bashInvoke,
			withShellHalluc,
		)
	}

	withoutShell := confabForcePrompt([]ToolDef{writeToolDef()})
	if strings.Contains(withoutShell, "Emit ONE "+bashInvoke) {
		t.Errorf(
			"confabForcePrompt without a shell tool must not ask for a bash invoke, got %q",
			withoutShell,
		)
	}
	if !strings.Contains(withoutShell, "no bash or code-interpreter tool") {
		t.Errorf(
			"confabForcePrompt without a shell tool should explain there is none, got %q",
			withoutShell,
		)
	}
	withoutShellHalluc := hallucinationForcePrompt([]ToolDef{writeToolDef()})
	if strings.Contains(withoutShellHalluc, "Emit ONE "+bashInvoke) {
		t.Errorf(
			"hallucinationForcePrompt without a shell tool must not ask for a bash invoke, got %q",
			withoutShellHalluc,
		)
	}
}

func TestParseFencedShellRouting(t *testing.T) {
	specs := BuildSpecMap([]ToolDef{shellToolDef()})
	calls, _ := ParseFencedToolCalls(
		`<invoke name="bash"><parameter name="command">ls -la</parameter></invoke>`,
		specs,
	)
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

func TestFileToolHints_EmptyWithoutDedicatedTools(t *testing.T) {
	if got := fileToolHints([]ToolDef{shellToolDef()}); got != "" {
		t.Errorf("no dedicated file tools registered, want empty hint, got %q", got)
	}
}

func TestFileToolHints_NamesRegisteredTools(t *testing.T) {
	got := fileToolHints([]ToolDef{shellToolDef(), writeToolDef(), editToolDef()})
	for _, want := range []string{"write_file", "edit_file"} {
		if !strings.Contains(got, want) {
			t.Errorf("hint missing %q: %q", want, got)
		}
	}
	if strings.Contains(got, "read_file") {
		t.Errorf("read_file not registered, must not appear in hint: %q", got)
	}
}

// Every framing variant must steer file operations to the dedicated tools,
// not just the shell, once those tools are registered -- otherwise a model
// defaults to heredocs/sed for a plain file write that has a dedicated tool.
func TestFramingVariants_PreferDedicatedFileTools(t *testing.T) {
	tools := []ToolDef{shellToolDef(), writeToolDef(), editToolDef()}
	for _, v := range []string{"baseline", "minimal", "softened"} {
		out := FormatFencedToolDefinitions(tools, v)
		if !strings.Contains(out, "write_file") || !strings.Contains(out, "DEDICATED TOOLS") {
			t.Errorf("variant %q does not steer file ops to dedicated tools:\n%s", v, out)
		}
	}
}

func TestFramingVariants_ShellOnlyWithoutDedicatedTools(t *testing.T) {
	tools := []ToolDef{shellToolDef()}
	for _, v := range []string{"baseline", "minimal", "softened"} {
		out := FormatFencedToolDefinitions(tools, v)
		if strings.Contains(out, "DEDICATED TOOLS") {
			t.Errorf(
				"variant %q should not mention dedicated tools when none are registered:\n%s",
				v,
				out,
			)
		}
	}
}

// --- tools.go ---

func TestGetMessageContent(t *testing.T) {
	if GetMessageContent(Message{Content: "hi"}) != "hi" {
		t.Error("string content")
	}
	parts := []any{
		map[string]any{"type": "text", "text": "a"},
		map[string]any{"type": "text", "text": "b"},
	}
	if GetMessageContent(Message{Content: parts}) != "ab" {
		t.Error("array content")
	}
	if GetMessageContent(Message{}) != "" {
		t.Error("nil content")
	}
}

func TestParseToolCallsFenced(t *testing.T) {
	tools := []ToolDef{shellToolDef()}
	res := ParseToolCalls(`<invoke name="bash"><parameter name="command">echo hi</parameter></invoke>`, tools)
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
	saveCachedAgent(
		cachedAgent{AgentID: "A.b.gpt.default", BotID: "b", InstructionsHash: "hash1234"},
	)
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

func TestSecretsPath(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "secrets.json")
	t.Setenv("M365_SECRETS_FILE", path)
	if got := SecretsPath(); got != path {
		t.Errorf("SecretsPath() = %q, want %q", got, path)
	}
}

func TestSaveCredentials(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "secrets.json")
	t.Setenv("M365_SECRETS_FILE", path)

	if err := SaveCredentials("", "p", "m"); err == nil {
		t.Error("empty email should error")
	}
	if err := SaveCredentials("e", "", "m"); err == nil {
		t.Error("empty password should error")
	}
	if err := SaveCredentials("e", "p", ""); err == nil {
		t.Error("empty mfaSecret should error")
	}

	if err := SaveCredentials("  e@x.com  ", "p", "m"); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}
	s := loadSecrets()
	if s == nil || s.Email != "e@x.com" || s.Password != "p" || s.MFASecret != "m" {
		t.Errorf("secrets after save = %+v", s)
	}
	// Windows' os.Stat doesn't report Unix-style permission bits, so
	// os.WriteFile's 0o600 mode has no equivalent to assert there.
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("secrets file perm = %v, want 0600", perm)
		}
	}
}

func TestCredentialsReady(t *testing.T) {
	t.Setenv("M365_SECRETS_FILE", filepath.Join(t.TempDir(), "none.json"))
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "cache.json"))
	resetCache()
	defer resetCache()

	if CredentialsReady() {
		t.Error("no secrets and no cached token should not be ready")
	}

	if err := SaveCredentials("e", "p", "m"); err != nil {
		t.Fatalf("SaveCredentials: %v", err)
	}
	if !CredentialsReady() {
		t.Error("a valid secrets file should be ready")
	}

	// A cached refresh token alone (no secrets file) is also ready.
	t.Setenv("M365_SECRETS_FILE", filepath.Join(t.TempDir(), "still-none.json"))
	resetCache()
	cacheMu.Lock()
	cacheState = &tokenCache{RefreshToken: "rt", Access: map[string]cachedToken{}}
	cacheMu.Unlock()
	if !CredentialsReady() {
		t.Error("a cached refresh token should be ready even without a secrets file")
	}
}

func TestForceReauthNoCredentials(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("M365_SECRETS_FILE", filepath.Join(dir, "none.json"))
	// Isolate from any real cached refresh token on the host (e.g.
	// ~/.config/kdeps/m365/token-cache.json from actual prior M365 auth) --
	// without this override the test's premise (no cache, no secrets) does
	// not hold on a machine that has ever authenticated for real.
	t.Setenv("M365_CACHE_FILE", filepath.Join(dir, "no-cache.json"))
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
	fn := &chatRequest{
		ToolChoice: json.RawMessage(`{"type":"function","function":{"name":"bash"}}`),
	}
	if tc := fn.parsedToolChoice(); tc == nil || tc.Mode != "function" ||
		tc.FunctionName != "bash" {
		t.Errorf("function tool_choice = %+v", tc)
	}
	empty := &chatRequest{}
	if empty.parsedToolChoice() != nil {
		t.Error("empty tool_choice should be nil")
	}
}

func TestFingerprintStable(t *testing.T) {
	a := fingerprint([]Message{{Role: "user", Content: "hello"}})
	b := fingerprint(
		[]Message{{Role: "user", Content: "hello"}, {Role: "assistant", Content: "hi"}},
	)
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
	req := httptest.NewRequest(
		http.MethodPost,
		"/v1/chat/completions",
		strings.NewReader("{bad json"),
	)
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
		t.Errorf(
			"claude tone should be agent-less: hasTools=%v useAgent=%v",
			claude.hasTools,
			claude.useToolAgent,
		)
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
	if p := finalizeToolCalls(replyOnly, "fallback"); p == nil || p.kind != producedText ||
		p.text != "the answer" {
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
	dir := t.TempDir()
	t.Setenv("M365_SECRETS_FILE", filepath.Join(dir, "none.json"))
	// See TestForceReauthNoCredentials for why the cache file also needs
	// isolating from any real cached refresh token on the host.
	t.Setenv("M365_CACHE_FILE", filepath.Join(dir, "no-cache.json"))
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
		scopeKey([]string{"s"}): {
			AccessToken: "cached-tok",
			ExpiresAt:   time.Now().Add(time.Hour).Unix(),
		},
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
	cacheState.Access[scopeKey(scopes)] = cachedToken{
		AccessToken: tok,
		ExpiresAt:   time.Now().Add(time.Hour).Unix(),
	}
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

// --- coverage: fenced.go pure-logic gaps ---

func TestFormatToolChoiceInstructionBranches(t *testing.T) {
	if formatToolChoiceInstruction(nil) != "" {
		t.Error("nil -> empty")
	}
	if formatToolChoiceInstruction(&ToolChoice{Mode: "auto"}) != "" {
		t.Error("auto -> empty")
	}
	if formatToolChoiceInstruction(&ToolChoice{Mode: "required"}) == "" {
		t.Error("required -> non-empty")
	}
	if got := formatToolChoiceInstruction(&ToolChoice{Mode: "function", FunctionName: "bash"}); !strings.Contains(
		got,
		"bash",
	) {
		t.Errorf("function -> %q", got)
	}
	if formatToolChoiceInstruction(&ToolChoice{Mode: "function"}) != "" {
		t.Error("function with no name -> empty")
	}
}

func TestToolCallSummaryBranches(t *testing.T) {
	if got := toolCallSummary(`{"path":"a.txt"}`); got != "a.txt" {
		t.Errorf("named-key = %q", got)
	}
	if got := toolCallSummary(`{"weird_key":"value here"}`); got != "value here" {
		t.Errorf("fallback-any-string = %q", got)
	}
	if toolCallSummary(`not json`) != "" {
		t.Error("invalid json -> empty")
	}
	if toolCallSummary(`{"n":1}`) != "" {
		t.Error("no string value -> empty")
	}
	long := strings.Repeat("x", 200)
	if got := toolCallSummary(`{"command":"` + long + `"}`); len(got) != summaryMaxLen {
		t.Errorf("truncation len = %d", len(got))
	}
}

func TestScalarToStringBranches(t *testing.T) {
	if scalarToString(nil) != "" {
		t.Error("nil")
	}
	if scalarToString("x") != "x" {
		t.Error("string")
	}
	if scalarToString(42) != "42" {
		t.Errorf("number = %q", scalarToString(42))
	}
	if scalarToString(true) != "true" {
		t.Errorf("bool = %q", scalarToString(true))
	}
}

func TestDeriveSpecFromArgs(t *testing.T) {
	spec := deriveSpecFromArgs("mystery_tool", map[string]any{"path": "x", "content": "y"})
	if spec.Name != "mystery_tool" {
		t.Errorf("name = %q", spec.Name)
	}
	if spec.BodyParam != "content" {
		t.Errorf("body param = %q", spec.BodyParam)
	}
}

func TestParseFencedInnerEditPairMismatch(t *testing.T) {
	spec := DeriveFencedSpec(editToolDef())
	if _, ok := parseFencedInner(spec, "no search/replace markers here"); ok {
		t.Error("malformed edit body should fail to parse")
	}
}

func TestRenderFencedTemplateEditPair(t *testing.T) {
	spec := DeriveFencedSpec(editToolDef())
	tmpl := renderFencedTemplate(spec)
	if !strings.Contains(tmpl, `<parameter name="old">`) || !strings.Contains(tmpl, `<parameter name="new">`) {
		t.Errorf("template = %q", tmpl)
	}
}

func TestRenderFencedCallEditPair(t *testing.T) {
	spec := DeriveFencedSpec(editToolDef())
	out := RenderFencedCall(spec, map[string]any{"path": "a.py", "old": "x", "new": "y"})
	if !strings.Contains(out, `<parameter name="old">x</parameter>`) ||
		!strings.Contains(out, `<parameter name="new">y</parameter>`) {
		t.Errorf("rendered edit call = %q", out)
	}
}

func TestCurrentFramingVariantEnvOverride(t *testing.T) {
	t.Setenv("M365_FRAMING_VARIANT", "minimal")
	if CurrentFramingVariant() != "minimal" {
		t.Errorf("variant = %q", CurrentFramingVariant())
	}
}

func TestShellNameNoShellTool(t *testing.T) {
	if got := shellName(nil); got != "bash" {
		t.Errorf("default shell name = %q", got)
	}
}

func TestFindShellToolSingleParamFallback(t *testing.T) {
	tool := ToolDef{Function: ToolFunction{
		Name: "run_it",
		Parameters: &ToolParameters{
			Properties: map[string]json.RawMessage{"cmd": json.RawMessage(`{"type":"string"}`)},
		},
	}}
	if findShellTool([]ToolDef{tool}) == nil {
		t.Error("single cmd-like param should be treated as the shell tool")
	}
}

func TestPropNamesNilParameters(t *testing.T) {
	if propNames(ToolDef{Function: ToolFunction{Name: "x"}}) != nil {
		t.Error("nil parameters -> nil props")
	}
}

// --- coverage: session.go / stream.go small gaps ---

func TestBuildChatArgsExtraOptionsSets(t *testing.T) {
	t.Setenv("M365_EXTRA_OPTIONSSETS", "flagA, flagB ,,")
	s := NewCopilotSession(CopilotSessionOptions{})
	args := s.buildChatArgs("req", "hi", "m365-copilot", true)
	opts, _ := args["optionsSets"].([]string)
	found := map[string]bool{}
	for _, o := range opts {
		found[o] = true
	}
	if !found["flagA"] || !found["flagB"] {
		t.Errorf("optionsSets = %v", opts)
	}
}

func TestBuildChatArgsNoCodeInterpreterEnv(t *testing.T) {
	t.Setenv("M365_NO_CODE_INTERPRETER", "1")
	s := NewCopilotSession(CopilotSessionOptions{})
	args := s.buildChatArgs("req", "hi", "m365-copilot", true)
	opts, _ := args["optionsSets"].([]string)
	for _, o := range opts {
		if o == "cwc_code_interpreter" {
			t.Error("code interpreter options should be suppressed")
		}
	}
}

// A turn that wanted the tool-calling agent but ended up agentless (agent
// resolution failed, or the caller hasn't provisioned one) must NOT unlock
// the server's own code interpreter: doing so lets the model silently answer
// from Microsoft's empty sandbox filesystem instead of the caller's real
// tools, which still went out in the prompt and are then ignored. Confirmed
// live: a think-deeper turn with tools registered used the server's own bash
// sandbox against an empty /mnt/data instead of kdeps' local-filesystem
// tools, reporting "no project found" for a real, populated repository. Gated
// on HasTools, not WantedAgent, so this holds even on a Claude-tone turn
// (which never wants an M365 agent) as long as real tools exist.
func TestBuildChatArgs_WantedAgentSuppressesCodeInterpreterEvenAgentless(t *testing.T) {
	s := NewCopilotSession(CopilotSessionOptions{WantedAgent: true, HasTools: true})
	args := s.buildChatArgs("req", "hi", "m365-copilot", true)
	opts, _ := args["optionsSets"].([]string)
	for _, o := range opts {
		if o == "cwc_code_interpreter" {
			t.Errorf(
				"code interpreter must stay off when the turn wanted an agent, got optionsSets = %v",
				opts,
			)
		}
	}
}

// The actual regression this gate was designed to prevent: a Claude-tone
// turn never wants an M365 agent (WantedAgent stays false, by convention --
// see ModelSession's doc comment), but if it has real tools to route through
// fenced calls, the server's code interpreter must stay off. Before HasTools
// existed, this case incorrectly unlocked it (the gate keyed on WantedAgent
// alone), which would let a Claude-tone tool session silently answer from
// M365's own sandbox exactly like the gpt-5.x reasoning-tone case above.
func TestBuildChatArgs_HasToolsSuppressesCodeInterpreterEvenWithoutWantedAgent(t *testing.T) {
	s := NewCopilotSession(CopilotSessionOptions{WantedAgent: false, HasTools: true})
	args := s.buildChatArgs("req", "hi", "m365-copilot", true)
	opts, _ := args["optionsSets"].([]string)
	for _, o := range opts {
		if o == "cwc_code_interpreter" {
			t.Errorf(
				"code interpreter must stay off when real tools exist, even agentless: got optionsSets = %v",
				opts,
			)
		}
	}
}

// The existing agentless-and-tool-less case is unaffected: code interpreter
// still unlocks when nothing asked for the tool-agent path at all.
func TestBuildChatArgs_NoAgentWantedEnablesCodeInterpreter(t *testing.T) {
	s := NewCopilotSession(CopilotSessionOptions{WantedAgent: false})
	args := s.buildChatArgs("req", "hi", "m365-copilot", true)
	opts, _ := args["optionsSets"].([]string)
	found := false
	for _, o := range opts {
		if o == "cwc_code_interpreter" {
			found = true
		}
	}
	if !found {
		t.Errorf(
			"code interpreter should still be enabled when no agent was ever wanted, got optionsSets = %v",
			opts,
		)
	}
}

func TestStreamSawActionDefaultsFalse(t *testing.T) {
	s := &CopilotStream{deltas: make(chan string, 1)}
	if s.SawAction() {
		t.Error("SawAction should default false (native actions unported)")
	}
}

// --- coverage: auth.go browser-login-driven flows (browserLoginFunc stubbed) ---

func withStubBrowserLogin(
	t *testing.T,
	fn func(ctx context.Context, authURL string, creds *Credentials) (string, error),
) {
	t.Helper()
	old := browserLoginFunc
	browserLoginFunc = fn
	t.Cleanup(func() { browserLoginFunc = old })
}

func withTokenEndpoint(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(
			[]byte(
				`{"access_token":"AT-` + r.FormValue(
					"scope",
				) + `","refresh_token":"RT","expires_in":3600}`,
			),
		)
	}))
	old := authority
	authority = srv.URL
	t.Cleanup(func() { authority = old; srv.Close() })
}

func TestLoginAutomatedSuccess(t *testing.T) {
	resetCache()
	defer resetCache()
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	withTokenEndpoint(t)
	withStubBrowserLogin(t, func(context.Context, string, *Credentials) (string, error) {
		return "auth-code", nil
	})

	tok, err := loginAutomated(
		context.Background(),
		&Credentials{Email: "e", Password: "p", MFASecret: "m"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tok, "AT-") {
		t.Errorf("token = %q", tok)
	}
}

func TestRunBrowserLoginRetriesThenSucceeds(t *testing.T) {
	resetCache()
	defer resetCache()
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	withTokenEndpoint(t)

	attempts := 0
	withStubBrowserLogin(t, func(context.Context, string, *Credentials) (string, error) {
		attempts++
		if attempts < 2 {
			return "", errBoom
		}
		return "auth-code", nil
	})

	// Use a near-zero retry wait so the test doesn't actually sleep 31s.
	tok, err := runBrowserLoginWithWait(
		context.Background(),
		chatScopes,
		&Credentials{Email: "e", Password: "p", MFASecret: "m"},
		2,
		time.Millisecond,
	)
	if err != nil {
		t.Fatal(err)
	}
	if tok == "" {
		t.Error("expected a token after the retry succeeded")
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestRunBrowserLoginExhausted(t *testing.T) {
	withStubBrowserLogin(t, func(context.Context, string, *Credentials) (string, error) {
		return "", errBoom
	})
	_, err := runBrowserLoginWithWait(
		context.Background(),
		chatScopes,
		&Credentials{Email: "e", Password: "p", MFASecret: "m"},
		2,
		time.Millisecond,
	)
	if err == nil {
		t.Fatal("want error after exhausting retries")
	}
}

func withStubHeadedBrowserLogin(
	t *testing.T,
	fn func(ctx context.Context, authURL string) (string, error),
) {
	t.Helper()
	old := headedBrowserLoginFunc
	headedBrowserLoginFunc = fn
	t.Cleanup(func() { headedBrowserLoginFunc = old })
}

func TestInteractiveLoginSuccess(t *testing.T) {
	resetCache()
	defer resetCache()
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	withTokenEndpoint(t)

	var gotAuthURL string
	withStubHeadedBrowserLogin(t, func(_ context.Context, authURL string) (string, error) {
		gotAuthURL = authURL
		return "auth-code", nil
	})

	tok, err := InteractiveLogin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(tok, "AT-") {
		t.Errorf("token = %q", tok)
	}
	if !strings.Contains(gotAuthURL, "response_type=code") {
		t.Errorf("authURL = %q, want an authorize URL", gotAuthURL)
	}
}

func TestInteractiveLoginBrowserFails(t *testing.T) {
	withStubHeadedBrowserLogin(t, func(context.Context, string) (string, error) {
		return "", errBoom
	})
	_, err := InteractiveLogin(context.Background())
	if err == nil {
		t.Fatal("want error when the headed browser login fails")
	}
}

func TestInteractiveLoginExchangeFails(t *testing.T) {
	resetCache()
	defer resetCache()
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	// No token endpoint stub: exchangeCode's request will fail against the
	// real (unstubbed) authority, or return a non-200 -- either way an error.
	old := authority
	authority = "http://127.0.0.1:0"
	t.Cleanup(func() { authority = old })

	withStubHeadedBrowserLogin(t, func(context.Context, string) (string, error) {
		return "auth-code", nil
	})
	_, err := InteractiveLogin(context.Background())
	if err == nil {
		t.Fatal("want error when code exchange fails")
	}
}

func TestGetTokenAutomatedLoginPath(t *testing.T) {
	resetCache()
	defer resetCache()
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	secretsPath := filepath.Join(t.TempDir(), "secrets.json")
	t.Setenv("M365_SECRETS_FILE", secretsPath)
	_ = os.WriteFile(secretsPath, []byte(`{"email":"e","password":"p","mfaSecret":"m"}`), 0o600)
	withTokenEndpoint(t)
	withStubBrowserLogin(
		t,
		func(context.Context, string, *Credentials) (string, error) { return "code", nil },
	)

	tok, err := getToken(context.Background())
	if err != nil || tok == "" {
		t.Fatalf("getToken = %q, %v", tok, err)
	}
}

func TestGetTokenForScopeAutomatedLoginPath(t *testing.T) {
	resetCache()
	defer resetCache()
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	secretsPath := filepath.Join(t.TempDir(), "secrets.json")
	t.Setenv("M365_SECRETS_FILE", secretsPath)
	_ = os.WriteFile(secretsPath, []byte(`{"email":"e","password":"p","mfaSecret":"m"}`), 0o600)
	withTokenEndpoint(t)
	withStubBrowserLogin(
		t,
		func(context.Context, string, *Credentials) (string, error) { return "code", nil },
	)

	tok, err := getTokenForScope(context.Background(), []string{"custom-scope"})
	if err != nil || tok == "" {
		t.Fatalf("getTokenForScope = %q, %v", tok, err)
	}
}

func TestForceReauthViaSecrets(t *testing.T) {
	resetCache()
	defer resetCache()
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	secretsPath := filepath.Join(t.TempDir(), "secrets.json")
	t.Setenv("M365_SECRETS_FILE", secretsPath)
	_ = os.WriteFile(secretsPath, []byte(`{"email":"e","password":"p","mfaSecret":"m"}`), 0o600)
	withTokenEndpoint(t)
	withStubBrowserLogin(
		t,
		func(context.Context, string, *Credentials) (string, error) { return "code", nil },
	)

	if !forceReauth(context.Background()) {
		t.Error("forceReauth should succeed via the automated-login fallback")
	}
}

func TestLoadSecretsInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	t.Setenv("M365_SECRETS_FILE", path)
	_ = os.WriteFile(path, []byte(`not json`), 0o600)
	if loadSecrets() != nil {
		t.Error("invalid JSON should yield nil secrets")
	}
}

func TestLoadSecretsIncomplete(t *testing.T) {
	path := filepath.Join(t.TempDir(), "secrets.json")
	t.Setenv("M365_SECRETS_FILE", path)
	_ = os.WriteFile(path, []byte(`{"email":"e"}`), 0o600)
	if loadSecrets() != nil {
		t.Error("incomplete secrets should yield nil")
	}
}

func TestConfigDirNoHome(t *testing.T) {
	// configDir falls back to "." when the home directory can't be resolved;
	// exercising the happy path here is enough to cover the function's body
	// (the error branch requires breaking os.UserHomeDir, which isn't portable).
	if configDir() == "" {
		t.Error("configDir should never be empty")
	}
}

func TestLoadCacheCorruptFile(t *testing.T) {
	resetCache()
	defer resetCache()
	path := filepath.Join(t.TempDir(), "c.json")
	t.Setenv("M365_CACHE_FILE", path)
	_ = os.WriteFile(path, []byte(`not json`), 0o600)
	c := loadCache()
	if c == nil || c.Access == nil {
		t.Errorf("corrupt cache should fall back to an empty cache: %+v", c)
	}
}

func TestPostTokenHTTPError(t *testing.T) {
	resetCache()
	defer resetCache()
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`not json`))
	}))
	defer srv.Close()
	old := authority
	authority = srv.URL
	defer func() { authority = old }()

	if _, err := exchangeCode(context.Background(), "code", "verifier", []string{"s"}); err == nil {
		t.Error("want decode error from a non-JSON error body")
	}
}

// --- coverage: agent.go remaining provisioning branches ---

func TestGetOrCreateAgentFindsExisting(t *testing.T) {
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	t.Setenv("M365_AGENT_CACHE_FILE", filepath.Join(t.TempDir(), "agent.json"))
	resetCache()
	defer resetCache()
	seedAccessToken([]string{bapScope}, "bap-tok")
	seedAccessToken([]string{powerPlatformScope}, "pp-tok")

	wantName := getAgentName()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "minimalBots"):
			_, _ = w.Write([]byte(`[{"botId":"existing-bot","shortBotName":"` + wantName + `"}]`))
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/publish"):
			_, _ = w.Write([]byte(`{"TitleId":"T-existing"}`))
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
	if id != "T-existing.existing-bot.gpt.default" {
		t.Errorf("agent id = %q", id)
	}
}

func TestGetOrCreateAgentPublishFailsThenRecreates(t *testing.T) {
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	t.Setenv("M365_AGENT_CACHE_FILE", filepath.Join(t.TempDir(), "agent.json"))
	resetCache()
	defer resetCache()
	seedAccessToken([]string{bapScope}, "bap-tok")
	seedAccessToken([]string{powerPlatformScope}, "pp-tok")

	publishAttempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "minimalBots"):
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/publish"):
			publishAttempts++
			if publishAttempts == 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			_, _ = w.Write([]byte(`{"TitleId":"T-recreated"}`))
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"bot":{"schemaName":"botY"}}`))
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
	if id != "T-recreated.botY.gpt.default" {
		t.Errorf("agent id = %q", id)
	}
	if publishAttempts != 2 {
		t.Errorf("publish attempts = %d, want 2 (fail then recreate+succeed)", publishAttempts)
	}
}

func TestGetOrCreateAgentPublishFailsPermanently(t *testing.T) {
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	t.Setenv("M365_AGENT_CACHE_FILE", filepath.Join(t.TempDir(), "agent.json"))
	resetCache()
	defer resetCache()
	seedAccessToken([]string{bapScope}, "bap-tok")
	seedAccessToken([]string{powerPlatformScope}, "pp-tok")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "minimalBots"):
			_, _ = w.Write([]byte(`[]`))
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusOK)
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/publish"):
			w.WriteHeader(http.StatusBadRequest)
		case r.Method == http.MethodPost:
			_, _ = w.Write([]byte(`{"bot":{"schemaName":"botZ"}}`))
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
	if id != "" {
		t.Errorf("permanently failing publish should yield empty id, got %q", id)
	}
}

func TestGetOrCreateAgentListBotsFails(t *testing.T) {
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	t.Setenv("M365_AGENT_CACHE_FILE", filepath.Join(t.TempDir(), "agent.json"))
	resetCache()
	defer resetCache()
	seedAccessToken([]string{bapScope}, "bap-tok")
	seedAccessToken([]string{powerPlatformScope}, "pp-tok")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	oldEnv, oldBap := envURLOverride, bapAPI
	envURLOverride, bapAPI = srv.URL, srv.URL
	defer func() { envURLOverride, bapAPI = oldEnv, oldBap }()

	id, err := getOrCreateAgent(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if id != "" {
		t.Errorf("listBots failure should yield empty id, got %q", id)
	}
}

func TestGetOrCreateAgentNoBAPToken(t *testing.T) {
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	t.Setenv("M365_AGENT_CACHE_FILE", filepath.Join(t.TempDir(), "agent.json"))
	t.Setenv("M365_SECRETS_FILE", filepath.Join(t.TempDir(), "none.json"))
	resetCache()
	defer resetCache()

	id, err := getOrCreateAgent(context.Background(), false)
	if err != nil || id != "" {
		t.Errorf("no BAP token -> empty id, no error: id=%q err=%v", id, err)
	}
}

func TestGetOrCreateAgentNoPowerPlatformToken(t *testing.T) {
	t.Setenv("M365_CACHE_FILE", filepath.Join(t.TempDir(), "c.json"))
	t.Setenv("M365_AGENT_CACHE_FILE", filepath.Join(t.TempDir(), "agent.json"))
	t.Setenv("M365_SECRETS_FILE", filepath.Join(t.TempDir(), "none.json"))
	resetCache()
	defer resetCache()
	seedAccessToken([]string{bapScope}, "bap-tok")

	id, err := getOrCreateAgent(context.Background(), false)
	if err != nil || id != "" {
		t.Errorf("no PowerPlatform token -> empty id, no error: id=%q err=%v", id, err)
	}
}

// --- coverage: final small gaps ---

func TestDecodeJWTBadBase64(t *testing.T) {
	if _, err := DecodeJWT("hdr.not-valid-base64!!!.sig"); err == nil {
		t.Error("want base64 decode error")
	}
}

func TestParseFencedToolCallsNoMatch(t *testing.T) {
	specs := BuildSpecMap([]ToolDef{shellToolDef()})
	calls, leftover := ParseFencedToolCalls(
		`<invoke name="python"><parameter name="code">print(1)</parameter></invoke>`,
		specs,
	)
	if len(calls) != 0 {
		t.Errorf("unknown tool name should not parse as a tool call: %+v", calls)
	}
	if !strings.Contains(leftover, "python") {
		t.Errorf("leftover should retain the untouched invoke block: %q", leftover)
	}
}

func TestBrowserProfileDirAndChromiumPath(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("M365_BROWSER_PROFILE", dir+"/profile")
	if browserProfileDir() != dir+"/profile" {
		t.Errorf("browserProfileDir = %q", browserProfileDir())
	}

	t.Setenv("CHROMIUM_PATH", "/custom/chromium")
	if resolveChromiumPath() != "/custom/chromium" {
		t.Errorf("resolveChromiumPath override = %q", resolveChromiumPath())
	}
}

func TestPPFetchDelete(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	res, err := ppFetch(context.Background(), http.MethodDelete, srv.URL, "tok", nil)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q", gotMethod)
	}
}

func TestListBotsHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	if _, err := listBots(context.Background(), srv.URL, "tok"); err == nil {
		t.Error("want error on non-2xx status")
	}
}

func TestCreateBotHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()
	if _, err := createBot(context.Background(), srv.URL, "tok"); err == nil {
		t.Error("want error on non-2xx status")
	}
}

func TestPublishBotMissingTitleID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	if _, err := publishBot(context.Background(), srv.URL, "tok", "bot1"); err == nil {
		t.Error("want error when TitleId is missing")
	}
}

func TestPublishBotHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if _, err := publishBot(context.Background(), srv.URL, "tok", "bot1"); err == nil {
		t.Error("want error on non-2xx status")
	}
}

func TestGetEnvironmentURLHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	old := bapAPI
	bapAPI = srv.URL
	defer func() { bapAPI = old }()
	if _, err := getEnvironmentURL(context.Background(), "tok"); err == nil {
		t.Error("want error on non-2xx BAP status")
	}
}
