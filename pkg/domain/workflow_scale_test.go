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

func TestScaleConfig_Check(t *testing.T) {
	ok := []domain.ScaleConfig{
		{}, {Min: 1, Max: 1}, {Max: 5, Idle: "5m", Engine: "keda"}, {Engine: "knative", Idle: "0s"},
	}
	for _, s := range ok {
		assert.NoError(t, s.Check(), "%+v", s)
	}
	bad := map[string]domain.ScaleConfig{
		"deploy.scale.min":          {Min: -1},
		"deploy.scale.max: -":       {Max: -1},
		"is below deploy.scale.min": {Min: 3, Max: 2},
		"deploy.scale.engine":       {Engine: "lambda"},
		"deploy.scale.idle":         {Idle: "later"},
	}
	for want, s := range bad {
		assert.ErrorContains(t, s.Check(), want)
	}
	_, err := (&domain.ScaleConfig{Idle: "-1s"}).IdleDuration()
	assert.Error(t, err)
}

func TestScaleConfig_Helpers(t *testing.T) {
	assert.True(t, (&domain.ScaleConfig{Min: 2, Max: 2}).Fixed())
	assert.False(t, (&domain.ScaleConfig{}).Fixed())
	assert.False(t, (&domain.ScaleConfig{Min: 1, Max: 3}).Fixed())
	assert.Equal(t, "knative", (&domain.ScaleConfig{}).EngineName())
	assert.Equal(t, "keda", (&domain.ScaleConfig{Engine: "keda"}).EngineName())
	d, err := (&domain.ScaleConfig{Idle: "90s"}).IdleDuration()
	require.NoError(t, err)
	assert.Equal(t, 90*time.Second, d)
	d, err = (&domain.ScaleConfig{}).IdleDuration()
	require.NoError(t, err)
	assert.Zero(t, d)
}

func TestWorkflow_DeployScaleYAML(t *testing.T) {
	var wf domain.Workflow
	require.NoError(t, yaml.Unmarshal([]byte(`apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: a
deploy:
  scale:
    min: 1
    max: 4
    idle: 5m
    engine: keda
`), &wf))
	require.NotNil(t, wf.Settings.Scale)
	assert.Equal(t, domain.ScaleConfig{Min: 1, Max: 4, Idle: "5m", Engine: "keda"}, *wf.Settings.Scale)
}
