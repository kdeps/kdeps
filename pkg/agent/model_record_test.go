package agent

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func withRecordHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestModelRecord_CollectsDistinctRefusalsPerModel(t *testing.T) {
	withRecordHome(t)
	l := &Loop{config: Config{Backend: "m365", Model: "m365-copilot"}}
	l.recordModelRefusal(refusalKindLimit, "I can't emit or execute the kdeps invoke blocks. They are not available.")
	l.recordModelRefusal(refusalKindLimit, "I CAN'T emit or execute the kdeps invoke blocks.")
	l.recordModelRefusal(refusalKindUnbacked, "Created [notes.md](https://x.invalid) containing hello")

	rec := modelRecordFor("m365", "m365-copilot")
	require.NotNil(t, rec)
	assert.Equal(t, 3, rec.Count, "every catch counts")
	require.Len(t, rec.Refusals, 2, "the same words are kept once")
	assert.Equal(
		t,
		"I CAN'T emit or execute the kdeps invoke blocks.",
		rec.Refusals[0].Text,
		"first sentence, newest wording",
	)
	assert.Equal(t, refusalKindUnbacked, rec.Refusals[1].Kind)
	assert.Nil(t, modelRecordFor("openai", "gpt-x"), "another model has its own record")

	assert.True(t, clearModelRecord("m365", "m365-copilot"))
	assert.Nil(t, modelRecordFor("m365", "m365-copilot"))
	assert.False(t, clearModelRecord("m365", "m365-copilot"))
}

func TestModelRecord_BoundedByEvent(t *testing.T) {
	withRecordHome(t)
	withEvent(t, Event{Name: eventModelRecordLimit, Items: 2, Run: actionCap})
	l := &Loop{config: Config{Backend: "b", Model: "m"}}
	for _, s := range []string{"One.", "Two.", "Three."} {
		l.recordModelRefusal(refusalKindSandbox, s)
	}
	rec := modelRecordFor("b", "m")
	require.NotNil(t, rec)
	assert.Equal(t, 3, rec.Count)
	assert.Equal(t, []string{"Two.", "Three."}, []string{rec.Refusals[0].Text, rec.Refusals[1].Text})
}

func TestModelRecordPreamble_RendersTheRecord(t *testing.T) {
	withRecordHome(t)
	l := &Loop{config: Config{Backend: "m365", Model: "quick"}}
	assert.Empty(t, l.modelRecordPreamble(), "no record, no text")
	l.recordModelRefusal(refusalKindLimit, "I can't access the repository.")
	got := l.modelRecordPreamble()
	assert.Contains(t, got, "m365/quick: 1 times")
	assert.Contains(t, got, `[invented limit] "I can't access the repository."`)
	assert.Contains(t, got, "self-imposed")
}

// A caught refusal in a real turn lands in the model's record, and the next
// preamble for that model carries it.
func TestRunStreaming_RefusalIsRecordedAndShownNextTime(t *testing.T) {
	withRecordHome(t)
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: []mockStreamResponse{
		{content: "I can't access the filesystem from here.", toolCalls: nil},
		{content: "The answer is 4.", toolCalls: nil},
	}}}
	loop := newStreamingLoop(ms, 10)
	var buf bytes.Buffer
	_, err := loop.RunStreaming(context.Background(), "list files", &buf)
	require.NoError(t, err)
	rec := modelRecordFor(loop.config.Backend, loop.config.Model)
	require.NotNil(t, rec)
	assert.Contains(t, rec.Refusals[0].Text, "I can't access the filesystem")
	assert.Contains(t, loop.modelRecordPreamble(), "I can't access the filesystem")
}
