//go:build !js

package tui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ItemKind classifies a selectable item in the startup TUI.
type ItemKind int

const (
	KindWorkflow ItemKind = iota
	KindAgency
	KindComponent
	KindSkill
)

// Item is a single selectable tool/skill entry.
type Item struct {
	Name    string
	Path    string
	Kind    ItemKind
	Enabled bool
}

// Selection is the result of the TUI; contains enabled items per category.
type Selection struct {
	Workflows  []Item
	Agencies   []Item
	Components []Item
	Skills     []Item
}

const (
	numTabs         = 4
	tabWorkflows    = 0
	tabAgencies     = 1
	tabComponents   = 2
	tabSkills       = 3
	tabBarSeparator = 60
	tabPadH         = 2 // horizontal padding for tab labels
	helpText        = "↑/↓ navigate  space toggle  tab/shift+tab switch section  enter confirm  q quit"
)

//nolint:gochecknoglobals // lipgloss styles must be package-level vars
var (
	styleBase        = lipgloss.NewStyle()
	styleTab         = lipgloss.NewStyle().Padding(0, tabPadH)
	styleTabSel      = lipgloss.NewStyle().Padding(0, tabPadH).Bold(true).Foreground(lipgloss.Color("#00E5FF"))
	styleEnabled     = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E5FF"))
	styleDim         = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555"))
	styleHelp        = lipgloss.NewStyle().Foreground(lipgloss.Color("#555555")).Italic(true)
	styleCursor      = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E5FF")).Bold(true)
	styleAccent      = lipgloss.NewStyle().Foreground(lipgloss.Color("#00E5FF"))
	styleGroupHeader = lipgloss.NewStyle().Foreground(lipgloss.Color("#7AA2F7")).Bold(true)
)

//nolint:gochecknoglobals // const slice
var tabLabels = [numTabs]string{"Workflows", "Agencies", "Components", "Skills"}

// model is the bubbletea model for the startup selector.
type model struct {
	tabs    [numTabs][]Item
	tab     int
	cursor  int
	quitted bool
}

func newModel(items [numTabs][]Item) model {
	return model{tabs: items}
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "q", "ctrl+c":
		m.quitted = true
		return m, tea.Quit
	case "enter":
		return m, tea.Quit
	case "tab":
		m.tab = (m.tab + 1) % numTabs
		m.cursor = 0
	case "shift+tab":
		m.tab = (m.tab + numTabs - 1) % numTabs
		m.cursor = 0
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.tabs[m.tab])-1 {
			m.cursor++
		}
	case " ":
		if len(m.tabs[m.tab]) > 0 {
			m.tabs[m.tab][m.cursor].Enabled = !m.tabs[m.tab][m.cursor].Enabled
		}
	}
	return m, nil
}

func (m model) View() string {
	var sb strings.Builder
	m.renderTabBar(&sb)
	m.renderItems(&sb)
	fmt.Fprintf(&sb, "\n%s\n", styleHelp.Render(helpText))
	return styleBase.Render(sb.String())
}

func (m model) renderTabBar(sb *strings.Builder) {
	for i, label := range tabLabels {
		count := len(m.tabs[i])
		enabled := countEnabled(m.tabs[i])
		badge := fmt.Sprintf("%s (%d/%d)", label, enabled, count)
		if i == m.tab {
			sb.WriteString(styleTabSel.Render("[ " + badge + " ]"))
		} else {
			sb.WriteString(styleTab.Render(badge))
		}
	}
	fmt.Fprintf(sb, "\n%s\n\n", strings.Repeat("-", tabBarSeparator))
}

func countEnabled(items []Item) int {
	n := 0
	for _, it := range items {
		if it.Enabled {
			n++
		}
	}
	return n
}

func (m model) renderItems(sb *strings.Builder) {
	items := m.tabs[m.tab]
	if len(items) == 0 {
		fmt.Fprintf(sb, "%s\n", styleDim.Render("  (none installed)"))
		return
	}
	for i, it := range items {
		checkbox := "[ ]"
		nameStyle := styleBase
		if it.Enabled {
			checkbox = "[x]"
			nameStyle = styleEnabled
		}
		cursor := "  "
		if i == m.cursor {
			cursor = styleCursor.Render("> ")
		}
		fmt.Fprintf(sb, "%s%s %s\n", cursor, checkbox, nameStyle.Render(it.Name))
	}
}

func (m model) toSelection() Selection {
	sel := Selection{}
	for _, it := range m.tabs[tabWorkflows] {
		if it.Enabled {
			sel.Workflows = append(sel.Workflows, it)
		}
	}
	for _, it := range m.tabs[tabAgencies] {
		if it.Enabled {
			sel.Agencies = append(sel.Agencies, it)
		}
	}
	for _, it := range m.tabs[tabComponents] {
		if it.Enabled {
			sel.Components = append(sel.Components, it)
		}
	}
	for _, it := range m.tabs[tabSkills] {
		if it.Enabled {
			sel.Skills = append(sel.Skills, it)
		}
	}
	return sel
}

// Run shows the settings TUI with persisted selections pre-applied.
// Saves the resulting selection to disk. Returns the selection and updated settings.
func Run() (Selection, Settings, error) {
	items := discoverItems()
	settings, err := LoadSettings()
	if err != nil {
		settings = DefaultSettings()
	}

	m := applyToModel(newModel(items), settings)
	p := tea.NewProgram(m)
	final, runErr := p.Run()
	if runErr != nil {
		return Selection{}, settings, fmt.Errorf("tui: %w", runErr)
	}

	fm, ok := final.(model)
	if !ok || fm.quitted {
		return Selection{}, settings, nil
	}

	sel := fm.toSelection()
	newSettings := SettingsFromSelection(sel, items)
	if saveErr := newSettings.Save(); saveErr != nil {
		// non-fatal: warn but don't block the user
		fmt.Fprintf(os.Stderr, "warning: could not save settings: %v\n", saveErr)
	}
	return sel, newSettings, nil
}

// discoverItems scans configured directories for installed workflows, agencies,
// components, and skills available as tools for the agent loop.
func discoverItems() [numTabs][]Item {
	var items [numTabs][]Item

	home, homeErr := os.UserHomeDir()
	if homeErr != nil {
		return items
	}

	agentsDir := dirFromEnv("KDEPS_AGENTS_DIR", home, ".kdeps", "agents")
	compsDir := dirFromEnv("KDEPS_COMPONENTS_DIR", home, ".kdeps", "components")

	discoverAgentsDir(agentsDir, &items)
	discoverComponentsDir(compsDir, &items)
	discoverSkills(skillSearchDirs(home), &items)

	return items
}

func discoverAgentsDir(agentsDir string, items *[numTabs][]Item) {
	entries, _ := os.ReadDir(agentsDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		dir := filepath.Join(agentsDir, e.Name())
		switch {
		case hasFile(dir, "agency.yaml"):
			items[tabAgencies] = append(items[tabAgencies], Item{
				Name: e.Name(), Path: filepath.Join(dir, "agency.yaml"), Kind: KindAgency,
			})
		case hasFile(dir, "workflow.yaml"):
			items[tabWorkflows] = append(items[tabWorkflows], Item{
				Name: e.Name(), Path: filepath.Join(dir, "workflow.yaml"), Kind: KindWorkflow,
			})
		case hasFile(dir, "workflow.yml"):
			items[tabWorkflows] = append(items[tabWorkflows], Item{
				Name: e.Name(), Path: filepath.Join(dir, "workflow.yml"), Kind: KindWorkflow,
			})
		}
	}
}

func discoverComponentsDir(compsDir string, items *[numTabs][]Item) {
	entries, _ := os.ReadDir(compsDir)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		items[tabComponents] = append(items[tabComponents], Item{
			Name: e.Name(), Path: filepath.Join(compsDir, e.Name()), Kind: KindComponent,
		})
	}
}

func discoverSkills(dirs []string, items *[numTabs][]Item) {
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil || d.IsDir() {
				return nil //nolint:nilerr // skip unreadable entries silently
			}
			if strings.EqualFold(d.Name(), "SKILL.md") {
				name := filepath.Base(filepath.Dir(path))
				items[tabSkills] = append(items[tabSkills], Item{
					Name: name, Path: path, Kind: KindSkill,
				})
			}
			return nil
		})
	}
}

// skillSearchDirs returns the list of directories to search for SKILL.md files.
// KDEPS_SKILL_DIRS overrides the defaults (colon-separated list).
func skillSearchDirs(home string) []string {
	if override := os.Getenv("KDEPS_SKILL_DIRS"); override != "" {
		return strings.Split(override, ":")
	}
	return []string{
		filepath.Join(home, ".kdeps", "skills"),
		filepath.Join(home, ".agents", "skills"),
		filepath.Join(home, ".claude", "skills"),
		".",
	}
}

func dirFromEnv(envKey, home string, relParts ...string) string {
	if d := os.Getenv(envKey); d != "" {
		return d
	}
	parts := append([]string{home}, relParts...)
	return filepath.Join(parts...)
}

func agentsDirFromEnv(home string) string {
	return dirFromEnv("KDEPS_AGENTS_DIR", home, ".kdeps", "agents")
}

func hasFile(dir string, names ...string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
}
