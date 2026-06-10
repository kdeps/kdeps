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
	"runtime"
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

func (m *mockRunner) Build(
	_ context.Context,
	configPath, format, arch, outputDir, size string,
) error {
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

// mockNoOutputRunner succeeds without producing output files.
// Used to test the findLinuxKitOutput error path in iso-efi builds.
type mockNoOutputRunner struct{}

func (m *mockNoOutputRunner) Build(_ context.Context, _, _, _, _, _ string) error {
	return nil
}

func (m *mockNoOutputRunner) CacheImport(_ context.Context, _ string) error {
	return nil
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
	assert.Contains(t, config.Kernel.Cmdline, iso.KernelCmdline(runtime.GOARCH))
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

func TestBuilder_Build_CreateTempError(t *testing.T) {
	tmpDir := t.TempDir()
	readOnlyDir := filepath.Join(tmpDir, "readonly")
	require.NoError(t, os.Mkdir(readOnlyDir, 0555))
	t.Setenv("TMPDIR", readOnlyDir)

	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "create-temp-test",
		},
	}

	outputPath := filepath.Join(tmpDir, "output.iso")
	err := builder.Build(t.Context(), "create-temp-test:1.0.0", workflow, outputPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temp config file")
}

func TestBuilder_Build_MkdirAllOutputDirError(t *testing.T) {
	tmpDir := t.TempDir()

	blockFile := filepath.Join(tmpDir, "block")
	require.NoError(t, os.WriteFile(blockFile, []byte("block"), 0644))

	outputPath := filepath.Join(blockFile, "output.iso")

	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "mkdir-test",
		},
	}

	err := builder.Build(t.Context(), "mkdir-test:1.0.0", workflow, outputPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create output directory")
}

func TestBuilder_Build_RenameError(t *testing.T) {
	tmpDir := t.TempDir()

	outputDir := filepath.Join(tmpDir, "readonly-output")
	require.NoError(t, os.Mkdir(outputDir, 0555))
	outputPath := filepath.Join(outputDir, "output.iso")

	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "rename-test",
		},
	}

	err := builder.Build(t.Context(), "rename-test:1.0.0", workflow, outputPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to move output")
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
				Chat: &domain.ChatConfig{
					Backend: "ollama",
					Model:   "llama2:7b",
				},
			},
		},
	}

	config, err := iso.GenerateConfig("app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)
	assert.Contains(t, config.Services[2].Binds, "/dev:/dev")
}

func TestOllamaDetection_OnlineProvider(t *testing.T) {
	t.Setenv("KDEPS_DEFAULT_BACKEND", "openai")
	workflow := &domain.Workflow{
		Resources: []*domain.Resource{
			{
				Chat: &domain.ChatConfig{},
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

// ========================
// LinuxKit Runner Tests
// ========================

func TestDefaultLinuxKitRunner_Build_ErrorPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that would execute external command")
	}

	runner := &iso.DefaultLinuxKitRunner{
		BinaryPath: "/nonexistent/linuxkit",
	}

	ctx := t.Context()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	_ = os.WriteFile(configPath, []byte("kernel: {}"), 0644)

	err := runner.Build(ctx, configPath, "iso-efi", "amd64", tmpDir, "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit build failed")
}

func TestDefaultLinuxKitRunner_Build_WithSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that would execute external command")
	}

	runner := &iso.DefaultLinuxKitRunner{
		BinaryPath: "/nonexistent/linuxkit",
	}

	ctx := t.Context()
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	_ = os.WriteFile(configPath, []byte("kernel: {}"), 0644)

	err := runner.Build(ctx, configPath, "iso-efi", "amd64", tmpDir, "4096M")

	// Should attempt to run with size parameter
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit build failed")
}

func TestDefaultLinuxKitRunner_CacheImport_ErrorPath(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that would execute external command")
	}

	runner := &iso.DefaultLinuxKitRunner{
		BinaryPath: "/nonexistent/linuxkit",
	}

	ctx := t.Context()
	err := runner.CacheImport(ctx, "/fake/image.tar")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit cache import failed")
}

func TestDefaultLinuxKitRunner_Build_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires a fake linuxkit binary")
	}

	tmpDir := t.TempDir()
	fakeLinuxKit := filepath.Join(tmpDir, "linuxkit")
	linuxKitScript := "#!/bin/sh\nexit 0\n"
	require.NoError(t, os.WriteFile(fakeLinuxKit, []byte(linuxKitScript), 0755))

	runner := &iso.DefaultLinuxKitRunner{
		BinaryPath: fakeLinuxKit,
	}

	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("kernel: {}"), 0644))

	err := runner.Build(t.Context(), configPath, "iso-efi", "amd64", tmpDir, "")
	require.NoError(t, err)
}

func TestDefaultLinuxKitRunner_CacheImport_Success(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires a fake linuxkit binary")
	}

	tmpDir := t.TempDir()
	fakeLinuxKit := filepath.Join(tmpDir, "linuxkit")
	linuxKitScript := "#!/bin/sh\nexit 0\n"
	require.NoError(t, os.WriteFile(fakeLinuxKit, []byte(linuxKitScript), 0755))

	runner := &iso.DefaultLinuxKitRunner{
		BinaryPath: fakeLinuxKit,
	}

	tarPath := filepath.Join(tmpDir, "image.tar")
	require.NoError(t, os.WriteFile(tarPath, []byte("fake-tar"), 0644))

	err := runner.CacheImport(t.Context(), tarPath)
	require.NoError(t, err)
}

// ========================
// EnsureLinuxKit Tests
// ========================

func TestEnsureLinuxKit_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// This test attempts to find or download linuxkit
	// It may succeed or fail depending on environment
	_, err := iso.EnsureLinuxKit(t.Context())

	// We just ensure it doesn't panic
	// In CI without linuxkit, it should fail gracefully
	_ = err
}

// ========================
// Builder Constructor Tests
// ========================

func TestNewBuilder_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test that may download linuxkit")
	}

	builder, err := iso.NewBuilder()

	if err != nil {
		// Expected in environments without linuxkit
		assert.Contains(t, err.Error(), "linuxkit not available")
		return
	}

	// If successful, verify builder is properly initialized
	require.NotNil(t, builder)
	require.NotNil(t, builder.Runner)
	assert.Equal(t, "kdeps", builder.Hostname)
	assert.Equal(t, "iso-efi", builder.Format)
}

func TestNewBuilder_EnsureLinuxKitError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that manipulates HOME/PATH")
	}
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("PATH", "")

	cacheParent := filepath.Join(tmpDir, ".cache", "kdeps")
	require.NoError(t, os.MkdirAll(cacheParent, 0750))
	cacheDirPath := filepath.Join(cacheParent, "linuxkit")
	require.NoError(t, os.WriteFile(cacheDirPath, []byte("block"), 0644))

	_, err := iso.NewBuilder()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit not available")
}

func TestBuilder_Build_EmptyFormatDefaultsToISOEFI(t *testing.T) {
	runner := &mockRunner{}
	builder := &iso.Builder{
		Runner: runner,
		Arch:   runtime.GOARCH,
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "empty-format-app",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.iso")
	err := builder.Build(t.Context(), "empty-format-app:1.0.0", workflow, outputPath, false)
	require.NoError(t, err)
	require.Len(t, runner.buildCalls, 1)
	assert.Equal(t, "iso-efi", runner.buildCalls[0].Format)
}

func TestBuilder_Build_EmptyArchDefaultsToRuntimeGOARCH(t *testing.T) {
	runner := &mockRunner{}
	builder := &iso.Builder{
		Runner: runner,
		Format: "iso-efi",
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "empty-arch-app",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.iso")
	err := builder.Build(t.Context(), "empty-arch-app:1.0.0", workflow, outputPath, false)
	require.NoError(t, err)
	require.Len(t, runner.buildCalls, 1)
	assert.Equal(t, runtime.GOARCH, runner.buildCalls[0].Arch)
}

func TestBuilder_Build_RawBIOS_FallbackAssembler(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test that requires a fake docker binary")
	}

	// Set up a fake docker on PATH that produces the expected outputs.
	tmpDir := t.TempDir()
	fakeDocker := filepath.Join(tmpDir, "docker")
	dockerScript := `#!/bin/sh
set -e
if [ "$1" = "save" ]; then
    touch "$3"
    exit 0
fi
if [ "$1" = "run" ]; then
    WORK_DIR=""
    for arg in "$@"; do
        if echo "$arg" | grep -q ":/work"; then
            WORK_DIR=$(echo "$arg" | cut -d: -f1)
        fi
    done
    if [ -n "$WORK_DIR" ]; then
        echo "fake disk image" > "$WORK_DIR/disk.img"
    fi
    exit 0
fi
exit 0
`
	if err := os.WriteFile(fakeDocker, []byte(dockerScript), 0755); err != nil {
		t.Fatalf("failed to create fake docker: %v", err)
	}

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tmpDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Builder with nil RawBIOSAssembleFunc — exercises the fallback to assembleRawBIOS.
	runner := &mockRunner{}
	builder := iso.NewBuilderWithRunner(runner)
	builder.Format = "raw-bios"
	// RawBIOSAssembleFunc is nil by default, triggering the fallback path.

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "fallback-test",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.raw")
	err := builder.Build(t.Context(), "fallback-test:1.0.0", workflow, outputPath, false)
	require.NoError(t, err)

	require.Len(t, runner.buildCalls, 1)
	assert.Equal(t, "kernel+initrd", runner.buildCalls[0].Format)
	assert.FileExists(t, outputPath)
}

func TestBuilder_Build_RawBIOSError(t *testing.T) {
	runner := &mockRunner{
		buildErr: errors.New("linuxkit kernel+initrd failed"),
	}
	builder := iso.NewBuilderWithRunner(runner)
	builder.Format = "raw-bios"
	builder.RawBIOSAssembleFunc = mockAssembleRawBIOS

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "raw-bios-error-app",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.raw")
	err := builder.Build(t.Context(), "raw-bios-error-app:1.0.0", workflow, outputPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit kernel+initrd failed")
}

// ========================
// GetFormatExtension unknown format
// ========================

func TestGetFormatExtension_Unknown(t *testing.T) {
	assert.Equal(t, "", iso.GetFormatExtension("unknown-format"))
	assert.Equal(t, "", iso.GetFormatExtension("vdi"))
	assert.Equal(t, "", iso.GetFormatExtension(""))
}

// ========================
// Builder iso-efi findLinuxKitOutput failure
// ========================

func TestBuilder_Build_ISOEFINoOutput(t *testing.T) {
	runner := &mockNoOutputRunner{}
	builder := iso.NewBuilderWithRunner(runner)

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "no-output-app",
		},
	}

	outputPath := filepath.Join(t.TempDir(), "output.iso")
	err := builder.Build(t.Context(), "no-output-app:1.0.0", workflow, outputPath, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no output file found")
}

// ========================
// Ollama detection via KDEPS_LLM_ROUTER env var
// ========================

func TestOllamaDetection_RouterConfig(t *testing.T) {
	t.Setenv("KDEPS_LLM_ROUTER", `{"backend":"ollama","models":["llama2:7b"]}`)
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "router-app",
		},
	}

	config, err := iso.GenerateConfig("router-app:1.0.0", "kdeps", "", workflow)
	require.NoError(t, err)

	kdepsService := config.Services[2]
	assert.Contains(t, kdepsService.Binds, "/dev:/dev")
	assert.Contains(t, kdepsService.Env, "OLLAMA_HOST=127.0.0.1")
	assert.Contains(t, kdepsService.Env, "OLLAMA_MODELS=/root/.ollama/models")
}

// ========================
// GenerateConfigYAMLExtended with empty image name
// ========================

func TestBuilder_GenerateConfigYAMLExtended_EmptyImageName(t *testing.T) {
	builder := iso.NewBuilderWithRunner(nil)
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name: "app",
		},
	}

	_, err := builder.GenerateConfigYAMLExtended("", workflow, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "image name cannot be empty")
}

// TestBuilder_Build_WriteStringError moved to builder_write_error_internal_test.go:
// the WriteString error path is now triggered via the osCreateTemp seam instead of
// RLIMIT_FSIZE, which broke the test runtime's own bookkeeping writes.
