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

//go:build !js

package python

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindPythonExecutable_ScriptsOnly(t *testing.T) {
	venv := t.TempDir()
	scripts := filepath.Join(venv, "Scripts")
	require.NoError(t, os.MkdirAll(scripts, 0755))
	exe := filepath.Join(scripts, "python.exe")
	require.NoError(t, os.WriteFile(exe, []byte("x"), 0755))

	got, err := findPythonExecutable(venv)
	require.NoError(t, err)
	assert.Equal(t, exe, got)
}

func TestInstallPackages_FallbackPythonPath(t *testing.T) {
	m := NewManager(t.TempDir())
	venv := t.TempDir()
	err := m.InstallPackages(venv, []string{"__kdeps_nonexistent_pkg_xyz__"})
	require.Error(t, err)
}

func TestInstallRequirements_FallbackPythonPath(t *testing.T) {
	m := NewManager(t.TempDir())
	venv := t.TempDir()
	req := filepath.Join(t.TempDir(), "requirements.txt")
	require.NoError(t, os.WriteFile(req, []byte("nonexistent-pkg-xyz\n"), 0644))
	err := m.InstallRequirements(venv, req)
	require.Error(t, err)
}

func TestUvVenvEnv_IncludesPythonDir(t *testing.T) {
	env := uvVenvEnv("/venv", "/venv/bin/python")
	found := false
	for _, e := range env {
		if e == "VIRTUAL_ENV=/venv" {
			found = true
		}
	}
	assert.True(t, found)
}
