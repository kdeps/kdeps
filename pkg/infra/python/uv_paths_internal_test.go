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
	"context"
	"errors"
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

func withIOToolsBaseDir(t *testing.T, base string) {
	t.Helper()
	orig := userCacheDirFunc
	t.Cleanup(func() { userCacheDirFunc = orig })
	userCacheDirFunc = func() (string, error) { return base, nil }
}

func TestIOToolPythonBin_ExistsInTempBase(t *testing.T) {
	withIOToolsBaseDir(t, t.TempDir())
	toolName := "whisper-test"
	binPath := filepath.Join(IOToolVenvPath(toolName), "bin", "python")
	require.NoError(t, os.MkdirAll(filepath.Dir(binPath), 0755))
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))
	assert.Equal(t, binPath, IOToolPythonBin(toolName))
}

func TestIOToolBin_ExistsInTempBase(t *testing.T) {
	withIOToolsBaseDir(t, t.TempDir())
	toolName := "whisper-test"
	binName := "whisper-cli"
	binPath := filepath.Join(IOToolVenvPath(toolName), "bin", binName)
	require.NoError(t, os.MkdirAll(filepath.Dir(binPath), 0755))
	require.NoError(t, os.WriteFile(binPath, []byte("x"), 0755))
	assert.Equal(t, binPath, IOToolBin(toolName, binName))
}

func TestRunUVFunc_NilEnv(t *testing.T) {
	orig := runUVFunc
	t.Cleanup(func() { runUVFunc = orig })
	runUVFunc = func(_ context.Context, args []string, env []string) error {
		assert.Nil(t, env)
		assert.Equal(t, []string{"venv", "--python", "3.12", "/tmp/venv"}, args)
		return nil
	}
	require.NoError(t, runUVFunc(context.Background(), []string{"venv", "--python", "3.12", "/tmp/venv"}, nil))
}

func TestEnsureVenv_InstallsPackagesWithMockedUV(t *testing.T) {
	orig := runUVFunc
	t.Cleanup(func() { runUVFunc = orig })
	runUVFunc = func(_ context.Context, args []string, _ []string) error {
		if args[0] == "venv" {
			venvPath := args[len(args)-1]
			bin := filepath.Join(venvPath, "bin")
			if err := os.MkdirAll(bin, 0755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(bin, "python"), []byte("x"), 0755)
		}
		return nil
	}
	m := NewManager(t.TempDir())
	venvPath, err := m.EnsureVenv("3.12", []string{"pkg-a"}, "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, venvPath)
}

func TestEnsureVenv_InstallsPackagesAndRequirementsWithMockedUV(t *testing.T) {
	orig := runUVFunc
	t.Cleanup(func() { runUVFunc = orig })
	runUVFunc = func(_ context.Context, args []string, _ []string) error {
		if args[0] == "venv" {
			venvPath := args[len(args)-1]
			bin := filepath.Join(venvPath, "bin")
			if err := os.MkdirAll(bin, 0755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(bin, "python"), []byte("x"), 0755)
		}
		return nil
	}
	m := NewManager(t.TempDir())
	req := filepath.Join(t.TempDir(), "requirements.txt")
	require.NoError(t, os.WriteFile(req, []byte("pkg-b\n"), 0644))
	venvPath, err := m.EnsureVenv("3.12", []string{"pkg-a"}, req, "")
	require.NoError(t, err)
	assert.NotEmpty(t, venvPath)
}

func TestEnsureVenv_BaseDirCreateFailure(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0644))
	m := NewManager(filepath.Join(blocker, "venv"))
	_, err := m.EnsureVenv("3.12", nil, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create base directory")
}

func TestEnsureVenv_InstallPackagesError(t *testing.T) {
	orig := runUVFunc
	t.Cleanup(func() { runUVFunc = orig })
	runUVFunc = func(_ context.Context, args []string, _ []string) error {
		if args[0] == "venv" {
			venvPath := args[len(args)-1]
			bin := filepath.Join(venvPath, "bin")
			if err := os.MkdirAll(bin, 0755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(bin, "python"), []byte("x"), 0755)
		}
		return errors.New("pip install failed")
	}
	m := NewManager(t.TempDir())
	_, err := m.EnsureVenv("3.12", []string{"pkg-a"}, "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to install packages")
}

func TestEnsureVenv_InstallRequirementsError(t *testing.T) {
	orig := runUVFunc
	t.Cleanup(func() { runUVFunc = orig })
	runUVFunc = func(_ context.Context, args []string, _ []string) error {
		if args[0] == "venv" {
			venvPath := args[len(args)-1]
			bin := filepath.Join(venvPath, "bin")
			if err := os.MkdirAll(bin, 0755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(bin, "python"), []byte("x"), 0755)
		}
		return errors.New("pip install failed")
	}
	m := NewManager(t.TempDir())
	req := filepath.Join(t.TempDir(), "requirements.txt")
	require.NoError(t, os.WriteFile(req, []byte("pkg-a\n"), 0644))
	_, err := m.EnsureVenv("3.12", nil, req, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to install requirements")
}

func TestEnsureVenv_InstallsRequirementsWithMockedUV(t *testing.T) {
	orig := runUVFunc
	t.Cleanup(func() { runUVFunc = orig })
	runUVFunc = func(_ context.Context, args []string, _ []string) error {
		if args[0] == "venv" {
			venvPath := args[len(args)-1]
			bin := filepath.Join(venvPath, "bin")
			if err := os.MkdirAll(bin, 0755); err != nil {
				return err
			}
			return os.WriteFile(filepath.Join(bin, "python"), []byte("x"), 0755)
		}
		return nil
	}
	m := NewManager(t.TempDir())
	req := filepath.Join(t.TempDir(), "requirements.txt")
	require.NoError(t, os.WriteFile(req, []byte("pkg-a\n"), 0644))
	venvPath, err := m.EnsureVenv("3.12", nil, req, "")
	require.NoError(t, err)
	assert.NotEmpty(t, venvPath)
}

func TestInstallTool_WithExtraArgs(t *testing.T) {
	m := NewManager(t.TempDir())
	err := m.InstallTool("__kdeps_missing_bin_xyz__", "nonexistent-pkg", "--no-build-isolation")
	require.Error(t, err)
}
