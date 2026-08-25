package m365

import (
	"encoding/json"
	"regexp"
	"strings"

	"github.com/google/uuid"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
	"github.com/kdeps/kdeps/v2/pkg/executor/llm/toolguard"
)

const (
	// summaryMaxLen truncates a tool-call summary label.
	summaryMaxLen = 100
	// proseDocMinFences / proseDocManyFences / proseDocMinChars classify a reply as
	// a written document rather than an action turn.
	proseDocMinFences  = 2
	proseDocManyFences = 4
	proseDocMinChars   = 300
)

// This file adapts OpenAI-style chat messages and tool definitions to the
// prompt-injection protocol the chat model understands, and parses the model's
// replies back into structured tool calls. The model has no native
// function-calling, so tools are described in the prompt and calls are recovered
// from <invoke>/<parameter> XML blocks (see fenced.go).

// Message is one OpenAI-style chat message. Content may be a plain string or a
// content-part array; both decode into this shape.
type Message struct {
	Role       string     `json:"role"`
	Content    any        `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

// ToolCall is an assistant tool call in the transcript.
type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type,omitempty"`
	Function ToolCallFunction `json:"function"`
}

// ToolCallFunction carries the called tool name and its JSON arguments string.
type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// ToolChoice constrains tool selection: "auto", "none", "required", or a
// specific function name.
type ToolChoice struct {
	Mode         string // auto | none | required | function
	FunctionName string
}

var toolCallJSONRegex = regexp.MustCompile(`(?s)\{\s*"tool"\s*:\s*"[^"]+"\s*,\s*"arguments"\s*:\s*\{.*?\}\s*\}`)

var confidenceRegex = regexp.MustCompile(`\{\s*"confidence"\s*:\s*-?[0-9.]+\s*\}`)

var finalObjectRegex = regexp.MustCompile(`\{\s*"final"\s*:\s*"(?:[^"\\]|\\.)*"\s*\}`)

var emptyFenceRegex = regexp.MustCompile("```(?:json|tool_call)?\\s*```")

var danglingFenceRegex = regexp.MustCompile("```(?:json|tool_call)?")

// newCallID generates an OpenAI-style tool-call id.
func newCallID() string {
	return "call_" + strings.ReplaceAll(uuid.NewString(), "-", "")[:24]
}

// GetMessageContent flattens a message's content to a string, handling both the
// plain-string and content-part-array encodings.
func GetMessageContent(msg Message) string {
	switch c := msg.Content.(type) {
	case nil:
		return ""
	case string:
		return c
	case []any:
		var b strings.Builder
		for _, part := range c {
			if pm, ok := part.(map[string]any); ok {
				if t, isStr := pm["text"].(string); isStr {
					b.WriteString(t)
				}
			}
		}
		return b.String()
	default:
		return ""
	}
}

// formatToolChoiceInstruction renders the tool-choice constraint as a prompt line.
func formatToolChoiceInstruction(tc *ToolChoice) string {
	if tc == nil || tc.Mode == "" || tc.Mode == "auto" {
		return ""
	}
	switch tc.Mode {
	case "none":
		return "\nDo NOT call tools. Text only."
	case "required":
		return "\nYou MUST call at least one tool."
	case "function":
		if tc.FunctionName != "" {
			return "\nYou MUST call \"" + tc.FunctionName + "\"."
		}
	}
	return ""
}

// toolCallSummary extracts a short label describing what a call did, for
// annotating its result in the transcript.
func toolCallSummary(rawArgs string) string {
	var args map[string]any
	if json.Unmarshal([]byte(rawArgs), &args) != nil {
		return ""
	}
	var primary string
	for _, key := range []string{"command", "cmd", "script", "path", "file", "filename", "query"} {
		if v, ok := args[key].(string); ok {
			primary = v
			break
		}
	}
	if primary == "" {
		for _, v := range args {
			if s, ok := v.(string); ok {
				primary = s
				break
			}
		}
	}
	if primary == "" {
		return ""
	}
	primary = strings.Join(strings.Fields(primary), " ")
	primary = strings.ReplaceAll(primary, "\"", "'")
	if len(primary) > summaryMaxLen {
		primary = primary[:summaryMaxLen]
	}
	return primary
}

// FormatMessages renders the OpenAI-style transcript into the single prompt
// string sent to the chat model, injecting tool definitions and re-rendering
// prior assistant tool calls and tool results in the fenced protocol.
//
//nolint:gocognit // per-message rendering across assistant/tool/other roles
func FormatMessages(
	messages []Message,
	tools []ToolDef,
	toolChoice *ToolChoice,
	conversationID, framingVariant string,
) string {
	kdeps_debug.Log("enter: FormatMessages")
	var parts []string

	if conversationID != "" {
		parts = append(parts, "<conversation_id>"+conversationID+"</conversation_id>")
	}

	var specMap map[string]FencedToolSpec
	if len(tools) > 0 {
		specMap = BuildSpecMap(tools)
		if toolChoice == nil || toolChoice.Mode != "none" {
			parts = append(
				parts,
				"<system>\n"+FormatFencedToolDefinitions(
					tools,
					framingVariant,
				)+formatToolChoiceInstruction(
					toolChoice,
				)+"\n</system>",
			)
		}
	}

	// Map each tool-call id to its name and a short summary so a tool result can
	// be labelled with the command that produced it.
	callMeta := map[string]struct{ name, summary string }{}
	for _, m := range messages {
		if m.Role == "assistant" {
			for _, tc := range m.ToolCalls {
				if tc.ID != "" {
					callMeta[tc.ID] = struct{ name, summary string }{
						tc.Function.Name,
						toolCallSummary(tc.Function.Arguments),
					}
				}
			}
		}
	}

	for _, m := range messages {
		switch {
		case m.Role == "assistant" && len(m.ToolCalls) > 0:
			var rendered []string
			for _, tc := range m.ToolCalls {
				var argsObj map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &argsObj)
				var spec FencedToolSpec
				if specMap != nil {
					if s, ok := specMap[tc.Function.Name]; ok {
						spec = s
					}
				}
				if spec.Name == "" {
					spec = deriveSpecFromArgs(tc.Function.Name, argsObj)
				}
				rendered = append(rendered, RenderFencedCall(spec, argsObj))
			}
			content := GetMessageContent(m)
			body := ""
			if content != "" {
				body = "\n" + content
			}
			parts = append(parts, "<assistant>"+body+"\n"+strings.Join(rendered, "\n")+"\n</assistant>")
		case m.Role == "tool":
			meta := callMeta[m.ToolCallID]
			name := m.Name
			if name == "" {
				name = meta.name
			}
			if name == "" {
				name = "tool"
			}
			cmdAttr := ""
			if meta.summary != "" {
				cmdAttr = ` command="` + meta.summary + `"`
			}
			parts = append(
				parts,
				"<tool_response tool=\""+name+"\""+cmdAttr+">\n"+GetMessageContent(m)+"\n</tool_response>",
			)
		default:
			parts = append(parts, "<"+m.Role+">\n"+GetMessageContent(m)+"\n</"+m.Role+">")
		}
	}

	return strings.Join(parts, "\n\n")
}

// deriveSpecFromArgs synthesizes a fenced spec from recorded argument keys when
// a tool is no longer in the request's tool list.
func deriveSpecFromArgs(name string, args map[string]any) FencedToolSpec {
	props := make(map[string]json.RawMessage, len(args))
	for k := range args {
		props[k] = json.RawMessage(`{"type":"string"}`)
	}
	return DeriveFencedSpec(ToolDef{
		Function: ToolFunction{
			Name:       name,
			Parameters: &ToolParameters{Properties: props},
		},
	})
}

// ParseResult is the outcome of parsing a model reply for tool calls.
type ParseResult struct {
	HasToolCalls bool
	ToolCalls    []ParsedToolCall
	TextContent  string
}

// cleanLooseText strips invented confidence/final bookkeeping objects and
// unwraps a lone {"final": "..."} answer. Returns "" when nothing remains.
func cleanLooseText(text string) string {
	out := text
	for _, m := range finalObjectRegex.FindAllString(out, -1) {
		var obj struct {
			Final string `json:"final"`
		}
		if json.Unmarshal([]byte(m), &obj) == nil {
			out = strings.Replace(out, m, obj.Final, 1)
		}
	}
	out = strings.TrimSpace(confidenceRegex.ReplaceAllString(out, ""))
	return out
}

// ParseToolCalls parses a model reply into tool calls plus leftover text. It
// prefers the fenced protocol and tolerates a stray JSON tool-call object.
func ParseToolCalls(text string, tools []ToolDef) ParseResult {
	kdeps_debug.Log("enter: ParseToolCalls")
	if len(tools) > 0 {
		calls, leftover := ParseFencedToolCalls(text, BuildSpecMap(tools))
		if len(calls) > 0 {
			return ParseResult{HasToolCalls: true, ToolCalls: calls, TextContent: cleanLooseText(leftover)}
		}
	}

	var calls []ParsedToolCall
	for _, m := range toolCallJSONRegex.FindAllString(text, -1) {
		var parsed struct {
			Tool      string          `json:"tool"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if json.Unmarshal([]byte(m), &parsed) != nil || parsed.Tool == "" {
			continue
		}
		calls = append(calls, ParsedToolCall{
			ID:        newCallID(),
			Name:      parsed.Tool,
			Arguments: string(parsed.Arguments),
		})
	}

	if len(calls) == 0 {
		return ParseResult{HasToolCalls: false, TextContent: cleanLooseText(text)}
	}

	remaining := toolCallJSONRegex.ReplaceAllString(text, "")
	remaining = confidenceRegex.ReplaceAllString(remaining, "")
	remaining = finalObjectRegex.ReplaceAllString(remaining, "")
	remaining = emptyFenceRegex.ReplaceAllString(remaining, "")
	remaining = danglingFenceRegex.ReplaceAllString(remaining, "")
	remaining = strings.TrimSpace(remaining)

	return ParseResult{HasToolCalls: true, ToolCalls: calls, TextContent: remaining}
}

var markdownHeaderRegex = regexp.MustCompile(`(?m)^#{1,6}\s`)

// IsProseDocument reports whether a reply is a written document (prose with many
// embedded code fences) rather than a genuine action turn. The shell-routing
// parser turns every well-formed <invoke> block into a call, so a document
// answer that happens to illustrate the syntax would get executed; this flags
// that shape so the caller returns it as text.
//
// This guard is invoke-format-specific: it only matters when tool calls are
// recovered by parsing XML blocks out of free text (as m365 does). Backends
// with native structured tool-calling never have this ambiguity, so this stays
// local to m365 rather than moving to toolguard.
func IsProseDocument(p ParseResult) bool {
	if !p.HasToolCalls || len(p.ToolCalls) < proseDocMinFences {
		return false
	}
	prose := strings.TrimSpace(p.TextContent)
	return len(p.ToolCalls) >= proseDocManyFences || markdownHeaderRegex.MatchString(prose) ||
		len(prose) >= proseDocMinChars
}

// LooksLikeConfabulation reports whether a no-tool-call reply looks like the
// model claiming it cannot act (rather than a genuine final answer). Shared with
// every other backend via toolguard.
func LooksLikeConfabulation(text string) bool {
	return toolguard.LooksLikeConfabulation(text)
}

// LooksLikeHallucinatedCompletion reports whether a no-tool-call reply claims a
// file mutation it may not have performed. Shared with every other backend via
// toolguard.
func LooksLikeHallucinatedCompletion(text string) bool {
	return toolguard.LooksLikeHallucinatedCompletion(text)
}
