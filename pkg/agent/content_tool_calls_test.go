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
