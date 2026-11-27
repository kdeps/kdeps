// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package executor_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/executor"
	"github.com/kdeps/kdeps/v3/pkg/parser/expression"
	"github.com/kdeps/kdeps/v3/pkg/parser/yaml"
	"github.com/kdeps/kdeps/v3/pkg/validator"
)

// parseLimitsWorkflow writes a workflow with the given deploy.limits block and
// one chat step, and parses it with the schema.
func parseLimitsWorkflow(t *testing.T, limits string) (*domain.Workflow, error) {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "resources"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(`apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: limits-integration
  version: "1.0.0"
  targetActionId: wait
deploy:
  limits:
`+limits+`
`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "resources", "wait.yaml"), []byte(`actionId: wait
name: Wait
chat:
  model:
    name: fake
  context:
    prompt: wait
`), 0o600))
	sv, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	return yaml.NewParser(sv, expression.NewParser()).ParseWorkflow(filepath.Join(dir, "workflow.yaml"))
}

// TestLimits_TimeoutStopsAParsedWorkflow runs a parsed workflow whose model
// answers after 5s under deploy.limits.timeout: 300ms.
func TestLimits_TimeoutStopsAParsedWorkflow(t *testing.T) {
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(5 * time.Second):
		case <-r.Context().Done():
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"choices":[{"index":0,"message":{"role":"assistant","content":"late"},`+
			`"finish_reason":"stop"}]}`)
	}))
	defer model.Close()
	t.Setenv("KDEPS_LLM_OVERRIDE_URL", model.URL)
	wf, err := parseLimitsWorkflow(t, "    timeout: 300ms\n    concurrency: 2")
	require.NoError(t, err)
	require.NotNil(t, wf.Settings.Limits)
	assert.Equal(t, 2, wf.Settings.Limits.Concurrency)

	start := time.Now()
	_, err = newExecEngine(t).Execute(wf, &executor.RequestContext{Ctx: t.Context()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deploy.limits.timeout")
	assert.Less(t, time.Since(start), 4*time.Second)
}

// TestLimits_SchemaRejectsBadLimits checks the schema (unknown key, negative
// concurrency) and the workflow validator (a timeout that is not a duration).
func TestLimits_SchemaRejectsBadLimits(t *testing.T) {
	for name, limits := range map[string]string{
		"unknown key":          "    wait: 1s",
		"negative concurrency": "    concurrency: -1",
	} {
		_, err := parseLimitsWorkflow(t, limits)
		require.Error(t, err, name)
	}

	wf, err := parseLimitsWorkflow(t, "    timeout: soon")
	require.NoError(t, err)
	err = validator.NewWorkflowValidator(nil).ValidateSettings(wf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deploy.limits.timeout")
}
