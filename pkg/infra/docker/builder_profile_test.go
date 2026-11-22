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

package docker_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/infra/docker"
)

func TestResolveOllamaImageTag(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "rocm", docker.ResolveOllamaImageTag("rocm", "0.5.0"))
	assert.Equal(t, "0.5.0", docker.ResolveOllamaImageTag("cuda", "0.5.0"))
	assert.Equal(t, "0.5.0", docker.ResolveOllamaImageTag("", "0.5.0"))
}

func TestBuilder_GenerateDockerfile_workflowBaseOSOverride(t *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{BaseOS: "ubuntu"},
		},
	}

	dockerfile, err := builder.GenerateDockerfile(t.Context(), workflow)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "FROM ubuntu:latest")
}

func TestBuilder_GenerateDockerfile_rejectDebianBaseOS(t *testing.T) {
	builder := &docker.Builder{BaseOS: "alpine"}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{BaseOS: "debian"},
		},
	}

	_, err := builder.GenerateDockerfile(t.Context(), workflow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid baseOS")
}

func TestBuilder_GenerateDockerfile_rocmOllamaUbuntu(t *testing.T) {
	installOllama := true
	builder := &docker.Builder{BaseOS: "ubuntu", GPUType: "rocm"}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "rocm-app", Version: "1.0.0"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{InstallOllama: &installOllama},
		},
	}

	dockerfile, err := builder.GenerateDockerfile(t.Context(), workflow)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "FROM ollama/ollama:rocm")
	assert.NotContains(t, dockerfile, "COPY --from=ollama/ollama")
}

func TestBuilder_GenerateDockerfile_ollamaSkipsCopy(t *testing.T) {
	installOllama := true
	builder := &docker.Builder{BaseOS: "alpine"}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "cpu-ollama", Version: "1.0.0"},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{InstallOllama: &installOllama},
		},
	}

	dockerfile, err := builder.GenerateDockerfile(t.Context(), workflow)
	require.NoError(t, err)
	assert.Contains(t, dockerfile, "FROM alpine/ollama:0.5.0")
	assert.NotContains(t, dockerfile, "COPY --from=alpine/ollama")
	assert.Contains(t, dockerfile, "Ollama included in base image")
}

// A chat that names no model runs on PyBot, so the image bakes it in.
func TestBuilder_GenerateDockerfile_bakesPybotForChatWithoutModel(t *testing.T) {
	t.Setenv("KDEPS_LLM_MODELS", "")
	t.Setenv("KDEPS_DEFAULT_BACKEND", "")
	t.Setenv("KDEPS_LLM_ROUTER", "")
	for _, base := range []string{"alpine", "ubuntu"} {
		builder := &docker.Builder{BaseOS: base}
		workflow := &domain.Workflow{
			Metadata:  domain.WorkflowMetadata{Name: "test", Version: "1.0.0"},
			Resources: []*domain.Resource{{ActionID: "llm", Chat: &domain.ChatConfig{Prompt: "hi"}}},
		}
		dockerfile, err := builder.GenerateDockerfile(t.Context(), workflow)
		require.NoError(t, err)
		assert.Contains(t, dockerfile, "codeload.github.com/huggingface/pybot", base)
		assert.Contains(t, dockerfile, "sha256sum -c -", base)
		assert.Contains(t, dockerfile, "KDEPS_MODELS_DIR=/app/.kdeps/models", base)

		workflow.Resources[0].Chat.Model = "ministral3"
		dockerfile, err = builder.GenerateDockerfile(t.Context(), workflow)
		require.NoError(t, err)
		assert.NotContains(t, dockerfile, "huggingface/pybot", base)
	}
}
