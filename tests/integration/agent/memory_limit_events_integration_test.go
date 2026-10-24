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

package agent_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/agent"
)

// memory_search's caps and the auto-capture cap are events every preset
// tunes: applying one writes its values and the registry reads them back.
func TestPresets_TuneMemoryLimitEvents(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("USERPROFILE", t.TempDir())
	cases := map[string][3]int{
		"frugal":   {10, 300, 200},
		"balanced": {20, 500, 300},
		"thorough": {40, 1000, 600},
	}
	for preset, want := range cases {
		require.NoError(t, agent.ApplyPreset(preset), preset)
		search, ok := agent.EventByName("memory-search-limit")
		require.True(t, ok)
		value, _ := agent.EventByName("memory-value-limit")
		capture, _ := agent.EventByName("memory-capture-limit")
		assert.Equal(t, want, [3]int{search.Items, value.On.Bytes, capture.On.Bytes}, preset)
	}
	require.NoError(t, agent.ApplyPreset("balanced"))
}
