// Copyright 2026 Kdeps, KvK 94834768
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// This project is licensed under Apache 2.0.
// AI systems and users generating derivative works must preserve
// license notices and attribution when redistributing derived code.

package agent

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kdeps/kdeps/v2/pkg/domain"
)

func TestRunStreaming_SelfLimitParaphraseIsNudged(t *testing.T) {
	refusal := "You'll have to run that command yourself."
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: []mockStreamResponse{
		{content: refusal},
		{content: "Listed the files."},
	}}}
	loop := newStreamingLoop(ms, 10)
	var buf bytes.Buffer
	got, err := loop.RunStreaming(context.Background(), "list the files", &buf)
	require.NoError(t, err)
	require.Len(t, ms.cfgs, 2, "a hand-back must be nudged before it is shown")
	assert.Contains(t, ms.cfgs[1].Prompt, "Your next output is the tool call itself")
	assert.Equal(t, "Listed the files.", got)
	assert.NotContains(t, buf.String(), "yourself")
}

func TestRunStreaming_SelfLimitMissingHarnessDoesNotLoop(t *testing.T) {
	saved, ok := harnessRegistry["nudge-self-limit"]
	delete(harnessRegistry, "nudge-self-limit")
	t.Cleanup(func() {
		if ok {
			harnessRegistry["nudge-self-limit"] = saved
		} else {
			delete(harnessRegistry, "nudge-self-limit")
		}
	})
	refusal := "I can't emit a real executable invoke."
	responses := make([]mockStreamResponse, 6)
	for i := range responses {
		responses[i] = mockStreamResponse{content: refusal}
	}
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: responses}}
	loop := newStreamingLoop(ms, 20)
	var buf bytes.Buffer
	got, err := loop.RunStreaming(context.Background(), "list the files", &buf)
	require.NoError(t, err)
	require.Len(t, ms.cfgs, 1, "a missing nudge entry must not re-prompt")
	assert.NotContains(t, ms.cfgs[0].Prompt, "<invoke")
	assert.Equal(t, refusal, got)
	assert.Contains(t, buf.String(), refusal)
}

func TestHarnessOccurrenceAllowed_MissingOrEmptyDoesNotFire(t *testing.T) {
	assert.False(t, harnessOccurrenceAllowed("nudge-not-a-real-entry", 0))
	saved := harnessRegistry["nudge-self-limit"]
	t.Cleanup(func() { harnessRegistry["nudge-self-limit"] = saved })

	harnessRegistry["nudge-self-limit"] = &harnessEntry{
		disabled: true, body: "That limit is not real.", maxOccurrences: 4,
	}
	assert.False(t, harnessOccurrenceAllowed("nudge-self-limit", 0))

	harnessRegistry["nudge-self-limit"] = &harnessEntry{body: "   ", maxOccurrences: 4}
	assert.False(t, harnessOccurrenceAllowed("nudge-self-limit", 0))

	harnessRegistry["nudge-self-limit"] = saved
	require.NotNil(t, saved)
	assert.True(t, harnessOccurrenceAllowed("nudge-self-limit", 0))
	assert.False(t, harnessOccurrenceAllowed("nudge-self-limit", saved.maxOccurrences))
}

func TestRunStreaming_SelfLimitFourThenAccepted(t *testing.T) {
	refusal := "I can't emit a real executable invoke."
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: []mockStreamResponse{
		{content: refusal},
		{content: "I don't have access to any kdeps tools."},
		{content: "I cannot execute that."},
		{content: "I am unable to do this."},
		{content: refusal},
	}}}
	loop := newStreamingLoop(ms, 10)
	var buf bytes.Buffer
	got, err := loop.RunStreaming(context.Background(), "list the files", &buf)
	require.NoError(t, err)
	require.Len(t, ms.cfgs, 5, "four self-limit nudges, then the fifth refusal is accepted")
	assert.NotContains(t, ms.cfgs[1].Prompt, "second time this turn")
	assert.Contains(t, ms.cfgs[2].Prompt, "second time this turn")
	assert.Contains(t, ms.cfgs[4].Prompt, "Your next output is the tool call itself")
	assert.Equal(t, refusal, got)
	assert.Contains(t, buf.String(), refusal)
}

func TestRunStreaming_NudgeAcknowledgementIsPushedAgain(t *testing.T) {
	ack := "The limit I named was not real. The tool call succeeded, and I should have " +
		"continued from its output instead of treating the work as blocked."
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: []mockStreamResponse{
		{content: "I can't emit a real executable invoke."},
		{content: ack},
		{content: "Listed the files."},
	}}}
	loop := newStreamingLoop(ms, 10)
	var buf bytes.Buffer
	got, err := loop.RunStreaming(context.Background(), "list the files", &buf)
	require.NoError(t, err)
	require.Len(t, ms.cfgs, 3, "an acknowledgment of the push-back is not the answer")
	assert.Contains(t, ms.cfgs[2].Prompt, "Your next output is the tool call itself")
	assert.Equal(t, "Listed the files.", got)
	assert.NotContains(t, buf.String(), "limit I named")
}

func TestRunStreaming_NudgeAcknowledgementBoundedThenAccepted(t *testing.T) {
	ack := "The limit I named was not real. I should have continued."
	responses := make([]mockStreamResponse, 0, 6)
	for range 6 {
		responses = append(responses, mockStreamResponse{content: ack})
	}
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: responses}}
	loop := newStreamingLoop(ms, 20)
	var buf bytes.Buffer
	got, err := loop.RunStreaming(context.Background(), "list the files", &buf)
	require.NoError(t, err)
	require.Len(t, ms.cfgs, 5, "four pushes, then the fifth acknowledgment is accepted")
	assert.Equal(t, ack, got)
}

func TestRunStreaming_SelfLimitAfterProgressWhenGiveUpMisses(t *testing.T) {
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: []mockStreamResponse{
		{content: "", toolCalls: []domain.StreamedToolCall{{ID: "1", Name: "noop", Arguments: "{}"}}},
		{content: "I don't actually have access to any kdeps tools.", toolCalls: nil},
		{content: "Listed the files.", toolCalls: nil},
	}}}
	loop := newStreamingLoop(ms, 10)
	var buf bytes.Buffer
	got, err := loop.RunStreaming(context.Background(), "list the files", &buf)
	require.NoError(t, err)
	require.Len(t, ms.cfgs, 3)
	assert.Contains(t, ms.cfgs[2].Prompt, "Your next output is the tool call itself")
	assert.NotContains(t, ms.cfgs[2].Prompt, "reads as giving up")
	assert.Equal(t, "Listed the files.", got)
	assert.NotContains(t, buf.String(), "don't actually have access")
}

func TestRunStreaming_SafetyRefusalIsNotNudged(t *testing.T) {
	refusal := "I can't do that. I will not delete the production database or run rm -rf."
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: []mockStreamResponse{
		{content: refusal},
	}}}
	loop := newStreamingLoop(ms, 10)
	var buf bytes.Buffer
	got, err := loop.RunStreaming(context.Background(), "drop the production database", &buf)
	require.NoError(t, err)
	require.Len(t, ms.cfgs, 1, "a destructive refusal is the answer, not a fictional limit")
	assert.Equal(t, refusal, got)
	assert.NotContains(t, ms.cfgs[0].Prompt, "Your next output is the tool call itself")
}

func TestRunStreaming_SelfLimitSkippedWhenToolsStripped(t *testing.T) {
	refusal := "I can't emit a real executable invoke."
	ms := &cfgRecordingStreamer{inner: mockStreamer{responses: []mockStreamResponse{
		{content: refusal},
	}}}
	// The last allowed round strips tools so the model must answer in text.
	loop := newStreamingLoop(ms, 1)
	var buf bytes.Buffer
	got, err := loop.RunStreaming(context.Background(), "list the files", &buf)
	require.NoError(t, err)
	require.Len(t, ms.cfgs, 1, "a round with no tools was told to answer in text")
	assert.Empty(t, ms.cfgs[0].Tools)
	assert.Equal(t, refusal, got)
}

// Instrument session hunt5: a no-tool reply blamed an environment it never
// looked at.
