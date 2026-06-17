// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0.
//go:build !js

package executor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type mockResourceExec struct{}

func (m *mockResourceExec) Execute(_ *ExecutionContext, _ interface{}) (interface{}, error) {
	return nil, nil
}

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

func TestSetGetGitExecutor(t *testing.T) {
	reg := NewRegistry()
	exec := &mockResourceExec{}
	reg.SetGitExecutor(exec)
	assert.NotNil(t, reg.GetGitExecutor())
}

func TestSetGetCodeIntelligenceExecutor(t *testing.T) {
	reg := NewRegistry()
	exec := &mockResourceExec{}
	reg.SetCodeIntelligenceExecutor(exec)
	assert.NotNil(t, reg.GetCodeIntelligenceExecutor())
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

func TestSetGetTranscribeExecutor(t *testing.T) {
	reg := NewRegistry()
	exec := &mockResourceExec{}
	reg.SetTranscribeExecutor(exec)
	assert.NotNil(t, reg.GetTranscribeExecutor())
}

func TestGetGitExecutor_Nil(t *testing.T) {
	reg := NewRegistry()
	assert.Nil(t, reg.GetGitExecutor())
}

func TestGetLoaderExecutor_Nil(t *testing.T) {
	reg := NewRegistry()
	assert.Nil(t, reg.GetLoaderExecutor())
}

func TestGetVectorStoreExecutor_Nil(t *testing.T) {
	reg := NewRegistry()
	assert.Nil(t, reg.GetVectorStoreExecutor())
}

func TestGetTranscribeExecutor_Nil(t *testing.T) {
	reg := NewRegistry()
	assert.Nil(t, reg.GetTranscribeExecutor())
}

func TestGetCodeIntelligenceExecutor_Nil(t *testing.T) {
	reg := NewRegistry()
	assert.Nil(t, reg.GetCodeIntelligenceExecutor())
}
