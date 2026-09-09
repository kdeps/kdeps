set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop session save/list + resume picker flags..."

SESS_HOME=$(mktemp -d)
SESS_PROJ=$(mktemp -d)
trap 'rm -rf "$SESS_HOME" "$SESS_PROJ"' EXIT

# /session save writes a session regardless of whether an LLM turn happened;
# /session list must then show it.
OUTPUT=$(cd "$SESS_PROJ" && printf '/session save probe\n/session list\n/quit\n' \
    | HOME="$SESS_HOME" USERPROFILE="$SESS_HOME" timeout 60 "$KDEPS_BIN" 2>&1 || true)

if output_grep_fixed "kdeps agent" "$OUTPUT"; then
    test_passed "sessions - agent loop starts"
else
    test_failed "sessions - agent loop did not start" "Output: $OUTPUT"
fi

if output_grep_i "session saved" "$OUTPUT"; then
    test_passed "sessions - /session save persists a session"
else
    test_failed "sessions - /session save did not confirm" "Output: $OUTPUT"
fi

if output_grep_i "probe" "$OUTPUT"; then
    test_passed "sessions - /session list shows the saved session"
else
    test_failed "sessions - /session list did not list the session" "Output: $OUTPUT"
fi

# --new must start without hanging on the picker even though a session exists.
OUTPUT2=$(cd "$SESS_PROJ" && printf '/quit\n' \
    | HOME="$SESS_HOME" USERPROFILE="$SESS_HOME" timeout 60 "$KDEPS_BIN" --new 2>&1 || true)
if output_grep_fixed "kdeps agent" "$OUTPUT2"; then
    test_passed "sessions - --new starts without hanging on the picker"
else
    test_failed "sessions - --new did not start" "Output: $OUTPUT2"
fi

if "$KDEPS_BIN" --help 2>&1 | grep -q -- "--new"; then
    test_passed "sessions - --new flag is in --help"
else
    test_failed "sessions - --new missing from --help" ""
fi
