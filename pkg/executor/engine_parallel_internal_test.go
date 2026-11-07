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

package executor

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func chatRes(id, prompt string, requires ...string) *domain.Resource {
	return &domain.Resource{
		ActionID: id,
		Requires: requires,
		Chat:     &domain.ChatConfig{Model: "m", Prompt: prompt},
	}
}

func withParallelCap(t *testing.T, n int) {
	t.Helper()
	prev := maxParallelResources.Load()
	SetMaxParallelResources(n)
	t.Cleanup(func() { maxParallelResources.Store(prev) })
}

// peakExecutor records how many calls overlap and fails prompts named in fail.
type peakExecutor struct {
	mu      sync.Mutex
	active  int
	peak    int
	order   []string
	fail    string
	panicOn string
	want    int // hold each call until this many overlap (or 2s); 0 holds 40ms
}

func (p *peakExecutor) hold() {
	if p.want == 0 {
		time.Sleep(40 * time.Millisecond)
		return
	}
	for deadline := time.Now().Add(2 * time.Second); time.Now().Before(deadline); {
		p.mu.Lock()
		reached := p.peak >= p.want
		p.mu.Unlock()
		if reached {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func (p *peakExecutor) Execute(_ *ExecutionContext, cfg interface{}) (interface{}, error) {
	prompt := cfg.(*domain.ChatConfig).Prompt
	p.mu.Lock()
	p.active++
	p.peak = max(p.peak, p.active)
	p.order = append(p.order, prompt)
	p.mu.Unlock()
	p.hold()
	p.mu.Lock()
	p.active--
	p.mu.Unlock()
	if prompt == p.panicOn {
		panic("boom")
	}
	if prompt == p.fail {
		return nil, errors.New("failed " + prompt)
	}
	return prompt, nil
}

func runPeak(t *testing.T, p *peakExecutor, resources ...*domain.Resource) error {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetLLMExecutor(p)
	e.SetRegistry(reg)
	_, err := e.Execute(covWorkflow(resources...), nil)
	return err
}

func TestAccessOf(t *testing.T) {
	a := accessOf(chatRes("x", "{{ set('k', get('in')) }}"))
	assert.False(t, a.barrier)
	assert.True(t, a.writes["k"])
	assert.True(t, a.reads["in"])

	for name, r := range map[string]*domain.Resource{
		"dynamic key":  chatRes("x", "{{ get(name) }}"),
		"item storage": chatRes("x", "{{ set('k', 1, 'item') }}"),
		"memory func":  chatRes("x", "{{ memory_save('k') }}"),
		"session map":  chatRes("x", "{{ session() }}"),
		"dynamic out":  chatRes("x", "{{ output(id) }}"),
		"items":        {ActionID: "x", Items: []string{"a"}, Chat: &domain.ChatConfig{}},
		"loop":         {ActionID: "x", Loop: &domain.LoopConfig{While: "false"}},
		"exec":         {ActionID: "x", Exec: &domain.ExecConfig{}},
		"response":     {ActionID: "x", APIResponse: &domain.APIResponseConfig{}},
		"filters":      {ActionID: "x", Validations: &domain.ValidationsConfig{Headers: []string{"h"}}},
		"chat tools": {ActionID: "x", Chat: &domain.ChatConfig{
			Tools: []domain.Tool{{Name: "t"}},
		}},
	} {
		assert.True(t, accessOf(r).barrier, name)
	}
}

func TestResourceDeps(t *testing.T) {
	deps := resourceDeps([]*domain.Resource{
		chatRes("w", "{{ set('k', 1) }}"),             // 0
		chatRes("free", "hello"),                      // 1
		chatRes("rd", "{{ get('k') }}"),               // 2 reads w's key
		chatRes("out", "{{ output('free') }}"),        // 3 reads free's output
		chatRes("req", "x", "w"),                      // 4 requires w
		chatRes("ww", "{{ set('k', 2) }}"),            // 5 rewrites k after w and rd
		{ActionID: "bar", Exec: &domain.ExecConfig{}}, // 6 barrier
		chatRes("after", "y"),                         // 7 after the barrier
	})
	assert.Empty(t, deps[1])
	assert.Equal(t, []int{0}, deps[2])
	assert.Equal(t, []int{1}, deps[3])
	assert.Equal(t, []int{0}, deps[4])
	assert.Equal(t, []int{0, 2}, deps[5])
	assert.Equal(t, []int{0, 1, 2, 3, 4, 5}, deps[6])
	assert.Equal(t, []int{6}, deps[7])
}

func TestRunResources_IndependentRunTogether(t *testing.T) {
	withParallelCap(t, 4)
	p := &peakExecutor{want: 3}
	require.NoError(t, runPeak(t, p,
		chatRes("a", "a"), chatRes("b", "b"), chatRes("c", "c"),
		chatRes("r", "r", "a", "b", "c"),
	))
	assert.Equal(t, 3, p.peak)
	assert.Equal(t, "r", p.order[len(p.order)-1], "r waits for its requires")
}

func TestRunResources_CapOneRunsInOrder(t *testing.T) {
	withParallelCap(t, 1)
	p := &peakExecutor{}
	require.NoError(t, runPeak(t, p,
		chatRes("a", "a"), chatRes("b", "b"), chatRes("r", "r", "a", "b"),
	))
	assert.Equal(t, 1, p.peak)
	assert.Equal(t, []string{"a", "b", "r"}, p.order)
}

func TestRunResources_CapLimitsOverlap(t *testing.T) {
	withParallelCap(t, 2)
	p := &peakExecutor{want: 2}
	require.NoError(t, runPeak(t, p,
		chatRes("a", "a"), chatRes("b", "b"), chatRes("c", "c"), chatRes("d", "d"),
		chatRes("r", "r", "a", "b", "c", "d"),
	))
	assert.Equal(t, 2, p.peak)
}

func TestRunResources_SharedKeyKeepsOrder(t *testing.T) {
	withParallelCap(t, 4)
	p := &peakExecutor{}
	require.NoError(t, runPeak(t, p,
		chatRes("a", "{{ set('k', 1) }}"), chatRes("b", "{{ get('k') }}"),
		chatRes("r", "r", "a", "b"),
	))
	assert.Equal(t, 1, p.peak)
}

func TestRunResources_ErrorStopsAndReports(t *testing.T) {
	withParallelCap(t, 4)
	p := &peakExecutor{fail: "b"}
	err := runPeak(t, p, chatRes("a", "a"), chatRes("b", "b"), chatRes("r", "r", "a", "b"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed b")
	assert.NotContains(t, strings.Join(p.order, ","), "r")
}

func TestRunResources_PanicReachesCaller(t *testing.T) {
	withParallelCap(t, 4)
	p := &peakExecutor{panicOn: "a"}
	assert.Panics(t, func() {
		_ = runPeak(t, p, chatRes("a", "a"), chatRes("b", "b"), chatRes("r", "r", "a", "b"))
	})
}

func TestParallelCapDefault(t *testing.T) {
	withParallelCap(t, 0)
	assert.Equal(t, defaultParallelResources, parallelCap())
}
