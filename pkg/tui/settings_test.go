//go:build !js

package tui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultSettings(t *testing.T) {
	s := DefaultSettings()
	assert.True(t, s.SelectAll)
}

func TestSettings_SaveLoad(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	s := Settings{
		SelectAll:        false,
		EnabledWorkflows: []string{"wf1", "wf2"},
		EnabledSkills:    []string{"lint"},
	}
	require.NoError(t, s.Save())

	got, err := LoadSettings()
	require.NoError(t, err)
	assert.False(t, got.SelectAll)
	assert.Equal(t, []string{"wf1", "wf2"}, got.EnabledWorkflows)
	assert.Equal(t, []string{"lint"}, got.EnabledSkills)
}

func TestLoadSettings_Missing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	s, err := LoadSettings()
	require.NoError(t, err)
	assert.True(t, s.SelectAll)
}

func TestLoadSettings_CorruptFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".kdeps", "agent-loop-settings.yaml")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(":\tinvalid: yaml: {["), 0o600))

	_, err := LoadSettings()
	assert.Error(t, err)
}

func TestApplyToModel_SelectAll(t *testing.T) {
	var items [numTabs][]Item
	items[tabWorkflows] = []Item{{Name: "wf1"}, {Name: "wf2"}}
	items[tabSkills] = []Item{{Name: "sk1"}}

	m := applyToModel(newModel(items), Settings{SelectAll: true})
	for _, it := range m.tabs[tabWorkflows] {
		assert.True(t, it.Enabled)
	}
	for _, it := range m.tabs[tabSkills] {
		assert.True(t, it.Enabled)
	}
}

func TestApplyToModel_PartialSelection(t *testing.T) {
	var items [numTabs][]Item
	items[tabWorkflows] = []Item{{Name: "wf1"}, {Name: "wf2"}}
	items[tabSkills] = []Item{{Name: "sk1"}, {Name: "sk2"}}

	s := Settings{
		SelectAll:        false,
		EnabledWorkflows: []string{"wf1"},
		EnabledSkills:    []string{"sk2"},
	}
	m := applyToModel(newModel(items), s)
	assert.True(t, m.tabs[tabWorkflows][0].Enabled)  // wf1
	assert.False(t, m.tabs[tabWorkflows][1].Enabled) // wf2
	assert.False(t, m.tabs[tabSkills][0].Enabled)    // sk1
	assert.True(t, m.tabs[tabSkills][1].Enabled)     // sk2
}

func TestSettingsFromSelection_AllEnabled(t *testing.T) {
	var allItems [numTabs][]Item
	allItems[tabWorkflows] = []Item{{Name: "wf1"}}
	allItems[tabSkills] = []Item{{Name: "sk1"}}

	sel := Selection{
		Workflows: []Item{{Name: "wf1"}},
		Skills:    []Item{{Name: "sk1"}},
	}

	s := SettingsFromSelection(sel, allItems)
	assert.True(t, s.SelectAll)
}

func TestSettingsFromSelection_PartialEnabled(t *testing.T) {
	var allItems [numTabs][]Item
	allItems[tabWorkflows] = []Item{{Name: "wf1"}, {Name: "wf2"}}

	sel := Selection{
		Workflows: []Item{{Name: "wf1"}},
	}

	s := SettingsFromSelection(sel, allItems)
	assert.False(t, s.SelectAll)
	assert.Equal(t, []string{"wf1"}, s.EnabledWorkflows)
}

func TestSelectionFromSettings_SelectAll(t *testing.T) {
	agentsDir := t.TempDir()
	wfDir := filepath.Join(agentsDir, "my-wf")
	require.NoError(t, os.MkdirAll(wfDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "workflow.yaml"), []byte(""), 0o644))

	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)
	t.Setenv("KDEPS_COMPONENTS_DIR", agentsDir)
	t.Setenv("KDEPS_SKILL_DIRS", agentsDir)

	sel := SelectionFromSettings(Settings{SelectAll: true})
	require.Len(t, sel.Workflows, 1)
	assert.Equal(t, "my-wf", sel.Workflows[0].Name)
}

func TestSelectionFromSettings_Filtered(t *testing.T) {
	agentsDir := t.TempDir()
	for _, name := range []string{"wf-a", "wf-b"} {
		dir := filepath.Join(agentsDir, name)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(""), 0o644))
	}
	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)
	t.Setenv("KDEPS_COMPONENTS_DIR", agentsDir)
	t.Setenv("KDEPS_SKILL_DIRS", agentsDir)

	sel := SelectionFromSettings(Settings{
		SelectAll:        false,
		EnabledWorkflows: []string{"wf-a"},
	})
	require.Len(t, sel.Workflows, 1)
	assert.Equal(t, "wf-a", sel.Workflows[0].Name)
}
