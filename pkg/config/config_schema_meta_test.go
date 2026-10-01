package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func byPath(t *testing.T) map[string]SettingField {
	t.Helper()
	out := map[string]SettingField{}
	for _, f := range (&Config{}).Settings() {
		out[f.Path] = f
	}
	return out
}

func TestSettingsMeta_Options(t *testing.T) {
	f := byPath(t)
	assert.Equal(t, []string{"fail", "continue", "retry"}, f["resource_defaults.onError.action"].Options)
	assert.Contains(t, f["llm.strategy"].Options, "fallback")

	backends := f["llm.backend"].Options
	assert.Contains(t, backends, "ollama")
	assert.Contains(t, backends, "file")
	assert.Greater(t, len(backends), 3, "cloud providers are listed")
}

func TestSettingsMeta_Ranges(t *testing.T) {
	f := byPath(t)
	temp := f["resource_defaults.chat.temperature"]
	require.NotNil(t, temp.Min)
	require.NotNil(t, temp.Max)
	require.NotNil(t, temp.Step)
	assert.InDelta(t, 2.0, *temp.Max, 0)
	assert.InDelta(t, -2.0, *f["resource_defaults.chat.presence_penalty"].Min, 0)
}

func TestSettingsMeta_Suggestions(t *testing.T) {
	f := byPath(t)
	assert.Contains(t, f["resource_defaults.chat.timeout"].Suggestions, "30s")
	assert.Contains(t, f["resource_defaults.http.retry_backoff"].Suggestions, "1m")
	assert.Contains(t, f["resource_defaults.exec.max_output_bytes"].Suggestions, "1048576")
	assert.Contains(t, f["defaults.python_version"].Suggestions, "3.12")
	assert.Contains(t, f["llm.ctx_size"].Suggestions, "8192")
}

func TestSettingsMeta_SecretsHaveNoControlsMeta(t *testing.T) {
	for _, f := range byPath(t) {
		if f.Secret {
			assert.Empty(t, f.Options, f.Path)
			assert.Empty(t, f.Suggestions, f.Path)
			assert.NotEmpty(t, f.Help, f.Path)
		}
	}
}

func TestSettingsMeta_OptionsAreAcceptedByPersist(t *testing.T) {
	t.Setenv("KDEPS_CONFIG_PATH", t.TempDir()+"/config.yaml")
	f := byPath(t)
	for _, p := range []string{"resource_defaults.onError.action", "llm.strategy", "llm.backend"} {
		for _, o := range f[p].Options {
			require.NoError(t, PersistField(p, o), "%s=%s", p, o)
		}
	}
}
