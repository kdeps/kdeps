// Copyright 2026 kdeps KVK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

//go:build !js

package cmd

import "testing"

func TestDefaultName(t *testing.T) {
	if got := defaultName("svc", "ollama"); got != "svc" {
		t.Fatalf("got %q", got)
	}
	if got := defaultName("", "ollama"); got != "kdeps-llm-ollama" {
		t.Fatalf("got %q", got)
	}
}

func TestIsInteractiveTTY(_ *testing.T) {
	// CI / pipes: typically false; just call
	_ = isInteractiveTTY()
}
