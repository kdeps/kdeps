package main

import (
	"testing"

	"github.com/spf13/afero"
)

func TestSetupEnvironmentSuccess(t *testing.T) {
	fs := afero.NewMemMapFs()
	env, err := setupEnvironment(fs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env == nil {
		t.Fatalf("expected non-nil environment")
	}
}
