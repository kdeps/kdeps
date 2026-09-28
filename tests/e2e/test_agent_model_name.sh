# shellcheck shell=bash
# E2E: /model name (show|hide|abbreviate|auto) overrides how the modeline
# displays the model name, independent of the active theme.
#
# Color/legibility of the palettes themselves is verified by the Go tests
# (pkg/agent/theme_test.go, tests/integration/cmd). This script checks the
# cross-platform user-visible behavior: /help advertises /model name, and
# each mode switches and confirms.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing /model name display modes..."

MODELNAME_HOME=$(mktemp -d)
trap 'rm -rf "$MODELNAME_HOME"' EXIT

OUTPUT=$(printf '/help\n/model name show\n/model name hide\n/model name abbreviate\n/model name auto\n/quit\n' \
    | HOME="$MODELNAME_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true)

if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "model-name - agent loop starts"
else
    test_failed "model-name - agent loop did not start" "Output: $OUTPUT"
fi

if output_grep_fixed "/model name" "$OUTPUT"; then
    test_passed "model-name - /help lists /model name"
else
    test_failed "model-name - /help missing /model name" "Output: $OUTPUT"
fi

if output_grep_i "model name display set to show" "$OUTPUT" \
    && output_grep_i "model name display set to hide" "$OUTPUT" \
    && output_grep_i "model name display set to abbreviate" "$OUTPUT" \
    && output_grep_i "model name display set to auto" "$OUTPUT"; then
    test_passed "model-name - all four modes confirm"
else
    test_failed "model-name - a mode switch did not confirm" "Output: $OUTPUT"
fi
