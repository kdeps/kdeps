// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package executor_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func TestNewRegistry(t *testing.T) {
	registry := executor.NewRegistry()
	assert.NotNil(t, registry)
}

func TestExecutorRegistry_Getters(t *testing.T) {
	registry := executor.NewRegistry()

	// Test that getters return nil initially (lazy initialization)
	assert.Nil(t, registry.GetLLMExecutor())
	assert.Nil(t, registry.GetHTTPExecutor())
	assert.Nil(t, registry.GetSQLExecutor())
	assert.Nil(t, registry.GetPythonExecutor())
	assert.Nil(t, registry.GetExecExecutor())
}

func TestExecutorRegistry_Setters(t *testing.T) {
	registry := executor.NewRegistry()

	// Create mock executors (just need non-nil values for testing)
	mockLLM := &mockExecutor{}
	mockHTTP := &mockExecutor{}
	mockSQL := &mockExecutor{}
	mockPython := &mockExecutor{}
	mockExec := &mockExecutor{}

	// Set executors
	registry.SetLLMExecutor(mockLLM)
	registry.SetHTTPExecutor(mockHTTP)
	registry.SetSQLExecutor(mockSQL)
	registry.SetPythonExecutor(mockPython)
	registry.SetExecExecutor(mockExec)

	// Verify getters return the set values
	assert.Equal(t, mockLLM, registry.GetLLMExecutor())
	assert.Equal(t, mockHTTP, registry.GetHTTPExecutor())
	assert.Equal(t, mockSQL, registry.GetSQLExecutor())
	assert.Equal(t, mockPython, registry.GetPythonExecutor())
	assert.Equal(t, mockExec, registry.GetExecExecutor())
}

// mockExecutor is a simple mock implementation for testing.
type mockExecutor struct{}

func (m *mockExecutor) Execute(_ *executor.ExecutionContext, _ interface{}) (interface{}, error) {
	return "mock result", nil
}
