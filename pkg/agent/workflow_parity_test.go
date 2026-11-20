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
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/plancouncil"
)

func TestWithTuro_WorkflowChoiceWinsOverREPL(t *testing.T) {
	resetTuroState(t)
	turoOnce.Do(func() {})
	old := turoEnabled
	turoEnabled = true
	t.Cleanup(func() { turoEnabled = old })

	SetTuroRuntimeOff(true)
	assert.False(t, turoActive(context.Background()), "the REPL's /turo off")
	assert.True(t, turoActive(withTuro(context.Background(), true)), "harness.turo: true")
	SetTuroRuntimeOff(false)
	assert.False(t, turoActive(withTuro(context.Background(), false)), "a workflow chat without harness.turo")

	turoEnabled = false
	assert.False(t, turoActive(withTuro(context.Background(), true)), "turo is not installed")
}

func TestConfigureCouncil_ReadsTheHarness(t *testing.T) {
	configureCouncil()
	s := plancouncil.Configured()
	require.NotNil(t, s.Members)
	assert.NotEmpty(t, s.Members(), "plan-council-members lists the seats")
	assert.Positive(t, s.Rounds())
	assert.Positive(t, s.MaxOrders())
	note := s.Note("1. #1 step\n")
	assert.Contains(t, note, "1. #1 step")
	assert.NotContains(t, note, "plan_update", "a workflow chat has no progress file")
}
