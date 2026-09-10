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
