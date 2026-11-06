# shellcheck shell=bash
# E2E: the subagent tool runs a task in a fresh loop that sees only the task
# and read-only tools, and refuses a tool that writes. Against the shared mock
# LLM, whose request log holds each request.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing the subagent tool (mock LLM)..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "subagent - the shared mock LLM is not running (python3 missing or KDEPS_E2E_REAL_LLM set)"
    return 0 2>/dev/null || exit 0
fi

SA_DIR=$(cd "$(mktemp -d)" && pwd -P)
trap 'rm -rf "$SA_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
mkdir -p "$SA_DIR/home/.kdeps" "$SA_DIR/work"
printf 'knowledge:\n  enabled: false\n' > "$SA_DIR/home/.kdeps/config.yaml"

# run_turn sends one line to a fresh session and prints the mock requests it made.
run_turn() {
    local before
    before=$(wc -c < "$E2E_MOCK_LLM_LOG")
    (cd "$SA_DIR/work" && printf '%s\n/quit\n' "$1" \
        | env HOME="$SA_DIR/home" USERPROFILE="$(native "$SA_DIR/home")" \
            KDEPS_CONFIG_PATH="$SA_DIR/home/.kdeps/config.yaml" KDEPS_TUI=0 \
            timeout 180 "$KDEPS_BIN" --new --backend openai --model fake >/dev/null 2>&1) || true
    tail -c +"$((before + 1))" "$E2E_MOCK_LLM_LOG"
}

# subagent_requests prints, for each request that is not offered bash_exec
# (only the main loop is), its tool names and whether its messages carry
# the given marker and the main turn's CALL line.
subagent_requests() {
    python3 -c '
import json, sys
marker = sys.argv[1]
for line in sys.stdin:
    try:
        d = json.loads(line)
    except ValueError:
        continue
    b = d.get("body", d)
    if isinstance(b, str):
        b = json.loads(b)
    names = [t["function"]["name"] for t in b.get("tools") or []]
    if not names or "bash_exec" in names or "judge_verdict" in names:
        continue
    text = json.dumps(b.get("messages", []))
    print(",".join(sorted(names)), marker in text, "CALL subagent" in text)
' "$1"
}

SENT=$(run_turn 'CALL subagent {"task":"Check README.md for gaps.","tools":"read_file"}')
SUB=$(subagent_requests "Check README.md for gaps." <<<"$SENT")
if [ -n "$SUB" ] && [[ "$SUB" == *"read_file"* ]] && [[ "$SUB" != *"write_file"* ]] && [[ "$SUB" != *"subagent,"* ]]; then
    test_passed "subagent - the subagent gets read-only tools and no subagent tool"
else
    test_failed "subagent - the subagent request was missing or offered the wrong tools" "$SUB"
fi
if [[ "$SUB" == *" True False"* ]]; then
    test_passed "subagent - the subagent sees the task, not the conversation"
else
    test_failed "subagent - the subagent did not get just the task" "$SUB"
fi

SENT=$(run_turn 'CALL subagent {"task":"Rewrite README.md.","tools":"write_file"}')
SUB=$(subagent_requests "Rewrite README.md." <<<"$SENT")
if [ -z "$SUB" ]; then
    test_passed "subagent - a write tool is refused before any subagent runs"
else
    test_failed "subagent - a subagent ran with a write tool" "$SUB"
fi

OUT=$(cd "$SA_DIR/work" && printf '/subagent\n/quit\n' \
    | env HOME="$SA_DIR/home" USERPROFILE="$(native "$SA_DIR/home")" \
        KDEPS_CONFIG_PATH="$SA_DIR/home/.kdeps/config.yaml" KDEPS_TUI=0 NO_COLOR=1 \
        timeout 180 "$KDEPS_BIN" --new --backend openai --model fake 2>&1) || true
if [[ "$OUT" == *"subagent calls use the session model"* ]]; then
    test_passed "subagent - /subagent shows the channel's model"
else
    test_failed "subagent - /subagent did not show the channel" "$(tail -5 <<<"$OUT")"
fi

echo ""
echo "subagent tool E2E: done"
