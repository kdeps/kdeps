// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0

package docker

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestParseTemplateError_AllBranches(t *testing.T) {
	baseErr := errors.New("parse failed")

	cases := []struct {
		name    string
		wantSub string
	}{
		{"dockerfile", "failed to parse Dockerfile template"},
		{"entrypoint", "failed to parse entrypoint template"},
		{"supervisord", "failed to parse supervisord template"},
		{"custom", "failed to parse custom template"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := parseTemplateError(tc.name, baseErr)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantSub)
		})
	}
}

func TestRenderTemplateError_AllBranches(t *testing.T) {
	baseErr := errors.New("render failed")

	cases := []struct {
		name    string
		wantSub string
	}{
		{"dockerfile", "failed to render Dockerfile"},
		{"entrypoint", "failed to render entrypoint"},
		{"supervisord", "failed to render supervisord config"},
		{"custom", "failed to render custom"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := renderTemplateError(tc.name, baseErr)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantSub)
		})
	}
}

func TestRenderTemplate_ParseAndExecuteErrors(t *testing.T) {
	_, err := renderTemplate("dockerfile", "{{.Missing", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse Dockerfile template")

	_, err = renderTemplate("dockerfile", "{{call .Value}}", struct{ Value int }{Value: 1})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to render Dockerfile")
}

func TestRenderWorkflowTemplate_BuildTemplateDataError(t *testing.T) {
	orig := backendInstallTemplate
	t.Cleanup(func() { backendInstallTemplate = orig })
	backendInstallTemplate = "{{call .InstallOllama}}"

	builder := &Builder{BaseOS: "alpine"}
	workflow := &domain.Workflow{
		Metadata: domain.WorkflowMetadata{Name: "test"},
	}

	_, err := builder.renderWorkflowTemplate("dockerfile", "FROM alpine", workflow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to build template data")
}

func TestRenderBackendInstall_ParseAndExecuteErrors(t *testing.T) {
	orig := backendInstallTemplate
	t.Cleanup(func() { backendInstallTemplate = orig })

	builder := &Builder{BaseOS: "alpine"}

	backendInstallTemplate = "{{.Broken"
	_, err := builder.renderBackendInstall(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse backend install template")

	backendInstallTemplate = "{{call .InstallOllama}}"
	_, err = builder.renderBackendInstall(false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to render backend install")
}
