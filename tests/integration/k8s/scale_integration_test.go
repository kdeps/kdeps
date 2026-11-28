// Copyright 2026 Kdeps, KvK 94834768
// Licensed under the Apache License, Version 2.0

package k8s_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yamlv3 "gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v3/pkg/domain"
	"github.com/kdeps/kdeps/v3/pkg/infra/k8s"
	"github.com/kdeps/kdeps/v3/pkg/parser/expression"
	"github.com/kdeps/kdeps/v3/pkg/parser/yaml"
	"github.com/kdeps/kdeps/v3/pkg/validator"
)

// parseScaleWorkflow writes a workflow with an API server and the given
// deploy lines, and parses it with the schema.
func parseScaleWorkflow(t *testing.T, deploy string) (*domain.Workflow, error) {
	t.Helper()
	dir := t.TempDir()
	wf := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: agent
  version: "1.0.0"
  targetActionId: act
deploy:
  apiServer:
    portNum: 8080
    routes:
      - path: /api/v1/run
        methods: [POST]
` + deploy + `resources:
  - actionId: act
    name: Act
    apiResponse:
      success: true
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(wf), 0o600))
	sv, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	return yaml.NewParser(sv, expression.NewParser()).ParseWorkflow(filepath.Join(dir, "workflow.yaml"))
}

// kinds returns the kind of each YAML document in the manifests.
func kinds(t *testing.T, manifests string) []string {
	t.Helper()
	var out []string
	dec := yamlv3.NewDecoder(strings.NewReader(manifests))
	for {
		var doc struct {
			APIVersion string `yaml:"apiVersion"`
			Kind       string `yaml:"kind"`
		}
		if err := dec.Decode(&doc); err != nil {
			break
		}
		out = append(out, doc.APIVersion+" "+doc.Kind)
	}
	return out
}

func TestScale_ExportModes(t *testing.T) {
	cases := map[string]struct {
		deploy string
		want   []string
	}{
		"default scales to zero with knative": {"", []string{"serving.knative.dev/v1 Service"}},
		"keda": {"  scale:\n    engine: keda\n    max: 4\n", []string{
			"apps/v1 Deployment", "v1 Service", "http.keda.sh/v1alpha1 HTTPScaledObject",
		}},
		"fixed count": {"  scale:\n    min: 2\n    max: 2\n", []string{"apps/v1 Deployment", "v1 Service"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			wf, err := parseScaleWorkflow(t, tc.deploy+"  limits:\n    timeout: 30s\n    concurrency: 2\n")
			require.NoError(t, err)
			out, err := k8s.NewGenerator("agent:1").GenerateManifests(wf)
			require.NoError(t, err)
			assert.Equal(t, tc.want, kinds(t, out))
		})
	}
}

func TestScale_SchemaRejects(t *testing.T) {
	for name, deploy := range map[string]string{
		"unknown key":      "  scale:\n    replicas: 2\n",
		"bad engine":       "  scale:\n    engine: lambda\n",
		"negative min":     "  scale:\n    min: -1\n",
		"removed replicas": "  runtime:\n    replicas: 3\n",
	} {
		_, err := parseScaleWorkflow(t, deploy)
		require.Error(t, err, name)
	}
}

func TestScale_ValidatorRejectsMaxBelowMin(t *testing.T) {
	wf, err := parseScaleWorkflow(t, "  scale:\n    min: 3\n    max: 1\n")
	require.NoError(t, err)
	err = validator.NewWorkflowValidator(nil).ValidateSettings(wf)
	require.ErrorContains(t, err, "deploy.scale.max")
}

const secondPort = "  webServer:\n    portNum: 9090\n    routes:\n      - path: /\n        publicPath: /public\n"

// With no deploy.scale, a workflow that no request can wake runs one pod.
func TestScale_DefaultWithoutWakeIsOnePod(t *testing.T) {
	wf, err := parseScaleWorkflow(t, secondPort)
	require.NoError(t, err)
	require.NoError(t, validator.NewWorkflowValidator(nil).ValidateSettings(wf))
	assert.Equal(t, domain.ScaleConfig{Min: 1, Max: 1}, wf.Settings.EffectiveScale())
	out, err := k8s.NewGenerator("agent:1").GenerateManifests(wf)
	require.NoError(t, err)
	assert.Equal(t, []string{"apps/v1 Deployment", "v1 Service"}, kinds(t, out))
}

// An explicit scale.min 0 that no request can wake fails validation.
func TestScale_ValidatorRejectsZeroWithoutWake(t *testing.T) {
	wf, err := parseScaleWorkflow(t, secondPort+"  scale:\n    max: 3\n")
	require.NoError(t, err)
	err = validator.NewWorkflowValidator(nil).ValidateSettings(wf)
	require.ErrorContains(t, err, "scale to zero serves one port")
}
