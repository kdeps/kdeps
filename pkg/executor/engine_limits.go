// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package executor

import (
	"context"
	"errors"
	"fmt"

	"github.com/kdeps/kdeps/v3/pkg/domain"
)

// startRun applies deploy.limits to one run. It waits for a free slot when
// the workflow sets concurrency, then starts the timeout. release frees the
// slot and stops the timer.
func (e *Engine) startRun(
	parent context.Context,
	workflow *domain.Workflow,
) (context.Context, func(), error) {
	limits := workflow.Settings.Limits
	if limits == nil {
		return parent, func() {}, nil
	}
	timeout, err := limits.TimeoutDuration()
	if err != nil {
		return nil, nil, err
	}
	freeSlot, err := e.takeRunSlot(parent, limits.Concurrency)
	if err != nil {
		return nil, nil, err
	}
	if timeout == 0 {
		return parent, freeSlot, nil
	}
	ctx, cancel := context.WithTimeoutCause(parent, timeout, errRunTimeout)
	return ctx, func() { cancel(); freeSlot() }, nil
}

// errRunTimeout is the cause of a run context that deploy.limits.timeout ended.
var errRunTimeout = errors.New("deploy.limits.timeout")

// takeRunSlot waits for one of the concurrency slots of this engine.
func (e *Engine) takeRunSlot(ctx context.Context, concurrency int) (func(), error) {
	if concurrency <= 0 {
		return func() {}, nil
	}
	e.runSlotsOnce.Do(func() { e.runSlots = make(chan struct{}, concurrency) })
	select {
	case e.runSlots <- struct{}{}:
		return func() { <-e.runSlots }, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("waiting for a run slot (deploy.limits.concurrency): %w", ctx.Err())
	}
}

// runLimitError names deploy.limits.timeout when the run stopped on it.
func runLimitError(ctx context.Context, workflow *domain.Workflow, err error) error {
	if err == nil || !errors.Is(context.Cause(ctx), errRunTimeout) {
		return err
	}
	return fmt.Errorf("run stopped after %s (deploy.limits.timeout): %w",
		workflow.Settings.Limits.Timeout, err)
}
