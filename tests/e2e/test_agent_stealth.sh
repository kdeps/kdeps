# shellcheck shell=bash
# E2E: stealth ("Muted") mode for the agent loop.
#
# Color correctness (near-black palette, no bright accents, no bold) is verified
# by the Go tests (pkg/agent/theme_test.go, tests/integration/cmd -
# TestStealth_EndToEndPalette forces a truecolor profile). This script checks
# the user-visible behavior: the flag/env start the loop, /help advertises the
# command, /stealth toggles, and the toggle persists.
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

# /stealth on persists to the settings file for the next launch.
SETTINGS="$STEALTH_HOME/.kdeps/agent-loop-settings.yaml"
printf '/stealth on\n/quit\n' | HOME="$STEALTH_HOME" timeout 60 "$KDEPS_BIN" >/dev/null 2>&1 || true
if [ -f "$SETTINGS" ] && grep -q 'stealth: true' "$SETTINGS"; then
    test_passed "stealth - /stealth on persists (stealth: true)"
else
    test_failed "stealth - /stealth on did not persist" "$(cat "$SETTINGS" 2>/dev/null)"
fi
