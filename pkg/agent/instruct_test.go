package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

func TestBuildInstruct_AllTopics(t *testing.T) {
	loop := makeTestLoop(nil)
	out, ok := buildInstruct(loop, "")
	require.True(t, ok)

	assert.Contains(t, out, "# kdeps agent briefing")
	// every topic title is present
	for _, tp := range instructTopicList() {
		assert.Contains(t, out, "## "+tp.title)
	}
	// the how-to-call section keeps its indented <invoke> example verbatim
	assert.Contains(t, out, `  <invoke name="read_file">`)
	// hard-wrapped prose is collapsed back to flowing lines
	assert.Contains(t, out, "knowledge dependencies")
	assert.NotContains(t, out, "APIs\n-- not chatbots")
}

func TestBuildInstruct_SingleTopic(t *testing.T) {
	loop := makeTestLoop(nil)
	out, ok := buildInstruct(loop, "memory")
	require.True(t, ok)

	assert.Contains(t, out, "## Memory bridge (critical)")
	assert.Contains(t, out, "memory_search BEFORE every read")
	assert.NotContains(t, out, "## How to call a tool")
}

func TestBuildInstruct_CaseInsensitiveTopic(t *testing.T) {
	loop := makeTestLoop(nil)
	_, ok := buildInstruct(loop, "MEMORY")
	assert.True(t, ok)
}

func TestBuildInstruct_UnknownTopic(t *testing.T) {
	loop := makeTestLoop(nil)
	_, ok := buildInstruct(loop, "does-not-exist")
	assert.False(t, ok)
}

func TestBuildInstruct_AvailableToolsReflectsRegistry(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{
		Name:        "widget_probe",
		Description: "probes a widget",
		Category:    "code",
		Parameters:  map[string]domain.ToolParam{},
		Execute:     func(_ map[string]any) (string, error) { return "ok", nil },
	})
	loop := &Loop{registry: reg, session: NewSession(0), config: Config{Model: "test"}}

	out, ok := buildInstruct(loop, "available")
	require.True(t, ok)
	assert.Contains(t, out, "widget_probe")
	assert.Contains(t, out, "probes a widget")
}

func TestBuildInstruct_AvailableToolsEmpty(t *testing.T) {
	loop := makeTestLoop(nil) // no registry
	out, ok := buildInstruct(loop, "available")
	require.True(t, ok)
	assert.Contains(t, out, "No tools are registered")
}

func TestCollapseWrap(t *testing.T) {
	in := "line one\nline two\n\n  indented block\n  stays as-is\n\nlast para"
	got := collapseWrap(in)
	assert.Equal(t, "line one line two\n\n  indented block\n  stays as-is\n\nlast para", got)
}

func TestCmdInstruct_InjectsBriefingTurn(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(context.Background(), loop)
	defer repl.cancel()

	before := loop.Session().TurnCount()
	out := testCaptureStdout(t, func() {
		require.NoError(t, repl.dispatchCommand("/instruct memory"))
	})
	assert.Equal(t, before+1, loop.Session().TurnCount(), "briefing must be added as a turn")
	assert.Contains(t, out, "Briefing added to the model's context (memory)")

	msgs := loop.Session().Messages()
	require.NotEmpty(t, msgs)
	assert.Contains(t, msgs[len(msgs)-2].Content, "Memory bridge (critical)")
	assert.Equal(t, instructAck, msgs[len(msgs)-1].Content)
}

func TestCmdInstruct_ListDoesNotInject(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(context.Background(), loop)
	defer repl.cancel()

	before := loop.Session().TurnCount()
	out := testCaptureStdout(t, func() {
		require.NoError(t, repl.dispatchCommand("/instruct list"))
	})
	assert.Equal(t, before, loop.Session().TurnCount(), "/instruct list must not brief the model")
	assert.Contains(t, out, "Briefing topics:")
	assert.Contains(t, out, "overview")
}

func TestCmdInstruct_UnknownTopic(t *testing.T) {
	loop := makeTestLoop(nil)
	repl := NewREPL(context.Background(), loop)
	defer repl.cancel()

	before := loop.Session().TurnCount()
	out := testCaptureStdout(t, func() {
		require.NoError(t, repl.dispatchCommand("/instruct bogus"))
	})
	assert.Equal(t, before, loop.Session().TurnCount())
	assert.Contains(t, out, `Unknown topic "bogus"`)
}

func TestInstructCommand_InBuiltinList(t *testing.T) {
	assert.Contains(t, builtinCmds, "/instruct")
}

func TestInstructTopicNamesUnique(t *testing.T) {
	seen := map[string]bool{}
	for _, tp := range instructTopicList() {
		assert.False(t, seen[tp.name], "duplicate topic name %q", tp.name)
		seen[tp.name] = true
		assert.Equal(t, strings.ToLower(tp.name), tp.name, "topic names must be lowercase")
	}
}
