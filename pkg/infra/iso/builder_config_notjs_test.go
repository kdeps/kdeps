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
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestGenerateConfigYAMLExtended_Thin(t *testing.T) {
	b := &Builder{
		Hostname: "test-host",
		Format:   "raw-bios",
		Arch:     "amd64",
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "thin-test",
			Version: "1.0.0",
		},
	}

	yaml, err := b.GenerateConfigYAMLExtended("my-image:latest", workflow, true)
	if err != nil {
		t.Fatalf("GenerateConfigYAMLExtended failed: %v", err)
	}

	if len(yaml) == 0 {
		t.Error("expected non-empty YAML output")
	}

	// Thin builds should contain mount-data and import-image steps
	if !strings.Contains(yaml, "mount-data") {
		t.Error("expected thin build YAML to contain 'mount-data'")
	}

	if !strings.Contains(yaml, "import-image") {
		t.Error("expected thin build YAML to contain 'import-image'")
	}
}

func TestGenerateConfigYAMLExtended_Fat(t *testing.T) {
	b := &Builder{
		Hostname: "test-host",
		Format:   "iso-efi",
		Arch:     "amd64",
	}

	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{
			Name:    "fat-test",
			Version: "1.0.0",
		},
	}

	yaml, err := b.GenerateConfigYAMLExtended("my-image:latest", workflow, false)
	if err != nil {
		t.Fatalf("GenerateConfigYAMLExtended failed: %v", err)
	}

	if len(yaml) == 0 {
		t.Error("expected non-empty YAML output")
	}
}

func TestGenerateConfigYAMLExtended_NilWorkflow(t *testing.T) {
	b := &Builder{Hostname: "test-host"}

	_, err := b.GenerateConfigYAMLExtended("my-image:latest", nil, false)
	if err == nil {
		t.Error("expected error for nil workflow, got nil")
	}
}

func TestGenerateConfigYAMLExtended_EmptyHostname(t *testing.T) {
	b := &Builder{} // empty hostname → should default

	workflow := &domain.Workflow{}

	_, err := b.GenerateConfigYAMLExtended("my-image:latest", workflow, false)
	if err != nil {
		t.Fatalf("expected no error with empty hostname, got: %v", err)
	}
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
