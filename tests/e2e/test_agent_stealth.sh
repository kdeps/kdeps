# shellcheck shell=bash
# E2E: stealth ("Muted") mode for the agent loop.
#
# Color correctness (near-black palette, no bright accents, no bold) is verified
# by the Go tests (pkg/agent/theme_test.go, tests/integration/cmd -
# TestStealth_EndToEndPalette forces a truecolor profile). Persistence of
# /stealth is covered by pkg/agent/repl_stealth_test.go. This script checks the
# cross-platform user-visible behavior: the flag/env start the loop, /help
# advertises the command, and /stealth toggles both ways.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop stealth mode..."

STEALTH_HOME=$(mktemp -d)
trap 'rm -rf "$STEALTH_HOME"' EXIT

# One session exercises the banner, /help, and both /stealth directions.
OUTPUT=$(printf '/help\n/stealth off\n/stealth on\n/quit\n' \
    | HOME="$STEALTH_HOME" timeout 60 "$KDEPS_BIN" --stealth 2>&1 || true)

if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "stealth - --stealth flag starts the agent loop"
else
    test_failed "stealth - --stealth flag did not start the loop" "Output: $OUTPUT"
fi

if output_grep_fixed "/stealth" "$OUTPUT"; then
    test_passed "stealth - /help lists /stealth"
else
    test_failed "stealth - /help missing /stealth" "Output: $OUTPUT"
fi

if output_grep_i "stealth mode on" "$OUTPUT" && output_grep_i "stealth mode off" "$OUTPUT"; then
    test_passed "stealth - /stealth on|off both confirm"
else
    test_failed "stealth - /stealth toggle did not confirm" "Output: $OUTPUT"
fi

# KDEPS_STEALTH env starts the loop the same way.
OUTPUT=$(printf '/quit\n' | HOME="$STEALTH_HOME" KDEPS_STEALTH=1 timeout 60 "$KDEPS_BIN" 2>&1 || true)
if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "stealth - KDEPS_STEALTH=1 starts the agent loop"
else
    test_failed "stealth - KDEPS_STEALTH=1 did not start the loop" "Output: $OUTPUT"
fi
