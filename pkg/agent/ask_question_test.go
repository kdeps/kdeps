package agent

import (
	"context"
	"encoding/json"
	"io"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

// scriptedCallStreamer answers the main loop's first round with calls, and
// every later main round with "after". A request offered no subagent tool is
// a subagent's, answered with "report" after a pause.
type scriptedCallStreamer struct {
	mu        sync.Mutex
	calls     []domain.StreamedToolCall
	mainHits  int
	running   int
	peak      int
	subagents int
}

func (s *scriptedCallStreamer) StreamChat(
	_ context.Context, cfg *domain.ChatConfig, w io.Writer,
) (string, []domain.StreamedToolCall, error) {
	names := []string{}
	for _, t := range cfg.Tools {
		names = append(names, t.Name)
	}
	if !slices.Contains(names, toolNameSubagent) {
		s.mu.Lock()
		s.subagents++
		s.running++
		s.peak = max(s.peak, s.running)
		s.mu.Unlock()
		time.Sleep(150 * time.Millisecond)
		s.mu.Lock()
		s.running--
		s.mu.Unlock()
		_, _ = io.WriteString(w, "report")
		return "report", nil, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.mainHits++
	if s.mainHits == 1 {
		return "", s.calls, nil
	}
	_, _ = io.WriteString(w, "after")
	return "after", nil, nil
}

func call(id, name string, args map[string]any) domain.StreamedToolCall {
	b, _ := json.Marshal(args)
	return domain.StreamedToolCall{ID: id, Name: name, Arguments: string(b)}
}

// TestAskQuestion_EndsTheTurnWithTheQuestion checks the question, with its
// options numbered, is the turn's reply and the model is not called again.
func TestAskQuestion_EndsTheTurnWithTheQuestion(t *testing.T) {
	s := &scriptedCallStreamer{calls: []domain.StreamedToolCall{call("1", toolNameAskQuestion, map[string]any{
		"question": "Which review mode?", "options": []string{"Interactive", "Consolidated"},
	})}}
	l := newSubagentTestLoop(t, s)

	out, err := l.RunStreaming(context.Background(), "review my design", io.Discard)
	require.NoError(t, err)
	assert.Contains(t, out, "Which review mode?")
	assert.Contains(t, out, "1. Interactive")
	assert.Contains(t, out, "2. Consolidated")
	assert.Equal(t, 1, s.mainHits, "the turn ends on the question")
}

func TestAskQuestion_NeedsAQuestion(t *testing.T) {
	l := newSubagentTestLoop(t, &scriptedCallStreamer{})
	_, err := l.askQuestion(map[string]any{"question": " "})
	require.Error(t, err)
}

func TestAskQuestion_OnlyOnATopLevelLoop(t *testing.T) {
	l := newSubagentTestLoop(t, &scriptedCallStreamer{})
	require.NotNil(t, l.registry.Get(toolNameAskQuestion))
	reg, err := l.subagentRegistry(nil)
	require.NoError(t, err)
	assert.Nil(t, reg.Get(toolNameAskQuestion), "a subagent has no user to ask")
}

func TestOptionList_ListOrLines(t *testing.T) {
	assert.Equal(t, []string{"a", "b"}, optionList([]any{"a", " b "}))
	assert.Equal(t, []string{"a", "b"}, optionList("a\n\nb"))
	assert.Empty(t, optionList(nil))
}

// TestRound_RunsLeadingSubagentCallsTogether checks a reply that opens with
// several subagent calls runs them all in one round, in parallel.
func TestRound_RunsLeadingSubagentCallsTogether(t *testing.T) {
	s := &scriptedCallStreamer{calls: []domain.StreamedToolCall{
		call("1", toolNameSubagent, map[string]any{"task": "review A"}),
		call("2", toolNameSubagent, map[string]any{"task": "review B"}),
		call("3", toolNameSubagent, map[string]any{"task": "review C"}),
	}}
	l := newSubagentTestLoop(t, s)

	out, err := l.RunStreaming(context.Background(), "review my design", io.Discard)
	require.NoError(t, err)
	assert.Contains(t, out, "after")
	assert.Equal(t, 3, s.subagents, "every subagent call ran")
	assert.Equal(t, 3, s.peak, "and they ran at once")
}

func TestRoundCalls_OnlyLeadingSubagents(t *testing.T) {
	a := call("1", toolNameSubagent, nil)
	b := call("2", "read_file", nil)
	assert.Len(t, roundCalls([]domain.StreamedToolCall{a, a, b, a}), 2)
	assert.Len(t, roundCalls([]domain.StreamedToolCall{b, a, a}), 1)
}
