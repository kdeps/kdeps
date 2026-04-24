// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package cmd

import (
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestParseOllamaURL_Internal(t *testing.T) {
	tests := []struct {
		name         string
		ollamaURL    string
		expectedHost string
		expectedPort int
	}{
		{"empty", "", "localhost", 11434},
		{"host", "1.2.3.4", "1.2.3.4", 11434},
		{"host-port", "1.2.3.4:5678", "1.2.3.4", 5678},
		{"http", "http://ollama:11434", "ollama", 11434},
		{"https", "https://ollama", "ollama", 11434},
		{"invalid-port", "localhost:abc", "localhost", 11434},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host, port := ParseOllamaURL(tt.ollamaURL)
			assert.Equal(t, tt.expectedHost, host)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

func TestWorkflowNeedsOllama_Internal(t *testing.T) {
	tests := []struct {
		name     string
		workflow *domain.Workflow
		expected bool
	}{
		{
			"no resources",
			&domain.Workflow{Resources: []*domain.Resource{}},
			false,
		},
		{
			"ollama resource",
			&domain.Workflow{
				Resources: []*domain.Resource{
					{Run: domain.RunConfig{Chat: &domain.ChatConfig{Backend: "ollama"}}},
				},
			},
			true,
		},
		{
			"default backend resource",
			&domain.Workflow{
				Resources: []*domain.Resource{
					{Run: domain.RunConfig{Chat: &domain.ChatConfig{Backend: ""}}},
				},
			},
			true,
		},
		{
			"non-ollama resource",
			&domain.Workflow{
				Resources: []*domain.Resource{
					{Run: domain.RunConfig{Chat: &domain.ChatConfig{Backend: "openai"}}},
				},
			},
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, workflowNeedsOllama(tt.workflow))
		})
	}
}

func TestIsOllamaRunning_Internal(t *testing.T) {
	// We can test the negative case easily
	assert.False(t, IsOllamaRunning("127.0.0.1", 0))

	// Test positive case with a dummy listener
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()

	port := l.Addr().(*net.TCPAddr).Port
	// Since it's not actually Ollama responding with 200 OK to /, it should return false
	assert.False(t, IsOllamaRunning("127.0.0.1", port))
}

func TestResolveWorkflowPath_Internal(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("absolute path file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "workflow.yaml")
		_ = os.WriteFile(path, []byte("test"), 0644)
		resolved, cleanup, err := resolveWorkflowPath(path)
		require.NoError(t, err)
		assert.Equal(t, path, resolved)
		assert.Nil(t, cleanup)
	})

	t.Run("directory", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "mydir")
		_ = os.Mkdir(dir, 0755)
		path := filepath.Join(dir, "workflow.yaml")
		_ = os.WriteFile(path, []byte("test"), 0644)
		resolved, cleanup, err := resolveWorkflowPath(dir)
		require.NoError(t, err)
		assert.Equal(t, path, resolved)
		assert.Nil(t, cleanup)
	})

	t.Run("package", func(t *testing.T) {
		// This tests the ExtractPackage path
		// We'll just mock a .kdeps file
		pkg := filepath.Join(tmpDir, "test.kdeps")
		_ = os.WriteFile(pkg, []byte("invalid content"), 0644)
		_, _, err := resolveWorkflowPath(pkg)
		assert.Error(t, err) // Should fail to extract
	})
}

func TestEnsureOllamaRunning_Internal(t *testing.T) {
	// This function is complex, let's test some branches

	t.Run("already running", func(_ *testing.T) {
		// Mock IsOllamaRunning to return true if possible?
		// Not easy without mocking the network.
	})

	t.Run("command not found", func(t *testing.T) {
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", "")
		defer os.Setenv("PATH", oldPath)

		err := ensureOllamaRunning("http://localhost:11434")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ollama not found in PATH")
	})
}

func TestPrintIORequirements_Internal(_ *testing.T) {
	// Just ensure it doesn't panic with various configs
	w := &domain.Workflow{
		Resources: []*domain.Resource{
			{Run: domain.RunConfig{APIResponse: &domain.APIResponseConfig{}}},
		},
	}
	printIORequirements(w)
}

func TestFindWorkflowFile_Internal(t *testing.T) {
	tmpDir := t.TempDir()

	path := FindWorkflowFile(tmpDir)
	assert.Equal(t, "", path)

	wPath := filepath.Join(tmpDir, "workflow.yaml")
	_ = os.WriteFile(wPath, []byte("test"), 0644)
	path = FindWorkflowFile(tmpDir)
	assert.Equal(t, wPath, path)
}
