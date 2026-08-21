package toolguard

import (
	"context"
	"testing"
	"time"
)

func TestLooksLikeConfabulation(t *testing.T) {
	if !LooksLikeConfabulation("I cannot access the files, please paste them") {
		t.Error("should detect confabulation")
	}
	if LooksLikeConfabulation("Here is the answer to your question.") {
		t.Error("false positive")
	}
	if LooksLikeConfabulation("short") {
		t.Error("too short should be false")
	}
}

func TestLooksLikeHallucinatedCompletion(t *testing.T) {
	if !LooksLikeHallucinatedCompletion("I've created the file and updated the README") {
		t.Error("should detect hallucination")
	}
	if LooksLikeHallucinatedCompletion("What would you like to do?") {
		t.Error("false positive")
	}
	if LooksLikeHallucinatedCompletion("short") {
		t.Error("too short should be false")
	}
}

// /mnt/data is the mount of the provider's own sandbox filesystem (e.g. M365
// Copilot's code interpreter), not the caller's real disk. A reply reporting
// facts about it is truthful about the wrong machine, confirmed as a real
// live failure mode (a model explored an empty /mnt/data and reported "no
// project found" for a real, populated local repository).
func TestLooksLikeHallucinatedCompletion_ProviderSandboxPath(t *testing.T) {
	if !LooksLikeHallucinatedCompletion(
		"## Current project status\n\n- Project directory: `/mnt/data`\n- Files present: None",
	) {
		t.Error("should flag a reply reporting facts about /mnt/data as a hallucinated completion")
	}
}

func TestBackoffControllerDistinctConversationGating(t *testing.T) {
	b := NewBackoffController(BackoffOptions{Threshold: 2, BaseCooldown: time.Minute})

	// Same conversation repeated does not trip (distinct-conversation gating).
	b.Note(true, "conv-a")
	b.Note(true, "conv-a")
	if b.WaitForSlot(context.Background()) != 0 {
		t.Error("single-conversation empties must not back off")
	}
	if b.IsBackingOff() {
		t.Error("should not be backing off yet")
	}

	// Distinct conversations reach threshold.
	b.Note(true, "conv-b")
	if !b.IsBackingOff() {
		t.Error("distinct-conversation empties should engage backoff")
	}

	// A clean response lifts backoff.
	b.Note(false, "conv-b")
	if b.IsBackingOff() {
		t.Error("clean response should lift backoff")
	}
}

func TestBackoffControllerEscalation(t *testing.T) {
	b := NewBackoffController(BackoffOptions{
		Threshold:    2,
		BaseCooldown: time.Hour, // large so we can observe escalation without racing the clock
		MaxCooldown:  10 * time.Hour,
	})
	b.Note(true, "a")
	b.Note(true, "b") // trips level 1
	b.mu.Lock()
	level1 := b.level
	until1 := b.backoffUntil
	b.mu.Unlock()
	if level1 != 1 {
		t.Fatalf("level = %d, want 1", level1)
	}

	// Force the window closed so a second trip is possible, and escalate.
	b.mu.Lock()
	b.backoffUntil = time.Now().Add(-time.Second)
	b.mu.Unlock()
	b.Note(true, "c")
	b.Note(true, "d")
	b.mu.Lock()
	level2 := b.level
	until2 := b.backoffUntil
	b.mu.Unlock()
	if level2 != 2 {
		t.Fatalf("level = %d, want 2", level2)
	}
	if !until2.After(until1) {
		t.Error("escalated cooldown should extend further than the first")
	}
}

func TestBackoffControllerDefaults(t *testing.T) {
	b := NewBackoffController(BackoffOptions{})
	if b.window != defaultBackoffWindow || b.threshold != defaultBackoffThreshold {
		t.Errorf("defaults not applied: window=%v threshold=%d", b.window, b.threshold)
	}
}

func TestWaitForSlotContextCancel(t *testing.T) {
	b := NewBackoffController(BackoffOptions{Threshold: 1, BaseCooldown: time.Hour})
	b.Note(true, "x")
	if !b.IsBackingOff() {
		t.Fatal("expected backoff engaged")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	// A cancelled context must not hang; WaitForSlot returns promptly either way.
	done := make(chan struct{})
	go func() {
		b.WaitForSlot(ctx)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("WaitForSlot did not respect context cancellation")
	}
}
