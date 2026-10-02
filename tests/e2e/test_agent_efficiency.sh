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

# One fresh launch (same HOME) covers persistence, preset and reset; each REPL
# launch is slow on Windows runners, so keep the launch count low. The Nth
# "Efficiency enforcement" status block is the state after the Nth /efficiency.
OUTPUT=$(run_eff "$(printf '/efficiency\n/efficiency preset frugal\n/efficiency\n/efficiency reads 9\n/efficiency reset\n/efficiency')")

status_block() {
    awk -v n="$1" '/Efficiency enforcement/ {c++} c == n' <<< "$OUTPUT"
}

# Persisted from the first launch: off, verbose on, reads 3.
B1=$(status_block 1)
if output_grep_i "state *off" "$B1" && output_grep_i "verbose *on" "$B1" \
    && output_grep_i "reads *3" "$B1"; then
    test_passed "efficiency - changes persist across launches"
else
    test_failed "efficiency - changes did not persist" "Output: $B1"
fi

# A harness preset overwrites everything: on, verbose off, reads = frugal value.
B2=$(status_block 2)
if output_grep_i "state *on" "$B2" && output_grep_i "verbose *off" "$B2" \
    && output_grep_i "reads *4" "$B2"; then
    test_passed "efficiency - preset overwrites all values"
else
    test_failed "efficiency - preset did not overwrite all values" "Output: $B2"
fi

B3=$(status_block 3)
if output_grep_i "reset to defaults" "$OUTPUT" && output_grep_i "reads *6" "$B3"; then
    test_passed "efficiency - reset restores defaults"
else
    test_failed "efficiency - reset did not restore defaults" "Output: $B3"
fi
