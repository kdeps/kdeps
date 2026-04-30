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

package chat

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanCatalog_Empty(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("KDEPS_COMPONENT_DIR", tmp)

	entries := ScanCatalog()
	assert.Empty(t, entries)
}

func TestScanCatalog_WithComponent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("KDEPS_COMPONENT_DIR", tmp)

	compDir := filepath.Join(tmp, "my-tool")
	require.NoError(t, os.MkdirAll(compDir, 0o755))

	yaml := `apiVersion: kdeps.io/v1
kind: Component
metadata:
  name: my-tool
  description: Does something useful
  version: 1.2.0
interface:
  inputs:
    - name: query
      type: string
      required: true
      description: Search query
    - name: limit
      type: integer
      required: false
`
	require.NoError(t, os.WriteFile(filepath.Join(compDir, "component.yaml"), []byte(yaml), 0o644))

	entries := ScanCatalog()
	require.Len(t, entries, 1)
	assert.Equal(t, "my-tool", entries[0].Name)
	assert.Equal(t, "1.2.0", entries[0].Version)
	assert.Equal(t, "Does something useful", entries[0].Description)
	assert.Len(t, entries[0].Inputs, 2)
	assert.Contains(t, entries[0].Inputs[0], "query")
	assert.Contains(t, entries[0].Inputs[0], "[required]")
}

func TestScanCatalog_Deduplication(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("KDEPS_COMPONENT_DIR", tmp)

	// Add same component twice (same dir = only scanned once anyway)
	for range 2 {
		compDir := filepath.Join(tmp, "dup-tool")
		require.NoError(t, os.MkdirAll(compDir, 0o755))
		yaml := `metadata:
  name: dup-tool
  version: 1.0.0
`
		require.NoError(t, os.WriteFile(filepath.Join(compDir, "component.yaml"), []byte(yaml), 0o644))
	}

	entries := ScanCatalog()
	assert.Len(t, entries, 1)
}

func TestFormatCatalog_Empty(t *testing.T) {
	out := FormatCatalog(nil)
	assert.Contains(t, out, "No components")
}

func TestFormatCatalog_WithEntries(t *testing.T) {
	entries := []ComponentEntry{
		{
			Name:        "search",
			Version:     "1.0.0",
			Description: "Web search",
			Inputs:      []string{"query (string) [required]"},
		},
		{
			Name:    "memory",
			Version: "2.0.0",
		},
	}

	out := FormatCatalog(entries)
	assert.Contains(t, out, "search@1.0.0")
	assert.Contains(t, out, "Web search")
	assert.Contains(t, out, "query (string) [required]")
	assert.Contains(t, out, "memory@2.0.0")
	assert.True(t, strings.HasPrefix(out, "Available components"))
}

func TestScanComponentDir_NoYAML(t *testing.T) {
	dir := t.TempDir()
	entry := scanComponentDir(dir)
	assert.Nil(t, entry)
}

func TestScanComponentDir_WorkflowFallback(t *testing.T) {
	dir := t.TempDir()
	yaml := `metadata:
  name: fallback-tool
  version: 0.1.0
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(yaml), 0o644))

	entry := scanComponentDir(dir)
	require.NotNil(t, entry)
	assert.Equal(t, "fallback-tool", entry.Name)
}
