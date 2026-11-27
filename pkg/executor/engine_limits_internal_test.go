// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package executor

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
)

// waitExecutor holds each call for hold, or until the run context ends. The
// counters are pointers: the registry copies the executor for each run.
type waitExecutor struct {
	hold   time.Duration
	active *atomic.Int32
	peak   *atomic.Int32
}

func newWait(hold time.Duration) *waitExecutor {
	return &waitExecutor{hold: hold, active: new(atomic.Int32), peak: new(atomic.Int32)}
}

func (w *waitExecutor) Execute(ctx *ExecutionContext, _ interface{}) (interface{}, error) {
	n := w.active.Add(1)
	defer w.active.Add(-1)
	for {
		p := w.peak.Load()
		if n <= p || w.peak.CompareAndSwap(p, n) {
			break
		}
	}
	select {
	case <-time.After(w.hold):
		return "done", nil
	case <-ctx.Ctx.Done():
		return nil, ctx.Ctx.Err()
	}
}

func limitsEngine(t *testing.T, w *waitExecutor) *Engine {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	e := covTestEngine()
	reg := NewRegistry()
	reg.SetExecExecutor(w)
	e.SetRegistry(reg)
	return e
}

func limitsWorkflow(limits *domain.LimitsConfig) *domain.Workflow {
	wf := covWorkflow(&domain.Resource{ActionID: "r", Exec: &domain.ExecConfig{Command: "true"}})
	wf.Settings.Limits = limits
	return wf
}

func TestExecute_TimeoutStopsTheRun(t *testing.T) {
	e := limitsEngine(t, newWait(5*time.Second))
	start := time.Now()
	_, err := e.Execute(limitsWorkflow(&domain.LimitsConfig{Timeout: "100ms"}), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "run stopped after 100ms (deploy.limits.timeout)")
	assert.Less(t, time.Since(start), 3*time.Second)
}

func TestExecute_TimeoutNotReached(t *testing.T) {
	e := limitsEngine(t, newWait(10*time.Millisecond))
	out, err := e.Execute(limitsWorkflow(&domain.LimitsConfig{Timeout: "5s"}), nil)
	require.NoError(t, err)
	assert.Equal(t, "done", out)
}

func TestExecute_BadTimeout(t *testing.T) {
	for _, bad := range []string{"soon", "-1s", "0s"} {
		e := limitsEngine(t, newWait(0))
		_, err := e.Execute(limitsWorkflow(&domain.LimitsConfig{Timeout: bad}), nil)
		require.Error(t, err, bad)
		assert.Contains(t, err.Error(), "deploy.limits.timeout")
	}
}

func TestExecute_ParentCancelIsNotATimeout(t *testing.T) {
	e := limitsEngine(t, newWait(5*time.Second))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := e.Execute(limitsWorkflow(&domain.LimitsConfig{Timeout: "10s"}),
		&RequestContext{Ctx: ctx})
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "run stopped after")
}

func TestExecute_ConcurrencyQueuesRuns(t *testing.T) {
	w := newWait(time.Second)
	e := limitsEngine(t, w)
	wf := limitsWorkflow(&domain.LimitsConfig{Concurrency: 2})
	var wg sync.WaitGroup
	for i := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.Execute(wf, &RequestContext{SessionID: fmt.Sprint("s", i)})
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(2), w.peak.Load())
}

func TestExecute_WaitForSlotEndsWithTheRequest(t *testing.T) {
	w := newWait(2 * time.Second)
	e := limitsEngine(t, w)
	wf := limitsWorkflow(&domain.LimitsConfig{Concurrency: 1})
	go func() { _, _ = e.Execute(wf, nil) }()
	require.Eventually(t, func() bool { return w.active.Load() == 1 }, time.Second, 5*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := e.Execute(wf, &RequestContext{Ctx: ctx})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "waiting for a run slot (deploy.limits.concurrency)")
}

func TestExecute_NoLimits(t *testing.T) {
	w := newWait(time.Second)
	e := limitsEngine(t, w)
	wf := limitsWorkflow(nil)
	var wg sync.WaitGroup
	for i := range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := e.Execute(wf, &RequestContext{SessionID: fmt.Sprint("s", i)})
			assert.NoError(t, err)
		}()
	}
	wg.Wait()
	assert.Equal(t, int32(3), w.peak.Load())
}
