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

package agent

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v3/pkg/domain"
)

func TestLoadBuiltinPresets_HasEveryShippedPreset(t *testing.T) {
	built := loadBuiltinPresets()
	want := []string{"frugal", "balanced", "thorough", "design-review"}
	require.Len(t, built, len(want))
	for _, name := range want {
		require.Contains(t, built, name)
		assert.NotEmpty(t, built[name].Description, "preset %q must have a description", name)
		assert.True(t, len(built[name].Events) > 0 || len(built[name].Judges) > 0,
			"preset %q must tune an event or set a judge panel", name)
	}
}

func TestPresetByName_UnknownReturnsFalse(t *testing.T) {
	_, ok := PresetByName("does-not-exist")
	assert.False(t, ok)
}

func TestPresetByName_KnownReturnsTrue(t *testing.T) {
	p, ok := PresetByName("frugal")
	require.True(t, ok)
	assert.Equal(t, "frugal", p.Name)
}

// isolateEventsAndPresetsHome mirrors isolateEventsHome/isolateActionsHome,
// isolating both registries since ApplyPreset touches both.
func isolateEventsAndPresetsHome(t *testing.T) {
	t.Helper()
	// Registered before t.Setenv below so it runs LAST -- see
	// isolateEventsHome (events_test.go) for why the order matters.
	t.Cleanup(initEvents)
	t.Cleanup(initPresets)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// Load the registries under the isolated HOME now: the package-level init()
	// already loaded them from the developer's real ~/.kdeps, and a user event
	// override (e.g. efficiency-failures rounds) would otherwise leak into the
	// test's expected defaults.
	initEvents()
	initPresets()
	// Registered after the temp dirs: t.Cleanup runs LIFO, so this waits for
	// lazy index writes before the home dir's RemoveAll runs. Without it a
	// fire-and-forget edit_file/read_file index write races the cleanup
	// ("directory not empty" on .kdeps/projects/<name>-<hash>).
	t.Cleanup(WaitForBackgroundIndexing)
}

func TestInitPresets_UserOverrideReplacesBuiltinByName(t *testing.T) {
	isolateEventsAndPresetsHome(t)
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	dir := filepath.Join(home, ".kdeps", "presets")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "frugal.yaml"), []byte(`
name: frugal
description: custom override
`), 0o644))

	initPresets()

	p, ok := PresetByName("frugal")
	require.True(t, ok)
	assert.Equal(t, "custom override", p.Description)
}

func TestInitPresets_UserFileAddsNewPreset(t *testing.T) {
	isolateEventsAndPresetsHome(t)
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	dir := filepath.Join(home, ".kdeps", "presets")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "custom.yaml"), []byte(`
name: my-custom-preset
description: a user-defined preset
`), 0o644))

	initPresets()

	p, ok := PresetByName("my-custom-preset")
	require.True(t, ok)
	assert.Equal(t, "a user-defined preset", p.Description)
}

func TestParseYAMLPreset_ReadsJudges(t *testing.T) {
	p, err := parseYAMLPreset([]byte(`name: panel
judges:
  - name: scope
    criteria: solves the stated problem
    allowedTools: [read_file]
    maxRounds: 3
`), "panel.yaml")
	require.NoError(t, err)
	want := []domain.Judge{{
		Name: "scope", Criteria: "solves the stated problem", Tools: []string{"read_file"}, MaxRounds: 3,
	}}
	assert.Equal(t, want, p.Judges)
}

func TestParseYAMLPreset_JudgeNeedsNameAndCriteria(t *testing.T) {
	_, err := parseYAMLPreset([]byte("name: panel\njudges:\n  - name: scope\n"), "panel.yaml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "judge 1")
}
