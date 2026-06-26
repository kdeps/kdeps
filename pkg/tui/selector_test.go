//go:build !js

package tui

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- model unit tests ---

func makeItems(n int) []Item {
	items := make([]Item, n)
	for i := range items {
		items[i] = Item{Name: "item", Path: "/tmp/item", Kind: KindWorkflow}
	}
	return items
}

func TestModel_Init(t *testing.T) {
	m := newModel([numTabs][]Item{})
	cmd := m.Init()
	assert.Nil(t, cmd)
}

func TestModel_QuitKey(t *testing.T) {
	m := newModel([numTabs][]Item{})
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	assert.True(t, out.(model).quitted)
}

func TestModel_CtrlC(t *testing.T) {
	m := newModel([numTabs][]Item{})
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	assert.True(t, out.(model).quitted)
}

func TestModel_Enter(t *testing.T) {
	m := newModel([numTabs][]Item{})
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, out.(model).quitted)
}

func TestModel_TabNavigation(t *testing.T) {
	m := newModel([numTabs][]Item{})
	assert.Equal(t, 0, m.tab)
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 1, out.(model).tab)
	out, _ = out.Update(tea.KeyMsg{Type: tea.KeyTab})
	out, _ = out.Update(tea.KeyMsg{Type: tea.KeyTab})
	out, _ = out.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 0, out.(model).tab) // wraps around
}

func TestModel_ShiftTabNavigation(t *testing.T) {
	m := newModel([numTabs][]Item{})
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	assert.Equal(t, numTabs-1, out.(model).tab)
}

func TestModel_CursorDown(t *testing.T) {
	var tabs [numTabs][]Item
	tabs[0] = makeItems(3)
	m := newModel(tabs)
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("j")})
	assert.Equal(t, 1, out.(model).cursor)
}

func TestModel_CursorUp(t *testing.T) {
	var tabs [numTabs][]Item
	tabs[0] = makeItems(3)
	m := newModel(tabs)
	m.cursor = 2
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("k")})
	assert.Equal(t, 1, out.(model).cursor)
}

func TestModel_CursorBounds(t *testing.T) {
	m := newModel([numTabs][]Item{})
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, out.(model).cursor) // empty tab, stays at 0

	var tabs [numTabs][]Item
	tabs[0] = makeItems(1)
	m2 := newModel(tabs)
	out2, _ := m2.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 0, out2.(model).cursor) // only 1 item, stays at 0
}

func TestModel_SpaceToggle(t *testing.T) {
	var tabs [numTabs][]Item
	tabs[0] = makeItems(2)
	m := newModel(tabs)
	assert.False(t, m.tabs[0][0].Enabled)

	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	assert.True(t, out.(model).tabs[0][0].Enabled)

	out2, _ := out.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	assert.False(t, out2.(model).tabs[0][0].Enabled)
}

func TestModel_SpaceOnEmpty(t *testing.T) {
	m := newModel([numTabs][]Item{})
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	// no panic, cursor stays at 0
	assert.Equal(t, 0, out.(model).cursor)
}

func TestModel_TabSwitchResetsCursor(t *testing.T) {
	var tabs [numTabs][]Item
	tabs[0] = makeItems(3)
	m := newModel(tabs)
	m.cursor = 2
	out, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	assert.Equal(t, 0, out.(model).cursor)
}

func TestModel_View_Empty(t *testing.T) {
	m := newModel([numTabs][]Item{})
	v := m.View()
	assert.Contains(t, v, "none installed")
}

func TestModel_View_Items(t *testing.T) {
	var tabs [numTabs][]Item
	tabs[0] = []Item{{Name: "my-workflow", Path: "/tmp/wf.yaml", Kind: KindWorkflow, Enabled: true}}
	m := newModel(tabs)
	v := m.View()
	assert.Contains(t, v, "my-workflow")
	assert.Contains(t, v, "[x]")
}

func TestModel_ToSelection(t *testing.T) {
	var tabs [numTabs][]Item
	tabs[tabWorkflows] = []Item{
		{Name: "wf1", Enabled: true},
		{Name: "wf2", Enabled: false},
	}
	tabs[tabAgencies] = []Item{
		{Name: "ag1", Enabled: true},
	}
	tabs[tabComponents] = []Item{
		{Name: "comp1", Enabled: false},
	}
	tabs[tabSkills] = []Item{
		{Name: "sk1", Enabled: true},
		{Name: "sk2", Enabled: true},
	}
	m := newModel(tabs)
	sel := m.toSelection()
	require.Len(t, sel.Workflows, 1)
	assert.Equal(t, "wf1", sel.Workflows[0].Name)
	require.Len(t, sel.Agencies, 1)
	assert.Equal(t, "ag1", sel.Agencies[0].Name)
	assert.Empty(t, sel.Components)
	require.Len(t, sel.Skills, 2)
}

// --- discoverItems tests ---

func TestDiscoverItems_Empty(t *testing.T) {
	// When all dirs point to empty temp dirs, results should be empty slices.
	dir := t.TempDir()
	t.Setenv("KDEPS_AGENTS_DIR", dir)
	t.Setenv("KDEPS_COMPONENTS_DIR", dir)
	t.Setenv("KDEPS_SKILL_DIRS", dir)

	items := discoverItems()

	for _, tab := range items {
		assert.Empty(t, tab)
	}
}

func setIsolatedDirs(t *testing.T, agentsDir string) {
	t.Helper()
	empty := t.TempDir()
	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)
	t.Setenv("KDEPS_COMPONENTS_DIR", empty)
	t.Setenv("KDEPS_SKILL_DIRS", empty)
}

func TestDiscoverItems_Workflow(t *testing.T) {
	agentsDir := t.TempDir()
	setIsolatedDirs(t, agentsDir)

	wfDir := filepath.Join(agentsDir, "my-wf")
	require.NoError(t, os.MkdirAll(wfDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(wfDir, "workflow.yaml"), []byte(""), 0o644))

	items := discoverItems()

	require.Len(t, items[tabWorkflows], 1)
	assert.Equal(t, "my-wf", items[tabWorkflows][0].Name)
	assert.Equal(t, KindWorkflow, items[tabWorkflows][0].Kind)
}

func TestDiscoverItems_Agency(t *testing.T) {
	agentsDir := t.TempDir()
	setIsolatedDirs(t, agentsDir)

	agDir := filepath.Join(agentsDir, "my-agency")
	require.NoError(t, os.MkdirAll(agDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(agDir, "agency.yaml"), []byte(""), 0o644))

	items := discoverItems()

	require.Len(t, items[tabAgencies], 1)
	assert.Equal(t, "my-agency", items[tabAgencies][0].Name)
	assert.Equal(t, KindAgency, items[tabAgencies][0].Kind)
}

func TestDiscoverItems_AgencyTakesPrecedenceOverWorkflow(t *testing.T) {
	agentsDir := t.TempDir()
	setIsolatedDirs(t, agentsDir)

	dir := filepath.Join(agentsDir, "both")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	// Both agency.yaml and workflow.yaml present — agency wins
	require.NoError(t, os.WriteFile(filepath.Join(dir, "agency.yaml"), []byte(""), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yaml"), []byte(""), 0o644))

	items := discoverItems()

	assert.Len(t, items[tabAgencies], 1)
	assert.Empty(t, items[tabWorkflows])
}

func TestDiscoverItems_Component(t *testing.T) {
	agentsDir := t.TempDir()
	compsDir := t.TempDir()
	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)
	t.Setenv("KDEPS_COMPONENTS_DIR", compsDir)
	t.Setenv("KDEPS_SKILL_DIRS", agentsDir)

	cDir := filepath.Join(compsDir, "my-comp")
	require.NoError(t, os.MkdirAll(cDir, 0o755))

	items := discoverItems()

	require.Len(t, items[tabComponents], 1)
	assert.Equal(t, "my-comp", items[tabComponents][0].Name)
	assert.Equal(t, KindComponent, items[tabComponents][0].Kind)
}

func TestDiscoverItems_Skill(t *testing.T) {
	agentsDir := t.TempDir()
	skillsDir := t.TempDir()
	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)
	t.Setenv("KDEPS_COMPONENTS_DIR", agentsDir)
	t.Setenv("KDEPS_SKILL_DIRS", skillsDir)

	skDir := filepath.Join(skillsDir, "my-skill")
	require.NoError(t, os.MkdirAll(skDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(skDir, "SKILL.md"), []byte("# skill"), 0o644))

	items := discoverItems()

	require.Len(t, items[tabSkills], 1)
	assert.Equal(t, "my-skill", items[tabSkills][0].Name)
	assert.Equal(t, KindSkill, items[tabSkills][0].Kind)
}

func TestSkillSearchDirs_Override(t *testing.T) {
	t.Setenv("KDEPS_SKILL_DIRS", "/a:/b:/c")
	dirs := skillSearchDirs("/home/user")
	assert.Equal(t, []string{"/a", "/b", "/c"}, dirs)
}

func TestSkillSearchDirs_Default(t *testing.T) {
	t.Setenv("KDEPS_SKILL_DIRS", "")
	dirs := skillSearchDirs("/home/user")
	assert.Contains(t, dirs, "/home/user/.kdeps/skills")
	assert.Contains(t, dirs, "/home/user/.agents/skills")
	assert.Contains(t, dirs, "/home/user/.claude/skills")
}

func TestDirFromEnv_EnvSet(t *testing.T) {
	t.Setenv("MY_TEST_DIR", "/custom")
	assert.Equal(t, "/custom", dirFromEnv("MY_TEST_DIR", "/home", "sub"))
}

func TestDirFromEnv_EnvUnset(t *testing.T) {
	t.Setenv("MY_TEST_DIR2", "")
	assert.Equal(t, "/home/sub", dirFromEnv("MY_TEST_DIR2", "/home", "sub"))
}

func TestHasFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "test.yaml")
	require.NoError(t, os.WriteFile(f, []byte(""), 0o644))

	assert.True(t, hasFile(dir, "test.yaml"))
	assert.False(t, hasFile(dir, "nope.yaml"))
	assert.True(t, hasFile(dir, "nope.yaml", "test.yaml"))
}

func TestAgentsDirFromEnv(t *testing.T) {
	t.Setenv("KDEPS_AGENTS_DIR", "/custom/agents")
	assert.Equal(t, "/custom/agents", agentsDirFromEnv("/home/user"))

	t.Setenv("KDEPS_AGENTS_DIR", "")
	assert.Equal(t, "/home/user/.kdeps/agents", agentsDirFromEnv("/home/user"))
}

func TestModel_Update_NonKeyMsg(t *testing.T) {
	m := newModel([numTabs][]Item{})
	// Send a non-key message (e.g. WindowSizeMsg); should be a no-op.
	out, cmd := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	assert.Nil(t, cmd)
	assert.False(t, out.(model).quitted)
}

func TestDiscoverAgentsDir_WorkflowYml(t *testing.T) {
	agentsDir := t.TempDir()
	setIsolatedDirs(t, agentsDir)

	dir := filepath.Join(agentsDir, "my-wf-yml")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "workflow.yml"), []byte(""), 0o644))

	items := discoverItems()
	require.Len(t, items[tabWorkflows], 1)
	assert.Equal(t, "my-wf-yml", items[tabWorkflows][0].Name)
	assert.Equal(t, KindWorkflow, items[tabWorkflows][0].Kind)
}

func TestDiscoverAgentsDir_SkipsFiles(t *testing.T) {
	agentsDir := t.TempDir()
	setIsolatedDirs(t, agentsDir)

	// Place a regular file (not dir) in the agents dir - should be skipped.
	require.NoError(t, os.WriteFile(filepath.Join(agentsDir, "notadir.yaml"), []byte(""), 0o644))

	items := discoverItems()
	assert.Empty(t, items[tabWorkflows])
	assert.Empty(t, items[tabAgencies])
}

func TestDiscoverComponentsDir_SkipsFiles(t *testing.T) {
	agentsDir := t.TempDir()
	compsDir := t.TempDir()
	t.Setenv("KDEPS_AGENTS_DIR", agentsDir)
	t.Setenv("KDEPS_COMPONENTS_DIR", compsDir)
	t.Setenv("KDEPS_SKILL_DIRS", agentsDir)

	// Place a regular file (not dir) in the components dir - should be skipped.
	require.NoError(t, os.WriteFile(filepath.Join(compsDir, "notadir.yaml"), []byte(""), 0o644))

	items := discoverItems()
	assert.Empty(t, items[tabComponents])
}

// TestDiscoverItems_HomeDirError covers the homeErr branch in discoverItems.
// When HOME is unset, UserHomeDir fails and the function returns empty items.
func TestDiscoverItems_HomeDirError(t *testing.T) {
	t.Setenv("HOME", "")
	items := discoverItems()
	for _, tab := range items {
		assert.Empty(t, tab)
	}
}
