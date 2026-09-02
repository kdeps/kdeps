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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeLoopArtifacts(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "memory marker on its own line is removed",
			in:   "The build passes.\n[MEMORY: build] passing on main\nAll tests green.",
			want: "The build passes.\nAll tests green.",
		},
		{
			name: "trailing memory marker is removed",
			in:   "Done.\n\n[MEMORY: progress] finished the refactor",
			want: "Done.",
		},
		{
			name: "inline memory marker is trimmed to end of line",
			in:   "Fixed it. [MEMORY: fix] done\nNext step follows.",
			want: "Fixed it.\nNext step follows.",
		},
		{
			name: "echoed goal directive lines are removed",
			in:   "GOAL: ship the parser\nACTIVE TASK 2 of 3: write tests\nHere is the actual answer.",
			want: "Here is the actual answer.",
		},
		{
			name: "plain prose is untouched",
			in:   "The memory usage looks fine. My goal here is clarity.",
			want: "The memory usage looks fine. My goal here is clarity.",
		},
		{
			name: "no markers, multiple newlines preserved as paragraph break",
			in:   "Para one.\n\nPara two.",
			want: "Para one.\n\nPara two.",
		},
		{
			name: "empty stays empty",
			in:   "",
			want: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, sanitizeLoopArtifacts(c.in))
		})
	}
}

func TestSanitizeLoopArtifacts_KeepsMemoryWordInProse(t *testing.T) {
	// "GOAL:" only triggers the directive strip when the distinctive header text
	// is present; a sentence that merely contains the word "goal" is safe.
	in := "Memory is the bridge between models. The goal: keep it in sync."
	assert.Equal(t, in, sanitizeLoopArtifacts(in))
}
