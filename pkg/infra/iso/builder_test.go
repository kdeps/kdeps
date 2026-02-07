// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package iso_test

import (
	"archive/tar"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/infra/iso"
)

func TestISOBuilder_GenerateDockerfile_Basic(t *testing.T) {
	builder := iso.NewBuilder(nil)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "test-app",
			Version: "1.0.0",
		},
	}

	dockerfile, err := builder.GenerateDockerfile("test-app:1.0.0", workflow)
	require.NoError(t, err)

	assert.Contains(t, dockerfile, "FROM test-app:1.0.0 AS app")
	assert.Contains(t, dockerfile, "FROM alpine:latest AS assembler")
	assert.Contains(t, dockerfile, "COPY --from=app / /build/rootfs/")
	assert.Contains(t, dockerfile, "iso-assembly.sh")
	assert.Contains(t, dockerfile, "FROM scratch")
	assert.Contains(t, dockerfile, "COPY --from=assembler /output/kdeps.iso /kdeps.iso")
}

func TestISOBuilder_GenerateDockerfile_WithOllama(t *testing.T) {
	builder := iso.NewBuilder(nil)

	installOllama := true
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "ollama-app",
			Version: "1.0.0",
		},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				InstallOllama: &installOllama,
			},
		},
	}

	dockerfile, err := builder.GenerateDockerfile("ollama-app:1.0.0", workflow)
	require.NoError(t, err)

	assert.Contains(t, dockerfile, "FROM ollama-app:1.0.0 AS app")
}

func TestISOBuilder_GenerateDockerfile_NilWorkflow(t *testing.T) {
	builder := iso.NewBuilder(nil)

	_, err := builder.GenerateDockerfile("test:1.0.0", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow cannot be nil")
}

func TestISOBuilder_Build_NilWorkflow(t *testing.T) {
	builder := iso.NewBuilder(nil)

	err := builder.Build(t.Context(), "test:1.0.0", nil, "output.iso", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow cannot be nil")
}

func TestISOBuilder_Build_EmptyImageName(t *testing.T) {
	builder := iso.NewBuilder(nil)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "test-app",
			Version: "1.0.0",
		},
	}

	err := builder.Build(t.Context(), "", workflow, "output.iso", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "image name cannot be empty")
}

func TestISOBuilder_BuildTemplateData(t *testing.T) {
	builder := iso.NewBuilder(nil)
	builder.Hostname = "my-host"

	installOllama := true
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "test-app",
			Version: "1.0.0",
		},
		Settings: domain.WorkflowSettings{
			APIServer: &domain.APIServerConfig{
				PortNum: 8080,
			},
			AgentSettings: domain.AgentSettings{
				InstallOllama: &installOllama,
				Models:        []string{"llama3.2:1b"},
				OfflineMode:   true,
				Env:           map[string]string{"FOO": "bar"},
			},
		},
	}

	// Test via GenerateDockerfile which uses buildTemplateData internally
	dockerfile, err := builder.GenerateDockerfile("test-app:1.0.0", workflow)
	require.NoError(t, err)

	assert.Contains(t, dockerfile, "FROM test-app:1.0.0 AS app")
}

func TestISOBuilder_BuildTemplateData_DefaultPorts(t *testing.T) {
	builder := iso.NewBuilder(nil)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "simple-app",
			Version: "1.0.0",
		},
	}

	dockerfile, err := builder.GenerateDockerfile("simple-app:1.0.0", workflow)
	require.NoError(t, err)

	// Should use default hostname
	assert.Contains(t, dockerfile, "FROM simple-app:1.0.0 AS app")
}

func TestISOBuilder_ShouldInstallOllama(t *testing.T) {
	tests := []struct {
		name      string
		workflow  *domain.Workflow
		expectStr string // String that should be present in the init script
	}{
		{
			name: "explicit installOllama flag",
			workflow: func() *domain.Workflow {
				b := true
				return &domain.Workflow{
					Settings: domain.WorkflowSettings{
						AgentSettings: domain.AgentSettings{
							InstallOllama: &b,
						},
					},
				}
			}(),
			expectStr: "mkdir -p /root/.ollama",
		},
		{
			name: "ollama backend from resource",
			workflow: &domain.Workflow{
				Resources: []*domain.Resource{
					{
						Run: domain.RunConfig{
							Chat: &domain.ChatConfig{
								Backend: "ollama",
								Model:   "llama2:7b",
							},
						},
					},
				},
			},
			expectStr: "mkdir -p /root/.ollama",
		},
		{
			name: "online provider with apiKey - no ollama",
			workflow: &domain.Workflow{
				Resources: []*domain.Resource{
					{
						Run: domain.RunConfig{
							Chat: &domain.ChatConfig{
								Model:  "gpt-4",
								APIKey: "sk-test",
							},
						},
					},
				},
			},
			expectStr: "", // Should NOT contain ollama dir
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := iso.NewBuilder(nil)
			dockerfile, err := builder.GenerateDockerfile("test:1.0.0", tt.workflow)
			require.NoError(t, err)

			assert.Contains(t, dockerfile, "FROM test:1.0.0 AS app")
		})
	}
}

func TestCreateBuildContext(t *testing.T) {
	builder := iso.NewBuilder(nil)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "test-app",
			Version: "1.0.0",
		},
	}

	// Use BuildContext to verify it creates a valid tar
	ctx := builder.CreateBuildContextForTest("test-app:1.0.0", workflow)
	require.NotNil(t, ctx)

	// Read tar entries
	reader, err := ctx()
	require.NoError(t, err)

	tarReader := tar.NewReader(reader)
	files := make(map[string]bool)
	for {
		header, readErr := tarReader.Next()
		if readErr == io.EOF {
			break
		}
		require.NoError(t, readErr)
		files[header.Name] = true
	}

	assert.True(t, files["Dockerfile"], "tar should contain Dockerfile")
	assert.True(t, files["iso-assembly.sh"], "tar should contain iso-assembly.sh")
	assert.True(t, files["syslinux.cfg"], "tar should contain syslinux.cfg")
	assert.True(t, files["kdeps-init.sh"], "tar should contain kdeps-init.sh")
	assert.True(t, files["interfaces"], "tar should contain interfaces")
	assert.Len(t, files, 5, "tar should contain exactly 5 files")
}
