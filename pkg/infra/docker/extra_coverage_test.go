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
	"net/http"
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
				ActionID: "py-res",
				Python:   &domain.PythonConfig{Script: "print('hello')"},
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
				ActionID: "chat-res",
				Chat: &domain.ChatConfig{
					Model:   "llama3.2:1b",
					Backend: "ollama",
					Role:    "user",
					Prompt:  "hello",
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
				ActionID: "chat-res",
				Chat: &domain.ChatConfig{
					Model:   "llama3.2:1b",
					Backend: "ollama",
					Role:    "user",
					Prompt:  "hello",
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
			APIServer: &domain.APIServerConfig{PortNum: 16395},
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

// TestBuilder_CreateBuildContext_ResourcesWithFile verifies that addDirectoryToTar
// (builder.go:629) correctly walks regular files inside resources/ and data/
// directories and adds them to the build context tar.
func TestBuilder_CreateBuildContext_ResourcesWithFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create workflow.yaml (required by CreateBuildContext)
	workflowYAML := filepath.Join(tmpDir, "workflow.yaml")
	require.NoError(t, os.WriteFile(workflowYAML, []byte("name: resources-test\n"), 0644))

	// Create resources/ directory with a regular file
	resourcesDir := filepath.Join(tmpDir, "resources")
	require.NoError(t, os.MkdirAll(resourcesDir, 0755))
	resFile := filepath.Join(resourcesDir, "test.txt")
	require.NoError(t, os.WriteFile(resFile, []byte("resources content"), 0644))

	// Create data/ directory with a regular file
	dataDir := filepath.Join(tmpDir, "data")
	require.NoError(t, os.MkdirAll(dataDir, 0755))
	dataFile := filepath.Join(dataDir, "config.json")
	require.NoError(t, os.WriteFile(dataFile, []byte(`{"key": "value"}`), 0644))

	t.Chdir(tmpDir)

	builder := &docker.Builder{BaseOS: "alpine"}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "resources-test", Version: "1.0.0"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
			},
		},
	}

	dockerfile := "FROM alpine:latest\n"
	reader, err := builder.CreateBuildContext(workflow, dockerfile)
	require.NoError(t, err)
	require.NotNil(t, reader)

	// Read tar and collect entries
	tr := tar.NewReader(reader)
	entries := make(map[string]string)
	for {
		hdr, tarErr := tr.Next()
		if tarErr == io.EOF {
			break
		}
		require.NoError(t, tarErr)
		raw, readErr := io.ReadAll(tr)
		require.NoError(t, readErr)
		entries[hdr.Name] = string(raw)
	}

	// Standard build context files must be present
	assert.Contains(t, entries, "Dockerfile")
	assert.Contains(t, entries, "entrypoint.sh")
	assert.Contains(t, entries, "supervisord.conf")

	// resources/test.txt must be present with correct content
	assert.Equal(t, "resources content", entries["resources/test.txt"],
		"resources/test.txt should be in tar with correct content")

	// data/config.json must be present with correct content
	assert.Equal(t, `{"key": "value"}`, entries["data/config.json"],
		"data/config.json should be in tar with correct content")
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

// ---------------------------------------------------------------------------
// Build prune output paths (builder.go lines 259-264) and workflow BaseOS
// override (builder.go line 230)
// ---------------------------------------------------------------------------

func TestBuilder_Build_WorkflowBaseOSOverride(t *testing.T) {
	mockClient := newMockDockerClient(t, func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/build") {
			return bytesResponse("application/x-ndjson",
				[]byte(`{"stream":"Successfully built"}`+"\n")), nil
		}
		if strings.Contains(r.URL.Path, "/images/prune") {
			return jsonResponse(http.StatusOK, map[string]any{"SpaceReclaimed": uint64(0)}), nil
		}
		return jsonResponse(http.StatusNotFound, map[string]string{"message": "unexpected"}), nil
	})

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "workflow.yaml"),
		[]byte("metadata:\n  name: override-test\n  version: 1.0.0\n"),
		0644,
	))
	t.Chdir(tmpDir)

	builder := &docker.Builder{Client: mockClient}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "override-test",
			Version: "1.0.0",
		},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
				BaseOS:        "ubuntu",
			},
		},
	}

	_, err := builder.Build(workflow, "output.tar", false)
	require.NoError(t, err)
	assert.Equal(t, "ubuntu", builder.BaseOS,
		"Build should apply workflow BaseOS when builder BaseOS is empty")
}

func TestBuilder_Build_PruneSuccessWithSpace(t *testing.T) {
	mockClient := newMockDockerClient(t, func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/build") {
			return bytesResponse("application/x-ndjson",
				[]byte(`{"stream":"Successfully built"}`+"\n")), nil
		}
		if strings.Contains(r.URL.Path, "/images/prune") {
			return jsonResponse(http.StatusOK, map[string]any{"SpaceReclaimed": uint64(5242880)}), nil
		}
		return jsonResponse(http.StatusNotFound, map[string]string{"message": "unexpected"}), nil
	})

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "workflow.yaml"),
		[]byte("metadata:\n  name: prune-space-test\n  version: 1.0.0\n"),
		0644,
	))
	t.Chdir(tmpDir)

	builder := &docker.Builder{BaseOS: "alpine", Client: mockClient}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "prune-space-test",
			Version: "1.0.0",
		},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
			},
		},
	}

	imageName, err := builder.Build(workflow, "output.tar", false)
	require.NoError(t, err)
	assert.Contains(t, imageName, "prune-space-test")
}

func TestBuilder_Build_PruneError(t *testing.T) {
	mockClient := newMockDockerClient(t, func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/build") {
			return bytesResponse("application/x-ndjson",
				[]byte(`{"stream":"Successfully built"}`+"\n")), nil
		}
		if strings.Contains(r.URL.Path, "/images/prune") {
			return jsonResponse(http.StatusInternalServerError, map[string]string{"message": "prune failed"}), nil
		}
		return jsonResponse(http.StatusNotFound, map[string]string{"message": "unexpected"}), nil
	})

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "workflow.yaml"),
		[]byte("metadata:\n  name: prune-error-test\n  version: 1.0.0\n"),
		0644,
	))
	t.Chdir(tmpDir)

	builder := &docker.Builder{BaseOS: "alpine", Client: mockClient}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "prune-error-test",
			Version: "1.0.0",
		},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
			},
		},
	}

	imageName, err := builder.Build(workflow, "output.tar", false)
	require.NoError(t, err)
	assert.Contains(t, imageName, "prune-error-test")
}

func TestBuilder_Build_PruneZeroSpace(t *testing.T) {
	mockClient := newMockDockerClient(t, func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/build") {
			return bytesResponse("application/x-ndjson",
				[]byte(`{"stream":"Successfully built"}`+"\n")), nil
		}
		if strings.Contains(r.URL.Path, "/images/prune") {
			return jsonResponse(http.StatusOK, map[string]any{"SpaceReclaimed": uint64(0)}), nil
		}
		return jsonResponse(http.StatusNotFound, map[string]string{"message": "unexpected"}), nil
	})

	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(tmpDir, "workflow.yaml"),
		[]byte("metadata:\n  name: prune-zero-test\n  version: 1.0.0\n"),
		0644,
	))
	t.Chdir(tmpDir)

	builder := &docker.Builder{BaseOS: "alpine", Client: mockClient}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "prune-zero-test",
			Version: "1.0.0",
		},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				PythonVersion: "3.12",
			},
		},
	}

	imageName, err := builder.Build(workflow, "output.tar", false)
	require.NoError(t, err)
	assert.Contains(t, imageName, "prune-zero-test")
}

// ---------------------------------------------------------------------------
// RunContainer mock tests (client.go lines 172-176)
// ---------------------------------------------------------------------------

func TestClient_RunContainer_Mock_Success(t *testing.T) {
	c := newMockDockerClient(t, func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/containers/create") {
			return jsonResponse(http.StatusCreated, map[string]any{"Id": "container-123"}), nil
		}
		if strings.Contains(r.URL.Path, "/containers/") && strings.Contains(r.URL.Path, "/start") {
			return &http.Response{
				StatusCode: http.StatusNoContent,
				Proto:      "HTTP/1.1",
				ProtoMajor: 1,
				ProtoMinor: 1,
				Body:       http.NoBody,
			}, nil
		}
		return jsonResponse(http.StatusNotFound, map[string]string{"message": "unexpected"}), nil
	})

	ctx := t.Context()
	config := &docker.ContainerConfig{
		PortBindings: map[string]string{"8080": "8080"},
	}

	id, err := c.RunContainer(ctx, "test-image:latest", config)
	require.NoError(t, err)
	assert.Equal(t, "container-123", id)
}

func TestClient_RunContainer_Mock_StartError(t *testing.T) {
	c := newMockDockerClient(t, func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/containers/create") {
			return jsonResponse(http.StatusCreated, map[string]any{"Id": "container-456"}), nil
		}
		if strings.Contains(r.URL.Path, "/containers/") && strings.Contains(r.URL.Path, "/start") {
			return jsonResponse(http.StatusInternalServerError, map[string]string{"message": "start failed"}), nil
		}
		return jsonResponse(http.StatusNotFound, map[string]string{"message": "unexpected"}), nil
	})

	ctx := t.Context()
	config := &docker.ContainerConfig{
		PortBindings: map[string]string{"8080": "8080"},
	}

	_, err := c.RunContainer(ctx, "test-image:latest", config)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start container")
}
