package codegraph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

func TestExposedToolsAreKartographerTools(t *testing.T) {
	names, err := exposedNames()
	require.NoError(t, err)
	assert.Contains(t, names, "find_symbol")
	assert.NotContains(t, names, "remember", "memory tools stay with kdeps")
}

func TestRegisterOffersEachExposedToolWithPrefix(t *testing.T) {
	names, err := exposedNames()
	require.NoError(t, err)

	reg := kdepstools.NewRegistry()
	require.NoError(t, Register(reg, t.TempDir()))
	for _, name := range names {
		tool := reg.Get(ToolPrefix + name)
		require.NotNil(t, tool, name)
		assert.Equal(t, "code", tool.Category)
		assert.NotEmpty(t, tool.Description)
	}
}

func TestToolParamsReadsTypesRequiredAndDefaults(t *testing.T) {
	params := toolParams(map[string]any{
		"properties": map[string]any{
			"term":  map[string]any{"type": "string", "description": "Substring of the symbol name."},
			"limit": map[string]any{"type": "integer", "default": 30},
		},
		"required": []string{"term"},
	})
	assert.Equal(t, "string", params["term"].Type)
	assert.True(t, params["term"].Required)
	assert.Equal(t, "integer", params["limit"].Type)
	assert.False(t, params["limit"].Required)
	assert.EqualValues(t, 30, params["limit"].Default)
}

func TestCodeGraphToolRunsOnTheWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/hello\n\ngo 1.21\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "hello.go"), []byte("package hello\n\nfunc Greet() string { return \"hi\" }\n"), 0o600))

	reg := kdepstools.NewRegistry()
	require.NoError(t, Register(reg, dir))
	out, err := reg.Get(ToolPrefix + "find_symbol").Execute(map[string]any{"term": "Greet"})
	require.NoError(t, err)
	assert.True(t, strings.Contains(out, "Greet"), out)
}
