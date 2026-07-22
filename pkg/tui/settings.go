//go:build !js

package tui

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/spf13/afero"
)

const settingsFile = "agent-loop-settings.yaml"

// Settings persists the user's tool/skill selections across sessions.
type Settings struct {
	// nil means "all" (default-all semantics); non-nil means explicit list.
	EnabledWorkflows  []string `yaml:"enabled_workflows,omitempty"`
	EnabledAgencies   []string `yaml:"enabled_agencies,omitempty"`
	EnabledComponents []string `yaml:"enabled_components,omitempty"`
	EnabledSkills     []string `yaml:"enabled_skills,omitempty"`

	// When true, every newly discovered item is enabled by default.
	// Stored as false only after the user has made explicit selections.
	SelectAll bool `yaml:"select_all"`

	// DefaultModel is the model used at startup when no --model flag is given.
	// Set via /model default <name> in the REPL.
	DefaultModel string `yaml:"default_model,omitempty"`

	// CustomOpenAIModels are user-added OpenAI-compatible endpoints registered
	// via "/model <base-url>". Downloaded .gguf/.llamafile URLs persist in
	// their own version registries; only network endpoints live here.
	CustomOpenAIModels []CustomOpenAIModel `yaml:"custom_openai_models,omitempty"`

	// FavoriteModels are models the user starred via "/model favorite <name>".
	// They are surfaced first in "/model" and "/model <tab>" and persist across
	// sessions.
	FavoriteModels []string `yaml:"favorite_models,omitempty"`

	// AgentLoop persists the /model tool settings (rounds, stall timeout, autokill,
	// retries, compaction, history caps). nil means "never customized — use
	// built-in defaults".
	AgentLoop *AgentLoopTuning `yaml:"agent_loop,omitempty"`
}

// AgentLoopTuning is the persisted form of the /model tool settings. Durations
// are strings (e.g. "10m"); ToolStallTimeout is "off" when stall detection is
// disabled. Field order and types match agent.ToolTuning so the two convert
// directly.
type AgentLoopTuning struct {
	MaxToolRounds        int    `yaml:"max_tool_rounds"`
	AutoRetryMax         int    `yaml:"auto_retry_max"`
	AutoRetryBaseDelay   string `yaml:"auto_retry_base_delay"`
	ToolStallTimeout     string `yaml:"tool_stall_timeout"`
	AutoCompactThreshold int    `yaml:"auto_compact_threshold"`
	CompactTokenBudget   int    `yaml:"compact_token_budget"`
	MaxTurns             int    `yaml:"max_turns"`
	MaxHistoryTokens     int    `yaml:"max_history_tokens"`
	WebLimit             int    `yaml:"web_limit"`
	BashLimit            int    `yaml:"bash_limit"`
	FileLimit            int    `yaml:"file_limit"`
	CodeLimit            int    `yaml:"code_limit"`
}

// SaveAgentLoopTuning persists the agent-loop tool settings, preserving the rest
// of the settings file.
func SaveAgentLoopTuning(t AgentLoopTuning) error {
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	s.AgentLoop = &t
	return s.Save()
}

// CustomOpenAIModel is a user-registered OpenAI-compatible endpoint. No API
// key is stored; the client uses OPENAI_API_KEY / KDEPS_CUSTOM_API_KEY from
// the environment when the endpoint requires one.
type CustomOpenAIModel struct {
	Alias   string `yaml:"alias"`
	BaseURL string `yaml:"base_url"`
}

// AddCustomOpenAIModel adds or replaces a custom OpenAI-compatible endpoint by
// alias and persists the settings file.
func AddCustomOpenAIModel(alias, baseURL string) error {
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	replaced := false
	for i := range s.CustomOpenAIModels {
		if s.CustomOpenAIModels[i].Alias == alias {
			s.CustomOpenAIModels[i].BaseURL = baseURL
			replaced = true
			break
		}
	}
	if !replaced {
		s.CustomOpenAIModels = append(s.CustomOpenAIModels, CustomOpenAIModel{Alias: alias, BaseURL: baseURL})
	}
	return s.Save()
}

// SaveDefaultModel updates only the DefaultModel field and persists the settings file.
func SaveDefaultModel(model string) error {
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	s.DefaultModel = model
	return s.Save()
}

// SetFavoriteModel adds (fav=true) or removes (fav=false) a model from the
// persisted favorites list and saves the settings file.
func SetFavoriteModel(name string, fav bool) error {
	s, err := LoadSettings()
	if err != nil {
		return err
	}
	out := s.FavoriteModels[:0]
	for _, m := range s.FavoriteModels {
		if m != name {
			out = append(out, m)
		}
	}
	if fav {
		out = append(out, name)
	}
	s.FavoriteModels = out
	return s.Save()
}

// DefaultSettings returns settings with all items selected by default.
func DefaultSettings() Settings {
	return Settings{SelectAll: true}
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("settings: home dir: %w", err)
	}
	return filepath.Join(home, ".kdeps", settingsFile), nil
}

// LoadSettings reads persisted settings from ~/.kdeps/agent-loop-settings.yaml.
// Returns DefaultSettings if the file does not exist.
func LoadSettings() (Settings, error) {
	path, err := settingsPath()
	if err != nil {
		return DefaultSettings(), nil //nolint:nilerr // non-fatal: use defaults
	}

	data, err := afero.ReadFile(AppFS, path)
	if os.IsNotExist(err) {
		return DefaultSettings(), nil
	}
	if err != nil {
		return DefaultSettings(), fmt.Errorf("settings: read: %w", err)
	}

	var s Settings
	if unmarshalErr := yaml.Unmarshal(data, &s); unmarshalErr != nil {
		return DefaultSettings(), fmt.Errorf("settings: parse: %w", unmarshalErr)
	}
	return s, nil
}

// Save writes settings to ~/.kdeps/agent-loop-settings.yaml.
func (s Settings) Save() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if mkErr := AppFS.MkdirAll(filepath.Dir(path), 0o750); mkErr != nil {
		return fmt.Errorf("settings: mkdir: %w", mkErr)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("settings: marshal: %w", err)
	}
	if writeErr := afero.WriteFile(AppFS, path, data, 0o600); writeErr != nil {
		return fmt.Errorf("settings: write: %w", writeErr)
	}
	return nil
}

// applyToModel returns a new model with items pre-toggled to match settings.
// When SelectAll is true (or settings is fresh/default), all items are enabled.
func applyToModel(m model, s Settings) model {
	if s.SelectAll {
		for tab := range m.tabs {
			for i := range m.tabs[tab] {
				m.tabs[tab][i].Enabled = true
			}
		}
		return m
	}

	enabledMap := func(names []string) map[string]bool {
		out := make(map[string]bool, len(names))
		for _, n := range names {
			out[n] = true
		}
		return out
	}

	wfSet := enabledMap(s.EnabledWorkflows)
	agSet := enabledMap(s.EnabledAgencies)
	compSet := enabledMap(s.EnabledComponents)
	skSet := enabledMap(s.EnabledSkills)

	setEnabled := func(items []Item, allowed map[string]bool) []Item {
		for i := range items {
			items[i].Enabled = allowed[items[i].Name]
		}
		return items
	}

	m.tabs[tabWorkflows] = setEnabled(m.tabs[tabWorkflows], wfSet)
	m.tabs[tabAgencies] = setEnabled(m.tabs[tabAgencies], agSet)
	m.tabs[tabComponents] = setEnabled(m.tabs[tabComponents], compSet)
	m.tabs[tabSkills] = setEnabled(m.tabs[tabSkills], skSet)
	return m
}

// SelectionFromSettings discovers all items from ~/.kdeps and returns those
// permitted by settings. When SelectAll is true, all discovered items are returned.
func SelectionFromSettings(s Settings) Selection {
	items := discoverItems()
	m := applyToModel(newModel(items), s)
	return m.toSelection()
}

// SettingsFromSelection converts a TUI selection back into a Settings struct
// so it can be persisted. If all items in all tabs are enabled, SelectAll is
// set to true to preserve the default-all semantics.
func SettingsFromSelection(sel Selection, allItems [numTabs][]Item) Settings {
	toNames := func(items []Item) []string {
		names := make([]string, 0, len(items))
		for _, it := range items {
			names = append(names, it.Name)
		}
		return names
	}

	// Check if user enabled everything (equivalent to SelectAll)
	totalItems := 0
	totalEnabled := 0
	for _, tab := range allItems {
		totalItems += len(tab)
	}
	totalEnabled += len(sel.Workflows) + len(sel.Agencies) + len(sel.Components) + len(sel.Skills)

	if totalEnabled == totalItems {
		return Settings{SelectAll: true}
	}

	return Settings{
		SelectAll:         false,
		EnabledWorkflows:  toNames(sel.Workflows),
		EnabledAgencies:   toNames(sel.Agencies),
		EnabledComponents: toNames(sel.Components),
		EnabledSkills:     toNames(sel.Skills),
	}
}
