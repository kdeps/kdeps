// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0.
package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockResourceExec struct{}

func (m *mockResourceExec) Execute(_ *ExecutionContext, _ interface{}) (interface{}, error) {
	return nil, nil
}

func TestSetGetLoaderExecutor(t *testing.T) {
	reg := NewRegistry()
	exec := &mockResourceExec{}
	reg.SetLoaderExecutor(exec)
	assert.NotNil(t, reg.GetLoaderExecutor())
}

func TestSetGetVectorStoreExecutor(t *testing.T) {
	reg := NewRegistry()
	exec := &mockResourceExec{}
	reg.SetVectorStoreExecutor(exec)
	assert.NotNil(t, reg.GetVectorStoreExecutor())
}

func TestGetLoaderExecutor_Nil(t *testing.T) {
	reg := NewRegistry()
	assert.Nil(t, reg.GetLoaderExecutor())
}

func TestGetVectorStoreExecutor_Nil(t *testing.T) {
	reg := NewRegistry()
	assert.Nil(t, reg.GetVectorStoreExecutor())
}
