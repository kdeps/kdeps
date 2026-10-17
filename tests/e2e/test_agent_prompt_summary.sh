set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing harness prompt summaries (/efficiency prompt)..."

PROMPT_HOME=$(mktemp -d)

run_prompt() {
    printf '%s\n/quit\n' "$1" | HOME="$PROMPT_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true
}

OUTPUT=$(run_prompt "$(printf '/efficiency\n/efficiency prompt summary\n/efficiency prompt bogus\n/efficiency prompt full')")

if output_grep_i "prompt *auto" "$OUTPUT"; then
    test_passed "prompt - auto is the default"
else
    test_failed "prompt - status did not show auto" "Output: $OUTPUT"
fi

if output_grep_fixed "efficiency prompt summary (saved)" "$OUTPUT"; then
    test_passed "prompt - summary mode saves"
else
    test_failed "prompt - summary mode did not save" "Output: $OUTPUT"
fi

if output_grep_fixed "efficiency prompt full (saved)" "$OUTPUT"; then
    test_passed "prompt - full mode saves"
else
    test_failed "prompt - full mode did not save" "Output: $OUTPUT"
fi

if output_grep_fixed "usage: /efficiency prompt auto|full|summary" "$OUTPUT"; then
    test_passed "prompt - a bad mode shows usage"
else
    test_failed "prompt - a bad mode did not show usage" "Output: $OUTPUT"
fi

OUTPUT=$(run_prompt "/efficiency")

if output_grep_i "prompt *full" "$OUTPUT"; then
    test_passed "prompt - mode persists across launches"
else
    test_failed "prompt - mode did not persist" "Output: $OUTPUT"
fi

rm -rf "$PROMPT_HOME"
