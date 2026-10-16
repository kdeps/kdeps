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
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package yaml

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	goyaml "gopkg.in/yaml.v3"

	"github.com/kdeps/kdeps/v2/pkg/assets"
	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/validator"
)

func componentBundle(t *testing.T, component string) []byte {
	t.Helper()
	data, err := goyaml.Marshal(assets.Bundle{
		Header: assets.Header{Version: "1.0.0"},
		Files:  map[string]string{componentsFile: component},
	})
	require.NoError(t, err)
	return data
}

func TestValidateComponentBundle(t *testing.T) {
	ok := "apiVersion: kdeps.io/v1\nkind: Component\nmetadata:\n  name: ocr\n"
	require.NoError(t, validateComponentBundle(componentBundle(t, ok), "ocr"))
	require.ErrorContains(t, validateComponentBundle(componentBundle(t, ok), "git"), `names "ocr", not "git"`)
	require.ErrorContains(
		t,
		validateComponentBundle(componentBundle(t, "kind: Workflow\n"), "ocr"),
		"not a named Component",
	)
	require.ErrorContains(t, validateComponentBundle(componentBundle(t, "kind: [\n"), "ocr"), "yaml")
	require.Error(t, validateComponentBundle([]byte("version: 1.0.0\nfiles: {}\n"), "ocr"))
}

func TestShippedComponentsValidate(t *testing.T) {
	entries, err := os.ReadDir(filepath.Join(assets.Source(), componentsSet))
	require.NoError(t, err)
	require.NotEmpty(t, entries)
	sv, err := validator.NewSchemaValidator()
	require.NoError(t, err)
	p := NewParser(sv, nil)
	for _, e := range entries {
		files, sErr := assets.SourceFiles(assets.Source(), componentsSet)
		require.NoError(t, sErr)
		require.NoError(t, validateComponentBundle(files[e.Name()], e.Name()), e.Name())
		data, rErr := os.ReadFile(filepath.Join(assets.Source(), componentsSet, e.Name(), componentsFile))
		require.NoError(t, rErr)
		_, pErr := p.parseComponentData(data)
		require.NoError(t, pErr, "%s passes the component schema", e.Name())
	}
}

func TestLoadAssetComponents_BrokenAssetFails(t *testing.T) {
	src := t.TempDir()
	dir := filepath.Join(src, componentsSet, "ocr")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "template.yaml"), []byte("version: 1.0.0\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, componentsFile), []byte("kind: Workflow\n"), 0o600))
	t.Setenv(assets.SourceEnv, src)
	wf := &domain.Workflow{
		Components: map[string]*domain.Component{},
		Resources:  []*domain.Resource{{ActionID: "a", Component: &domain.ComponentCallConfig{Name: "ocr"}}},
	}
	require.ErrorContains(t, NewParser(nil, nil).loadAssetComponents(wf), "kdeps component ocr")
}

func TestComponentContext(t *testing.T) {
	got := componentContext(componentBundle(t, `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: sql
  description: Runs SQL.
interface:
  inputs:
    - name: query
      type: string
      description: The statement.
`), "sql")
	require.Contains(t, got, "Runs SQL.")
	require.Contains(t, got, "The statement.")
	require.Empty(t, componentContext([]byte("not: a bundle"), "x"))
	require.Empty(t, componentContext(componentBundle(t, "kind: Workflow\n"), "x"))
	require.True(t, assets.InContext("components"))
}
