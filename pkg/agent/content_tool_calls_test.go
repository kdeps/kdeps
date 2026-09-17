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

package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSalvageContentToolCalls_ToolCallTag(t *testing.T) {
	calls, cleaned, fake := salvageContentToolCalls(
		`<tool_call>{"name":"bash_exec","arguments":{"command":"ls -la"}}</tool_call>`)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.JSONEq(t, `{"command":"ls -la"}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)
	assert.False(t, fake)
}

func TestSalvageContentToolCalls_MultilineJSON(t *testing.T) {
	calls, _, _ := salvageContentToolCalls(
		"Reading the file.\n<tool_call>\n{\n  \"name\": \"read_file\",\n  \"arguments\": {\n    \"file_path\": \"/a/b.go\"\n  }\n}\n</tool_call>",
	)
	require.Len(t, calls, 1)
	assert.Equal(t, "read_file", calls[0].Name)
	assert.JSONEq(t, `{"file_path":"/a/b.go"}`, calls[0].Arguments)
}

func TestSalvageContentToolCalls_TwoBlocks(t *testing.T) {
	calls, cleaned, _ := salvageContentToolCalls(
		`<tool_call>{"name":"a","arguments":{}}</tool_call> keep me <tool_call>{"name":"b","arguments":{}}</tool_call>`,
	)
	require.Len(t, calls, 2)
	assert.Equal(t, "a", calls[0].Name)
	assert.Equal(t, "b", calls[1].Name)
	assert.Equal(t, "keep me", cleaned)
}

func TestSalvageContentToolCalls_ParametersKey(t *testing.T) {
	calls, _, _ := salvageContentToolCalls(
		`<tool_call>{"name":"x","parameters":{"k":1}}</tool_call>`)
	require.Len(t, calls, 1)
	assert.JSONEq(t, `{"k":1}`, calls[0].Arguments)
}

func TestSalvageContentToolCalls_FunctionAttr(t *testing.T) {
	calls, cleaned, _ := salvageContentToolCalls(
		`<function=web_search>{"query":"kdeps"}</function>`)
	require.Len(t, calls, 1)
	assert.Equal(t, "web_search", calls[0].Name)
	assert.JSONEq(t, `{"query":"kdeps"}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)
}

func TestSalvageContentToolCalls_BareJSONObject(t *testing.T) {
	calls, cleaned, _ := salvageContentToolCalls(
		`{"name":"list_files","arguments":{"path":"."}}`)
	require.Len(t, calls, 1)
	assert.Equal(t, "list_files", calls[0].Name)
	assert.Equal(t, "", cleaned)
}

func TestSalvageContentToolCalls_JSONAnswerNotACall(t *testing.T) {
	// A JSON *answer* (no name+arguments shape) must not be eaten.
	calls, cleaned, fake := salvageContentToolCalls(`{"result":"ok","count":3}`)
	assert.Empty(t, calls)
	assert.Equal(t, `{"result":"ok","count":3}`, cleaned)
	assert.False(t, fake)
}

func TestSalvageContentToolCalls_HallucinatedResponse(t *testing.T) {
	calls, cleaned, fake := salvageContentToolCalls(
		"I ran it.\n<tool_response>\ntotal 0\ndrwxr-xr-x  2 me me\n</tool_response>\nAll done! The directory is empty.",
	)
	assert.Empty(t, calls)
	assert.True(t, fake)
	assert.NotContains(t, cleaned, "tool_response")
	assert.NotContains(t, cleaned, "total 0")
	assert.Contains(t, cleaned, "All done!")
}

func TestSalvageContentToolCalls_PlainProse(t *testing.T) {
	calls, cleaned, fake := salvageContentToolCalls("The timeout is 30 seconds and configurable.")
	assert.Empty(t, calls)
	assert.False(t, fake)
	assert.Equal(t, "The timeout is 30 seconds and configurable.", cleaned)
}

func TestSalvageContentToolCalls_DSML(t *testing.T) {
	in := "before <｜DSML｜tool_calls>{\"name\":\"bash_exec\",\"arguments\":{\"command\":\"pwd\"}}</｜DSML｜tool_calls> after"
	calls, cleaned, _ := salvageContentToolCalls(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.Equal(t, "before after", strings.Join(strings.Fields(cleaned), " "))
	assert.NotContains(t, cleaned, "DSML")
}

func TestStripContentToolCalls_StripsTags(t *testing.T) {
	got := stripContentToolCalls(
		"Result:\n<tool_call>{\"name\":\"x\",\"arguments\":{}}</tool_call>\n<tool_response>fake</tool_response>\nHere is the answer.",
	)
	assert.NotContains(t, got, "tool_call")
	assert.NotContains(t, got, "tool_response")
	assert.Contains(t, got, "Here is the answer.")
}

func TestStripContentToolCalls_JSONArrayStillEmptied(t *testing.T) {
	assert.Equal(t, "", stripContentToolCalls(`[{"name":"a","arguments":{}}]`))
}

func TestStripContentToolCalls_PlainTextUnchanged(t *testing.T) {
	assert.Equal(t, "just an answer", stripContentToolCalls("just an answer"))
}

func TestSalvageContentToolCalls_AnthropicInvoke(t *testing.T) {
	in := "Listing the directory.\n<function_calls>\n<invoke name=\"bash_exec\">\n<parameter name=\"command\">ls -la</parameter>\n</invoke>\n</function_calls>"
	calls, cleaned, fake := salvageContentToolCalls(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.JSONEq(t, `{"command":"ls -la"}`, calls[0].Arguments)
	assert.Equal(t, "Listing the directory.", cleaned)
	assert.False(t, fake)
}

func TestSalvageContentToolCalls_InvokeMultipleParams(t *testing.T) {
	in := `<invoke name="edit_file"><parameter name="file_path">/a.go</parameter><parameter name="start_line">3</parameter><parameter name="new_string">x</parameter></invoke>`
	calls, _, _ := salvageContentToolCalls(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "edit_file", calls[0].Name)
	assert.JSONEq(t, `{"file_path":"/a.go","start_line":3,"new_string":"x"}`, calls[0].Arguments)
}

// TestSalvageContentToolCalls_InvokeDecodesHTMLEntities covers a model (seen
// live on GPT-5.6) that writes the text-fallback <invoke> form as if it were
// real markup and entity-encodes it accordingly: "a && b" arrives as
// "a &amp;&amp; b", which then fails when bash_exec runs it verbatim.
func TestSalvageContentToolCalls_InvokeDecodesHTMLEntities(t *testing.T) {
	in := `<invoke name="bash_exec"><parameter name="command">ls foo &amp;&amp; ls bar</parameter></invoke>`
	calls, _, _ := salvageContentToolCalls(in)
	require.Len(t, calls, 1)
	assert.JSONEq(t, `{"command":"ls foo && ls bar"}`, calls[0].Arguments)
}

// TestSalvageContentToolCalls_InvokeDecodesOtherEntities covers the rest of
// the commonly-seen entity set in one pass, not just &amp;.
func TestSalvageContentToolCalls_InvokeDecodesOtherEntities(t *testing.T) {
	in := `<invoke name="bash_exec"><parameter name="command">if [ 1 -lt 2 ]; then echo &quot;a&apos;b&quot; &gt; out.txt; fi</parameter></invoke>`
	calls, _, _ := salvageContentToolCalls(in)
	require.Len(t, calls, 1)
	assert.JSONEq(t,
		`{"command":"if [ 1 -lt 2 ]; then echo \"a'b\" > out.txt; fi"}`,
		calls[0].Arguments)
}

// TestSalvageContentToolCalls_RepairsEscapedClosingTags covers a model that
// treats the text-fallback <invoke> markup as a JSON string it's embedding
// and escapes every "/" as "\/" -- turning "</invoke>" into "<\/invoke>",
// which the tag regexes don't match at all, silently dropping the call.
func TestSalvageContentToolCalls_RepairsEscapedClosingTags(t *testing.T) {
	in := `<invoke name="bash_exec"><parameter name="command">ls -la<\/parameter><\/invoke>`
	calls, cleaned, _ := salvageContentToolCalls(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "bash_exec", calls[0].Name)
	assert.JSONEq(t, `{"command":"ls -la"}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)
}

// TestSalvageContentToolCalls_InvokeDecodesEscapedSlashesInValue covers the
// same stray JSON escaping inside a parameter's value, not just around the
// closing tags: a file path arrives as "a\/b\/c" instead of "a/b/c".
func TestSalvageContentToolCalls_InvokeDecodesEscapedSlashesInValue(t *testing.T) {
	in := `<invoke name="read_file"><parameter name="file_path">pkg\/agent\/loop.go</parameter></invoke>`
	calls, _, _ := salvageContentToolCalls(in)
	require.Len(t, calls, 1)
	assert.JSONEq(t, `{"file_path":"pkg/agent/loop.go"}`, calls[0].Arguments)
}

func TestSalvageContentToolCalls_InvokeNamespaced(t *testing.T) {
	// A leaked call may carry a namespace prefix on the tags.
	prefix := "an" + "tml:"
	in := "<" + prefix + `invoke name="web_search"><` + prefix +
		`parameter name="query">kdeps</` + prefix + "parameter></" + prefix + "invoke>"
	calls, cleaned, _ := salvageContentToolCalls(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "web_search", calls[0].Name)
	assert.JSONEq(t, `{"query":"kdeps"}`, calls[0].Arguments)
	assert.Equal(t, "", cleaned)
}

func TestSalvageContentToolCalls_LoneParameterStripped(t *testing.T) {
	_, cleaned, _ := salvageContentToolCalls(
		`Here is the plan.<parameter name="x">y</parameter> Done.`)
	assert.NotContains(t, cleaned, "parameter")
	assert.Contains(t, cleaned, "Here is the plan.")
	assert.Contains(t, cleaned, "Done.")
}
