//go:build js

package tui

// Settings persists the user's tool/skill selections across sessions.
type Settings struct {
	SelectAll         bool     `yaml:"select_all"`
	EnabledWorkflows  []string `yaml:"enabled_workflows,omitempty"`
	EnabledAgencies   []string `yaml:"enabled_agencies,omitempty"`
	EnabledComponents []string `yaml:"enabled_components,omitempty"`
	EnabledSkills     []string `yaml:"enabled_skills,omitempty"`
}

// DefaultSettings returns settings with all items selected by default.
func DefaultSettings() Settings { return Settings{SelectAll: true} }

// LoadSettings is a no-op stub for js builds.
func LoadSettings() (Settings, error) { return DefaultSettings(), nil }

// Save is a no-op stub for js builds.
func (Settings) Save() error { return nil }

// SelectionFromSettings is a no-op stub for js builds.
func SelectionFromSettings(_ Settings) Selection { return Selection{} }

// SettingsFromSelection is a no-op stub for js builds.
func SettingsFromSelection(_ Selection, _ [numTabs][]Item) Settings { return DefaultSettings() }
