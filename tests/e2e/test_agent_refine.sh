set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop prompt refinement (/refine)..."

REFINE_HOME=$(mktemp -d)
trap 'rm -rf "$REFINE_HOME"' EXIT

run_refine() {
    printf '%s\n' "$1" | HOME="$REFINE_HOME" USERPROFILE="$(to_native_path "$REFINE_HOME")" timeout 60 "$KDEPS_BIN" 2>&1 || true
}

STATUS=$(run_refine $'/refine\n/quit')
OUTPUT=$(run_refine $'/help\n/refine on\n/refine\n/refine off\n/refine\n/refine bogus\n/quit')

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

if output_grep_i "prompt refinement is off" "$STATUS" \
    && ! output_grep_i "prompt refinement is on" "$STATUS"; then
    test_passed "refine - off by default"
else
    test_failed "refine - not off by default" "Output: $STATUS"
fi

if output_grep_i "prompt refinement disabled" "$OUTPUT" \
    && output_grep_i "prompt refinement enabled" "$OUTPUT"; then
    test_passed "refine - /refine off|on both confirm"
else
    test_failed "refine - toggle did not confirm" "Output: $OUTPUT"
fi

# A bare /refine after each toggle must report that state.
if output_grep_i "prompt refinement is on" "$OUTPUT" \
    && output_grep_i "prompt refinement is off" "$OUTPUT"; then
    test_passed "refine - /refine on and off take effect for the session"
else
    test_failed "refine - toggle not reflected by status" "Output: $OUTPUT"
fi

if output_grep_i "Usage: /refine" "$OUTPUT"; then
    test_passed "refine - bad argument shows usage"
else
    test_failed "refine - bad argument did not show usage" "Output: $OUTPUT"
fi
