# shellcheck shell=bash
# E2E: theme selection for the agent loop (replaces the deprecated stealth
# on/off toggle -- /theme is now the sole control; "normal" is "stealth off",
# any other theme is "stealth on").
#
# Color correctness (near-black palette, no bright accents, no bold for the
# black theme) is verified by the Go tests (pkg/agent/theme_test.go,
# tests/integration/cmd - TestTheme_EndToEndPalette forces a truecolor
# profile). This script checks the cross-platform user-visible behavior: the
# --theme flag/KDEPS_THEME env start the loop, /help advertises /theme, and
# /theme <name> switches and confirms.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop theme selection..."

THEME_HOME=$(mktemp -d)
trap 'rm -rf "$THEME_HOME"' EXIT

# One session exercises the banner, /help, /theme list, and switching themes
# both ways.
OUTPUT=$(printf '/help\n/theme list\n/theme black\n/theme normal\n/quit\n' \
    | HOME="$THEME_HOME" timeout 60 "$KDEPS_BIN" --theme black 2>&1 || true)

if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "theme - --theme flag starts the agent loop"
else
    test_failed "theme - --theme flag did not start the loop" "Output: $OUTPUT"
fi

if output_grep_fixed "/theme" "$OUTPUT"; then
    test_passed "theme - /help lists /theme"
else
    test_failed "theme - /help missing /theme" "Output: $OUTPUT"
fi

if output_grep_i "theme set to black" "$OUTPUT" && output_grep_i "theme set to normal" "$OUTPUT"; then
    test_passed "theme - /theme <name> confirms both switches"
else
    test_failed "theme - /theme switch did not confirm" "Output: $OUTPUT"
fi

if output_grep_i "built-in" "$OUTPUT" && output_grep_fixed "vim" "$OUTPUT" && output_grep_i "custom" "$OUTPUT"; then
    test_passed "theme - /theme list shows built-in and custom themes separately"
else
    test_failed "theme - /theme list did not show the expected grouping" "Output: $OUTPUT"
fi

# KDEPS_THEME env starts the loop the same way.
OUTPUT=$(printf '/quit\n' | HOME="$THEME_HOME" KDEPS_THEME=vim timeout 60 "$KDEPS_BIN" 2>&1 || true)
if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "theme - KDEPS_THEME=vim starts the agent loop"
else
    test_failed "theme - KDEPS_THEME=vim did not start the loop" "Output: $OUTPUT"
fi
