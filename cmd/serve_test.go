//go:build !js

package cmd

import (
	"testing"
)

func TestNewServeCmd_Flags(t *testing.T) {
	cmd := newServeCmd()
	if cmd == nil {
		t.Fatal("expected non-nil command")
	}
	if cmd.Use != "serve [workflow.yaml | agency.yaml]" {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}
	for _, flagName := range []string{"model", "backend", "base-url", "system"} {
		if cmd.Flags().Lookup(flagName) == nil {
			t.Errorf("expected flag --%s", flagName)
		}
	}
}

func TestNewServeCmd_RequiresOneArg(t *testing.T) {
	cmd := newServeCmd()
	// Zero args should error.
	if err := cmd.Args(cmd, []string{}); err == nil {
		t.Error("expected error for zero args")
	}
	// Two args should error.
	if err := cmd.Args(cmd, []string{"a", "b"}); err == nil {
		t.Error("expected error for two args")
	}
	// Exactly one arg should be accepted.
	if err := cmd.Args(cmd, []string{"workflow.yaml"}); err != nil {
		t.Errorf("unexpected error for one arg: %v", err)
	}
}
