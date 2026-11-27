// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package domain_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v3/pkg/domain"
)

func TestLimitsConfig_TimeoutDuration(t *testing.T) {
	d, err := (&domain.LimitsConfig{}).TimeoutDuration()
	require.NoError(t, err)
	assert.Zero(t, d)

	d, err = (&domain.LimitsConfig{Timeout: "90s"}).TimeoutDuration()
	require.NoError(t, err)
	assert.Equal(t, 90*time.Second, d)

	for _, bad := range []string{"soon", "0s", "-5s"} {
		_, err = (&domain.LimitsConfig{Timeout: bad}).TimeoutDuration()
		require.Error(t, err, bad)
		assert.Contains(t, err.Error(), "deploy.limits.timeout")
	}
}

func TestLimitsConfig_Check(t *testing.T) {
	require.NoError(t, (&domain.LimitsConfig{Timeout: "1m", Concurrency: 4}).Check())
	err := (&domain.LimitsConfig{Concurrency: -1}).Check()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "deploy.limits.concurrency")
	require.Error(t, (&domain.LimitsConfig{Timeout: "x"}).Check())
}

func TestWorkflow_DeployLimitsDecode(t *testing.T) {
	var wf domain.Workflow
	require.NoError(t, yaml.Unmarshal([]byte(`apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: w
  targetActionId: a
deploy:
  limits:
    timeout: 60s
    concurrency: 4
`), &wf))
	require.NotNil(t, wf.Settings.Limits)
	assert.Equal(t, "60s", wf.Settings.Limits.Timeout)
	assert.Equal(t, 4, wf.Settings.Limits.Concurrency)
}
