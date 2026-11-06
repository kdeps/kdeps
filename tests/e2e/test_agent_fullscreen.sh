# shellcheck shell=bash
# E2E: the full-screen agent loop in a real terminal (a pty). Prompts typed
# before a turn ends wait and run next, in order, and /exit leaves the
# alternate screen. Against the shared mock LLM.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the full-screen agent loop (mock LLM, pty)..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "full-screen - the shared mock LLM is not running (python3 missing or KDEPS_E2E_REAL_LLM set)"
    return 0 2>/dev/null || exit 0
fi

FS_DIR=$(cd "$(mktemp -d)" && pwd -P)
trap 'rm -rf "$FS_DIR"' EXIT
mkdir -p "$FS_DIR/home/.kdeps" "$FS_DIR/work"
printf 'knowledge:\n  enabled: false\n' > "$FS_DIR/home/.kdeps/config.yaml"

before=$(wc -c < "$E2E_MOCK_LLM_LOG")
(cd "$FS_DIR/work" && env -u KDEPS_TUI HOME="$FS_DIR/home" TERM=xterm-256color \
    KDEPS_CONFIG_PATH="$FS_DIR/home/.kdeps/config.yaml" \
    python3 "$PROJECT_ROOT/tests/e2e/helpers/pty_run.py" "$FS_DIR/screen.out" 20 \
        "fsfirst prompt" "fssecond prompt" "/exit" \
        -- "$KDEPS_BIN" --new --backend openai --model fake)
status=$?
if [ "$status" -eq 3 ]; then
    test_skipped "full-screen - no pty on this platform"
    return 0 2>/dev/null || exit 0
fi
SENT=$(tail -c +"$((before + 1))" "$E2E_MOCK_LLM_LOG")

if [ "$status" -eq 0 ]; then
    test_passed "full-screen - /exit typed during a turn ends the session (exit 0)"
else
    test_failed "full-screen - exit status $status" "$(tail -c 400 "$FS_DIR/screen.out" 2>/dev/null)"
fi

if LC_ALL=C grep -q $'\x1b\[?1049h' "$FS_DIR/screen.out" && LC_ALL=C grep -q $'\x1b\[?1049l' "$FS_DIR/screen.out"; then
    test_passed "full-screen - enters and leaves the alternate screen"
else
    test_failed "full-screen - no alternate screen in a terminal" "$(tail -c 400 "$FS_DIR/screen.out" 2>/dev/null)"
fi

# One request per line in the mock log. The second prompt's request must carry
# the first turn, which only happens when it ran after that turn ended.
FIRST=$(grep -n '"content":"fsfirst prompt"' <<<"$SENT" | head -1 | cut -d: -f1)
SECOND=$(grep -n '"content":"fssecond prompt"' <<<"$SENT" | head -1 | cut -d: -f1)
if [ -n "$FIRST" ] && [ -n "$SECOND" ] && [ "$FIRST" -lt "$SECOND" ]; then
    test_passed "full-screen - both prompts reach the model, in order"
else
    test_failed "full-screen - prompts missing or out of order" "first=$FIRST second=$SECOND"
fi
if [ -n "$SECOND" ] && sed -n "${SECOND}p" <<<"$SENT" | grep -q "fsfirst prompt"; then
    test_passed "full-screen - the second prompt waits for the first turn"
else
    test_failed "full-screen - the second prompt ran beside the first" "$(sed -n "${SECOND:-1}p" <<<"$SENT" | tail -c 400)"
fi

echo ""
echo "full-screen agent loop E2E: done"
