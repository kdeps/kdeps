// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package toolcall

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSalvage_ToolCallTag(t *testing.T) {
	calls, cleaned, fake := Salvage(
		`<tool_call>{"name":"bash_exec","arguments":{"command":"ls -la"}}</tool_call>`)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.JSONEq(t, `{"command":"ls -la"}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)
	assert.False(t, fake)
}

func TestSalvage_MultilineJSON(t *testing.T) {
	calls, _, _ := Salvage(
		"Reading the file.\n<tool_call>\n{\n  \"name\": \"read_file\",\n  \"arguments\": {\n    \"file_path\": \"/a/b.go\"\n  }\n}\n</tool_call>",
	)
	require.Len(t, calls, 1)
	assert.Equal(t, "read_file", calls[0].Name)
	assert.JSONEq(t, `{"file_path":"/a/b.go"}`, calls[0].Arguments)
}

func TestSalvage_TwoBlocks(t *testing.T) {
	calls, cleaned, _ := Salvage(
		`<tool_call>{"name":"a","arguments":{}}</tool_call> keep me <tool_call>{"name":"b","arguments":{}}</tool_call>`,
	)
	require.Len(t, calls, 2)
	assert.Equal(t, "a", calls[0].Name)
	assert.Equal(t, "b", calls[1].Name)
	assert.Equal(t, "keep me", cleaned)
}

func TestSalvage_ParametersKey(t *testing.T) {
	calls, _, _ := Salvage(
		`<tool_call>{"name":"x","parameters":{"k":1}}</tool_call>`)
	require.Len(t, calls, 1)
	assert.JSONEq(t, `{"k":1}`, calls[0].Arguments)
}

func TestSalvage_FunctionAttr(t *testing.T) {
	calls, cleaned, _ := Salvage(
		`<function=web_search>{"query":"kdeps"}</function>`)
	require.Len(t, calls, 1)
	assert.Equal(t, "web_search", calls[0].Name)
	assert.JSONEq(t, `{"query":"kdeps"}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)
}

// TestSalvage_FunctionWithParameterTags checks the
// <tool_call><function=NAME><parameter=KEY>VALUE</parameter></function> form,
// as a free OpenRouter model wrote it during a real session: the call was
// dropped, the model repeated it, and the turn ended without the read.
func TestSalvage_FunctionWithParameterTags(t *testing.T) {
	text := "<tool_call>\n<function=read_file>\n<parameter=file_path>\n/work/egm/standard.md\n</parameter>\n" +
		"<parameter=limit>\n40\n</parameter>\n</function>\n</tool_call>"
	calls, cleaned, _ := Salvage(text)
	require.Len(t, calls, 1)
	assert.Equal(t, "read_file", calls[0].Name)
	assert.JSONEq(t, `{"file_path":"/work/egm/standard.md","limit":40}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)

	calls, _, _ = Salvage("<function=list_files>\n<parameter=path>\n.\n</parameter>\n</function>")
	require.Len(t, calls, 1, "the same form without the tool_call wrapper")
	assert.JSONEq(t, `{"path":"."}`, calls[0].Arguments)
}

func TestSalvage_BareJSONObject(t *testing.T) {
	calls, cleaned, _ := Salvage(
		`{"name":"list_files","arguments":{"path":"."}}`)
	require.Len(t, calls, 1)
	assert.Equal(t, "list_files", calls[0].Name)
	assert.Equal(t, "", cleaned)
}

func TestSalvage_JSONAnswerNotACall(t *testing.T) {
	// A JSON *answer* (no name+arguments shape) must not be eaten.
	calls, cleaned, fake := Salvage(`{"result":"ok","count":3}`)
	assert.Empty(t, calls)
	assert.Equal(t, `{"result":"ok","count":3}`, cleaned)
	assert.False(t, fake)
}

func TestSalvage_HallucinatedResponse(t *testing.T) {
	calls, cleaned, fake := Salvage(
		"I ran it.\n<tool_response>\ntotal 0\ndrwxr-xr-x  2 me me\n</tool_response>\nAll done! The directory is empty.",
	)
	assert.Empty(t, calls)
	assert.True(t, fake)
	assert.NotContains(t, cleaned, "tool_response")
	assert.NotContains(t, cleaned, "total 0")
	assert.Contains(t, cleaned, "All done!")
}

func TestSalvage_PlainProse(t *testing.T) {
	calls, cleaned, fake := Salvage("The timeout is 30 seconds and configurable.")
	assert.Empty(t, calls)
	assert.False(t, fake)
	assert.Equal(t, "The timeout is 30 seconds and configurable.", cleaned)
}

func TestSalvage_DSML(t *testing.T) {
	in := "before <｜DSML｜tool_calls>{\"name\":\"bash_exec\",\"arguments\":{\"command\":\"pwd\"}}</｜DSML｜tool_calls> after"
	calls, cleaned, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.Equal(t, "before after", strings.Join(strings.Fields(cleaned), " "))
	assert.NotContains(t, cleaned, "DSML")
}

// DeepSeek's native DSML markup wraps the same invoke/parameter elements as
// the Anthropic text form, with a string="true" attribute added. The exact
// payload below was emitted as plain text by deepseek-v4-flash; it produced
// no tool call, so the model kept repeating it until the turn ended.
func TestSalvage_DSMLInvoke(t *testing.T) {
	in := "<｜｜DSML｜｜ calls>\n" +
		"<｜｜DSML｜｜ invoke name=\"bash_exec\">\n" +
		"<｜｜DSML｜｜ parameter name=\"command\" string=\"true\">rg -n 'WaitSearchIndex' pkg</｜｜DSML｜｜ parameter>\n" +
		"</｜｜DSML｜｜ invoke>\n" +
		"</｜｜DSML｜｜ calls>"
	calls, cleaned, fake := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.JSONEq(t, `{"command":"rg -n 'WaitSearchIndex' pkg"}`, calls[0].Arguments)
	assert.False(t, fake)
	assert.Empty(t, strings.TrimSpace(cleaned))
	assert.NotContains(t, cleaned, "DSML")
}

func TestSalvage_DSMLInvokeTwice(t *testing.T) {
	in := "<｜DSML｜ calls>\n" +
		"<｜DSML｜ invoke name=\"read_file\">\n" +
		"<｜DSML｜ parameter name=\"file_path\" string=\"true\">/tmp/a.go</｜DSML｜ parameter>\n" +
		"</｜DSML｜ invoke>\n" +
		"<｜DSML｜ invoke name=\"bash_exec\">\n" +
		"<｜DSML｜ parameter name=\"command\" string=\"true\">pwd</｜DSML｜ parameter>\n" +
		"</｜DSML｜ invoke>\n" +
		"</｜DSML｜ calls>"
	calls, cleaned, _ := Salvage(in)
	require.Len(t, calls, 2)
	assert.Equal(t, "read_file", calls[0].Name)
	assert.JSONEq(t, `{"file_path":"/tmp/a.go"}`, calls[0].Arguments)
	assert.Equal(t, "bash_exec", calls[1].Name)
	assert.JSONEq(t, `{"command":"pwd"}`, calls[1].Arguments)
	assert.Empty(t, strings.TrimSpace(cleaned))
}

func TestSalvage_ShellBodyIsTheCommand(t *testing.T) {
	cases := []struct {
		name string
		in   string
	}{
		{"plain", "<<invoke name=\"bash\">\nls -la && cat README.md\n</invoke>"},
		{"attribute", "<<invoke name=\"bash\" command=\"ls -la && cat README.md\">\n</invoke>"},
		{"assign", "<invoke name=\"bash\">\ncommand=\"ls -la && cat README.md\"\n</invoke>"},
		{"colon", "<invoke name=\"bash\">\ncommand: ls -la && cat README.md\n</invoke>"},
		{"json", "<invoke name=\"bash\">\n{\"command\":\"ls -la && cat README.md\"}\n</invoke>"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls, _, _ := Salvage(tc.in)
			require.Len(t, calls, 1)
			assert.Equal(t, "bash", calls[0].Name)
			assert.JSONEq(t, `{"command":"ls -la && cat README.md"}`, calls[0].Arguments)
		})
	}
}

func TestSalvage_PlainBodyIsNotARead(t *testing.T) {
	in := "<invoke name=\"read_file\">\npkg/api/calculator.go\n</invoke>"
	calls, _, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.JSONEq(t, `{}`, calls[0].Arguments)
}

func TestSalvage_AnthropicInvoke(t *testing.T) {
	in := "Listing the directory.\n<function_calls>\n<invoke name=\"bash_exec\">\n<parameter name=\"command\">ls -la</parameter>\n</invoke>\n</function_calls>"
	calls, cleaned, fake := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.JSONEq(t, `{"command":"ls -la"}`, calls[0].Arguments)
	assert.Equal(t, "Listing the directory.", cleaned)
	assert.False(t, fake)
}

func TestSalvage_InvokeMultipleParams(t *testing.T) {
	in := `<invoke name="edit_file"><parameter name="file_path">/a.go</parameter><parameter name="start_line">3</parameter><parameter name="new_string">x</parameter></invoke>`
	calls, _, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "edit_file", calls[0].Name)
	assert.JSONEq(t, `{"file_path":"/a.go","start_line":3,"new_string":"x"}`, calls[0].Arguments)
}

// TestSalvage_InvokeDecodesHTMLEntities covers a model (seen
// live on GPT-5.6) that writes the text-fallback <invoke> form as if it were
// real markup and entity-encodes it accordingly: "a && b" arrives as
// "a &amp;&amp; b", which then fails when bash_exec runs it verbatim.
func TestSalvage_InvokeDecodesHTMLEntities(t *testing.T) {
	in := `<invoke name="bash_exec"><parameter name="command">ls foo &amp;&amp; ls bar</parameter></invoke>`
	calls, _, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.JSONEq(t, `{"command":"ls foo && ls bar"}`, calls[0].Arguments)
}

// TestSalvage_InvokeDecodesOtherEntities covers the rest of
// the commonly-seen entity set in one pass, not just &amp;.
func TestSalvage_InvokeDecodesOtherEntities(t *testing.T) {
	in := `<invoke name="bash_exec"><parameter name="command">if [ 1 -lt 2 ]; then echo &quot;a&apos;b&quot; &gt; out.txt; fi</parameter></invoke>`
	calls, _, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.JSONEq(t,
		`{"command":"if [ 1 -lt 2 ]; then echo \"a'b\" > out.txt; fi"}`,
		calls[0].Arguments)
}

// TestSalvage_RepairsEscapedClosingTags covers a model that
// treats the text-fallback <invoke> markup as a JSON string it's embedding
// and escapes every "/" as "\/" -- turning "</invoke>" into "<\/invoke>",
// which the tag regexes don't match at all, silently dropping the call.
func TestSalvage_RepairsEscapedClosingTags(t *testing.T) {
	in := `<invoke name="bash_exec"><parameter name="command">ls -la<\/parameter><\/invoke>`
	calls, cleaned, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.JSONEq(t, `{"command":"ls -la"}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)
}

// TestSalvage_InvokeDecodesEscapedSlashesInValue covers the
// same stray JSON escaping inside a parameter's value, not just around the
// closing tags: a file path arrives as "a\/b\/c" instead of "a/b/c".
func TestSalvage_InvokeDecodesEscapedSlashesInValue(t *testing.T) {
	in := `<invoke name="read_file"><parameter name="file_path">pkg\/agent\/loop.go</parameter></invoke>`
	calls, _, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.JSONEq(t, `{"file_path":"pkg/agent/loop.go"}`, calls[0].Arguments)
}

// TestSalvage_RepairsDoubleSlashClosingTag covers a
// different, no-backslash mangling of the same closing tag: some models
// write "<//invoke>"/"<///parameter>" (a doubled/tripled literal slash)
// instead of "</invoke>"/"</parameter>", which the tag regexes don't match
// at all, silently dropping the call.
func TestSalvage_RepairsDoubleSlashClosingTag(t *testing.T) {
	in := `<invoke name="bash_exec"><parameter name="command">ls -la<//parameter><///invoke>`
	calls, cleaned, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.JSONEq(t, `{"command":"ls -la"}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)
}

func TestSalvage_InvokeNamespaced(t *testing.T) {
	// A leaked call may carry a namespace prefix on the tags.
	prefix := "an" + "tml:"
	in := "<" + prefix + `invoke name="web_search"><` + prefix +
		`parameter name="query">kdeps</` + prefix + "parameter></" + prefix + "invoke>"
	calls, cleaned, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "web_search", calls[0].Name)
	assert.JSONEq(t, `{"query":"kdeps"}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)
}

func TestSalvage_LoneParameterStripped(t *testing.T) {
	_, cleaned, _ := Salvage(
		`Here is the plan.<parameter name="x">y</parameter> Done.`)
	assert.NotContains(t, cleaned, "parameter")
	assert.Contains(t, cleaned, "Here is the plan.")
	assert.Contains(t, cleaned, "Done.")
}

// A code fence demonstrating the <invoke>/<parameter> tool-call syntax --
// exactly what use-kdeps-tools.yaml teaches models to write -- must survive
// intact: it is example text, not a real call and not a hallucinated result,
// and every regex in this file runs over the whole reply with no notion of
// "inside a fence" unless protectFencedCodeBlocks shields it first.
func TestSalvage_InvokeExampleInsideFenceSurvives(t *testing.T) {
	in := "Here is how to call it:\n\n```\n" +
		`<invoke name="read_file">` + "\n" +
		`<parameter name="file_path">cmd/serve.go</parameter>` + "\n" +
		"</invoke>\n```\n\nThen wait for the result."
	calls, cleaned, fake := Salvage(in)
	assert.Empty(t, calls, "example markup inside a fence must never be parsed as a real call")
	assert.False(t, fake)
	assert.Contains(t, cleaned, `<invoke name="read_file">`)
	assert.Contains(t, cleaned, `<parameter name="file_path">cmd/serve.go</parameter>`)
	assert.Contains(t, cleaned, "```")
	assert.Contains(t, cleaned, "Then wait for the result.")
}

// A <tool_response> shown inside a fence as a documentation example (e.g.
// explaining what NOT to write) must not be flagged as a hallucinated result
// -- only a real, unfenced self-authored <tool_response> is.
func TestSalvage_ToolResponseInsideFenceNotFlagged(t *testing.T) {
	in := "Never write this yourself:\n\n```\n<tool_response>fake</tool_response>\n```\n\nCall the real tool instead."
	calls, cleaned, fake := Salvage(in)
	assert.Empty(t, calls)
	assert.False(t, fake, "a fenced example must not be flagged as a hallucinated result")
	assert.Contains(t, cleaned, "<tool_response>fake</tool_response>")
	assert.Contains(t, cleaned, "Call the real tool instead.")
}

// Tilde fences must be protected the same as backtick fences.
func TestSalvage_InvokeExampleInsideTildeFenceSurvives(t *testing.T) {
	in := "Example:\n\n~~~\n" + `<invoke name="bash_exec"><parameter name="command">pwd</parameter></invoke>` + "\n~~~\n\nDone."
	calls, cleaned, _ := Salvage(in)
	assert.Empty(t, calls)
	assert.Contains(t, cleaned, `<invoke name="bash_exec">`)
	assert.Contains(t, cleaned, "~~~")
}

// A real (unfenced) tool call written as text right next to a fenced example
// must still be recovered -- fence protection must not swallow genuine calls
// outside the fence.
func TestSalvage_RealCallOutsideFenceStillRecovered(t *testing.T) {
	in := `<tool_call>{"name":"bash_exec","arguments":{"command":"pwd"}}</tool_call>` +
		"\n\nFor reference, the syntax looks like:\n\n```\n<invoke name=\"bash_exec\"></invoke>\n```"
	calls, cleaned, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.Contains(t, cleaned, "```")
	assert.Contains(t, cleaned, `<invoke name="bash_exec"></invoke>`)
}

// A literal invoke outside the fence is the call. The copy inside the fence
// is an example and must not run, even when the two blocks are identical.
func TestSalvage_UnfencedInvokeBesideFencedExample(t *testing.T) {
	in := "<invoke name=\"bash_exec\"><parameter name=\"command\">pwd</parameter></invoke>\n\n" +
		"```text\n<invoke name=\"bash_exec\"><parameter name=\"command\">rm -rf /</parameter></invoke>\n```\n"
	calls, cleaned, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.JSONEq(t, `{"command":"pwd"}`, calls[0].Arguments)
	assert.Contains(t, cleaned, "rm -rf /")
	assert.NotContains(t, cleaned, ">pwd<")
}

// A fence inside a parameter is file content. Protecting fences by swapping
// them out before argument parsing used to replace that content with a
// placeholder, so the call ran with the fence bytes missing.
func TestSalvage_FenceInsideParameterKept(t *testing.T) {
	in := "<invoke name=\"write_file\">\n" +
		"<parameter name=\"content\"># Plan\n\n```bash\necho hello\n```\n</parameter>\n" +
		"</invoke>"
	calls, cleaned, _ := Salvage(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "write_file", calls[0].Name)
	assert.Contains(t, calls[0].Arguments, "echo hello")
	assert.Contains(t, calls[0].Arguments, "```bash")
	assert.NotContains(t, cleaned, "echo hello")
}
