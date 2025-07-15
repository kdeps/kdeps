package resolver_test

import (
	"errors"
	"testing"
	"time"

	"github.com/apple/pkl-go/pkl"
	"github.com/kdeps/kdeps/pkg/logging"
	resolverpkg "github.com/kdeps/kdeps/pkg/resolver"
	pklRes "github.com/kdeps/schema/gen/resource"
)

// TestProcessResourceStep_Success verifies that the happy-path executes the handler
// and waits for the timestamp change without returning an error.
func TestProcessResourceStep_Success(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{Logger: logging.NewTestLogger(), DefaultTimeoutSec: -1}

	calledGet := false
	calledWait := false
	calledHandler := false

	dr.GetCurrentTimestampFn = func(resourceID, step string) (pkl.Duration, error) {
		calledGet = true
		return pkl.Duration{Value: 0, Unit: pkl.Second}, nil
	}
	dr.WaitForTimestampChangeFn = func(resourceID string, ts pkl.Duration, timeout time.Duration, step string) error {
		calledWait = true
		if timeout != 60*time.Second {
			t.Fatalf("expected default timeout 60s, got %v", timeout)
		}
		return nil
	}

	err := dr.ProcessResourceStep("resA", "exec", nil, func() error {
		calledHandler = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !calledGet || !calledWait || !calledHandler {
		t.Fatalf("expected all functions to be called; got get=%v wait=%v handler=%v", calledGet, calledWait, calledHandler)
	}
}

// TestProcessResourceStep_HandlerErr ensures that an error from the handler is propagated.
func TestProcessResourceStep_HandlerErr(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{Logger: logging.NewTestLogger(), DefaultTimeoutSec: -1}
	handlerErr := errors.New("boom")

	dr.GetCurrentTimestampFn = func(resourceID, step string) (pkl.Duration, error) {
		return pkl.Duration{Value: 0, Unit: pkl.Second}, nil
	}
	dr.WaitForTimestampChangeFn = func(resourceID string, ts pkl.Duration, timeout time.Duration, step string) error {
		return nil
	}

	err := dr.ProcessResourceStep("resA", "python", nil, func() error { return handlerErr })
	if err == nil || !errors.Is(err, handlerErr) {
		t.Fatalf("expected handler error to propagate, got %v", err)
	}
}

// TestProcessResourceStep_WaitErr ensures that an error from the wait helper is propagated.
func TestProcessResourceStep_WaitErr(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{Logger: logging.NewTestLogger(), DefaultTimeoutSec: -1}
	waitErr := errors.New("timeout")

	dr.GetCurrentTimestampFn = func(resourceID, step string) (pkl.Duration, error) {
		return pkl.Duration{Value: 0, Unit: pkl.Second}, nil
	}
	dr.WaitForTimestampChangeFn = func(resourceID string, ts pkl.Duration, timeout time.Duration, step string) error {
		return waitErr
	}

	err := dr.ProcessResourceStep("resA", "llm", nil, func() error { return nil })
	if err == nil || !errors.Is(err, waitErr) {
		t.Fatalf("expected wait error to propagate, got %v", err)
	}
}

// TestProcessResourceStep_CustomTimeout verifies that the timeout value from the Pkl duration is used.
func TestProcessResourceStep_CustomTimeout(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{Logger: logging.NewTestLogger(), DefaultTimeoutSec: -1}
	customDur := &pkl.Duration{Value: 5, Unit: pkl.Second} // 5 seconds

	dr.GetCurrentTimestampFn = func(resourceID, step string) (pkl.Duration, error) {
		return pkl.Duration{Value: 0, Unit: pkl.Second}, nil
	}

	waited := false
	dr.WaitForTimestampChangeFn = func(resourceID string, ts pkl.Duration, timeout time.Duration, step string) error {
		waited = true
		if timeout != 5*time.Second {
			t.Fatalf("expected timeout 5s, got %v", timeout)
		}
		return nil
	}

	if err := dr.ProcessResourceStep("resA", "exec", customDur, func() error { return nil }); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !waited {
		t.Fatal("WaitForTimestampChangeFn not invoked")
	}
}

// TestProcessRunBlock_NoRunBlock verifies that when Run is nil the function returns without error
// but still increments the FileRunCounter.
func TestProcessRunBlock_NoRunBlock(t *testing.T) {
	dr := &resolverpkg.DependencyResolver{
		Logger:         logging.NewTestLogger(),
		FileRunCounter: make(map[string]int),
		APIServerMode:  false,
	}

	resEntry := resolverpkg.ResourceNodeEntry{ActionID: "act1", File: "foo.pkl"}
	rsc := &pklRes.Resource{} // Run is nil by default

	proceed, err := dr.ProcessRunBlock(resEntry, rsc, "act1", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if proceed {
		t.Fatalf("expected proceed=false when Run is nil, got true")
	}
	if count := dr.FileRunCounter[resEntry.File]; count != 1 {
		t.Fatalf("expected FileRunCounter for %s to be 1, got %d", resEntry.File, count)
	}
}
