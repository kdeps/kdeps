set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop briefing command (/instruct)..."

INSTRUCT_HOME=$(mktemp -d)
trap 'rm -rf "$INSTRUCT_HOME"' EXIT

OUTPUT=$(printf '/help\n/instruct list\n/instruct tools\n/instruct bogus\n/instruct\n/quit\n' \
    | HOME="$INSTRUCT_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true)

if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "instruct - agent loop starts"
else
    test_failed "instruct - agent loop did not start" "Output: $OUTPUT"
fi

if output_grep_fixed "/instruct" "$OUTPUT"; then
    test_passed "instruct - /help lists /instruct"
else
    test_failed "instruct - /help missing /instruct" "Output: $OUTPUT"
fi

if output_grep_fixed "Briefing topics:" "$OUTPUT" && output_grep_fixed "overview" "$OUTPUT"; then
    test_passed "instruct - /instruct list names the topics"
else
    test_failed "instruct - /instruct list did not list topics" "Output: $OUTPUT"
fi

if output_grep_fixed "How to call a tool" "$OUTPUT" \
    && output_grep_fixed '<invoke name=' "$OUTPUT"; then
    test_passed "instruct - single topic prints the how-to-call briefing"
else
    test_failed "instruct - single topic briefing missing" "Output: $OUTPUT"
fi

if output_grep_fixed "Briefing added to the model's context (tools)" "$OUTPUT"; then
    test_passed "instruct - single topic reports it was added to context"
else
    test_failed "instruct - single topic not added to context" "Output: $OUTPUT"
fi

if output_grep_i 'Unknown topic "bogus"' "$OUTPUT"; then
    test_passed "instruct - unknown topic is rejected"
else
    test_failed "instruct - unknown topic not rejected" "Output: $OUTPUT"
fi

if output_grep_fixed "Briefing added to the model's context (all topics)" "$OUTPUT"; then
    test_passed "instruct - no-arg briefs on all topics"
else
    test_failed "instruct - no-arg did not brief all topics" "Output: $OUTPUT"
fi

if output_grep_fixed "/instruct!" "$OUTPUT"; then
    test_passed "instruct - /help lists /instruct!"
else
    test_failed "instruct - /help missing /instruct!" "Output: $OUTPUT"
fi
