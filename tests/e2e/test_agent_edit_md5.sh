set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop edit workflow harness (md5-verified edits)..."

EDIT_HOME=$(mktemp -d)
trap 'rm -rf "$EDIT_HOME"' EXIT

OUTPUT=$(printf '/harness\n/quit\n' \
    | HOME="$EDIT_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true)

if output_grep_fixed "edit-workflow" "$OUTPUT" \
    && output_grep_fixed "preamble-section, order 135 enabled" "$OUTPUT"; then
    test_passed "edit md5 - edit-workflow preamble section is registered and enabled"
else
    test_failed "edit md5 - edit-workflow section missing" "Output: $OUTPUT"
fi

if output_grep_fixed "nudge-edit-unchanged" "$OUTPUT" \
    && output_grep_fixed "standalone, max 4x enabled" "$OUTPUT"; then
    test_passed "edit md5 - nudge-edit-unchanged is registered, bounded at 4"
else
    test_failed "edit md5 - nudge-edit-unchanged missing" "Output: $OUTPUT"
fi
