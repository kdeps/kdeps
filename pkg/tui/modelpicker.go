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
	Repo      string // HuggingFace repo id (e.g. "googleai/gemma4"), llamafile/gguf only
	Cached    bool
	Enabled   bool    // cloud API key is set
	SizeGB    string  // formatted size string, or ""
	Score     float64 // llmfit composite score 0-100; 0 when unavailable
	FitLevel  string  // llmfit fit level: "Perfect", "Good", "Marginal", or "" when unavailable
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

// rankedEntry pairs an entry with its group order for stable sorting.
type rankedEntry struct {
	entry ModelEntry
	order int
}

// lessEntry returns true when a sorts before b in the model picker.
// Order by group, then by llmfit score descending, then current model first,
// then alphabetically by name.
func lessEntry(a, b rankedEntry, currentModel string) bool {
	if a.order != b.order {
		return a.order > b.order
	}
	// Same group: compare by score descending.
	if a.entry.Score != 0 && b.entry.Score != 0 && a.entry.Name != currentModel && b.entry.Name != currentModel {
		return a.entry.Score < b.entry.Score
	}
	// Score tie or one has no score: current model first, then alphabetical.
	if b.entry.Name == currentModel {
		return true
	}
	if a.entry.Name == currentModel {
		return false
	}
	return a.entry.Name > b.entry.Name
}

func sortEntries(entries []ModelEntry, currentModel string) []ModelEntry {
	sorted := make([]ModelEntry, len(entries))
	copy(sorted, entries)
	ranks := make([]rankedEntry, len(sorted))
	for i, e := range sorted {
		ranks[i] = rankedEntry{e, entryGroupOrder(e)}
	}
	// insertion sort (stable, small N is fine)
	for i := 1; i < len(ranks); i++ {
		for j := i; j > 0; j-- {
			a, b := ranks[j-1], ranks[j]
			if lessEntry(a, b, currentModel) {
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
	return fuzzyMatchEntries(m.allEntries, m.filter)
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
	case "up":
		if m.cursor > 0 {
			m.cursor--
		} else if total > 0 {
			m.cursor = total - 1
		}
	case "down":
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
	hint := styleDim.Render("type to filter  ↑↓  enter select  esc cancel")
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
	nameStyle := styleForFitLevel(e.FitLevel)
	nameStr := nameStyle.Render(name)
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

// styleForFitLevel returns the lipgloss style for the given llmfit fit level.
// Perfect = green, Good = cyan, Marginal = dim, Too Tight = pink,
// unrecognized = default.
func styleForFitLevel(level string) lipgloss.Style {
	switch level {
	case "Perfect":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF87"))
	case "Good":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#00E5FF"))
	case "Marginal":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
	case "Too Tight", "TooTight":
		return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF2D78"))
	default:
		return lipgloss.NewStyle()
	}
}

func tagForEntry(e ModelEntry) string {
	repoSuffix := ""
	if e.Repo != "" {
		repoSuffix = " " + e.Repo
	}
	scoreTag := ""
	if e.Score > 0 {
		level := ""
		switch e.FitLevel {
		case "Perfect":
			level = styleForFitLevel("Perfect").Render("Perfect") + " " // green
		case "Good":
			level = styleForFitLevel("Good").Render("Good") + " " // cyan
		case "Marginal":
			level = styleForFitLevel("Marginal").Render("Marginal") + " " // gray
		case "Too Tight", "TooTight":
			level = styleForFitLevel("Too Tight").Render("Too Tight") + " " // pink
		}
		if e.FitLevel != "" {
			scoreTag = fmt.Sprintf("%s%.0f ", level, e.Score)
		} else {
			scoreTag = fmt.Sprintf("%.0f ", e.Score)
		}
	}
	if e.Cached {
		switch e.ModelType {
		case modelTypeLLamafile:
			return scoreTag + "[llamafile installed" + repoSuffix + "]"
		case modelTypeGGUF:
			return scoreTag + "[gguf installed" + repoSuffix + "]"
		case modelTypeOllama:
			return scoreTag + "[ollama installed]"
		default:
			return scoreTag + "[installed]"
		}
	}
	switch e.ModelType {
	case modelTypeLLamafile:
		return scoreTag + "[llamafile" + repoSuffix + "]"
	case modelTypeGGUF:
		return scoreTag + "[gguf" + repoSuffix + "]"
	case modelTypeOllama:
		return scoreTag + "[ollama]"
	default:
		if e.Enabled {
			return scoreTag + "[cloud enabled]"
		}
		return scoreTag + "[cloud]"
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
	if !isInteractive() {
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
