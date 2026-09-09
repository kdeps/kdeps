set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop prompt refinement (/refine)..."

REFINE_HOME=$(mktemp -d)
trap 'rm -rf "$REFINE_HOME"' EXIT

OUTPUT=$(printf '/help\n/refine\n/refine off\n/refine\n/refine on\n/refine bogus\n/quit\n' \
    | HOME="$REFINE_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true)

if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "refine - agent loop starts"
else
    test_failed "refine - agent loop did not start" "Output: $OUTPUT"
fi

if output_grep_fixed "/refine" "$OUTPUT"; then
    test_passed "refine - /help lists /refine"
else
    test_failed "refine - /help missing /refine" "Output: $OUTPUT"
fi

if output_grep_i "prompt refinement is on" "$OUTPUT"; then
    test_passed "refine - on by default"
else
    test_failed "refine - not on by default" "Output: $OUTPUT"
fi

if output_grep_i "prompt refinement disabled" "$OUTPUT" \
    && output_grep_i "prompt refinement enabled" "$OUTPUT"; then
    test_passed "refine - /refine off|on both confirm"
else
    test_failed "refine - toggle did not confirm" "Output: $OUTPUT"
fi

# After "/refine off", a bare "/refine" must report the disabled state - proves
# the toggle held within the session.
if output_grep_i "prompt refinement is off" "$OUTPUT"; then
    test_passed "refine - /refine off takes effect for the session"
else
    test_failed "refine - /refine off not reflected by status" "Output: $OUTPUT"
fi

if output_grep_i "Usage: /refine" "$OUTPUT"; then
    test_passed "refine - bad argument shows usage"
else
    test_failed "refine - bad argument did not show usage" "Output: $OUTPUT"
fi
