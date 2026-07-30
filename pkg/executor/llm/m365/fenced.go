package m365

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// The chat model has no native tool-calling API. Tool calls are emulated with
// Markdown code fences: the model emits a fenced block whose info-string is the
// tool name, scalar arguments render as "key: value" header lines, and one
// free-form argument becomes the fence body. This file builds the prompt-side
// tool definitions and parses fenced blocks back into structured calls.
//	```bash
//	ls -la
//	```
//	```write_file
//	path: main.go
//	package main
//	```
//	```edit_file
//	path: app.go
//	<<<<<<< SEARCH
//	debug = false
//	=======
//	debug = true
//	>>>>>>> REPLACE
//	```

// ToolFunction is the function schema of an OpenAI-style tool definition.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  *ToolParameters `json:"parameters,omitempty"`
}

// ToolParameters holds the JSON-schema properties of a tool's arguments.
type ToolParameters struct {
	Type       string                     `json:"type,omitempty"`
	Properties map[string]json.RawMessage `json:"properties,omitempty"`
	Required   []string                   `json:"required,omitempty"`
}

// ToolDef is one OpenAI-style tool definition.
type ToolDef struct {
	Type     string       `json:"type,omitempty"`
	Function ToolFunction `json:"function"`
}

// bodyParamNames are argument names carried as the free-form fence body.
//
//nolint:gochecknoglobals // static lookup, read-only
var bodyParamNames = map[string]bool{
	"command": true, "content": true, "code": true, "body": true,
	"script": true, "text": true, "query": true, "input": true,
	"patch": true, "cmd": true, "data": true, "contents": true,
}

//nolint:gochecknoglobals // static lookup, read-only
var searchKeys = map[string]bool{
	"old": true, "search": true, "find": true,
	"old_str": true, "old_string": true, "target": true,
}

//nolint:gochecknoglobals // static lookup, read-only
var replaceKeys = map[string]bool{
	"new": true, "replace": true, "replacement": true,
	"new_str": true, "new_string": true,
}

// shellLangs are fence info-strings that mean "a shell script"; they route to
// whatever run-a-command tool the caller provided.
//
//nolint:gochecknoglobals,goconst // static lookup, read-only; "bash" here is a fence language, not a repeated literal
var shellLangs = []string{
	"bash", "sh", "shell", "zsh", "console",
	"shell-session", "shellsession", "shsession",
}

var shellToolName = regexp.MustCompile(
	`(?i)^(bash|sh|shell|zsh|run|exec|execute|command|cmd|terminal|run_command|run_terminal_cmd|execute_command|execute_bash|shell_exec|system)$`,
)

var commandParamName = regexp.MustCompile(`(?i)^(command|cmd|script|input)$`)

var fenceRegex = regexp.MustCompile("(?s)```([A-Za-z0-9_]+)[ \t]*\r?\n(.*?)\r?\n?```")

var searchReplaceRegex = regexp.MustCompile(
	`(?s)<{5,}\s*SEARCH\s*\r?\n(.*?)\r?\n={5,}\s*\r?\n(.*?)\r?\n>{5,}\s*REPLACE`,
)

var headerLineRegex = regexp.MustCompile(`^([A-Za-z0-9_]+):[ \t]?(.*)$`)

// FencedToolSpec describes how one tool maps onto the fenced shape.
type FencedToolSpec struct {
	Name        string
	Description string
	// HeaderParams render as "key: value" header lines.
	HeaderParams []string
	// BodyParam is the free-form argument carried as the fence body.
	BodyParam string
	// EditPair, when set, renders an (old -> new) SEARCH/REPLACE diff.
	EditPair *editPair
}

type editPair struct {
	search  string
	replace string
}

// propNames returns the declared argument names of a tool, preserving the
// declaration order recorded in the parameters schema.
func propNames(t ToolDef) []string {
	if t.Function.Parameters == nil {
		return nil
	}
	names := make([]string, 0, len(t.Function.Parameters.Properties))
	for k := range t.Function.Parameters.Properties {
		names = append(names, k)
	}
	return names
}

// DeriveFencedSpec derives the fenced shape for a single tool.
func DeriveFencedSpec(tool ToolDef) FencedToolSpec {
	kdeps_debug.Log("enter: DeriveFencedSpec")
	name := tool.Function.Name
	props := propNames(tool)

	var search, replace string
	for _, p := range props {
		if searchKeys[p] && search == "" {
			search = p
		}
		if replaceKeys[p] && replace == "" {
			replace = p
		}
	}
	if search != "" && replace != "" {
		return FencedToolSpec{
			Name:         name,
			Description:  tool.Function.Description,
			EditPair:     &editPair{search: search, replace: replace},
			HeaderParams: without(props, search, replace),
		}
	}

	body := ""
	for _, p := range props {
		if bodyParamNames[p] {
			body = p
			break
		}
	}
	if body == "" && len(props) == 1 {
		body = props[0]
	}
	return FencedToolSpec{
		Name:         name,
		Description:  tool.Function.Description,
		BodyParam:    body,
		HeaderParams: without(props, body),
	}
}

func without(items []string, drop ...string) []string {
	skip := make(map[string]bool, len(drop))
	for _, d := range drop {
		if d != "" {
			skip[d] = true
		}
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		if !skip[it] {
			out = append(out, it)
		}
	}
	return out
}

// findShellTool returns the caller's run-a-command tool, if any.
func findShellTool(tools []ToolDef) *ToolDef {
	for i := range tools {
		if shellToolName.MatchString(tools[i].Function.Name) {
			return &tools[i]
		}
	}
	for i := range tools {
		props := propNames(tools[i])
		if len(props) == 1 && commandParamName.MatchString(props[0]) {
			return &tools[i]
		}
	}
	return nil
}

// BuildSpecMap maps each tool name to its fenced spec, aliasing shell fence
// languages (```bash etc.) onto the caller's shell tool so the model can "just
// write bash" and still produce a structured call.
func BuildSpecMap(tools []ToolDef) map[string]FencedToolSpec {
	kdeps_debug.Log("enter: BuildSpecMap")
	m := make(map[string]FencedToolSpec, len(tools))
	for _, t := range tools {
		m[t.Function.Name] = DeriveFencedSpec(t)
	}
	if shell := findShellTool(tools); shell != nil {
		shellSpec := m[shell.Function.Name]
		for _, lang := range shellLangs {
			if _, ok := m[lang]; !ok {
				m[lang] = shellSpec
			}
		}
	}
	return m
}

func scalarToString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return string(b)
	}
}

// RenderFencedCall renders one concrete call (name + args) as a fenced block,
// for replaying a prior assistant tool call back into the transcript.
func RenderFencedCall(spec FencedToolSpec, args map[string]any) string {
	var lines []string
	for _, h := range spec.HeaderParams {
		if v, ok := args[h]; ok {
			lines = append(lines, h+": "+scalarToString(v))
		}
	}
	switch {
	case spec.EditPair != nil:
		lines = append(lines,
			"<<<<<<< SEARCH",
			scalarToString(args[spec.EditPair.search]),
			"=======",
			scalarToString(args[spec.EditPair.replace]),
			">>>>>>> REPLACE",
		)
	case spec.BodyParam != "":
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, scalarToString(args[spec.BodyParam]))
	}
	return "```" + spec.Name + "\n" + strings.Join(lines, "\n") + "\n```"
}

// renderFencedTemplate renders a self-documenting template for the <tools> block.
func renderFencedTemplate(spec FencedToolSpec) string {
	var lines []string
	for _, h := range spec.HeaderParams {
		lines = append(lines, h+": <"+h+">")
	}
	switch {
	case spec.EditPair != nil:
		lines = append(lines,
			"<<<<<<< SEARCH",
			"<"+spec.EditPair.search+">",
			"=======",
			"<"+spec.EditPair.replace+">",
			">>>>>>> REPLACE",
		)
	case spec.BodyParam != "":
		if len(lines) > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, "<"+spec.BodyParam+">")
	}
	header := spec.Name
	if spec.Description != "" {
		header = spec.Name + " - " + spec.Description
	}
	return header + "\n```" + spec.Name + "\n" + strings.Join(lines, "\n") + "\n```"
}

// toolsBlock renders the shared <tools> definition block.
func toolsBlock(tools []ToolDef) string {
	defs := make([]string, 0, len(tools))
	for _, t := range tools {
		defs = append(defs, renderFencedTemplate(DeriveFencedSpec(t)))
	}
	return "<tools>\n" + strings.Join(defs, "\n\n") + "\n</tools>"
}

// framingBuilder builds the instructional framing that precedes the <tools> block.
type framingBuilder func(tools []ToolDef) string

// framingVariants selects the behavioural framing. baseline is the default;
// softened drops the instruction-override shape that occasionally trips the
// service's prompt filter, and the handler retries with it on a Disengaged turn.
//
//nolint:gochecknoglobals // registry, read-only after init
var framingVariants = map[string]framingBuilder{
	"baseline": baselineFraming,
	"minimal":  minimalFraming,
	"softened": softenedFraming,
}

// CurrentFramingVariant resolves the active framing name from the environment,
// defaulting to baseline.
func CurrentFramingVariant() string {
	if v := strings.TrimSpace(os.Getenv("M365_FRAMING_VARIANT")); v != "" {
		return v
	}
	return "baseline"
}

// FormatFencedToolDefinitions renders the full framing + <tools> block for a
// request. An empty variantOverride uses the environment default.
func FormatFencedToolDefinitions(tools []ToolDef, variantOverride string) string {
	kdeps_debug.Log("enter: FormatFencedToolDefinitions")
	variant := variantOverride
	if variant == "" {
		variant = CurrentFramingVariant()
	}
	build, ok := framingVariants[variant]
	if !ok {
		build = baselineFraming
	}
	return build(tools)
}

func shellName(tools []ToolDef) string {
	if shell := findShellTool(tools); shell != nil {
		return shell.Function.Name
	}
	return "bash"
}

func baselineFraming(tools []ToolDef) string {
	var shellFraming string
	if shell := findShellTool(tools); shell != nil {
		shellFraming = "\n\nTHE WAY YOU DO ANYTHING IS BY WRITING A SHELL SCRIPT. You have a real shell (the `" + shell.Function.Name + "` tool). To perform a step, emit ONE ```bash block that does the whole thing end-to-end against the real files in the working directory: create/overwrite files with `cat > name <<'EOF' ... EOF` heredocs, edit files in place with `sed -i`, inspect with `cat`/`ls`/`grep`, run code with the available interpreters. The block is executed for real and you get its output back. Writing the commands IS doing the task; describing what you \"would\" run, or claiming you did it, accomplishes nothing.\n\nYou have NOT run any command yet and have NO results. NEVER claim a command \"returned no output\", that files are \"missing\", or that you \"cannot access\" / \"cannot list\" the environment before you have actually emitted a ```bash block and seen its <tool_response>. The files named in the task are present on a real filesystem right now. Your FIRST output must be a ```bash block (e.g. `ls -la` then `cat` the relevant files) - never open with prose, a question, or a request for the user to paste files. Do not assume a file's contents or a command's result; run a command and read the real output. One self-contained ```bash block per turn."
	}
	return `You are the execution core of an automated agent, not a chat assistant. Your output is parsed by a program - a real runtime that executes your tool calls against a live system and returns the actual results to you in <tool_response> blocks.` + shellFraming + `

Performing the task with tools is your PRIMARY JOB. Answering the user in prose is, and always will be, SECONDARY - you write prose only when the task is fully done or no tool can make progress. Default to acting, not talking.

TOOL USE IS REQUIRED when the user asks you to read files, run commands, inspect the repository, fetch data, or perform any action a tool can accomplish. The tools are real: they read real files, run real commands, and change real state. Never answer from memory or simulate a result when a tool can provide it.

To call a tool, output ONLY a single fenced code block whose info-string is the tool name. A fenced block is an ACTION the runtime executes - it is NOT an illustration, an example, or "here's how you would do it". No text before or after it:

` + "```<tool_name>" + `
<header lines: one "key: value" per scalar argument>

<body argument, if the tool has one>
` + "```" + `

STRICT RULES:
- Output ONLY the fenced block when calling a tool. No prose, no second fence, no commentary before or after.
- Never describe your intent ("I'll read the file...", "Let me check...") and never emit filler or acknowledgements. Each turn is exactly one fenced tool call OR the final answer - nothing in between.
- One tool call per response, then stop and wait for its <tool_response>. Never emit two fenced blocks in one response.
- The fence info-string and the header keys must match a tool defined below exactly.
- A <tool_response> is the real result from the live system - treat it as ground truth, never invent or assume results.
- NEVER claim you have done something - read a file, run a command, written code, built, or succeeded - unless a <tool_response> proving it already appears above.
- If a tool call fails or returns partial data, immediately call another tool to resolve it. Do not give up.
- Do not defer work or promise future results ("I'll do this next...").
- Do not ask the user questions unless tool execution is impossible.
- Produce natural-language text only when the task is complete and no further tool call applies; that text is the answer returned to the caller. When you do, output only the answer itself - no preamble, no sign-off.

` + toolsBlock(
		tools,
	)
}

func minimalFraming(tools []ToolDef) string {
	name := shellName(tools)
	return "You are an automated agent with a real shell (the `" + name + "` tool). You do the task by emitting ONE ```bash block per turn that acts on the real files in the working directory (heredocs to create, `sed -i` to edit, `cat`/`ls`/`grep` to inspect, interpreters to run). The block is executed for real; its output comes back in a <tool_response>. Writing the commands IS doing the task.\n\nYou have run nothing yet. Your FIRST output must be a ```bash block - never prose, a question, or \"I can't access the files\". Never claim a result you have not seen in a <tool_response>.\n\n" + toolsBlock(
		tools,
	)
}

func softenedFraming(tools []ToolDef) string {
	name := shellName(tools)
	var shellLine string
	if findShellTool(tools) != nil {
		shellLine = "You have a real shell available as the `" + name + "` tool. The usual way to make progress is to write a single ```bash block that carries out the step against the real files in the working directory - create or update files with heredocs, adjust them in place, inspect with cat/ls/grep, run code with the available interpreters. The runtime executes the block and returns its real output to you. Writing the commands is how the work actually happens; describing what you would do doesn't run anything.\n\n"
	}
	return `You are an automated coding agent working in a real working directory. Your replies are read by a program that runs your tool calls and returns the results.

` + shellLine + `To use a tool, reply with a single fenced code block whose info-string is the tool name (a fence is run as a real action, not shown as an illustration):

` + "```<tool_name>" + `
<one "key: value" header line per scalar argument>

<the body argument, if the tool has one>
` + "```" + `

A <tool_response> is the real result from the live system - rely on it rather than assuming what a command would print. Work one step at a time: one tool call per reply, then wait for its <tool_response>. Begin by running a ` + "```bash" + ` block that inspects the relevant files, rather than answering from memory, and keep going with tool calls until the task is finished. Reply in plain language only once the task is done and no further tool call would help.

` + toolsBlock(
		tools,
	)
}

// ParsedToolCall is one emulated tool call recovered from the model's output.
type ParsedToolCall struct {
	ID   string
	Name string
	// Arguments is the JSON-encoded arguments object.
	Arguments string
}

// parseFencedInner parses one fenced block body into an arguments object.
func parseFencedInner(spec FencedToolSpec, inner string) (map[string]any, bool) {
	lines := strings.Split(inner, "\n")
	args := map[string]any{}

	i := 0
	if len(spec.HeaderParams) > 0 {
		headerSet := make(map[string]bool, len(spec.HeaderParams))
		for _, h := range spec.HeaderParams {
			headerSet[h] = true
		}
		for ; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "" {
				i++
				break
			}
			m := headerLineRegex.FindStringSubmatch(lines[i])
			if m != nil && headerSet[m[1]] {
				args[m[1]] = m[2]
			} else {
				break
			}
		}
	}

	rest := strings.Join(lines[i:], "\n")

	switch {
	case spec.EditPair != nil:
		sr := searchReplaceRegex.FindStringSubmatch(rest)
		if sr == nil {
			return nil, false
		}
		args[spec.EditPair.search] = sr[1]
		args[spec.EditPair.replace] = sr[2]
	case spec.BodyParam != "":
		args[spec.BodyParam] = rest
	}
	return args, true
}

// ParseFencedToolCalls parses every fenced block whose info-string matches a
// known tool. leftover is the text with those fences removed.
func ParseFencedToolCalls(text string, specs map[string]FencedToolSpec) ([]ParsedToolCall, string) {
	kdeps_debug.Log("enter: ParseFencedToolCalls")
	var calls []ParsedToolCall
	leftover := text
	for _, m := range fenceRegex.FindAllStringSubmatch(text, -1) {
		spec, ok := specs[m[1]]
		if !ok {
			continue // not a tool - leave it in prose
		}
		args, ok := parseFencedInner(spec, m[2])
		if !ok {
			continue
		}
		argsJSON, err := json.Marshal(args)
		if err != nil {
			continue
		}
		calls = append(calls, ParsedToolCall{
			ID:        newCallID(),
			Name:      spec.Name,
			Arguments: string(argsJSON),
		})
		leftover = strings.Replace(leftover, m[0], "", 1)
	}
	return calls, leftover
}
