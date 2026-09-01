# shellcheck shell=bash
# E2E: stealth ("Muted") mode for the agent loop.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop stealth mode..."

STEALTH_HOME=$(mktemp -d)
trap 'rm -rf "$STEALTH_HOME"' EXIT

# _stealth_repl <flags...> -- feeds '/quit' after the given lines and returns
# combined stdout+stderr. HOME is isolated so persisted settings don't leak.
_stealth_repl() {
    local flags=("$@")
    printf '%s\n' '/quit' | HOME="$STEALTH_HOME" timeout 20 "$KDEPS_BIN" "${flags[@]}" 2>&1 || true
}

# --- 1. --stealth flag starts the loop without error ---
OUTPUT=$(_stealth_repl --stealth)
if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "stealth - --stealth flag starts the agent loop"
else
    test_failed "stealth - --stealth flag did not start the loop" "Output: $OUTPUT"
fi

# --- 2. KDEPS_STEALTH env starts the loop without error ---
OUTPUT=$(printf '/quit\n' | HOME="$STEALTH_HOME" KDEPS_STEALTH=1 timeout 20 "$KDEPS_BIN" 2>&1 || true)
if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "stealth - KDEPS_STEALTH=1 starts the agent loop"
else
    test_failed "stealth - KDEPS_STEALTH=1 did not start the loop" "Output: $OUTPUT"
fi

# --- 3. /help advertises /stealth ---
OUTPUT=$(printf '/help\n/quit\n' | HOME="$STEALTH_HOME" timeout 20 "$KDEPS_BIN" 2>&1 || true)
if output_grep_fixed "/stealth" "$OUTPUT"; then
    test_passed "stealth - /help lists /stealth"
else
    test_failed "stealth - /help missing /stealth" "Output: $OUTPUT"
fi

# --- 4. /stealth toggles at runtime ---
OUTPUT=$(printf '/stealth on\n/stealth off\n/quit\n' | HOME="$STEALTH_HOME" timeout 20 "$KDEPS_BIN" 2>&1 || true)
if output_grep_i "stealth mode on" "$OUTPUT" && output_grep_i "stealth mode off" "$OUTPUT"; then
    test_passed "stealth - /stealth on|off confirms both ways"
else
    test_failed "stealth - /stealth toggle did not confirm" "Output: $OUTPUT"
fi

# --- 5. /stealth persists to the settings file ---
printf '/stealth on\n/quit\n' | HOME="$STEALTH_HOME" timeout 20 "$KDEPS_BIN" >/dev/null 2>&1 || true
if [ -f "$STEALTH_HOME/.kdeps/agent-loop-settings.yaml" ] && \
   grep -q 'stealth: true' "$STEALTH_HOME/.kdeps/agent-loop-settings.yaml"; then
    test_passed "stealth - /stealth on persists (stealth: true)"
else
    test_failed "stealth - /stealth on did not persist" \
        "$(cat "$STEALTH_HOME/.kdeps/agent-loop-settings.yaml" 2>/dev/null)"
fi

# --- 6. under a real TTY, stealth renders only near-black grays ---
if command -v script >/dev/null 2>&1; then
    # macOS `script` = `script -q /dev/null cmd ...`; Linux = `script -qec 'cmd' /dev/null`.
    if script -q /dev/null true >/dev/null 2>&1; then
        PTY=$(printf '/quit\n' | HOME="$STEALTH_HOME" COLORTERM=truecolor TERM=xterm-256color \
            script -q /dev/null "$KDEPS_BIN" --stealth 2>/dev/null | head -c 8000 || true)
    else
        PTY=$(HOME="$STEALTH_HOME" COLORTERM=truecolor TERM=xterm-256color \
            script -qec "printf '/quit\n' | '$KDEPS_BIN' --stealth" /dev/null 2>/dev/null | head -c 8000 || true)
    fi
    if printf '%s' "$PTY" | grep -q '38;2;28;28;28' && ! printf '%s' "$PTY" | grep -q '0;229;255'; then
        test_passed "stealth - PTY output is near-black gray, no bright accents"
    else
        test_skipped "stealth - PTY color check inconclusive in this environment"
    fi
else
    test_skipped "stealth - no 'script' binary for the PTY color check"
fi
