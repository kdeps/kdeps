//go:build !js

package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	modelTypeLLamafile = "llamafile"
	modelTypeGGUF      = "gguf"
	modelTypeOllama    = "ollama"

	pickerMaxVisible = 12 // max rows shown at once before scrolling
	pickerNamePad    = 2  // spaces between name and tag
	pickerMinWidth   = 60 // minimum usable terminal width
)

// ModelEntry is a selectable model in the picker.
type ModelEntry struct {
	Name      string
	ModelType string // "llamafile", "gguf", "ollama", "" (cloud)
	Backend   string // cloud backend name (e.g. "deepseek"), or ""
	Cached    bool
	Enabled   bool   // cloud API key is set
	SizeGB    string // formatted size string, or ""
}

type modelPickerModel struct {
	allEntries   []ModelEntry // sorted, ungrouped
	groups       []pickerGroup
	currentModel string // active model name — gets ✓ marker
	filter       string
	cursor       int
	termWidth    int
	quitted      bool
	cancelled    bool
}

type pickerGroup struct {
	label   string
	entries []ModelEntry
}

const (
	groupOrderCached    = 0
	groupOrderLLamafile = 1
	groupOrderGGUF      = 2
	groupOrderOllama    = 3
	groupOrderCloud     = 4

	groupLabelCached    = "Cached"
	groupLabelLLamafile = "LLamafile"
	groupLabelGGUF      = "GGUF"
	groupLabelOllama    = "Ollama"
	groupLabelCloud     = "Cloud"
)

func entryGroupOrder(e ModelEntry) int {
	if e.Cached {
		return groupOrderCached
	}
	switch e.ModelType {
	case modelTypeLLamafile:
		return groupOrderLLamafile
	case modelTypeGGUF:
		return groupOrderGGUF
	case modelTypeOllama:
		return groupOrderOllama
	}
	return groupOrderCloud
}

func sortEntries(entries []ModelEntry, currentModel string) []ModelEntry {
	sorted := make([]ModelEntry, len(entries))
	copy(sorted, entries)
	// stable sort: current model first within its group, then alphabetical
	type ranked struct {
		entry ModelEntry
		order int
	}
	ranks := make([]ranked, len(sorted))
	for i, e := range sorted {
		ranks[i] = ranked{e, entryGroupOrder(e)}
	}
	// insertion sort (stable, small N is fine)
	for i := 1; i < len(ranks); i++ {
		for j := i; j > 0; j-- {
			a, b := ranks[j-1], ranks[j]
			less := a.order > b.order
			if a.order == b.order {
				aCurrent := a.entry.Name == currentModel
				bCurrent := b.entry.Name == currentModel
				switch {
				case !aCurrent && bCurrent:
					less = true
				case aCurrent == bCurrent:
					less = a.entry.Name > b.entry.Name
				default:
					less = false
				}
			}
			if less {
				ranks[j-1], ranks[j] = ranks[j], ranks[j-1]
			} else {
				break
			}
		}
	}
	for i, r := range ranks {
		sorted[i] = r.entry
	}
	return sorted
}

func buildGroups(entries []ModelEntry) []pickerGroup {
	groups := []pickerGroup{
		{label: groupLabelCached},
		{label: groupLabelLLamafile},
		{label: groupLabelGGUF},
		{label: groupLabelOllama},
		{label: groupLabelCloud},
	}
	for _, e := range entries {
		groups[entryGroupOrder(e)].entries = append(groups[entryGroupOrder(e)].entries, e)
	}
	return groups
}

func newModelPickerModel(entries []ModelEntry, currentModel, preFilter string) modelPickerModel {
	sorted := sortEntries(entries, currentModel)
	groups := buildGroups(sorted)
	m := modelPickerModel{
		allEntries:   sorted,
		groups:       groups,
		currentModel: currentModel,
		filter:       preFilter,
		termWidth:    pickerMinWidth,
	}
	// pre-select cursor to current model
	flat := m.flatFiltered()
	for i, e := range flat {
		if e.Name == currentModel {
			m.cursor = i
			break
		}
	}
	return m
}

func (m modelPickerModel) flatFiltered() []ModelEntry {
	if m.filter == "" {
		return m.allEntries
	}
	lower := strings.ToLower(m.filter)
	var out []ModelEntry
	for _, e := range m.allEntries {
		if strings.Contains(strings.ToLower(e.Name), lower) ||
			strings.Contains(strings.ToLower(tagForEntry(e)), lower) {
			out = append(out, e)
		}
	}
	return out
}

func (m modelPickerModel) Init() tea.Cmd { return nil }

func (m modelPickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m modelPickerModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	flat := m.flatFiltered()
	total := len(flat)
	switch msg.String() {
	case "ctrl+c", "esc":
		m.cancelled = true
		m.quitted = true
		return m, tea.Quit
	case "enter":
		if total > 0 {
			m.quitted = true
			return m, tea.Quit
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		} else if total > 0 {
			m.cursor = total - 1
		}
	case "down", "j":
		if m.cursor < total-1 {
			m.cursor++
		} else {
			m.cursor = 0
		}
	case "backspace":
		if len(m.filter) > 0 {
			m.filter = m.filter[:len(m.filter)-1]
			m.cursor = 0
		}
	default:
		s := msg.String()
		if len(s) == 1 && s >= " " && s <= "~" {
			m.filter += s
			m.cursor = 0
		}
	}
	return m, nil
}

func (m modelPickerModel) View() string {
	width := m.termWidth
	if width < pickerMinWidth {
		width = pickerMinWidth
	}
	inner := width - 2 //nolint:mnd // inner = total - left/right padding

	var sb strings.Builder
	sb.WriteString(m.viewHeader(inner))
	sb.WriteString(m.viewList(inner))
	sb.WriteString(m.viewFooter(inner))
	return lipgloss.NewStyle().Padding(1, 1).Render(sb.String())
}

func (m modelPickerModel) viewHeader(inner int) string {
	sep := styleDim.Render(strings.Repeat("─", inner))
	title := styleAccent.Bold(true).Render("Model Picker")
	hint := styleDim.Render("type to filter  ↑↓/jk  enter select  esc cancel")
	filterPrompt := styleAccent.Render("> ")
	filterText := m.filter
	if filterText == "" {
		filterText = styleDim.Render("(type to filter)")
	}
	return title + "  " + hint + "\n" +
		sep + "\n" +
		filterPrompt + filterText + "\n" +
		sep + "\n\n"
}

func (m modelPickerModel) viewList(inner int) string {
	flat := m.flatFiltered()
	total := len(flat)
	if total == 0 {
		return styleDim.Render("  No matching models") + "\n"
	}
	start, end := m.scrollWindow(total)
	flatSet := make(map[string]struct{}, total)
	for _, e := range flat {
		flatSet[e.Name] = struct{}{}
	}
	var sb strings.Builder
	shownGroup := -1
	for i := start; i < end; i++ {
		e := flat[i]
		g := entryGroupOrder(e)
		if g != shownGroup {
			shownGroup = g
			count := 0
			for _, ge := range m.groups[g].entries {
				if _, ok := flatSet[ge.Name]; ok {
					count++
				}
			}
			fmt.Fprintf(&sb, "%s\n", styleDim.Render(fmt.Sprintf("  %s (%d)", groupLabel(g), count)))
		}
		fmt.Fprintf(&sb, "%s\n", m.renderRow(e, i == m.cursor, inner))
	}
	return sb.String()
}

func (m modelPickerModel) scrollWindow(total int) (int, int) {
	if total <= pickerMaxVisible {
		return 0, total
	}
	half := pickerMaxVisible / 2 //nolint:mnd // half-window for cursor centering
	start := max(0, m.cursor-half)
	start = min(start, total-pickerMaxVisible)
	end := min(start+pickerMaxVisible, total)
	return start, end
}

func (m modelPickerModel) viewFooter(inner int) string {
	flat := m.flatFiltered()
	total := len(flat)
	sep := styleDim.Render(strings.Repeat("─", inner))
	scroll := ""
	if total > 0 {
		scroll = fmt.Sprintf("(%d/%d)", m.cursor+1, total)
	}
	selInfo := ""
	if total > 0 && m.cursor < total {
		sel := flat[m.cursor]
		selInfo = sel.Name + "  " + styleDim.Render(tagForEntry(sel))
		if sel.SizeGB != "" {
			selInfo += styleDim.Render("  " + sel.SizeGB + "GB")
		}
	}
	scrollRendered := styleDim.Render(scroll)
	gap := inner - lipgloss.Width(selInfo) - lipgloss.Width(scrollRendered)
	if gap < 1 {
		gap = 1
	}
	return "\n" + sep + "\n" + selInfo + strings.Repeat(" ", gap) + scrollRendered
}

func groupLabel(order int) string {
	switch order {
	case groupOrderCached:
		return groupLabelCached
	case groupOrderLLamafile:
		return groupLabelLLamafile
	case groupOrderGGUF:
		return groupLabelGGUF
	case groupOrderOllama:
		return groupLabelOllama
	}
	return groupLabelCloud
}

func (m modelPickerModel) renderRow(e ModelEntry, isCursor bool, width int) string {
	marker := "  "
	if isCursor {
		marker = styleAccent.Render("▸ ")
	}

	isCurrent := e.Name == m.currentModel
	checkmark := ""
	if isCurrent {
		checkmark = styleSuccess.Render(" ✓")
	}

	tag := styleDim.Render(tagForEntry(e))

	name := e.Name
	// compute available space for name: width - marker(2) - tag - checkmark - padding
	tagW := lipgloss.Width(tag)
	checkW := lipgloss.Width(checkmark)
	markerW := lipgloss.Width(marker)
	maxNameW := width - markerW - tagW - checkW - pickerNamePad
	if maxNameW < 1 {
		maxNameW = 1
	}
	if len(name) > maxNameW {
		name = name[:maxNameW-1] + "…"
	}
	nameStr := name
	if isCursor {
		nameStr = styleAccent.Bold(true).Render(name)
	}

	// right-align tag
	nameW := lipgloss.Width(nameStr)
	padW := width - markerW - nameW - tagW - checkW
	if padW < 1 {
		padW = 1
	}
	pad := strings.Repeat(" ", padW)

	return marker + nameStr + pad + tag + checkmark
}

func tagForEntry(e ModelEntry) string {
	if e.Cached {
		switch e.ModelType {
		case modelTypeLLamafile:
			return "[llamafile installed]"
		case modelTypeGGUF:
			return "[gguf installed]"
		case modelTypeOllama:
			return "[ollama installed]"
		default:
			return "[installed]"
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
		if e.Enabled {
			return "[cloud enabled]"
		}
		return "[cloud]"
	}
}

// RunModelPicker opens the interactive model picker TUI. currentModel is the
// currently active model (shown with ✓ and pre-selected). preFilter is an
// optional initial search string. Returns the selected model name, or empty if
// cancelled.
func RunModelPicker(entries []ModelEntry, currentModel, preFilter string) (string, error) {
	if len(entries) == 0 {
		return "", nil
	}
	m := newModelPickerModel(entries, currentModel, preFilter)
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("model picker: %w", err)
	}
	fm, ok := final.(modelPickerModel)
	if !ok || fm.cancelled || !fm.quitted {
		return "", nil
	}
	flat := fm.flatFiltered()
	if fm.cursor >= 0 && fm.cursor < len(flat) {
		return flat[fm.cursor].Name, nil
	}
	return "", nil
}
