package resolver

import (
	"errors"
	"testing"
)

func TestHandleFileImports_DelegatesToInjectedFns(t *testing.T) {
	dr := &DependencyResolver{}

	calledPrepend := false
	calledPlaceholder := false
	argPath := "dummy.pkl"

	dr.PrependDynamicImportsFn = func(p string) error {
		if p != argPath {
			t.Errorf("expected path %s, got %s", argPath, p)
		}
		calledPrepend = true
		return nil
	}

	dr.AddPlaceholderImportsFn = func(p string) error {
		if p != argPath {
			t.Errorf("expected path %s, got %s", argPath, p)
		}
		calledPlaceholder = true
		return nil
	}

	if err := dr.handleFileImports(argPath); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !calledPrepend || !calledPlaceholder {
		t.Errorf("delegated functions were not called: prepend=%v placeholder=%v", calledPrepend, calledPlaceholder)
	}
}

func TestHandleFileImports_PropagatesError(t *testing.T) {
	dr := &DependencyResolver{}

	dr.PrependDynamicImportsFn = func(p string) error {
		return errors.New("boom")
	}

	if err := dr.handleFileImports("file.pkl"); err == nil {
		t.Fatal("expected error but got nil")
	}
}
