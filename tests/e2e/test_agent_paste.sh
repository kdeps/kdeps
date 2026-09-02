# shellcheck shell=bash
# E2E: size-aware paste in the agent loop.
#
# A large bracketed paste is staged to a temp file rather than echoed line by
# line, and that temp dir is removed when the REPL exits. Small pastes and the
# exact edit-line rendering are covered by the Go tests (pkg/agent/paste*_test.go).
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

# The 300 raw lines must not all be echoed back - the paste was staged.
PASTED_ECHOED=$(printf '%s\n' "$OUTPUT" | grep -c '^pasted line ' || true)
if [ "$PASTED_ECHOED" -lt 50 ]; then
    test_passed "paste - large paste is not dumped line by line ($PASTED_ECHOED echoed)"
else
    test_failed "paste - large paste flooded the terminal" "$PASTED_ECHOED lines echoed"
fi

# The staging temp dir is cleaned up on exit.
if find "$PASTE_TMP" -maxdepth 1 -name 'kdeps-paste-*' -print -quit | grep -q .; then
    test_failed "paste - temp dir left behind" "$(find "$PASTE_TMP" -maxdepth 1 -name 'kdeps-paste-*')"
else
    test_passed "paste - staging temp dir removed on exit"
fi

echo ""
echo "agent-loop paste E2E: done"
