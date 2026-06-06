// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

//go:build !js

package iso

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestWriteLinuxKitConfigTempFile_CloseError(t *testing.T) {
	orig := closeTempFile
	t.Cleanup(func() { closeTempFile = orig })
	closeTempFile = func(_ *os.File) error {
		return errors.New("simulated close failure")
	}

	_, err := writeLinuxKitConfigTempFile("config: test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to close temp config file")
}

func TestBuilder_Build_GenerateConfigError(t *testing.T) {
	orig := yamlMarshal
	t.Cleanup(func() { yamlMarshal = orig })
	yamlMarshal = func(_ interface{}) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	builder := NewBuilderWithRunner(&mockRunnerForCoverage{})
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "marshal-test"},
	}

	err := builder.Build(t.Context(), "marshal-test:1.0.0", workflow, filepath.Join(t.TempDir(), "out.iso"), false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to generate LinuxKit config")
}

func TestBuilder_Build_MkdirTempBuildDirError(t *testing.T) {
	origMkdir := osMkdirTemp
	t.Cleanup(func() { osMkdirTemp = origMkdir })
	osMkdirTemp = func(_, _ string) (string, error) {
		return "", errors.New("mkdir temp failed")
	}

	builder := NewBuilderWithRunner(&mockRunnerForCoverage{})
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "build-dir-test"},
	}

	err := builder.Build(
		t.Context(),
		"build-dir-test:1.0.0",
		workflow,
		filepath.Join(t.TempDir(), "out.iso"),
		false,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create temp build directory")
}

func TestGenerateConfigYAMLExtended_MarshalError(t *testing.T) {
	orig := yamlMarshal
	t.Cleanup(func() { yamlMarshal = orig })
	yamlMarshal = func(_ interface{}) ([]byte, error) {
		return nil, errors.New("marshal failed")
	}

	builder := &Builder{Hostname: "test-host"}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "yaml-test"},
	}

	_, err := builder.GenerateConfigYAMLExtended("yaml-test:1.0.0", workflow, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal LinuxKit config")
}

func TestEnsureLinuxKit_CacheDirError(t *testing.T) {
	origHome := osUserHomeDir
	origPath := os.Getenv("PATH")
	t.Cleanup(func() {
		osUserHomeDir = origHome
		os.Setenv("PATH", origPath)
	})

	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home directory")
	}
	os.Setenv("PATH", "")

	_, err := EnsureLinuxKit(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get home directory")
}

func TestDownloadLinuxKit_ChmodError(t *testing.T) {
	tmpDir := t.TempDir()
	cachedBinary := filepath.Join(tmpDir, "linuxkit-test")

	origChmod := osChmod
	t.Cleanup(func() { osChmod = origChmod })
	osChmod = func(_ string, _ os.FileMode) error {
		return errors.New("chmod failed")
	}

	origDo := httpClientDo
	t.Cleanup(func() { httpClientDo = origDo })
	httpClientDo = func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("binary")),
			Header:     make(http.Header),
		}, nil
	}

	_, err := downloadLinuxKit(context.Background(), tmpDir, cachedBinary)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to make linuxkit executable")
}

func TestCreateRawBIOSWorkDir_HomeDirError(t *testing.T) {
	orig := osUserHomeDir
	t.Cleanup(func() { osUserHomeDir = orig })
	osUserHomeDir = func() (string, error) {
		return "", errors.New("no home directory")
	}

	_, err := createRawBIOSWorkDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get home directory")
}

type mockRunnerForCoverage struct{}

func (m *mockRunnerForCoverage) Build(_ context.Context, _, _, _, _, _ string) error {
	return nil
}

func (m *mockRunnerForCoverage) CacheImport(_ context.Context, _ string) error {
	return nil
}

func TestEnsureLinuxKit_CachedBinary(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })

	cacheDir, err := linuxkitCacheDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(cacheDir, 0750))

	cachedBinary := filepath.Join(cacheDir, "linuxkit-"+linuxkitVersion)
	require.NoError(t, os.WriteFile(cachedBinary, []byte("cached"), 0o700))

	path, err := EnsureLinuxKit(t.Context())
	require.NoError(t, err)
	assert.Equal(t, cachedBinary, path)
}

func TestNewBuilder_EnsureLinuxKitError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)
	t.Setenv("PATH", "")

	cacheParent := filepath.Join(tmpDir, ".cache", "kdeps")
	require.NoError(t, os.MkdirAll(cacheParent, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(cacheParent, "linuxkit"), []byte("block"), 0644))

	_, err := NewBuilder()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit not available")
}

func TestNewBuilder_CachedBinary(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", "")
	t.Cleanup(func() { os.Setenv("PATH", oldPath) })

	cacheDir, err := linuxkitCacheDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(cacheDir, 0750))

	cachedBinary := filepath.Join(cacheDir, "linuxkit-"+linuxkitVersion)
	require.NoError(t, os.WriteFile(cachedBinary, []byte("cached"), 0o700))

	builder, err := NewBuilder()
	require.NoError(t, err)
	require.NotNil(t, builder)
	require.NotNil(t, builder.Runner)
	assert.Equal(t, defaultHostname, builder.Hostname)
	assert.Equal(t, defaultFormat, builder.Format)
}

func TestDownloadLinuxKit_Success(t *testing.T) {
	tmpDir := t.TempDir()
	cachedBinary := filepath.Join(tmpDir, "linuxkit-"+linuxkitVersion)

	origDo := httpClientDo
	t.Cleanup(func() { httpClientDo = origDo })
	httpClientDo = func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("linuxkit-binary")),
			Header:     make(http.Header),
		}, nil
	}

	path, err := downloadLinuxKit(t.Context(), tmpDir, cachedBinary)
	require.NoError(t, err)
	assert.Equal(t, cachedBinary, path)
	assert.FileExists(t, cachedBinary)
}

func TestDefaultLinuxKitRunner_ExecMock(t *testing.T) {
	origExec := execCommandContext
	t.Cleanup(func() { execCommandContext = origExec })

	runner := &DefaultLinuxKitRunner{BinaryPath: "linuxkit"}
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yml")
	require.NoError(t, os.WriteFile(configPath, []byte("kernel: {}"), 0644))

	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "exit 0")
	}

	require.NoError(t, runner.Build(t.Context(), configPath, "iso-efi", "amd64", tmpDir, ""))
	require.NoError(t, runner.Build(t.Context(), configPath, "iso-efi", "amd64", tmpDir, "4096M"))
	require.NoError(t, runner.CacheImport(t.Context(), filepath.Join(tmpDir, "image.tar")))

	execCommandContext = func(ctx context.Context, _ string, _ ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sh", "-c", "exit 1")
	}

	err := runner.Build(t.Context(), configPath, "iso-efi", "amd64", tmpDir, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit build failed")

	err = runner.CacheImport(t.Context(), filepath.Join(tmpDir, "image.tar"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "linuxkit cache import failed")
}

func TestCreateRawBIOSWorkDir_MkdirTempError(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	cacheDir := filepath.Join(tmpDir, ".cache", "kdeps")
	require.NoError(t, os.MkdirAll(cacheDir, 0750))
	require.NoError(t, os.Chmod(cacheDir, 0555))

	_, err := createRawBIOSWorkDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create work directory")
}

func TestCreateRawBIOSWorkDir_MkdirAllError(t *testing.T) {
	tmpDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	t.Cleanup(func() { os.Setenv("HOME", oldHome) })

	cacheParent := filepath.Join(tmpDir, ".cache")
	require.NoError(t, os.MkdirAll(cacheParent, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(cacheParent, "kdeps"), []byte("block"), 0644))

	_, err := createRawBIOSWorkDir()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create cache directory")
}

type mockRawBIOSRunner struct{}

func (m *mockRawBIOSRunner) Build(
	_ context.Context,
	configPath, format, _, outputDir, _ string,
) error {
	if format != "kernel+initrd" {
		return nil
	}

	base := filepath.Base(configPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	files := map[string]string{
		base + "-kernel":     "fake-kernel",
		base + "-initrd.img": "fake-initrd",
		base + "-cmdline":    "console=ttyS0",
	}
	for name, content := range files {
		if writeErr := os.WriteFile(filepath.Join(outputDir, name), []byte(content), 0644); writeErr != nil {
			return writeErr
		}
	}
	return nil
}

func (m *mockRawBIOSRunner) CacheImport(_ context.Context, _ string) error {
	return nil
}

func TestAssembleRawBIOS_WorkDirError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cacheParent := filepath.Join(tmpDir, ".cache")
	require.NoError(t, os.MkdirAll(cacheParent, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(cacheParent, "kdeps"), []byte("block"), 0644))

	kernelFile := filepath.Join(tmpDir, "kernel")
	initrdFile := filepath.Join(tmpDir, "initrd.img")
	cmdlineFile := filepath.Join(tmpDir, "cmdline")
	outputFile := filepath.Join(tmpDir, "disk.img")
	for _, f := range []string{kernelFile, initrdFile, cmdlineFile} {
		require.NoError(t, os.WriteFile(f, []byte("fake"), 0644))
	}

	err := assembleRawBIOS(t.Context(), kernelFile, initrdFile, cmdlineFile, outputFile, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create cache directory")
}

func TestBuilder_BuildRawBIOS_FallbackAssembler_ShortMode(t *testing.T) {
	origExec := execCommandContext
	t.Cleanup(func() { execCommandContext = origExec })

	execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "docker" && len(args) > 0 {
			switch args[0] {
			case "save":
				return exec.CommandContext(ctx, "touch", args[2])
			case "run":
				var workDir string
				for _, arg := range args {
					if strings.Contains(arg, ":/work") {
						workDir = strings.TrimSuffix(arg, ":/work")
						break
					}
				}
				diskPath := filepath.Join(workDir, "disk.img")
				return exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("echo fake > %q", diskPath))
			}
		}
		return exec.CommandContext(ctx, "sh", "-c", "exit 0")
	}

	builder := NewBuilderWithRunner(&mockRawBIOSRunner{})
	builder.Format = "raw-bios"

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "fallback-short"},
	}
	outputPath := filepath.Join(t.TempDir(), "output.raw")

	err := builder.Build(t.Context(), "fallback-short:1.0.0", workflow, outputPath, false)
	require.NoError(t, err)
	assert.FileExists(t, outputPath)
}
