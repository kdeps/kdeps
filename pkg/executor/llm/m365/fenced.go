package m365

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"

	kdeps_debug "github.com/kdeps/kdeps/v2/pkg/debug"
)

// The chat model has no native tool-calling API. Tool calls are emulated with
// Anthropic-style XML invoke tags: the model emits <invoke name="tool_name">
// with one <parameter name="key">value</parameter> per argument. This shape
// was chosen over a Markdown-fence convention (```tool_name ... ```) because
// the model actually driving this backend is trained on this exact
// <invoke>/<parameter> format for its own native tool use, and because
// "</parameter>" / "</invoke>" collide with real file content far less often
// than a bare "```" does -- a written file or patch commonly contains its own
// nested Markdown code fence, which silently truncated fenced-style parsing.
// This file builds the prompt-side tool definitions and parses <invoke>
// blocks back into structured calls.
//	<invoke name="bash">
//	<parameter name="command">ls -la</parameter>
//	</invoke>
//	<invoke name="write_file">
//	<parameter name="path">main.go</parameter>
//	<parameter name="content">package main</parameter>
//	</invoke>
//	<invoke name="edit_file">
//	<parameter name="path">app.go</parameter>
//	<parameter name="old_string">debug = false</parameter>
//	<parameter name="new_string">debug = true</parameter>
//	</invoke>

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

// bodyParamNames are argument names carried as the free-form final parameter.
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

// shellLangs are legacy fence-language aliases still recognised as invoke
// names for "run this as a shell command"; they route to whatever
// run-a-command tool the caller provided.
//
//nolint:gochecknoglobals // static lookup, read-only; "bash" here is an alias, not a repeated literal
var shellLangs = []string{
	"bash", "sh", "shell", "zsh", "console",
	"shell-session", "shellsession", "shsession",
}

var shellToolName = regexp.MustCompile(
	`(?i)^(bash|sh|shell|zsh|run|exec|execute|command|cmd|terminal|run_command|run_terminal_cmd|execute_command|execute_bash|shell_exec|system)$`,
)

var commandParamName = regexp.MustCompile(`(?i)^(command|cmd|script|input)$`)

// invokeRegex's body capture is non-greedy ((?s).*?), matching through to the
// FIRST closing </invoke> after each opening tag -- NOT the last. This was
// greedy in an earlier version on the theory that the framing prompts'
// "ONE invoke block per turn" rule made a second real </invoke> impossible
// to over-consume into. That assumption was proven false live: the model
// chained a second call (write_file immediately followed by
// <invoke name="task_complete">...) in the same turn despite the
// instruction, and the greedy match swallowed the second invoke's raw XML
// into the first call's body parameter -- writing "</parameter></invoke>
// <invoke name=\"task_complete\">..." literally into the file. Multiple
// chained real invokes are a live-confirmed, real failure mode; a literal
// "</invoke>" substring inside a parameter's own value is a much rarer,
// theoretical one. Non-greedy correctly scopes each invoke to its own
// nearest close tag, so FindAllStringSubmatch below recovers every real call
// in the response instead of only the first (with the rest mangled into its
// body).
var invokeRegex = regexp.MustCompile(`(?s)<invoke name="([A-Za-z0-9_.\-]+)">(.*?)</invoke>`)

const parameterCloseTag = "</parameter>"

// FencedToolSpec describes how one tool maps onto the <invoke> shape.
type FencedToolSpec struct {
	Name        string
	Description string
	// HeaderParams render as single-line <parameter name="key">value</parameter>
	// tags, in declaration order, before the free-form parameter (if any).
	HeaderParams []string
	// BodyParam is the free-form argument carried as the final parameter,
	// parsed greedily through to the invoke's closing tag.
	BodyParam string
	// EditPair, when set, renders an (old -> new) pair of parameters instead
	// of a single body parameter.
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

// DeriveFencedSpec derives the <invoke> shape for a single tool.
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

// BuildSpecMap maps each tool name to its invoke spec, aliasing legacy shell
// names (bash, sh, ...) onto the caller's shell tool so the model can invoke
// e.g. <invoke name="bash"> and still produce a structured call.
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

func renderParameter(name, value string) string {
	return `<parameter name="` + name + `">` + value + parameterCloseTag
}

// RenderFencedCall renders one concrete call (name + args) as an <invoke>
// block, for replaying a prior assistant tool call back into the transcript.
func RenderFencedCall(spec FencedToolSpec, args map[string]any) string {
	var lines []string
	for _, h := range spec.HeaderParams {
		if v, ok := args[h]; ok {
			lines = append(lines, renderParameter(h, scalarToString(v)))
		}
	}
	switch {
	case spec.EditPair != nil:
		lines = append(lines,
			renderParameter(spec.EditPair.search, scalarToString(args[spec.EditPair.search])),
			renderParameter(spec.EditPair.replace, scalarToString(args[spec.EditPair.replace])),
		)
	case spec.BodyParam != "":
		lines = append(lines, renderParameter(spec.BodyParam, scalarToString(args[spec.BodyParam])))
	}
	return `<invoke name="` + spec.Name + `">` + "\n" + strings.Join(lines, "\n") + "\n</invoke>"
}

// renderFencedTemplate renders a self-documenting template for the <tools> block.
func renderFencedTemplate(spec FencedToolSpec) string {
	var lines []string
	for _, h := range spec.HeaderParams {
		lines = append(lines, renderParameter(h, "<"+h+">"))
	}
	switch {
	case spec.EditPair != nil:
		lines = append(lines,
			renderParameter(spec.EditPair.search, "<"+spec.EditPair.search+">"),
			renderParameter(spec.EditPair.replace, "<"+spec.EditPair.replace+">"),
		)
	case spec.BodyParam != "":
		lines = append(lines, renderParameter(spec.BodyParam, "<"+spec.BodyParam+">"))
	}
	header := spec.Name
	if spec.Description != "" {
		header = spec.Name + " - " + spec.Description
	}
	return header + "\n" + `<invoke name="` + spec.Name + `">` + "\n" + strings.Join(lines, "\n") + "\n</invoke>"
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

// dedicatedFileTools names the file-operating tools kdeps registers
// alongside the shell tool, in the order fileToolHints should mention them.
//
//nolint:gochecknoglobals // static lookup table
var dedicatedFileTools = []struct{ name, hint string }{
	{"write_file", "create or overwrite a file: `write_file`"},
	{"edit_file", "make a targeted change: `edit_file`"},
	{"read_file", "read a file: `read_file`"},
	{"list_files", "list a directory: `list_files`"},
}

// fileToolHints redirects file creation/editing/reading/listing to kdeps's
// dedicated tools when the tool list includes them. Without this, the
// forceful shell framing below (needed to stop weaker models from just
// describing what they'd run instead of acting) drowns out each tool's own
// description, and the model defaults to shelling out via heredocs/sed even
// for a plain file write that has a dedicated tool. Returns "" when none of
// the dedicated tools are registered, leaving the shell as the only option.
func fileToolHints(tools []ToolDef) string {
	has := func(name string) bool {
		for _, t := range tools {
			if t.Function.Name == name {
				return true
			}
		}
		return false
	}
	var hints []string
	for _, ft := range dedicatedFileTools {
		if has(ft.name) {
			hints = append(hints, ft.hint)
		}
	}
	if len(hints) == 0 {
		return ""
	}
	return " FILE OPERATIONS HAVE DEDICATED TOOLS -- prefer them over the shell: " +
		strings.Join(hints, "; ") +
		". Reach for the shell for what those don't cover: running scripts or tests, git, and inspecting process or system state."
}

func baselineFraming(tools []ToolDef) string {
	var shellFraming string
	if shell := findShellTool(tools); shell != nil {
		fileHints := fileToolHints(tools)
		firstAction := "an <invoke name=\"" + shell.Function.Name + "\"> block (e.g. `ls -la` then `cat` the relevant files)"
		if fileHints != "" {
			firstAction = "the invoke block for whichever tool the step needs " +
				"(e.g. `list_files` to see what's there, or `read_file` on a named file)"
		}
		shellFraming = "\n\nYou have a real shell (the `" + shell.Function.Name + "` tool)." + fileHints +
			" To perform a step, emit ONE <invoke> block for the single tool that step needs, acting end-to-end against the real files in the working directory: an invoke of the shell tool inspects with `cat`/`ls`/`grep` and runs code with the available interpreters." +
			" The block is executed for real and you get its output back. Writing the commands IS doing the task; describing what you \"would\" run, or claiming you did it, accomplishes nothing.\n\nYou have NOT run any command yet and have NO results. NEVER claim a command \"returned no output\", that files are \"missing\", or that you \"cannot access\" / \"cannot list\" the environment before you have actually emitted an invoke block and seen its <tool_response>. The files named in the task are present on a real filesystem right now. Your FIRST output must be " + firstAction + " - never open with prose, a question, or a request for the user to paste files. Do not assume a file's contents or a command's result; run a tool and read the real output. One self-contained invoke block per turn."
	}
	return `You are the execution core of an automated agent, not a chat assistant. Your output is parsed by a program - a real runtime that executes your tool calls against a live system and returns the actual results to you in <tool_response> blocks.` + shellFraming + `

Performing the task with tools is your PRIMARY JOB. Answering the user in prose is, and always will be, SECONDARY - you write prose only when the task is fully done or no tool can make progress. Default to acting, not talking.

TOOL USE IS REQUIRED when the user asks you to read files, run commands, inspect the repository, fetch data, or perform any action a tool can accomplish. The tools are real: they read real files, run real commands, and change real state. Never answer from memory or simulate a result when a tool can provide it.

To call a tool, output ONLY a single ` + "`<invoke name=\"tool_name\">`" + ` block. An invoke block is an ACTION the runtime executes - it is NOT an illustration, an example, or "here's how you would do it". No text before or after it:

<invoke name="<tool_name>">
<parameter name="<param_name>"><value></parameter>
...
</invoke>

STRICT RULES:
- Output ONLY the invoke block when calling a tool. No prose, no second invoke, no commentary before or after.
- Never describe your intent ("I'll read the file...", "Let me check...") and never emit filler or acknowledgements. Each turn is exactly one invoke block OR the final answer - nothing in between.
- One tool call per response, then stop and wait for its <tool_response>. Never emit two invoke blocks in one response.
- The invoke's "name" attribute and each parameter's "name" attribute must match a tool defined below exactly.
- Every tool below is called with ITS OWN invoke, name="<that tool's name>" (e.g. <invoke name="memory_search"> to search memory, <invoke name="read_file"> to read a file) - never call one tool by writing another tool's name as the shell command inside an invoke of the shell tool. The shell only runs real system commands; it does not know kdeps tool names and will fail with "command not found".
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
	fileHints := fileToolHints(tools)
	shellPurpose := "acts on the real files in the working directory (heredocs to create, `sed -i` to edit, `cat`/`ls`/`grep` to inspect, interpreters to run)"
	if fileHints != "" {
		shellPurpose = "runs scripts/tests, git, or inspects process/system state"
	}
	return "You are an automated agent with a real shell (the `" + name + "` tool)." + fileHints +
		` You do the task by emitting ONE <invoke> block per turn for whichever tool the step needs; an invoke of the shell tool ` + shellPurpose + `. The block is executed for real; its output comes back in a <tool_response>. Writing the commands IS doing the task. To call a tool other than the shell (e.g. memory_search), use ITS OWN invoke directly - never write another tool's name as a shell command, the shell will fail with "command not found".` +
		"\n\nYou have run nothing yet. Your FIRST output must be an invoke block - never prose, a question, or \"I can't access the files\". Never claim a result you have not seen in a <tool_response>.\n\n" + toolsBlock(
		tools,
	)
}

func softenedFraming(tools []ToolDef) string {
	name := shellName(tools)
	var shellLine string
	if findShellTool(tools) != nil {
		fileHints := fileToolHints(tools)
		shellPurpose := "create or update files with heredocs, adjust them in place, inspect with cat/ls/grep, run code with the available interpreters"
		if fileHints != "" {
			shellPurpose = "run scripts or tests, use git, or inspect process/system state"
		}
		shellLine = "You have a real shell available as the `" + name + "` tool." + fileHints +
			" The usual way to make progress on a step the shell covers is to write a single invoke of the shell tool that carries it out against the real files in the working directory - " + shellPurpose + ". The runtime executes the block and returns its real output to you. Writing the commands is how the work actually happens; describing what you would do doesn't run anything.\n\n"
	}
	return `You are an automated coding agent working in a real working directory. Your replies are read by a program that runs your tool calls and returns the results.

` + shellLine + `To use a tool, reply with a single ` + "`<invoke name=\"tool_name\">`" + ` block (an invoke is run as a real action, not shown as an illustration):

<invoke name="<tool_name>">
<parameter name="<param_name>"><value></parameter>
...
</invoke>

A <tool_response> is the real result from the live system - rely on it rather than assuming what a command would print. Work one step at a time: one tool call per reply, then wait for its <tool_response>. Begin by running an invoke of the shell tool that inspects the relevant files, rather than answering from memory, and keep going with tool calls until the task is finished. To call a tool other than the shell, use its own invoke directly - never write another tool's name as a shell command inside an invoke of the shell tool, the shell does not know kdeps tool names and will fail with "command not found". Reply in plain language only once the task is done and no further tool call would help.

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

// parseFencedInner parses one <invoke> block body into an arguments object.
// inner is everything between the opening <invoke name="..."> tag and the
// closing </invoke> tag.
func parseFencedInner(spec FencedToolSpec, inner string) (map[string]any, bool) {
	args := map[string]any{}
	rest := inner

	for _, h := range spec.HeaderParams {
		open := `<parameter name="` + h + `">`
		idx := strings.Index(rest, open)
		if idx == -1 {
			continue
		}
		afterOpen := rest[idx+len(open):]
		closeIdx := strings.Index(afterOpen, parameterCloseTag)
		if closeIdx == -1 {
			continue
		}
		args[h] = strings.TrimSpace(afterOpen[:closeIdx])
		rest = afterOpen[closeIdx+len(parameterCloseTag):]
	}

	switch {
	case spec.EditPair != nil:
		searchVal, replaceVal, ok := parseEditPairParams(rest, spec.EditPair)
		if !ok {
			return nil, false
		}
		args[spec.EditPair.search] = searchVal
		args[spec.EditPair.replace] = replaceVal
	case spec.BodyParam != "":
		val, ok := parseFinalParam(rest, spec.BodyParam)
		if !ok {
			return nil, false
		}
		args[spec.BodyParam] = stripRedundantOuterFence(val)
	}
	return args, true
}

// stripRedundantOuterFence removes a body's own self-wrapping Markdown code
// fence, e.g. a model asked to write_file a markdown document habitually
// wraps the whole thing in an extra ```markdown ... ``` block as if
// displaying it, even though the outer <parameter name="content"> tag
// already delimits the body -- kdeps then wrote that inner wrapper's own
// fence markers literally into the file (confirmed live: the written file's
// first line was a bare ``` line). This habit is independent of the
// tool-call delimiter convention (XML invoke tags here, Markdown fences
// before), so the strip still applies under either. Only strips when the
// body's FIRST line is itself a fence open and its LAST line is a bare fence
// close -- i.e. the entire body is wrapped in exactly one pair, not a
// document that happens to contain an internal code block partway through.
// minWrappedFenceLines is the fewest lines a self-wrapped body can have: an
// opening fence line and a closing fence line, with the wrapped content (if
// any) in between.
const minWrappedFenceLines = 2

func stripRedundantOuterFence(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) < minWrappedFenceLines {
		return content
	}
	first := strings.TrimRight(lines[0], "\r")
	if !strings.HasPrefix(first, "```") {
		return content
	}
	last := strings.TrimRight(lines[len(lines)-1], "\r")
	if strings.TrimSpace(last) != "```" {
		return content
	}
	return strings.Join(lines[1:len(lines)-1], "\n")
}

// parseFinalParam extracts the last parameter's value: everything after its
// open tag through to the LAST occurrence of the close tag in rest (greedy).
// rest is already scoped to a single invoke's own inner text (invokeRegex is
// non-greedy per call, so a second real invoke chained after this one is
// never part of rest) -- the only remaining risk this greedy match protects
// against is the value itself containing a literal "</parameter>"-shaped
// substring, which is far rarer than a second chained call turned out to be.
func parseFinalParam(rest, name string) (string, bool) {
	open := `<parameter name="` + name + `">`
	idx := strings.Index(rest, open)
	if idx == -1 {
		return "", false
	}
	afterOpen := rest[idx+len(open):]
	closeIdx := strings.LastIndex(afterOpen, parameterCloseTag)
	if closeIdx == -1 {
		return "", false
	}
	return afterOpen[:closeIdx], true
}

// parseEditPairParams extracts the search/replace pair. search is parsed
// non-greedily up to its own close tag (its value is expected to be a
// short, targeted snippet); replace is parsed greedily as the final
// parameter, same as parseFinalParam.
func parseEditPairParams(rest string, pair *editPair) (string, string, bool) {
	searchOpen := `<parameter name="` + pair.search + `">`
	idx := strings.Index(rest, searchOpen)
	if idx == -1 {
		return "", "", false
	}
	afterSearchOpen := rest[idx+len(searchOpen):]

	searchCloseIdx := strings.Index(afterSearchOpen, parameterCloseTag)
	if searchCloseIdx == -1 {
		return "", "", false
	}
	search := afterSearchOpen[:searchCloseIdx]
	afterSearchClose := afterSearchOpen[searchCloseIdx+len(parameterCloseTag):]

	replaceOpen := `<parameter name="` + pair.replace + `">`
	replaceIdx := strings.Index(afterSearchClose, replaceOpen)
	if replaceIdx == -1 {
		return "", "", false
	}
	afterReplaceOpen := afterSearchClose[replaceIdx+len(replaceOpen):]

	replaceCloseIdx := strings.LastIndex(afterReplaceOpen, parameterCloseTag)
	if replaceCloseIdx == -1 {
		return "", "", false
	}
	replace := afterReplaceOpen[:replaceCloseIdx]
	return search, replace, true
}

// ParseFencedToolCalls parses every <invoke> block whose name matches a known
// tool. leftover is the text with those blocks removed.
func ParseFencedToolCalls(text string, specs map[string]FencedToolSpec) ([]ParsedToolCall, string) {
	kdeps_debug.Log("enter: ParseFencedToolCalls")
	var calls []ParsedToolCall
	leftover := text
	for _, m := range invokeRegex.FindAllStringSubmatch(text, -1) {
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
