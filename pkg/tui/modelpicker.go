//go:build !js

package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	modelPickerWidth   = 90
	modelPickerColumns = 2 // side-by-side columns in the model grid
	modelPickerPadding = 4 // total horizontal padding subtracted before dividing into columns

	modelTypeLLamafile = "llamafile"
	modelTypeGGUF      = "gguf"
	modelTypeOllama    = "ollama"
)

// maxNameLen is the maximum length of a model name before truncation.
const maxNameLen = 35

// ModelEntry is a selectable model in the picker.
type ModelEntry struct {
	Name      string
	ModelType string // "llamafile", "gguf", "ollama", "" (cloud)
	Backend   string // cloud backend name (e.g. "deepseek"), or ""
	Cached    bool
	Enabled   bool   // cloud API key is set
	SizeGB    string // formatted size string, or ""
}

// group label ordering and labels.
const (
	groupCached    = 0
	groupLLamafile = 1
	groupGGUF      = 2
	groupOllama    = 3
	groupCloud     = 4

	labelCached    = "📦 Cached (Downloaded)"
	labelLLamafile = "🦙 LLamafile"
	labelGGUF      = "📄 GGUF"
	labelOllama    = "🦙 Ollama"
	labelCloud     = "☁️  Cloud APIs"
)

// modelPickerModel holds the full state of the picker.
type modelPickerModel struct {
	entries   []ModelEntry
	groups    []modelGroup
	cursor    int
	filter    string
	quitted   bool
	cancelled bool
}

type modelGroup struct {
	label   string
	entries []ModelEntry
}

func newModelPickerModel(entries []ModelEntry, preFilter string) modelPickerModel {
	// Sort globally first: cached, llamafile, gguf, ollama, cloud
	sort.Slice(entries, func(i, j int) bool {
		order := func(e ModelEntry) int {
			if e.Cached {
				return groupCached
			}
			switch e.ModelType {
			case modelTypeLLamafile:
				return groupLLamafile
			case modelTypeGGUF:
				return groupGGUF
			case modelTypeOllama:
				return groupOllama
			}
			return groupCloud
		}
		oi, oj := order(entries[i]), order(entries[j])
		if oi != oj {
			return oi < oj
		}
		return entries[i].Name < entries[j].Name
	})

	// Build groups
	groups := []modelGroup{
		{label: labelCached},
		{label: labelLLamafile},
		{label: labelGGUF},
		{label: labelOllama},
		{label: labelCloud},
	}
	for _, e := range entries {
		var idx int
		if e.Cached {
			idx = 0
		} else {
			switch e.ModelType {
			case modelTypeLLamafile:
				idx = 1
			case modelTypeGGUF:
				idx = 2
			case modelTypeOllama:
				idx = 3
			default:
				idx = 4
			}
		}
		groups[idx].entries = append(groups[idx].entries, e)
	}

	return modelPickerModel{
		entries: entries,
		groups:  groups,
		filter:  preFilter,
	}
}

func (m modelPickerModel) filtered() []ModelEntry {
	if m.filter == "" {
		return m.entries
	}
	var out []ModelEntry
	lower := strings.ToLower(m.filter)
	for _, e := range m.entries {
		if strings.Contains(strings.ToLower(e.Name), lower) {
			out = append(out, e)
		}
	}
	return out
}

func (m modelPickerModel) Init() tea.Cmd { return nil }

func (m modelPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	flat := m.flatIndexed()
	total := len(flat)

	switch key.String() {
	case "ctrl+c", "esc":
		m.cancelled = true
		m.quitted = true
		return m, tea.Quit
	case "enter":
		if total > 0 && m.cursor < total {
			m.quitted = true
			return m, tea.Quit
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < total-1 {
			m.cursor++
		}
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.cursor = 0
		}
	default:
		s := key.String()
		if len(s) == 1 && s >= " " && s <= "~" {
			m.filter += s
			m.cursor = 0
		}
	}
	return m, nil
}

// flatIndexed returns a deduplicated, ordered slice for navigation.
func (m modelPickerModel) flatIndexed() []ModelEntry {
	flat := m.filtered()
	return flat
}

func (m modelPickerModel) View() string {
	var sb strings.Builder

	// Header
	fmt.Fprintf(&sb, "%s", styleAccent.Render("╭"+strings.Repeat("─", modelPickerWidth)+"╮"))
	sb.WriteString("\n")
	const pickerHelpText = "│ Model Picker — type to filter, ↑↓/jk navigate, enter select, esc cancel │"
	fmt.Fprintf(&sb, "%s", styleDim.Render(pickerHelpText))
	if m.filter != "" {
		badge := styleAccent.Render(" filter: " + m.filter + " ")
		sb.WriteString("│ " + badge + "\n")
	}
	fmt.Fprintf(&sb, "%s", styleAccent.Render("╰"+strings.Repeat("─", modelPickerWidth)+"╯"))
	sb.WriteString("\n\n")

	colWidth := (modelPickerWidth - modelPickerPadding) / modelPickerColumns

	// Render entries grouped
	flat := m.flatIndexed()
	flatIdx := 0
	flatSet := make(map[string]bool)
	for _, e := range flat {
		flatSet[e.Name] = true
	}

	for _, g := range m.groups {
		var visible []ModelEntry
		for _, e := range g.entries {
			if flatSet[e.Name] {
				visible = append(visible, e)
			}
		}
		if len(visible) == 0 {
			continue
		}
		sb.WriteString(styleGroupHeader.Render("▎" + g.label + fmt.Sprintf(" (%d)", len(visible))))
		sb.WriteString("\n")

		// Two-column layout
		for i := 0; i < len(visible); i += modelPickerColumns {
			left := m.renderCompactRow(visible[i], flatIdx == m.cursor, colWidth)
			if i+1 < len(visible) {
				right := m.renderCompactRow(visible[i+1], flatIdx+1 == m.cursor, colWidth)
				fmt.Fprintf(&sb, "%s  %s\n", left, right)
				flatIdx += modelPickerColumns
			} else {
				sb.WriteString(left + "\n")
				flatIdx++
			}
		}
		sb.WriteString("\n")
	}

	// Footer
	footer := fmt.Sprintf(" %d models ", len(m.filtered()))
	if len(m.filtered()) > 0 && m.cursor < len(m.filtered()) {
		sel := m.filtered()[m.cursor]
		footer += fmt.Sprintf("· %s %s", sel.Name, tagForEntry(sel))
		if sel.SizeGB != "" {
			footer += " " + sel.SizeGB + "GB"
		}
	}
	fmt.Fprintf(&sb, "%s", styleDim.Render(footer))

	return lipgloss.NewStyle().Padding(0, 1).Render(sb.String())
}

func (m modelPickerModel) renderCompactRow(e ModelEntry, isCursor bool, width int) string {
	marker := "  "
	if isCursor {
		marker = styleCursor.Render("▸ ")
	}

	// Truncate long names to keep columns aligned.
	name := e.Name
	if len(name) > maxNameLen {
		name = name[:maxNameLen-1] + "…"
	}
	if isCursor {
		name = styleAccent.Bold(true).Render(name)
	}

	tag := styleDim.Render(" " + tagForEntry(e))

	// Build line and use lipgloss to set fixed width with right padding.
	content := marker + name + tag
	return lipgloss.NewStyle().Width(width).Render(content)
}

func tagForEntry(e ModelEntry) string {
	if e.Cached {
		switch e.ModelType {
		case modelTypeLLamafile:
			return "[llamafile installed]"
		case modelTypeGGUF:
			return "[gguf installed]"
		case modelTypeOllama:
			return "[ollama]"
		default:
			return "[✓]"
		}
	}
	switch e.ModelType {
	case modelTypeLLamafile:
		return "[llamafile]"
	case modelTypeGGUF:
		return "[gguf]"
	case modelTypeOllama:
		return "[ollama]"
	default:
		if e.Enabled && e.Backend != "" {
			return "[" + e.Backend + " ✓]"
		}
		return "[cloud]"
	}
}

// RunModelPicker opens the interactive model picker TUI. If preFilter is
// non-empty, the picker starts with that filter applied. Returns the selected
// model name, or empty string if cancelled.
func RunModelPicker(entries []ModelEntry, preFilter string) (string, error) {
	if len(entries) == 0 {
		return "", nil
	}
	m := newModelPickerModel(entries, preFilter)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("model picker: %w", err)
	}
	fm, ok := final.(modelPickerModel)
	if !ok || fm.cancelled || !fm.quitted {
		return "", nil
	}
	flat := fm.flatIndexed()
	if fm.cursor >= 0 && fm.cursor < len(flat) {
		return flat[fm.cursor].Name, nil
	}
	return "", nil
}
