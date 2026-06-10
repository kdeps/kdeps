// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

//go:build !js

package iso

import (
	"errors"
	"strings"
	"testing"
)

func TestMarshalConfig_MarshalError(t *testing.T) {
	origMarshal := yamlMarshal
	yamlMarshal = func(_ interface{}) ([]byte, error) {
		return nil, errors.New("simulated marshal failure")
	}
	defer func() { yamlMarshal = origMarshal }()

	config := &LinuxKitConfig{}
	_, err := MarshalConfig(config)
	if err == nil {
		t.Fatal("expected error from MarshalConfig, got nil")
	}
	if !strings.Contains(err.Error(), "failed to marshal LinuxKit config") {
		t.Fatalf("expected 'failed to marshal LinuxKit config' error, got: %v", err)
	}
	if !strings.Contains(err.Error(), "simulated marshal failure") {
		t.Fatalf("expected wrapped 'simulated marshal failure' error, got: %v", err)
	}
}
