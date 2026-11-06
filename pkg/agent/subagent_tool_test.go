package agent

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
	"github.com/kdeps/kdeps/v2/pkg/executor"
	"github.com/kdeps/kdeps/v2/pkg/tools"
)

// subagentStreamer answers a subagent's request with a fixed report and
// records what each subagent request carried. The main loop calls subagent
// once with args, then answers with the tool result.
type subagentStreamer struct {
	mu       sync.Mutex
	args     map[string]any
	report   string
	subTools [][]string
	subText  []string
	mainHits int
}

func chatText(cfg *domain.ChatConfig) string {
	parts := []string{cfg.Prompt}
	for _, s := range cfg.Scenario {
		parts = append(parts, s.Prompt)
	}
	return strings.Join(parts, "\n")
}

func (s *subagentStreamer) StreamChat(
	_ context.Context, cfg *domain.ChatConfig, w io.Writer,
) (string, []domain.StreamedToolCall, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	text := chatText(cfg)
	if strings.Contains(text, "You are a subagent") {
		names := make([]string, 0, len(cfg.Tools))
		for _, t := range cfg.Tools {
			names = append(names, t.Name)
		}
		s.subTools = append(s.subTools, names)
		s.subText = append(s.subText, text)
		_, _ = io.WriteString(w, s.report)
		return s.report, nil, nil
	}
	s.mainHits++
	if s.mainHits == 1 {
		args, _ := json.Marshal(s.args)
		return "", []domain.StreamedToolCall{{ID: "1", Name: toolNameSubagent, Arguments: string(args)}}, nil
	}
	return "done", nil, nil
}

func newSubagentTestLoop(t *testing.T, streamer Streamer) *Loop {
	t.Helper()
	eng := executor.NewEngine(context.Background(), nil)
	reg := tools.NewRegistry()
	reg.Register(&tools.Tool{Name: "read_file", Execute: func(map[string]any) (string, error) { return "x", nil }})
	reg.Register(&tools.Tool{Name: "bash_exec", Execute: func(map[string]any) (string, error) { return "x", nil }})
	return New(context.Background(), eng, newTestWorkflowForSession(), reg, Config{Model: "test", Streamer: streamer})
}

func TestSubagent_RegisteredOnATopLevelLoopOnly(t *testing.T) {
	l := newSubagentTestLoop(t, &subagentStreamer{})
	require.NotNil(t, l.registry.Get(toolNameSubagent))

	nested := New(context.Background(), l.engine, l.workflow, tools.NewRegistry(), Config{Model: "test", Nested: true})
	assert.Nil(t, nested.registry.Get(toolNameSubagent), "a side loop cannot start a subagent")
}

func TestSubagent_RunsAFreshLoopAndReturnsItsReport(t *testing.T) {
	s := &subagentStreamer{report: "REPORT: two gaps", args: map[string]any{
		"task": "Review plan.md", "instructions": "Be a strict reviewer.",
	}}
	l := newSubagentTestLoop(t, s)

	out, err := l.runSubagent(s.args)
	require.NoError(t, err)
	assert.Equal(t, "REPORT: two gaps", out)
	require.Len(t, s.subTools, 1)
	assert.Contains(t, s.subTools[0], "read_file", "read-only tools are offered by default")
	assert.NotContains(t, s.subTools[0], "bash_exec", "a write tool is never offered")
	assert.NotContains(t, s.subTools[0], toolNameSubagent, "a subagent cannot start another")
	assert.Contains(t, s.subText[0], "Review plan.md")
	assert.Contains(t, s.subText[0], "Be a strict reviewer.")
}

func TestSubagent_SeesNoConversation(t *testing.T) {
	s := &subagentStreamer{report: "ok", args: map[string]any{"task": "Check the README"}}
	l := newSubagentTestLoop(t, s)
	_, err := l.RunStreaming(context.Background(), "secret-earlier-turn-marker", io.Discard)
	require.NoError(t, err)

	require.Len(t, s.subText, 1)
	assert.NotContains(t, s.subText[0], "secret-earlier-turn-marker", "the subagent starts with no history")
}

func TestSubagent_RefusesAToolThatIsNotReadOnly(t *testing.T) {
	l := newSubagentTestLoop(t, &subagentStreamer{report: "ok"})
	_, err := l.runSubagent(map[string]any{"task": "x", "tools": "read_file, bash_exec"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "bash_exec")
}

func TestSubagent_NamedToolsNarrowTheSet(t *testing.T) {
	reg, err := newSubagentTestLoop(t, &subagentStreamer{}).subagentRegistry([]string{"read_file"})
	require.NoError(t, err)
	assert.NotNil(t, reg.Get("read_file"))
	assert.Len(t, reg.List(), 1)
}

func TestSubagent_NeedsATask(t *testing.T) {
	_, err := newSubagentTestLoop(t, &subagentStreamer{}).runSubagent(map[string]any{"task": "  "})
	require.Error(t, err)
}

func TestSubagent_EmptyReportIsAnError(t *testing.T) {
	l := newSubagentTestLoop(t, &subagentStreamer{report: ""})
	_, err := l.runSubagent(map[string]any{"task": "x"})
	require.Error(t, err, "the loop's empty-turn notice is not a report")
	assert.Contains(t, err.Error(), "returned nothing")
}

func TestSubagent_NoStreamerIsAnError(t *testing.T) {
	l := newSubagentTestLoop(t, nil)
	l.streamer = nil
	_, err := l.runSubagent(map[string]any{"task": "x"})
	require.Error(t, err)
}

func TestSideLoopConfig_UsesTheSubagentChannel(t *testing.T) {
	l := newSubagentTestLoop(t, &subagentStreamer{})
	l.config.Subagent = "pinned-model"
	cfg := l.sideLoopConfig("sys", 7)
	assert.Equal(t, "pinned-model", cfg.Model)
	assert.Equal(t, "sys", cfg.SystemPrompt)
	assert.Equal(t, 7, cfg.MaxToolRounds)
	assert.True(t, cfg.Nested)
}

// TestSideLoop_SendsEveryToolAndNoLoader checks a judge or subagent gets its
// whole scoped tool set on the first request, with no load_tools to misuse.
func TestSideLoop_SendsEveryToolAndNoLoader(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(defined(&tools.Tool{Name: "wikipedia", Execute: func(map[string]any) (string, error) { return "", nil }}))
	require.NotNil(t, reg.Get("wikipedia"), "wikipedia ships a definition")
	require.True(t, reg.Get("wikipedia").OnDemand, "and is deferred on a top-level loop")

	l := New(context.Background(), executor.NewEngine(context.Background(), nil), newTestWorkflowForSession(), reg,
		Config{Model: "test", Nested: true})
	assert.Nil(t, l.registry.Get(toolNameLoadTools))
	names := []string{}
	for _, td := range l.requestTools() {
		names = append(names, td.Name)
	}
	assert.Contains(t, names, "wikipedia", "a deferred tool is sent in full to a side loop")
}
