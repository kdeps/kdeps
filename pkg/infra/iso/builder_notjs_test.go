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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestBuilder_Build_NilWorkflow_Error(t *testing.T) {
	b := &Builder{}
	err := b.Build(context.Background(), "image", nil, "out", false)
	if err == nil {
		t.Error("expected error for nil workflow, got nil")
	}
}

func TestBuilder_Build_EmptyImage_Error(t *testing.T) {
	b := &Builder{}
	err := b.Build(context.Background(), "", &domain.Workflow{}, "out", false)
	if err == nil {
		t.Error("expected error for empty image name, got nil")
	}
}

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
