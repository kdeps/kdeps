// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0.
//go:build !js

package executor

import (
	"testing"
)

type mockFileExecutor struct{}

func (m *mockFileExecutor) Execute(_ *ExecutionContext, _ interface{}) (interface{}, error) {
	return nil, nil
}

func TestSetGetFileExecutor(t *testing.T) {
	reg := NewRegistry()
	exec := &mockFileExecutor{}
	reg.SetFileExecutor(exec)
	got := reg.GetFileExecutor()
	if got == nil {
		t.Fatal("expected non-nil executor")
	}
}
