// Copyright 2026 kdeps KVK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

//go:build !js

package cmd

import (
	"testing"

	"github.com/kdeps/kdeps/v2/pkg/tui"
)

func TestExtractToolNamesEmpty(t *testing.T) {
	if names := extractToolNames(nil); len(names) != 0 {
		t.Fatalf("nil: %v", names)
	}
}

func TestNamesEqual(t *testing.T) {
	if !namesEqual(nil, nil) {
		t.Fatal("nil")
	}
	a := []tui.Item{{Name: "x"}}
	b := []tui.Item{{Name: "x"}}
	c := []tui.Item{{Name: "y"}}
	if !namesEqual(a, b) {
		t.Fatal("equal")
	}
	if namesEqual(a, c) {
		t.Fatal("not equal")
	}
	if namesEqual(a, nil) {
		t.Fatal("len mismatch")
	}
}

func TestSelectionsEqual(t *testing.T) {
	s1 := tui.Selection{}
	s2 := tui.Selection{}
	if !selectionsEqual(s1, s2) {
		t.Fatal("empty")
	}
}
