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
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/parser/yaml"
)

// createKdepsArchiveWithEntry creates a .kdeps archive at dest containing a
// single file with the given tar header name and content.  The caller controls
// header.Name (e.g. "../outside") and header.Size to exercise specific
// validation branches in extractTarFile and safeJoinPath.
func createKdepsArchiveWithEntry(t *testing.T, destPath, entryName string, entrySize int64) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(destPath), 0o750))
	f, err := os.Create(destPath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	hdr := &tar.Header{
		Name: entryName,
		Size: entrySize,
		Mode: 0o600,
	}
	require.NoError(t, tw.WriteHeader(hdr))

	if entrySize > 0 && entrySize <= 100 {
		data := make([]byte, entrySize)
		for i := range data {
			data[i] = 'x'
		}
		_, wErr := tw.Write(data)
		require.NoError(t, wErr)
	}
}

// createKdepsArchiveWithoutWorkflow creates a .kdeps archive that contains
// a non-workflow file (data.txt) so extraction succeeds but no workflow is found.
func createKdepsArchiveWithoutWorkflow(t *testing.T, destPath string) {
	t.Helper()
	createKdepsArchiveWithEntry(t, destPath, "data.txt", 5)
}

// createTraversalKdepsArchive creates a .kdeps archive with a path-traversal
// entry name (../outside.txt) to trigger safeJoinPath rejection.
func createTraversalKdepsArchive(t *testing.T, destPath string) {
	t.Helper()
	createKdepsArchiveWithEntry(t, destPath, "../outside.txt", 5)
}

// createOversizedKdepsArchive creates a .kdeps archive with an entry whose
// declared Size exceeds maxKdepsExtractSize (100 MB).  No actual data beyond
// the header is written.
func createOversizedKdepsArchive(t *testing.T, destPath string) {
	t.Helper()
	// 100 MB + 1 byte
	createKdepsArchiveWithEntry(t, destPath, "bigfile.bin", 100*1024*1024+1)
}

// minimalWorkflowYAML is a standalone workflow that can be packaged into .kdeps.
const minimalPackagedWorkflowYAML = `apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: packed-agent
  version: "1.0.0"
  targetActionId: respond
settings:
  agentSettings:
    timezone: "UTC"
resources:
  - actionId: respond
    name: Respond
    apiResponse:
      success: true
      response: "packed agent response"
`

// createKdepsPackage creates a .kdeps (tar.gz) archive from a directory.
// The function mirrors the logic used in cmd.createTestPackage to produce
// valid archives that extractKdepsPackage can consume.
func createKdepsPackage(t *testing.T, sourceDir, destPath string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(destPath), 0o750))
	f, err := os.Create(destPath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	require.NoError(
		t,
		filepath.Walk(sourceDir, func(path string, info os.FileInfo, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			rel, relErr := filepath.Rel(sourceDir, path)
			if relErr != nil {
				return relErr
			}
			hdr, hdrErr := tar.FileInfoHeader(info, "")
			if hdrErr != nil {
				return hdrErr
			}
			hdr.Name = rel
			if wErr := tw.WriteHeader(hdr); wErr != nil {
				return wErr
			}
			if !info.IsDir() {
				data, rErr := os.ReadFile(path)
				if rErr != nil {
					return rErr
				}
				_, wErr := tw.Write(data)
				return wErr
			}
			return nil
		}),
	)
}

const validComponentYAML = `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: greeter
  description: A simple greeting component
  version: "1.0.0"
  targetActionId: hello
interface:
  inputs:
    - name: user_name
      type: string
      required: true
      description: The user's name
    - name: temperature
      type: number
      required: false
      default: 0.7
`

const componentYAMLWithResources = `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: processor
  version: "2.0.0"
interface:
  inputs:
    - name: data
      type: string
      required: true
resources:
  - actionId: process
    exec:
      command: echo "Processing"
`

func newMockComponentParser() *yaml.Parser {
	return yaml.NewParser(&mockSchemaValidator{}, &mockExprParser{})
}

// createTestKomponent creates a gzipped tar archive at path that contains a
// single file "component.yaml" with the given YAML content.
func createTestKomponent(t *testing.T, path, componentYAML string) {
	t.Helper()

	// Create temp directory to build the archive from
	tmp := t.TempDir()
	compFile := filepath.Join(tmp, "component.yaml")
	require.NoError(t, os.WriteFile(compFile, []byte(componentYAML), 0o600))

	// Create the tar.gz archive
	file, err := os.Create(path)
	require.NoError(t, err)
	defer file.Close()

	gz := gzip.NewWriter(file)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	// Add component.yaml
	require.NoError(t, addFileToTar(tw, tmp, "component.yaml"))

	// Could add resources/ subdir if needed in future tests
}

// addFileToTar adds a file from baseDir to the tar writer with the given
// relative path (relativeName). It uses the file's actual content.
func addFileToTar(tw *tar.Writer, baseDir, relativeName string) error {
	path := filepath.Join(baseDir, relativeName)
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = relativeName
	if err = tw.WriteHeader(header); err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(tw, f)
	return err
}

func TestGlobalComponentsDir_EnvOverride(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("KDEPS_COMPONENT_DIR", tmp)
	result := yaml.GlobalComponentsDir()
	assert.Equal(t, tmp, result)
}

func TestGlobalComponentsDir_Default(t *testing.T) {
	t.Setenv("KDEPS_COMPONENT_DIR", "")
	home := t.TempDir()
	t.Setenv("HOME", home)
	result := yaml.GlobalComponentsDir()
	assert.Equal(t, filepath.Join(home, ".kdeps", "components"), result)
}

func TestHasJ2Suffix(t *testing.T) {
	assert.True(t, yaml.HasJ2Suffix("file.j2"))
	assert.True(t, yaml.HasJ2Suffix("component.yaml.j2"))
	assert.False(t, yaml.HasJ2Suffix("file.yaml"))
	assert.False(t, yaml.HasJ2Suffix(".j2")) // len == 3, not > 3
	assert.False(t, yaml.HasJ2Suffix(""))
}

func TestTrimJ2Suffix(t *testing.T) {
	assert.Equal(t, "component.yaml", yaml.TrimJ2Suffix("component.yaml.j2"))
	assert.Equal(t, "file", yaml.TrimJ2Suffix("file.j2"))
	assert.Equal(t, "file.yaml", yaml.TrimJ2Suffix("file.yaml")) // no .j2
}

func TestIsKomponentFile(t *testing.T) {
	assert.True(t, yaml.IsKomponentFileInternal("email.komponent"))
	assert.True(t, yaml.IsKomponentFileInternal("my-component.komponent"))
	assert.False(t, yaml.IsKomponentFileInternal("email.kdeps"))
	assert.False(t, yaml.IsKomponentFileInternal("component.yaml"))
	assert.False(t, yaml.IsKomponentFileInternal(""))
}

func TestScanComponentsDir_NonExistent(t *testing.T) {
	p := newMockComponentParser()
	resources, _, err := p.ScanComponentsDir("/nonexistent/path/to/nowhere", map[string]struct{}{})
	require.NoError(t, err)
	assert.Nil(t, resources)
}

func TestScanComponentsDir_PathIsFile(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "notadir")
	require.NoError(t, os.WriteFile(f, []byte("x"), 0o600))

	p := newMockComponentParser()
	resources, _, err := p.ScanComponentsDir(f, map[string]struct{}{})
	require.NoError(t, err)
	assert.Nil(t, resources)
}

func TestScanComponentsDir_EmptyDir(t *testing.T) {
	tmp := t.TempDir()
	p := newMockComponentParser()
	resources, _, err := p.ScanComponentsDir(tmp, map[string]struct{}{})
	require.NoError(t, err)
	assert.Empty(t, resources)
}

const componentWithInterface = `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: scraper
  description: Test scraper
  version: "1.0.0"
interface:
  inputs:
    - name: url
      type: string
      required: true
      description: URL to scrape
    - name: selector
      type: string
      required: false
      description: CSS selector
resources:
  - actionId: scrape-url
    exec:
      command: echo ok
`

func TestScanComponentsDir_ReturnsComponentMap(t *testing.T) {
	tmp := t.TempDir()
	compDir := filepath.Join(tmp, "mycomp")
	require.NoError(t, os.Mkdir(compDir, 0o755))
	require.NoError(t, os.WriteFile(
		filepath.Join(compDir, "component.yaml"),
		[]byte(componentWithInterface),
		0o600,
	))

	p := newMockComponentParser()
	_, comps, err := p.ScanComponentsDir(tmp, map[string]struct{}{})
	require.NoError(t, err)
	require.NotNil(t, comps)
	comp, ok := comps["scraper"]
	require.True(t, ok)
	assert.Equal(t, "1.0.0", comp.Metadata.Version)
}
