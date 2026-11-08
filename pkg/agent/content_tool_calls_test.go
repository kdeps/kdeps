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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStripContentToolCalls_DSMLInvokeIsNotAnswer(t *testing.T) {
	in := "<｜｜DSML｜｜ calls>\n" +
		"<｜｜DSML｜｜ invoke name=\"bash_exec\">\n" +
		"<｜｜DSML｜｜ parameter name=\"command\" string=\"true\">pwd</｜｜DSML｜｜ parameter>\n" +
		"</｜｜DSML｜｜ invoke>\n" +
		"</｜｜DSML｜｜ calls>"
	assert.Empty(t, stripContentToolCalls(in))
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

// TestSalvageHandshakeToolCall_RecoversFencedInvoke is the direct regression
// test for a live failure mode: a small local model, asked to send a
// literal <invoke> block for session_handshake, wrote a genuine,
// correctly-coded one but wrapped it in a markdown code fence out of habit.
// salvageContentToolCalls correctly treats that as a protected "example" (see
// TestSalvageContentToolCalls_InvokeExampleInsideFenceSurvives) and drops
// it -- exactly right for an ordinary turn, but it meant a real, intentional
// handshake call was silently never dispatched, causing an endless retry
// loop even though the model "did it correctly." salvageHandshakeToolCall is
// the narrow, handshake-only fallback that recovers it.
func TestSalvageHandshakeToolCall_RecoversFencedInvoke(t *testing.T) {
	in := "Here is the literal invoke block:\n\n```\n" +
		`<invoke name="session_handshake">` + "\n" +
		`<parameter name="code">2705</parameter>` + "\n" +
		"</invoke>\n```\n\nI've sent the block exactly as specified."
	calls, _, _ := salvageHandshakeToolCall(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "session_handshake", calls[0].Name)
	// A purely-numeric value is legitimately encoded as a JSON number here
	// (parametersToJSON, isJSONScalar), not a string -- see
	// TestHandshakeCodeArg_CoercesJSONNumber for why the session_handshake
	// tool's own arg extraction must handle that.
	assert.JSONEq(t, `{"code":2705}`, calls[0].Arguments)
}

// TestSalvageHandshakeToolCall_IgnoresOtherFencedTools ensures the fallback
// stays narrowly scoped: a fenced example for a DIFFERENT tool (the exact
// false-positive protectFencedCodeBlocks exists to prevent) must not be
// recovered by this handshake-only path either.
func TestSalvageHandshakeToolCall_IgnoresOtherFencedTools(t *testing.T) {
	in := "Here is how to call it:\n\n```\n" +
		`<invoke name="bash_exec"><parameter name="command">pwd</parameter></invoke>` +
		"\n```"
	calls, _, _ := salvageHandshakeToolCall(in)
	assert.Empty(t, calls, "only a fenced session_handshake call may be recovered here")
}

// TestSalvageHandshakeToolCall_UnfencedStillWorks ensures the fallback is not
// fence-only -- an ordinary, unfenced literal invoke still recovers.
func TestSalvageHandshakeToolCall_UnfencedStillWorks(t *testing.T) {
	in := `<invoke name="session_handshake"><parameter name="code">1234</parameter></invoke>`
	calls, _, _ := salvageHandshakeToolCall(in)
	require.Len(t, calls, 1)
	assert.Equal(t, "session_handshake", calls[0].Name)
}
