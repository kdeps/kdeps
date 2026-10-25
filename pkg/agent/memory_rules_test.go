package agent

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withMemoryRules swaps in rules for one test.
func withMemoryRules(t *testing.T, r MemoryRules) {
	t.Helper()
	old := memoryRules
	memoryRules = r
	t.Cleanup(func() { memoryRules = old })
}

func TestMemoryRules_ShippedRulesLoad(t *testing.T) {
	initMemoryRules()
	assert.Equal(t, memTypeToolResult, inferType("tool:bash_exec:output"))
	assert.Equal(t, memTypeAction, inferType("last_action"))
	assert.Equal(t, memTypePrompt, inferType("current_task"))
	assert.Equal(t, memTypeNote, inferType("zzz"), "no rule matches: default_type")
	assert.Equal(t, memTypePrompt, memoryRules.Parents[memTypePurpose])
	assert.True(t, isToolMemoryWorthy("bash_exec"))
	assert.False(t, isToolMemoryWorthy("calculator"))
	assert.Greater(t, resumeTypeRank(memTypeProgress), resumeTypeRank(memTypeStatus))
	assert.Greater(t, resumeTypeRank(memTypeStatus), resumeTypeRank(memTypeResult))
	assert.True(t, isResumableType(memTypeResult))
	assert.False(t, isResumableType(memTypeNote))
}

func TestMemoryRules_YAMLDrivesTypingCaptureAndMarkers(t *testing.T) {
	r, err := parseMemoryRules([]byte(`
types:
  - {prefix: "x:", type: fact}
  - {equals: only, type: decision}
  - {contains: [blip], type: error}
default_type: context
parents: {fact: decision}
capture_tools: [calculator]
resume: [fact]
status_done: {not: [nope], markers: [yay]}
error_resolved: {markers: [gone]}
`))
	require.NoError(t, err)
	withMemoryRules(t, r)

	assert.Equal(t, memTypeFact, inferType("x:1"))
	assert.Equal(t, memTypeDecision, inferType("only"))
	assert.Equal(t, memTypeError, inferType("a_blip_b"))
	assert.Equal(t, memTypeContext, inferType("other"))
	assert.True(t, isToolMemoryWorthy("calculator"))
	assert.False(t, isToolMemoryWorthy("bash_exec"))
	assert.True(t, isResumableType(memTypeFact))
	assert.False(t, isResumableType(memTypeProgress))
	assert.True(t, statusIsDone("yay"))
	assert.False(t, statusIsDone("yay but nope"), "a not marker wins")
	assert.False(t, statusIsDone("done"), "only the YAML's markers count")
	assert.True(t, errorIsResolved("it is gone"))
	assert.False(t, errorIsResolved("fixed"))

	m := NewMemoryStore(t.TempDir())
	m.SetCwd(t.TempDir())
	require.NoError(t, m.Set("only", "parent"))
	require.NoError(t, m.Set("x:child", "child"))
	assert.Equal(t, "only", m.findParentKey(memTypeFact), "a fact links to the newest decision")
}

func TestParseMemoryRules_Rejects(t *testing.T) {
	for name, src := range map[string]string{
		"bad yaml":        "types: [",
		"rule no type":    "types: [{prefix: a}]\ndefault_type: note\n",
		"rule no match":   "types: [{type: note}]\ndefault_type: note\n",
		"no default_type": "types: []\n",
	} {
		_, err := parseMemoryRules([]byte(src))
		assert.Errorf(t, err, "%s is rejected", name)
	}
}

func TestInitMemoryRules_MissingFileLeavesNotes(t *testing.T) {
	t.Cleanup(initMemoryRules) // registered first, so it runs after the env is restored
	t.Setenv("KDEPS_ASSETS_SRC", t.TempDir())
	t.Setenv("KDEPS_ASSETS_DIR", t.TempDir())
	initMemoryRules() // prints why to stderr
	assert.Equal(t, memTypeNote, inferType("tool:x"))
	assert.False(t, isToolMemoryWorthy("bash_exec"))
	assert.False(t, statusIsDone("done"))
}

func TestDeclaredCommand_ToolStepSavesMemory(t *testing.T) {
	withCommands(t, map[string]string{
		"rem": "name: rem\nusage: [{help: h}]\nrun:\n  - tool: memory_save\n    args: {key: \"$1\", value: \"$2\"}\n",
	})
	store := NewMemoryStore(t.TempDir())
	store.SetCwd(t.TempDir())
	old := memoryStoreInstance
	memoryStoreInstance = store
	t.Cleanup(func() { memoryStoreInstance = old })

	repl := NewREPL(context.Background(), makeTestLoop(nil))
	defer repl.cancel()
	c := declaredCommand("rem")
	require.NotNil(t, c)
	require.NoError(t, repl.runDeclaredCommandEmit(c, []string{"my_decision", "use yaml"}, func(string) {}))
	e, ok := store.Get("my_decision")
	require.True(t, ok, "a command's memory_save step writes the session's memory store")
	assert.Equal(t, "use yaml", e.Value)
	assert.Equal(t, memTypeDecision, e.Type)
}

func TestMemoryCaps_AreEvents(t *testing.T) {
	assert.Equal(t, 20, memoryToolResultsLimit())
	assert.Equal(t, 20, memoryThinkingLimit())
	assert.Equal(t, 50, memoryLowSignalLimit())
	assert.Equal(t, 60, memoryStatusLimit())
	assert.Equal(t, []string{memTypeNote, memTypeFact}, memoryRules.LowSignal)

	withEvent(t, Event{Name: eventMemoryLowSignal, Items: 3, Run: actionCap})
	withEvent(t, Event{Name: eventMemoryToolResults, Items: 2, Run: actionCap})
	m := NewMemoryStore(t.TempDir())
	m.SetCwd(t.TempDir())
	for i := range 5 {
		k := fmt.Sprintf("n%d", i)
		m.entries[k] = MemoryEntry{Key: k, Type: memTypeNote, UpdatedAt: int64(i)}
	}
	assert.Equal(t, 2, m.pruneLowSignal(), "the event's items: is the cap")
	for range 4 {
		m.ExtractToolResult("bash_exec", fmt.Sprintf("out %d", time.Now().UnixNano()))
		time.Sleep(2 * time.Millisecond)
	}
	n := 0
	for k := range m.entries {
		if strings.HasPrefix(k, "tool:bash_exec:") {
			n++
		}
	}
	assert.Equal(t, 2, n, "captures per tool stay at the event's items:")
}
