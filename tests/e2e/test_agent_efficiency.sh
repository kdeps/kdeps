set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop efficiency enforcement (/efficiency)..."

EFF_HOME=$(mktemp -d)
trap 'rm -rf "$EFF_HOME"' EXIT

run_eff() {
    printf '%s\n/quit\n' "$1" | HOME="$EFF_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true
}

OUTPUT=$(run_eff "$(printf '/help\n/efficiency\n/efficiency reads 3\n/efficiency off\n/efficiency verbose on\n/efficiency bogus 1\n/efficiency reads nope')")

if output_grep_fixed "/efficiency" "$OUTPUT"; then
    test_passed "efficiency - /help lists /efficiency"
else
    test_failed "efficiency - /help missing /efficiency" "Output: $OUTPUT"
fi

if output_grep_i "Efficiency enforcement" "$OUTPUT" && output_grep_i "state *on" "$OUTPUT"; then
    test_passed "efficiency - on by default with status"
else
    test_failed "efficiency - status missing or not on by default" "Output: $OUTPUT"
fi

if output_grep_fixed "efficiency reads = 3 (saved)" "$OUTPUT"; then
    test_passed "efficiency - /efficiency reads 3 confirms and saves"
else
    test_failed "efficiency - setting a value did not confirm" "Output: $OUTPUT"
fi

if output_grep_fixed "efficiency enforcement off (saved)" "$OUTPUT" \
    && output_grep_fixed "efficiency verbose on (saved)" "$OUTPUT"; then
    test_passed "efficiency - off and verbose on confirm"
else
    test_failed "efficiency - toggles did not confirm" "Output: $OUTPUT"
fi

if output_grep_i "unknown efficiency setting" "$OUTPUT" && output_grep_i "Usage: /efficiency" "$OUTPUT"; then
    test_passed "efficiency - bad arguments show usage"
else
    test_failed "efficiency - bad arguments did not show usage" "Output: $OUTPUT"
fi

# A fresh launch with the same HOME must show every earlier change persisted.
OUTPUT=$(run_eff '/efficiency')
if output_grep_i "state *off" "$OUTPUT" && output_grep_i "verbose *on" "$OUTPUT" \
    && output_grep_i "reads *3" "$OUTPUT"; then
    test_passed "efficiency - changes persist across launches"
else
    test_failed "efficiency - changes did not persist" "Output: $OUTPUT"
fi

# A harness preset overwrites everything: on, verbose off, reads = frugal value.
OUTPUT=$(run_eff "$(printf '/efficiency preset frugal\n/efficiency')")
if output_grep_i "state *on" "$OUTPUT" && output_grep_i "verbose *off" "$OUTPUT" \
    && output_grep_i "reads *4" "$OUTPUT"; then
    test_passed "efficiency - preset overwrites all values"
else
    test_failed "efficiency - preset did not overwrite all values" "Output: $OUTPUT"
fi

OUTPUT=$(run_eff "$(printf '/efficiency reads 9\n/efficiency reset\n/efficiency')")
if output_grep_i "reset to defaults" "$OUTPUT" && output_grep_i "reads *6" "$OUTPUT"; then
    test_passed "efficiency - reset restores defaults"
else
    test_failed "efficiency - reset did not restore defaults" "Output: $OUTPUT"
fi
