# shellcheck shell=bash
# E2E: size-aware paste in the agent loop.
#
# A large bracketed paste is saved to memory and shown as "@paste:<id>" rather
# than echoed line by line, and REPL history keeps that reference instead of
# the paste placeholder rune. Small pastes and the exact edit-line rendering
# are covered by the Go tests (pkg/agent/paste*_test.go).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing agent-loop paste handling..."

PASTE_HOME=$(mktemp -d)
PASTE_TMP=$(mktemp -d)
trap 'rm -rf "$PASTE_HOME" "$PASTE_TMP"' EXIT

# 300-line body wrapped in bracketed-paste markers, then a newline to submit and
# /quit. ESC = \033.
BIG=$(seq 1 300 | sed 's/^/pasted line /')
INPUT=$(printf '\033[200~%s\033[201~\n/quit\n' "$BIG")

OUTPUT=$(printf '%s' "$INPUT" \
    | HOME="$PASTE_HOME" TMPDIR="$PASTE_TMP" timeout 60 "$KDEPS_BIN" 2>&1 || true)

PASTED_ECHOED=$(printf '%s\n' "$OUTPUT" | grep -c '^pasted line ' || true)
if [ "$PASTED_ECHOED" -lt 50 ]; then
    test_passed "paste - large paste is not dumped line by line ($PASTED_ECHOED echoed)"
else
    test_failed "paste - large paste flooded the terminal" "$PASTED_ECHOED lines echoed"
fi

# History keeps the @paste:<id> memory reference, never the placeholder rune
# (U+FFFC, bytes EF BF BC).
HISTORY="$PASTE_HOME/.kdeps/repl_history"
if [ ! -f "$HISTORY" ]; then
    test_skipped "paste - history (REPL did not reach the prompt)"
elif LC_ALL=C grep -q $'\xef\xbf\xbc' "$HISTORY"; then
    test_failed "paste - history holds the placeholder rune" "$(cat "$HISTORY")"
elif grep -q '@paste:' "$HISTORY"; then
    test_passed "paste - history keeps the @paste:<id> reference"
else
    test_failed "paste - history has no @paste:<id> reference" "$(cat "$HISTORY")"
fi

# No temp files are staged for a paste.
if find "$PASTE_TMP" -maxdepth 1 -name 'kdeps-paste-*' -print -quit | grep -q .; then
    test_failed "paste - temp files staged" "$(find "$PASTE_TMP" -maxdepth 1 -name 'kdeps-paste-*')"
else
    test_passed "paste - no temp files staged"
fi

echo ""
echo "agent-loop paste E2E: done"
