//go:build js

package tui

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

// Run is a no-op stub for WASM/js targets (no terminal TUI available).
func Run() (Selection, Settings, error) { return Selection{}, DefaultSettings(), nil }
