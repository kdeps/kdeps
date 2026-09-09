set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop prompt refinement (/refine)..."

REFINE_HOME=$(mktemp -d)
trap 'rm -rf "$REFINE_HOME"' EXIT

SETTINGS="$REFINE_HOME/.kdeps/agent-loop-settings.yaml"

OUTPUT=$(printf '/refine\n/refine off\n/refine on\n/refine bogus\n/quit\n' \
    | HOME="$REFINE_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true)

if output_grep_fixed "/refine" "$OUTPUT"; then
    test_passed "refine - /help or command output mentions /refine"
else
    test_failed "refine - no /refine reference" "Output: $OUTPUT"
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

if output_grep_i "Usage: /refine" "$OUTPUT"; then
    test_passed "refine - bad argument shows usage"
else
    test_failed "refine - bad argument did not show usage" "Output: $OUTPUT"
fi

# The last toggle in the script was "/refine on", so the persisted file must
# NOT record refine_off.
OUTPUT2=$(printf '/refine off\n/quit\n' \
    | HOME="$REFINE_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true)
if [ -f "$SETTINGS" ] && grep -q "refine_off: true" "$SETTINGS" \
    && grep -q "refine_configured: true" "$SETTINGS"; then
    test_passed "refine - /refine off persists to agent-loop-settings.yaml"
else
    test_failed "refine - choice not persisted" "settings: $(cat "$SETTINGS" 2>&1)"
fi

OUTPUT3=$(printf '/refine\n/quit\n' \
    | HOME="$REFINE_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true)
if output_grep_i "prompt refinement is off" "$OUTPUT3"; then
    test_passed "refine - persisted /refine off is restored next session"
else
    test_failed "refine - persisted off not restored" "Output: $OUTPUT3"
fi
