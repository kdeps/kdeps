// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package iso_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/infra/iso"
)

// mockRunner records calls and returns configurable results.
type mockRunner struct {
	buildCalls       []mockBuildCall
	cacheImportCalls []string
	buildErr         error
	cacheImportErr   error
}

type mockBuildCall struct {
	ConfigPath string
	Format     string
	Arch       string
	OutputDir  string
	Size       string
}

func (m *mockRunner) CacheImport(_ context.Context, tarPath string) error {
	m.cacheImportCalls = append(m.cacheImportCalls, tarPath)
	return m.cacheImportErr
}

func (m *mockRunner) Build(_ context.Context, configPath, format, arch, outputDir, size string) error {
	m.buildCalls = append(m.buildCalls, mockBuildCall{
		ConfigPath: configPath,
		Format:     format,
		Arch:       arch,
		OutputDir:  outputDir,
		Size:       size,
	})

	if m.buildErr != nil {
		return m.buildErr
	}

	// For kernel+initrd format (used by raw-bios two-step build),
	// produce the expected kernel, initrd, and cmdline files.
	if format == "kernel+initrd" {
		base := filepath.Base(configPath)
		base = base[:len(base)-len(filepath.Ext(base))] // strip .yml
		_ = os.WriteFile(filepath.Join(outputDir, base+"-kernel"), []byte("fake-kernel"), 0644)
		_ = os.WriteFile(filepath.Join(outputDir, base+"-initrd.img"), []byte("fake-initrd"), 0644)
		_ = os.WriteFile(filepath.Join(outputDir, base+"-cmdline"), []byte("console=ttyS0"), 0644)
		return nil
	}

	// Create a fake output file to simulate linuxkit producing output
	ext := iso.GetFormatExtension(format)
	if ext == "" {
		ext = ".iso"
	}

	fakeOutput := filepath.Join(outputDir, "kdeps"+ext)
	return os.WriteFile(fakeOutput, []byte("fake-image-data"), 0644)
}

// mockAssembleRawBIOS simulates the raw-bios disk assembly (no Docker needed).
func mockAssembleRawBIOS(_ context.Context, _, _, _, outputPath, _, _ string) error {
	return os.WriteFile(outputPath, []byte("fake-raw-bios-disk"), 0644)
}

// ========================
// Config generation tests
// ========================

func TestGenerateConfig_Basic(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "test-app",
			Version: "1.0.0",
		},
	}

	config, err := iso.GenerateConfig("test-app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)

	assert.Contains(t, config.Kernel.Image, "linuxkit/kernel:")
	assert.Contains(t, config.Kernel.Cmdline, "console=ttyS0")
	assert.Len(t, config.Init, 4)     // init + runc + containerd + ca-certificates
	assert.Len(t, config.Services, 3) // dhcpcd + getty + kdeps
	assert.Equal(t, "dhcpcd", config.Services[0].Name)
	assert.Equal(t, "getty", config.Services[1].Name)
	assert.Equal(t, "kdeps", config.Services[2].Name)
	assert.Equal(t, "test-app:1.0.0", config.Services[2].Image)
	assert.Equal(t, "host", config.Services[2].Net)
	assert.Contains(t, config.Services[2].Capabilities, "all")
}

func TestGenerateConfig_WithEnvVars(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "env-app",
		},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				Env: map[string]string{
					"FOO": "bar",
					"BAZ": "qux",
				},
			},
		},
	}

	config, err := iso.GenerateConfig("env-app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)

	kdepsService := config.Services[2]
	assert.Contains(t, kdepsService.Env, "KDEPS_BIND_HOST=0.0.0.0")
	assert.Contains(t, kdepsService.Env, "BAZ=qux")
	assert.Contains(t, kdepsService.Env, "FOO=bar")
	// KDEPS_BIND_HOST first, then KDEPS_PLATFORM, then sorted user env vars
	assert.Equal(t, "KDEPS_BIND_HOST=0.0.0.0", kdepsService.Env[0])
	assert.Equal(t, "KDEPS_PLATFORM=iso", kdepsService.Env[1])
	assert.Equal(t, "BAZ=qux", kdepsService.Env[2])
	assert.Equal(t, "FOO=bar", kdepsService.Env[3])
}

func TestGenerateConfig_Hostname(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "host-app",
		},
	}

	config, err := iso.GenerateConfig("host-app:1.0.0", "my-custom-host", "", workflow)
	require.NoError(t, err)

	require.Len(t, config.Files, 1)
	assert.Equal(t, "etc/hostname", config.Files[0].Path)
	assert.Equal(t, "my-custom-host", config.Files[0].Contents)
}

func TestGenerateConfig_DefaultHostname(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "app",
		},
	}

	config, err := iso.GenerateConfig("app:1.0.0", "", "", workflow)
	require.NoError(t, err)

	require.Len(t, config.Files, 1)
	assert.Equal(t, "kdeps", config.Files[0].Contents)
}

func TestGenerateConfig_NilWorkflow(t *testing.T) {
	_, err := iso.GenerateConfig("test:1.0.0", "kdeps", "", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow cannot be nil")
}

func TestGenerateConfig_EmptyImageName(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "app",
		},
	}

	_, err := iso.GenerateConfig("", "kdeps", "", workflow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "image name cannot be empty")
}

func TestGenerateConfig_WithOllama(t *testing.T) {
	installOllama := true
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "ollama-app",
		},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				InstallOllama: &installOllama,
			},
		},
	}

	config, err := iso.GenerateConfig("ollama-app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)

	// Ollama images should have /dev bind mount and ollama env vars
	kdepsService := config.Services[2]
	assert.Contains(t, kdepsService.Binds, "/dev:/dev")
	assert.Contains(t, kdepsService.Env, "OLLAMA_HOST=127.0.0.1")
	assert.Contains(t, kdepsService.Env, "OLLAMA_MODELS=/root/.ollama/models")
}

func TestGenerateConfig_WithoutOllama(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "simple-app",
		},
	}

	config, err := iso.GenerateConfig("simple-app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)

	kdepsService := config.Services[2]
	assert.NotContains(t, kdepsService.Binds, "/dev:/dev")
	assert.Contains(t, kdepsService.Binds, "/var/run:/var/run")
}

func TestGenerateConfig_YAMLRoundtrip(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "roundtrip-app",
		},
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				Env: map[string]string{"KEY": "value"},
			},
		},
	}

	config, err := iso.GenerateConfig("roundtrip-app:1.0.0", "test-host", "", workflow)
	require.NoError(t, err)

	data, err := iso.MarshalConfig(config)
	require.NoError(t, err)

	// Verify valid YAML
	var parsed iso.LinuxKitConfig
	require.NoError(t, yaml.Unmarshal(data, &parsed))

	assert.Equal(t, config.Kernel.Image, parsed.Kernel.Image)
	assert.Equal(t, config.Kernel.Cmdline, parsed.Kernel.Cmdline)
	assert.Equal(t, config.Init, parsed.Init)
	assert.Len(t, parsed.Services, 3)
	assert.Equal(t, "roundtrip-app:1.0.0", parsed.Services[2].Image)
	assert.Equal(t, "test-host", parsed.Files[0].Contents)
}

func TestMarshalConfig_ContainsExpectedKeys(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "yaml-app",
		},
	}

	config, err := iso.GenerateConfig("yaml-app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)

	data, err := iso.MarshalConfig(config)
	require.NoError(t, err)

	yamlStr := string(data)
	assert.Contains(t, yamlStr, "kernel:")
	assert.Contains(t, yamlStr, "init:")
	assert.Contains(t, yamlStr, "services:")
	assert.Contains(t, yamlStr, "files:")
	assert.Contains(t, yamlStr, "linuxkit/kernel:")
	assert.Contains(t, yamlStr, "yaml-app:1.0.0")
	assert.Contains(t, yamlStr, "dhcpcd")
}

// ========================
// Builder tests (mock)
// ========================

func TestBuilder_Build_CallsRunner(t *testing.T) {
	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "test-app",
			Version: "1.0.0",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.iso")

	err := builder.Build(t.Context(), "test-app:1.0.0", workflow, outputPath, false)
	require.NoError(t, err)

	require.Len(t, runner.buildCalls, 1)
	assert.Equal(t, "iso-efi", runner.buildCalls[0].Format)
	assert.FileExists(t, outputPath)
}

func TestBuilder_Build_NilWorkflow(t *testing.T) {
	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)

	err := builder.Build(t.Context(), "test:1.0.0", nil, "output.iso", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow cannot be nil")
	assert.Empty(t, runner.buildCalls)
}

func TestBuilder_Build_EmptyImage(t *testing.T) {
	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "app",
		},
	}

	err := builder.Build(t.Context(), "", workflow, "output.iso", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "image name cannot be empty")
	assert.Empty(t, runner.buildCalls)
}

func TestBuilder_Build_RunnerError(t *testing.T) {
	runner := &mockRunner{buildErr: errors.New("linuxkit failed")}
	builder := iso.NewBuilderWithRunner(runner)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "app",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.iso")

	err := builder.Build(t.Context(), "app:1.0.0", workflow, outputPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit failed")
}

func TestBuilder_DefaultFormat(t *testing.T) {
	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)
	assert.Equal(t, "iso-efi", builder.Format)
}

func TestBuilder_CustomFormat_RawBIOS(t *testing.T) {
	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)
	builder.Format = "raw-bios"
	builder.RawBIOSAssembleFunc = mockAssembleRawBIOS

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "app",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.raw")

	err := builder.Build(t.Context(), "app:1.0.0", workflow, outputPath, false)
	require.NoError(t, err)

	// raw-bios uses two-step build: linuxkit builds kernel+initrd first
	require.Len(t, runner.buildCalls, 1)
	assert.Equal(t, "kernel+initrd", runner.buildCalls[0].Format)
	assert.FileExists(t, outputPath)
}

func TestBuilder_GenerateConfigYAML(t *testing.T) {
	builder := iso.NewBuilderWithRunner(nil)
	builder.Hostname = "my-host"

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "preview-app",
			Version: "2.0.0",
		},
	}

	yamlStr, err := builder.GenerateConfigYAML("preview-app:2.0.0", workflow)
	require.NoError(t, err)

	assert.Contains(t, yamlStr, "preview-app:2.0.0")
	assert.Contains(t, yamlStr, "my-host")
	assert.Contains(t, yamlStr, "linuxkit/kernel:")
}

func TestBuilder_GenerateConfigYAML_NilWorkflow(t *testing.T) {
	builder := iso.NewBuilderWithRunner(nil)

	_, err := builder.GenerateConfigYAML("app:1.0.0", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "workflow cannot be nil")
}

// ========================
// Ollama detection tests
// ========================

func TestOllamaDetection_ExplicitFlag(t *testing.T) {
	installOllama := true
	workflow := &domain.Workflow{
		Settings: domain.WorkflowSettings{
			AgentSettings: domain.AgentSettings{
				InstallOllama: &installOllama,
			},
		},
	}

	config, err := iso.GenerateConfig("app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)
	assert.Contains(t, config.Services[2].Binds, "/dev:/dev")
}

func TestOllamaDetection_BackendOllama(t *testing.T) {
	workflow := &domain.Workflow{
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
	}

	config, err := iso.GenerateConfig("app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)
	assert.Contains(t, config.Services[2].Binds, "/dev:/dev")
}

func TestOllamaDetection_OnlineProvider(t *testing.T) {
	workflow := &domain.Workflow{
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
	}

	config, err := iso.GenerateConfig("app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)
	assert.NotContains(t, config.Services[2].Binds, "/dev:/dev")
}

// ========================
// LinuxKit URL tests
// ========================

func TestLinuxKitDownloadURL(t *testing.T) {
	url := iso.LinuxKitDownloadURL()
	assert.Contains(t, url, "github.com/linuxkit/linuxkit/releases/download")
	assert.Contains(t, url, "linuxkit-")
}

func TestGetFormatExtension(t *testing.T) {
	assert.Equal(t, ".iso", iso.GetFormatExtension("iso-efi"))
	assert.Equal(t, ".raw", iso.GetFormatExtension("raw-bios"))
	assert.Equal(t, ".qcow2", iso.GetFormatExtension("qcow2-bios"))
}

// ========================
// Architecture tests
// ========================

func TestGenerateConfig_ARM64Cmdline(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "arm-app",
		},
	}

	config, err := iso.GenerateConfig("arm-app:1.0.0", "kdeps", "arm64", workflow)
	require.NoError(t, err)

	assert.Contains(t, config.Kernel.Cmdline, "console=ttyAMA0")
	assert.NotContains(t, config.Kernel.Cmdline, "console=ttyS0")
}

func TestGenerateConfig_AMD64Cmdline(t *testing.T) {
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "amd-app",
		},
	}

	config, err := iso.GenerateConfig("amd-app:1.0.0", "kdeps", "amd64", workflow)
	require.NoError(t, err)

	assert.Contains(t, config.Kernel.Cmdline, "console=ttyS0")
	assert.NotContains(t, config.Kernel.Cmdline, "console=ttyAMA0")
}

func TestKernelCmdline(t *testing.T) {
	assert.Equal(t, "console=ttyAMA0 console=tty0", iso.KernelCmdline("arm64"))
	assert.Equal(t, "console=ttyS0 console=tty0", iso.KernelCmdline("amd64"))
	assert.Equal(t, "console=ttyS0 console=tty0", iso.KernelCmdline(""))
}

func TestBuilder_Build_PassesSize(t *testing.T) {
	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)
	builder.Size = "4096M"

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "big-app",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.iso")

	err := builder.Build(t.Context(), "big-app:1.0.0", workflow, outputPath, false)
	require.NoError(t, err)

	require.Len(t, runner.buildCalls, 1)
	assert.Equal(t, "4096M", runner.buildCalls[0].Size)
}

func TestBuilder_Build_EmptySize(t *testing.T) {
	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "small-app",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.iso")

	err := builder.Build(t.Context(), "small-app:1.0.0", workflow, outputPath, false)
	require.NoError(t, err)

	require.Len(t, runner.buildCalls, 1)
	assert.Empty(t, runner.buildCalls[0].Size)
}

func TestBuilder_Build_ARM64PassesArch(t *testing.T) {
	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)
	builder.Arch = "arm64"

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "arm-app",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.iso")

	err := builder.Build(t.Context(), "arm-app:1.0.0", workflow, outputPath, false)
	require.NoError(t, err)

	require.Len(t, runner.buildCalls, 1)
	assert.Equal(t, "arm64", runner.buildCalls[0].Arch)
}
