// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package docker_test

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/infra/docker"
)

// ---- shouldInstallUV (indirect via GenerateDockerfile) ----

func TestBuilder_ShouldInstallUV_PythonResource(t *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}

	workflow := &domain.Workflow{
		Resources: []*domain.Resource{
			{
				Metadata: domain.ResourceMetadata{ActionID: "py-res"},
				Run: domain.RunConfig{
					Python: &domain.PythonConfig{Script: "print('hello')"},
				},
			},
		},
	}

	dockerfile, err := builder.GenerateDockerfile(workflow)
	require.NoError(t, err)
	// When a Python resource exists, uv should be installed
	assert.Contains(t, dockerfile, "uv")
}

func TestBuilder_ShouldInstallUV_RequirementsFile(t *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}

	workflow := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				RequirementsFile: "requirements.txt",
			},
		},
	}

	dockerfile, err := builder.GenerateDockerfile(workflow)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "uv")
}

func TestBuilder_ShouldInstallUV_NoPython(t *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}

	// Workflow with no Python resources, no packages, no requirements file
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "simple", Version: "1.0.0"},
	}

	dockerfile, err := builder.GenerateDockerfile(workflow)
	require.NoError(t, err)
	// uv should not be mentioned in installation block when no Python usage
	assert.NotContains(t, dockerfile, "uv pip")
}

// ---- buildTemplateData (indirect via GenerateDockerfile) ----

func TestBuilder_GenerateDockerfile_DebianBase(t *testing.T) {
	builder := &docker.Builder{BaseOS: "debian"}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "debian-app", Version: "1.0.0"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
			},
		},
	}

	dockerfile, err := builder.GenerateDockerfile(workflow)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "debian:latest")
}

func TestBuilder_GenerateDockerfile_UbuntuBase(t *testing.T) {
	builder := &docker.Builder{BaseOS: "ubuntu"}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "ubuntu-app", Version: "1.0.0"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
			},
		},
	}

	dockerfile, err := builder.GenerateDockerfile(workflow)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "ubuntu:latest")
}

func TestBuilder_GenerateDockerfile_OllamaUbuntu(t *testing.T) {
	builder := &docker.Builder{BaseOS: "ubuntu"}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "ollama-app", Version: "1.0.0"},
		Resources: []*domain.Resource{
			{
				Metadata: domain.ResourceMetadata{ActionID: "chat-res"},
				Run: domain.RunConfig{
					Chat: &domain.ChatConfig{
						Model:   "llama3.2:1b",
						Backend: "ollama",
						Role:    "user",
						Prompt:  "hello",
					},
				},
			},
		},
	}

	dockerfile, err := builder.GenerateDockerfile(workflow)
	require.NoError(t, err)
	// For Ubuntu+Ollama, should use official ollama image
	assert.Contains(t, dockerfile, "ollama/ollama:latest")
}

func TestBuilder_GenerateDockerfile_OllamaAlpine(t *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "alpine-ollama", Version: "1.0.0"},
		Resources: []*domain.Resource{
			{
				Metadata: domain.ResourceMetadata{ActionID: "chat-res"},
				Run: domain.RunConfig{
					Chat: &domain.ChatConfig{
						Model:   "llama3.2:1b",
						Backend: "ollama",
						Role:    "user",
						Prompt:  "hello",
					},
				},
			},
		},
	}

	dockerfile, err := builder.GenerateDockerfile(workflow)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "alpine/ollama")
}

// ---- CreateBuildContext (exercises generateEntrypoint + generateSupervisord) ----

func TestBuilder_CreateBuildContext_Basic(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a workflow.yaml so addWorkflowFilesToContext can find it
	workflowYAML := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(workflowYAML, []byte("name: test\n"), 0644))

	t.Chdir(tmpDir)

	builder := &docker.Builder{BaseOS: "alpine"}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
			},
		},
	}

	dockerfile := "FROM alpine:latest\nWORKDIR /app\n"
	reader, err := builder.CreateBuildContext(workflow, dockerfile)
	require.NoError(t, err)
	require.NotNil(t, reader)

	// Validate tar contents
	tr := tar.NewReader(reader)
	files := map[string]bool{}
	for {
		hdr, nextErr := tr.Next()
		if nextErr == io.EOF {
			break
		}
		require.NoError(t, nextErr)
		files[hdr.Name] = true
	}

	assert.True(t, files["Dockerfile"], "tar should contain Dockerfile")
	assert.True(t, files["entrypoint.sh"], "tar should contain entrypoint.sh")
	assert.True(t, files["supervisord.conf"], "tar should contain supervisord.conf")
}

func TestBuilder_CreateBuildContext_WithAPIServer(t *testing.T) {
	tmpDir := t.TempDir()

	workflowYAML := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(workflowYAML, []byte("name: api-test\n"), 0644))

	t.Chdir(tmpDir)

	builder := &docker.Builder{BaseOS: "alpine"}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "api-test", Version: "1.0.0"},
		Settings: domain.WorkflowSettings{
			APIServerMode: true,
			PortNum:       16395,
			APIServer:     &domain.APIServerConfig{},
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
			},
		},
	}

	dockerfile, genErr := builder.GenerateDockerfile(workflow)
	require.NoError(t, genErr)

	reader, err := builder.CreateBuildContext(workflow, dockerfile)
	require.NoError(t, err)
	require.NotNil(t, reader)

	// Read the entrypoint.sh from the tar
	tr := tar.NewReader(reader)
	for {
		hdr, tarErr := tr.Next()
		if tarErr == io.EOF {
			break
		}
		require.NoError(t, tarErr)
		if hdr.Name == "entrypoint.sh" {
			content, readErr := io.ReadAll(tr)
			require.NoError(t, readErr)
			assert.True(t, len(content) > 0, "entrypoint.sh should not be empty")
			break
		}
	}
}

func TestBuilder_CreateBuildContext_SupervisordContent(t *testing.T) {
	tmpDir := t.TempDir()

	workflowYAML := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(workflowYAML, []byte("name: sup-test\n"), 0644))

	t.Chdir(tmpDir)

	builder := &docker.Builder{BaseOS: "alpine"}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "sup-test", Version: "1.0.0"},
	}

	dockerfile := "FROM alpine:latest\n"
	reader, err := builder.CreateBuildContext(workflow, dockerfile)
	require.NoError(t, err)

	tr := tar.NewReader(reader)
	for {
		hdr, tarErr := tr.Next()
		if tarErr == io.EOF {
			break
		}
		require.NoError(t, tarErr)
		if hdr.Name == "supervisord.conf" {
			content, readErr := io.ReadAll(tr)
			require.NoError(t, readErr)
			assert.True(t, strings.Contains(string(content), "supervisord") ||
				strings.Contains(string(content), "[supervisord]") ||
				len(content) > 0,
				"supervisord.conf should have content")
			break
		}
	}
}

// ---- Build with nil/empty workflow ----

func TestBuilder_Build_NilWorkflow(t *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}

	_, err := builder.Build(nil, "output.tar", false)
	assert.Error(t, err)
}

func TestBuilder_Build_EmptyName(_ *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}

	workflow := &domain.Workflow{
		// No name set
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
			},
		},
	}

	// Build will fail but should not panic - it may fail on Docker not available
	// or on missing workflow name (depending on validation)
	_, err := builder.Build(workflow, "output.tar", false)
	// Just ensure it doesn't panic
	_ = err
}
