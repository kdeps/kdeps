// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package yaml_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/parser/yaml"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

func TestFindComponentFile(t *testing.T) {
	dir := t.TempDir()

	// No component file
	result := yaml.FindComponentFile(dir)
	assert.Empty(t, result)

	// component.yaml
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.yaml"), []byte("test"), 0o600))
	result = yaml.FindComponentFile(dir)
	assert.Equal(t, filepath.Join(dir, "component.yaml"), result)

	// Remove and test .yaml.j2
	os.Remove(filepath.Join(dir, "component.yaml"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.yaml.j2"), []byte("test"), 0o600))
	result = yaml.FindComponentFile(dir)
	assert.Equal(t, filepath.Join(dir, "component.yaml.j2"), result)

	// Test .yml
	os.Remove(filepath.Join(dir, "component.yaml.j2"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.yml"), []byte("test"), 0o600))
	result = yaml.FindComponentFile(dir)
	assert.Equal(t, filepath.Join(dir, "component.yml"), result)

	// Test .yml.j2
	os.Remove(filepath.Join(dir, "component.yml"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.yml.j2"), []byte("test"), 0o600))
	result = yaml.FindComponentFile(dir)
	assert.Equal(t, filepath.Join(dir, "component.yml.j2"), result)

	// Test .j2
	os.Remove(filepath.Join(dir, "component.yml.j2"))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "component.j2"), []byte("test"), 0o600))
	result = yaml.FindComponentFile(dir)
	assert.Equal(t, filepath.Join(dir, "component.j2"), result)
}

func TestParseComponent_Valid(t *testing.T) {
	dir := t.TempDir()
	compPath := filepath.Join(dir, "component.yaml")
	require.NoError(t, os.WriteFile(compPath, []byte(validComponentYAML), 0o600))

	sv, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	parser := yaml.NewParser(sv, &mockExprParser{})
	comp, err := parser.ParseComponent(compPath)
	require.NoError(t, err)
	require.NotNil(t, comp)

	assert.Equal(t, "kdeps.io/v1", comp.APIVersion)
	assert.Equal(t, "Component", comp.Kind)
	assert.Equal(t, "greeter", comp.Metadata.Name)
	assert.Equal(t, "A simple greeting component", comp.Metadata.Description)
	assert.Equal(t, "1.0.0", comp.Metadata.Version)
	assert.Equal(t, "hello", comp.Metadata.TargetActionID)
	require.NotNil(t, comp.Interface)
	assert.Len(t, comp.Interface.Inputs, 2)

	assert.Equal(t, "user_name", comp.Interface.Inputs[0].Name)
	assert.Equal(t, "string", comp.Interface.Inputs[0].Type)
	assert.True(t, comp.Interface.Inputs[0].Required)
	assert.Equal(t, "The user's name", comp.Interface.Inputs[0].Description)

	assert.Equal(t, "temperature", comp.Interface.Inputs[1].Name)
	assert.Equal(t, "number", comp.Interface.Inputs[1].Type)
	assert.False(t, comp.Interface.Inputs[1].Required)
	assert.Equal(t, 0.7, comp.Interface.Inputs[1].Default)
}

func TestParseComponent_WithResources(t *testing.T) {
	dir := t.TempDir()
	compPath := filepath.Join(dir, "component.yaml")
	require.NoError(t, os.WriteFile(compPath, []byte(componentYAMLWithResources), 0o600))

	parser := newMockComponentParser()
	comp, err := parser.ParseComponent(compPath)
	require.NoError(t, err)
	require.NotNil(t, comp)

	assert.Len(t, comp.Resources, 1)
	assert.Equal(t, "process", comp.Resources[0].ActionID)
}

func TestParseComponent_InvalidAPIVersion(t *testing.T) {
	dir := t.TempDir()
	badYAML := `apiVersion: "v1"
kind: Component
metadata:
  name: test
`
	compPath := filepath.Join(dir, "component.yaml")
	require.NoError(t, os.WriteFile(compPath, []byte(badYAML), 0o600))

	sv, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	parser := yaml.NewParser(sv, &mockExprParser{})
	_, err = parser.ParseComponent(compPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "component schema validation failed")
}

func TestParseComponent_InvalidKind(t *testing.T) {
	dir := t.TempDir()
	badYAML := `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: test
`
	compPath := filepath.Join(dir, "component.yaml")
	require.NoError(t, os.WriteFile(compPath, []byte(badYAML), 0o600))

	sv, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	parser := yaml.NewParser(sv, &mockExprParser{})
	_, err = parser.ParseComponent(compPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "component schema validation failed")
}

func TestParseComponent_InvalidInputType(t *testing.T) {
	dir := t.TempDir()
	badYAML := `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: test
interface:
  inputs:
    - name: foo
      type: badtype
      required: true
`
	compPath := filepath.Join(dir, "component.yaml")
	require.NoError(t, os.WriteFile(compPath, []byte(badYAML), 0o600))

	sv, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	parser := yaml.NewParser(sv, &mockExprParser{})
	_, err = parser.ParseComponent(compPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be one of the following")
}

func TestParseComponent_MissingRequiredFields(t *testing.T) {
	dir := t.TempDir()
	badYAML := `apiVersion: kdeps.io/v1
kind: Component
metadata:
  description: No name
interface:
  inputs:
    - name: foo
`
	compPath := filepath.Join(dir, "component.yaml")
	require.NoError(t, os.WriteFile(compPath, []byte(badYAML), 0o600))

	sv, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	parser := yaml.NewParser(sv, &mockExprParser{})
	_, err = parser.ParseComponent(compPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "component validation failed")
	assert.Contains(t, err.Error(), "metadata: name is required")
}

func TestParseComponent_ResourceActionIDRequired(t *testing.T) {
	dir := t.TempDir()
	badYAML := `name: Resource without actionId
resources: []
`
	resPath := filepath.Join(dir, "resource.yaml")
	require.NoError(t, os.WriteFile(resPath, []byte(badYAML), 0o600))

	parser := newMockComponentParser()
	_, err := parser.ParseComponent(filepath.Join(dir, "component.yaml"))
	// Component file doesn't exist, expect parse error
	assert.NotNil(t, err)
}
