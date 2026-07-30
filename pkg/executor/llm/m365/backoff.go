package m365

import (
	"context"
	"os"
	"strconv"
	"time"

	"github.com/kdeps/kdeps/v2/pkg/executor/llm/toolguard"
)

// Degradation backoff: account throttle surfaces as empty responses across many
// distinct conversations. Re-authenticating does not clear it (the throttle is
// identity-keyed and a fresh token carries the same oid), so instead of logging
// in again we self-impose a paced delay before starting new turns, letting the
// account self-heal. A single long conversation never trips the trigger.
//
// The pacing mechanism itself (toolguard.BackoffController) is shared with the
// rest of kdeps; this file only wires it up with M365-specific env overrides and
// a process-wide default instance.

const (
	defaultBackoffWindowMs  = 120_000
	defaultBackoffThreshold = 3
	defaultBackoffBaseMs    = 90_000
	defaultBackoffMaxMs     = 600_000
)

//nolint:gochecknoglobals // process-wide degradation policy
var defaultBackoff = toolguard.NewBackoffController(toolguard.BackoffOptions{
	Window:       durationEnv("M365_BACKOFF_WINDOW_MS", defaultBackoffWindowMs),
	Threshold:    intEnv("M365_BACKOFF_THRESHOLD", defaultBackoffThreshold),
	BaseCooldown: durationEnv("M365_BACKOFF_BASE_MS", defaultBackoffBaseMs),
	MaxCooldown:  durationEnv("M365_BACKOFF_MAX_MS", defaultBackoffMaxMs),
})

func backoffDisabled() bool {
	return os.Getenv("M365_NO_BACKOFF") != "" || os.Getenv("M365_NO_AUTO_REAUTH") != ""
}

// noteRequestOutcome feeds the global degradation policy. No-op when disabled.
func noteRequestOutcome(empty bool, conversationID string) {
	if backoffDisabled() {
		return
	}
	defaultBackoff.Note(empty, conversationID)
}

// awaitDegradationBackoff paces a turn while the account is degraded, returning
// immediately when healthy or disabled.
func awaitDegradationBackoff(ctx context.Context) {
	if backoffDisabled() {
		return
	}
	defaultBackoff.WaitForSlot(ctx)
}

func intEnv(name string, def int) int {
	if v := os.Getenv(name); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func durationEnv(name string, defMs int) time.Duration {
	return time.Duration(intEnv(name, defMs)) * time.Millisecond
}
