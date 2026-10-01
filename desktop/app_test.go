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

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/desktop"
	"github.com/kdeps/kdeps/v2/pkg/executor"
)

func newTestApp(t *testing.T, statePath string) *App {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	t.Setenv("KDEPS_WORKSPACE_ROOT", "")

	app := NewApp(statePath)
	require.NoError(t, app.init(t.Context(), func(desktop.Event) {}, executor.NewEngine(nil)))
	return app
}

func TestApp_InitFallsBackToHome(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "state.json"))
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Equal(t, home, app.Workspace())
	assert.Empty(t, app.Recent())
}

func TestApp_OpenWorkspacePersistsAndRestores(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	app := newTestApp(t, statePath)
	dir := t.TempDir()

	require.NoError(t, app.OpenWorkspace(dir))
	assert.Equal(t, dir, app.Workspace())
	assert.Equal(t, []string{dir}, app.Recent())

	again := NewApp(statePath)
	require.NoError(t, again.init(t.Context(), func(desktop.Event) {}, executor.NewEngine(nil)))
	assert.Equal(t, dir, again.Workspace(), "last workspace is restored")
}

func TestApp_OpenWorkspaceRejectsBadDir(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "state.json"))
	assert.Error(t, app.OpenWorkspace(filepath.Join(t.TempDir(), "missing")))
	assert.Empty(t, app.Recent())
}

func TestApp_OpenWorkspaceSurfacesSaveError(t *testing.T) {
	blocker := filepath.Join(t.TempDir(), "blocker")
	require.NoError(t, os.WriteFile(blocker, []byte("x"), 0o600))
	app := newTestApp(t, filepath.Join(blocker, "state.json"))
	assert.Error(t, app.OpenWorkspace(t.TempDir()))
}

func TestApp_ReadyReportsInitResult(t *testing.T) {
	app := newTestApp(t, filepath.Join(t.TempDir(), "state.json"))
	close(app.ready)
	assert.NoError(t, app.Ready())

	failed := NewApp("")
	failed.initErr = os.ErrInvalid
	close(failed.ready)
	assert.ErrorIs(t, failed.Ready(), os.ErrInvalid)
}
