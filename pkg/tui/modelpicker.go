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
	modelTypeLlamafile = "llamafile"
	modelTypeGGUF      = "gguf"
	modelTypeOllama    = "ollama"

	modelSortCachedFirst   = 0
	modelSortLlamafile     = 1
	modelSortGGUF          = 2
	modelSortOllama        = 3
	modelSortCloudFallback = 4

	viewSeparatorWidth = 60
)

// ModelEntry is a selectable model in the picker.
type ModelEntry struct {
	Name      string
	ModelType string // "llamafile", "gguf", "" (cloud)
	Backend   string // cloud backend name (e.g. "deepseek"), or ""
	Cached    bool
	Enabled   bool   // cloud API key is set
	SizeGB    string // formatted size string, or ""
}

// ModelPickerResult holds the selected model or empty if cancelled.
type ModelPickerResult struct {
	Selected string
}

type modelPickerModel struct {
	entries   []ModelEntry
	cursor    int
	filter    string
	quitted   bool
	cancelled bool
}

func newModelPickerModel(entries []ModelEntry) modelPickerModel {
	sort.Slice(entries, func(i, j int) bool {
		// Sort: cached first, then llamafile, gguf, ollama, then cloud
		order := func(e ModelEntry) int {
			if e.Cached {
				return modelSortCachedFirst
			}
			switch e.ModelType {
			case modelTypeLlamafile:
				return modelSortLlamafile
			case modelTypeGGUF:
				return modelSortGGUF
			case modelTypeOllama:
				return modelSortOllama
			}
			return modelSortCloudFallback
		}
		oi, oj := order(entries[i]), order(entries[j])
		if oi != oj {
			return oi < oj
		}
		return entries[i].Name < entries[j].Name
	})
	return modelPickerModel{entries: entries}
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
	switch key.String() {
	case "ctrl+c", "esc":
		m.cancelled = true
		m.quitted = true
		return m, tea.Quit
	case "q":
		m.quitted = true
		return m, tea.Quit
	case "enter":
		filtered := m.filtered()
		if len(filtered) > 0 && m.cursor < len(filtered) {
			m.quitted = true
			return m, tea.Quit
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		filtered := m.filtered()
		if m.cursor < len(filtered)-1 {
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

func (m modelPickerModel) View() string {
	var sb strings.Builder
	sb.WriteString(styleHelp.Render("Model Picker — type to filter, ↑↓ navigate, enter select, esc cancel"))
	sb.WriteString("\n")
	if m.filter != "" {
		sb.WriteString(styleCursor.Render("  filter: " + m.filter))
		sb.WriteString("\n")
	}
	sb.WriteString(strings.Repeat("─", viewSeparatorWidth))
	sb.WriteString("\n\n")

	filtered := m.filtered()
	if len(filtered) == 0 {
		sb.WriteString(styleDim.Render("  no models match"))
		return lipgloss.NewStyle().Padding(1).Render(sb.String())
	}

	for i, e := range filtered {
		cursor := "  "
		nameStyle := styleBase
		if i == m.cursor {
			cursor = styleCursor.Render("▸ ")
			nameStyle = styleEnabled
		}
		tag := tagForEntry(e)
		fmt.Fprintf(&sb, "%s%s  %s\n", cursor, nameStyle.Render(e.Name), styleDim.Render(tag))
	}

	fmt.Fprintf(&sb, "\n%s\n", styleHelp.Render(fmt.Sprintf("%d models", len(filtered))))
	return lipgloss.NewStyle().Padding(1).Render(sb.String())
}

func tagForEntry(e ModelEntry) string {
	if e.Cached {
		switch e.ModelType {
		case modelTypeLlamafile:
			return "[✳ llamafile cached]"
		case modelTypeGGUF:
			return "[✳ gguf cached]"
		case modelTypeOllama:
			return "[✳ ollama cached]"
		default:
			return "[✳ cached]"
		}
	}
	switch e.ModelType {
	case modelTypeLlamafile:
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

// RunModelPicker opens the interactive model picker TUI. Returns the selected
// model name, or empty string if cancelled.
func RunModelPicker(entries []ModelEntry) (string, error) {
	if len(entries) == 0 {
		return "", nil
	}
	m := newModelPickerModel(entries)
	p := tea.NewProgram(m)
	final, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("model picker: %w", err)
	}
	fm, ok := final.(modelPickerModel)
	if !ok || fm.cancelled || !fm.quitted {
		return "", nil
	}
	filtered := fm.filtered()
	if fm.cursor >= 0 && fm.cursor < len(filtered) {
		return filtered[fm.cursor].Name, nil
	}
	return "", nil
}
