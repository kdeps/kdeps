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

func TestSettingsFromSelection_NoItems(t *testing.T) {
	// Zero discovered items + zero enabled → SelectAll: true so future items auto-enable.
	var allItems [numTabs][]Item
	s := SettingsFromSelection(Selection{}, allItems)
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

func TestSaveDefaultModel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	require.NoError(t, SaveDefaultModel("llama3.2:1b"))

	got, err := LoadSettings()
	require.NoError(t, err)
	assert.Equal(t, "llama3.2:1b", got.DefaultModel)
}

func TestSaveDefaultModel_UpdatesExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	require.NoError(t, SaveDefaultModel("first-model"))
	require.NoError(t, SaveDefaultModel("second-model"))

	got, err := LoadSettings()
	require.NoError(t, err)
	assert.Equal(t, "second-model", got.DefaultModel)
	assert.True(t, got.SelectAll, "SelectAll should remain default when not explicitly set")
}

func TestSettings_Save_PreservesOtherFields(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	s := Settings{
		SelectAll:        false,
		EnabledWorkflows: []string{"wf1"},
		DefaultModel:     "mymodel",
	}
	require.NoError(t, s.Save())

	got, err := LoadSettings()
	require.NoError(t, err)
	assert.Equal(t, "mymodel", got.DefaultModel)
	assert.Equal(t, []string{"wf1"}, got.EnabledWorkflows)
}

func TestLoadSettings_ReadError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a directory at the settings path so ReadFile fails with "is a directory"
	path := filepath.Join(home, ".kdeps", "agent-loop-settings.yaml")
	require.NoError(t, os.MkdirAll(path, 0o750))

	_, err := LoadSettings()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "settings: read")
}

func TestSettings_Save_WriteError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := filepath.Join(home, ".kdeps")
	require.NoError(t, os.MkdirAll(dir, 0o750))
	require.NoError(t, os.Chmod(dir, 0o500))
	defer func() { _ = os.Chmod(dir, 0o750) }()

	s := Settings{SelectAll: true}
	err := s.Save()
	assert.Error(t, err)
}

func TestSettings_Save_MkdirError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Place a file at ".kdeps" so MkdirAll fails (can't mkdir over a file).
	conflict := filepath.Join(home, ".kdeps")
	require.NoError(t, os.WriteFile(conflict, []byte("x"), 0o600))

	s := Settings{SelectAll: true}
	err := s.Save()
	assert.Error(t, err)
}

func TestSaveDefaultModel_LoadError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	path := filepath.Join(home, ".kdeps", "agent-loop-settings.yaml")
	require.NoError(t, os.MkdirAll(path, 0o750)) // dir at file path causes ReadFile to fail

	err := SaveDefaultModel("any")
	assert.Error(t, err)
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

func TestSettingsPath_HomeDirError(t *testing.T) {
	t.Setenv("HOME", "")

	path, err := settingsPath()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "settings: home dir")
	assert.Empty(t, path)
}

// TestLoadSettings_HomeDirError covers the settingsPath-error branch in LoadSettings.
// When HOME is unset, UserHomeDir fails; LoadSettings must return DefaultSettings with nil error.
func TestLoadSettings_HomeDirError(t *testing.T) {
	t.Setenv("HOME", "")

	s, err := LoadSettings()
	assert.NoError(t, err) // non-fatal: silently uses defaults
	assert.Equal(t, DefaultSettings(), s)
}

// TestSave_HomeDirError covers the settingsPath-error branch in Save.
func TestSave_HomeDirError(t *testing.T) {
	t.Setenv("HOME", "")

	err := Settings{SelectAll: true}.Save()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "settings: home dir")
}
