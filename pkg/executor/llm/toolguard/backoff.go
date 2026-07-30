package toolguard

import (
	"context"
	"math"
	"math/rand/v2"
	"sync"
	"time"
)

// BackoffController paces new turns after a burst of empty/failed responses
// spread across distinct conversations, giving a degraded upstream (rate
// limiting, transient outage) room to self-heal instead of hammering it.
// Repeated empties within a SINGLE conversation don't trip it — that's usually
// content-specific (a bad prompt), not upstream throttling.
type BackoffController struct {
	mu           sync.Mutex
	window       time.Duration
	threshold    int
	baseCooldown time.Duration
	maxCooldown  time.Duration
	jitterMin    time.Duration
	jitterMax    time.Duration

	empties      []emptyEvent
	backoffUntil time.Time
	level        int
}

type emptyEvent struct {
	t    time.Time
	conv string
}

// BackoffOptions configures a BackoffController. Zero values fall back to
// reasonable defaults.
type BackoffOptions struct {
	Window       time.Duration
	Threshold    int
	BaseCooldown time.Duration
	MaxCooldown  time.Duration
	JitterMin    time.Duration
	JitterMax    time.Duration
}

const (
	defaultBackoffWindow    = 120 * time.Second
	defaultBackoffThreshold = 3
	defaultBackoffBase      = 90 * time.Second
	defaultBackoffMax       = 600 * time.Second
	defaultJitterMin        = 10 * time.Second
	defaultJitterMax        = 25 * time.Second
	backoffFactor           = 2
)

// NewBackoffController creates a controller, applying defaults for any zero
// field in opts.
func NewBackoffController(opts BackoffOptions) *BackoffController {
	b := &BackoffController{
		window:       opts.Window,
		threshold:    opts.Threshold,
		baseCooldown: opts.BaseCooldown,
		maxCooldown:  opts.MaxCooldown,
		jitterMin:    opts.JitterMin,
		jitterMax:    opts.JitterMax,
	}
	if b.window == 0 {
		b.window = defaultBackoffWindow
	}
	if b.threshold == 0 {
		b.threshold = defaultBackoffThreshold
	}
	if b.baseCooldown == 0 {
		b.baseCooldown = defaultBackoffBase
	}
	if b.maxCooldown == 0 {
		b.maxCooldown = defaultBackoffMax
	}
	if b.jitterMin == 0 {
		b.jitterMin = defaultJitterMin
	}
	if b.jitterMax == 0 {
		b.jitterMax = defaultJitterMax
	}
	return b
}

// Note records one request outcome. empty means a throttle-shaped empty reply;
// conversationID identifies the conversation the request belongs to.
func (b *BackoffController) Note(empty bool, conversationID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()

	if !empty {
		b.empties = nil
		b.backoffUntil = time.Time{}
		b.level = 0
		return
	}

	if conversationID == "" {
		conversationID = "anon"
	}
	b.empties = append(b.empties, emptyEvent{t: now, conv: conversationID})
	kept := b.empties[:0]
	for _, e := range b.empties {
		if now.Sub(e.t) < b.window {
			kept = append(kept, e)
		}
	}
	b.empties = kept

	distinct := map[string]struct{}{}
	for _, e := range b.empties {
		distinct[e.conv] = struct{}{}
	}
	if len(distinct) < b.threshold {
		return
	}
	if now.Before(b.backoffUntil) {
		return // already backing off; don't stack
	}

	b.level++
	cooldown := time.Duration(math.Min(
		float64(b.baseCooldown)*math.Pow(backoffFactor, float64(b.level-1)),
		float64(b.maxCooldown),
	))
	b.backoffUntil = now.Add(cooldown)
	b.empties = nil
}

// WaitForSlot sleeps a jittered delay when in a backoff window, or returns
// immediately when healthy. Returns the slept duration.
func (b *BackoffController) WaitForSlot(ctx context.Context) time.Duration {
	b.mu.Lock()
	remaining := time.Until(b.backoffUntil)
	jitterMin, jitterMax := b.jitterMin, b.jitterMax
	b.mu.Unlock()

	if remaining <= 0 {
		return 0
	}
	span := jitterMax - jitterMin
	delay := jitterMin
	if span > 0 {
		//nolint:gosec // jitter timing, not security-sensitive
		delay += time.Duration(rand.Int64N(int64(span) + 1))
	}
	if delay > remaining {
		delay = remaining
	}
	if delay <= 0 {
		return 0
	}
	select {
	case <-ctx.Done():
	case <-time.After(delay):
	}
	return delay
}

// IsBackingOff reports whether the controller is currently in a backoff window.
func (b *BackoffController) IsBackingOff() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return time.Now().Before(b.backoffUntil)
}
