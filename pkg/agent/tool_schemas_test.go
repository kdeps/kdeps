package agent

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/subagent"
	kdepstools "github.com/kdeps/kdeps/v2/pkg/tools"
)

// withCoreTools makes coreToolSet return names for the test.
func withCoreTools(t *testing.T, names ...string) {
	t.Helper()
	toolSchemaCache.Lock()
	toolSchemaCache.loaded = true
	toolSchemaCache.core = nil
	if names != nil {
		toolSchemaCache.core = map[string]bool{}
		for _, n := range names {
			toolSchemaCache.core[n] = true
		}
	}
	toolSchemaCache.Unlock()
	t.Cleanup(initToolSchemaPolicy)
}

// schemaTestLoop has a core tool, two deferred tools, a project tool
// and load_tools.
func schemaTestLoop(t *testing.T) *Loop {
	t.Helper()
	l := makeTestLoop(nil)
	l.registry = kdepstools.NewRegistry()
	for _, tool := range []*kdepstools.Tool{
		{Name: "read_file", Description: "Read a file.", OnDemand: true},
		{Name: "code_query", Description: "Find the symbols that match a question. Long tail text.", OnDemand: true},
		{Name: "git_ops", Description: "Run one structured git operation.", OnDemand: true},
		{Name: "my_workflow", Description: "The project's workflow."},
	} {
		l.registry.Register(tool)
	}
	l.registerToolLoader()
	require.NotNil(t, l.registry.Get(toolNameLoadTools), "load_tools has a definition asset")
	return l
}

func toolNames(tools []domain.Tool) []string {
	out := make([]string, 0, len(tools))
	for _, t := range tools {
		out = append(out, t.Name)
	}
	return out
}

func TestRequestTools_SendsCoreAndPinnedAndNamesTheRest(t *testing.T) {
	withCoreTools(t, "read_file", toolNameLoadTools)
	l := schemaTestLoop(t)

	got := l.requestTools()
	assert.ElementsMatch(t, []string{"load_tools", "my_workflow", "read_file"}, toolNames(got))
	var desc string
	for _, tool := range got {
		if tool.Name == toolNameLoadTools {
			desc = tool.Description
		}
	}
	assert.Contains(t, desc, "- code_query: Find the symbols that match a question")
	assert.NotContains(t, desc, "Long tail text", "a catalog line keeps the first sentence")
	assert.Contains(t, desc, "- git_ops: Run one structured git operation.")
	assert.NotContains(t, desc, "my_workflow", "tools kdeps does not ship are sent, not listed")
}

func TestLoadTools_AddsDefinitionsToLaterRequests(t *testing.T) {
	withCoreTools(t, "read_file", toolNameLoadTools)
	l := schemaTestLoop(t)

	out, err := l.loadTools(subagent.ToolNames("code_query, nope"))
	require.NoError(t, err)
	assert.Contains(t, out, "Loaded: code_query")
	assert.Contains(t, out, "Not tools: nope")
	assert.Contains(t, toolNames(l.requestTools()), "code_query")
	assert.NotContains(t, toolNames(l.requestTools()), "git_ops")

	_, err = l.loadTools([]string{"nope"})
	require.Error(t, err, "nothing loaded is an error")
	_, err = l.loadTools(nil)
	require.Error(t, err)
}

func TestLoadTools_ToolCallLoadsThroughItsArguments(t *testing.T) {
	withCoreTools(t, "read_file", toolNameLoadTools)
	l := schemaTestLoop(t)
	out, err := l.registry.Get(toolNameLoadTools).Execute(map[string]any{"names": []any{"git_ops"}})
	require.NoError(t, err)
	assert.Contains(t, out, "git_ops")
	assert.Contains(t, toolNames(l.requestTools()), "git_ops")
}

func TestRequestTools_SendsEverythingWithoutAPolicyOrWhenNothingIsHeldBack(t *testing.T) {
	withCoreTools(t) // nil core: no policy file
	l := schemaTestLoop(t)
	assert.Len(t, l.requestTools(), 5)

	withCoreTools(t, "read_file", "code_query", "git_ops", toolNameLoadTools)
	got := l.requestTools()
	assert.Len(t, got, 5)
	for _, tool := range got {
		assert.NotContains(t, tool.Description, "Tools you can load", "no catalog when nothing is deferred")
	}
}

func TestToolSchemaPolicy_ParsesAndValidatesByFileName(t *testing.T) {
	p, err := parseToolSchemaPolicy([]byte("core: [read_file, load_tools]\n"))
	require.NoError(t, err)
	assert.Equal(t, []string{"read_file", "load_tools"}, p.Core)
	_, err = parseToolSchemaPolicy([]byte("core: []\n"))
	require.Error(t, err)

	require.NoError(t, validatePolicyAsset([]byte("core: [read_file]\n"), "tool-schemas"))
	require.NoError(t, validatePolicyAsset([]byte("core: [read_file]\n"), "policy/tool-schemas.yaml"))
	require.Error(t, validatePolicyAsset([]byte("core: [read_file]\n"), "permission"),
		"permission.yaml still needs its own fields")
	require.NoError(t, validatePolicyAsset([]byte("default: allow\n"), "permission"))
}

func TestCoreToolSet_ShippedPolicyKeepsTheBasics(t *testing.T) {
	initToolSchemaPolicy()
	t.Cleanup(initToolSchemaPolicy)
	t.Setenv("HOME", t.TempDir()) // no user override
	core := coreToolSet()
	require.NotNil(t, core, "the policy asset ships")
	for _, n := range []string{"bash_exec", "read_file", "edit_file", toolNameLoadTools, "load_skill", "session_handshake"} {
		assert.True(t, core[n], n)
	}
}

func TestToolCatalog_CutsLongDescriptions(t *testing.T) {
	got := toolCatalog([]*kdepstools.Tool{{Name: "x", Description: strings.Repeat("word ", 40)}})
	assert.LessOrEqual(t, len([]rune(strings.TrimPrefix(got, "- x: "))), toolCatalogDescChars)
}
