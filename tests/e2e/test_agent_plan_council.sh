# shellcheck shell=bash
# E2E: the planning council plans an agent task after refine - three seats
# draft and revise, append job orders, rank them - and the turn gets the sorted
# plan plus a progress file. /plan off skips it. Against the shared mock LLM.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"


if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "plan council - the shared mock LLM is not running (python3 missing or KDEPS_E2E_REAL_LLM set)"
    return 0 2>/dev/null || exit 0
fi

native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
PC_DIR=$(cd "$(mktemp -d)" && pwd -P)
trap 'rm -rf "$PC_DIR"' EXIT
mkdir -p "$PC_DIR/home/.kdeps" "$PC_DIR/work" "$PC_DIR/data"
printf 'knowledge:\n  enabled: false\n' > "$PC_DIR/home/.kdeps/config.yaml"

echo "Testing the agent planning council..."

run_agent() {
    (cd "$PC_DIR/work" && printf '%s\n' "$@" '/quit' \
        | env HOME="$PC_DIR/home" USERPROFILE="$(native "$PC_DIR/home")" KDEPS_DATA_DIR="$PC_DIR/data" \
            KDEPS_CONFIG_PATH="$PC_DIR/home/.kdeps/config.yaml" KDEPS_TUI=0 NO_COLOR=1 \
            timeout 180 "$KDEPS_BIN" --new --backend openai --model fake 2>&1) || true
}

before=$(wc -c < "$E2E_MOCK_LLM_LOG")
OUT=$(run_agent 'Please review the build scripts and summarize what each one does for a newcomer')
SENT=$(tail -c +"$((before + 1))" "$E2E_MOCK_LLM_LOG")

if [[ "$OUT" == *"[plan] council of 3"* ]] && [[ "$OUT" == *"job orders ->"* ]]; then
    test_passed "plan council - a council of three plans the task before the turn"
else
    test_failed "plan council - no council ran" "$(tail -15 <<<"$OUT")"
fi

# The turn's request carries the sorted plan as a system message.
if python3 -c '
import json, sys
for line in sys.stdin:
    try:
        d = json.loads(line)
    except ValueError:
        continue
    b = d.get("body", d)
    if isinstance(b, str):
        b = json.loads(b)
    sys_text = " ".join(json.dumps(m) for m in (b.get("messages") or []) if m.get("role") == "system")
    if "A planning council prepared this plan" in sys_text and "COUNCILJOB" in sys_text and "plan_update" in sys_text:
        sys.exit(0)
sys.exit(1)
' <<<"$SENT"; then
    test_passed "plan council - the turn receives the plan and the plan_update instruction"
else
    test_failed "plan council - the plan did not reach the turn" ""
fi

PLAN=$(find "$PC_DIR/data" -path '*plans*' -name '*.md' | head -1)
if [ -n "$PLAN" ] && grep -q 'COUNCILJOB' "$PLAN" && grep -q '## Progress' "$PLAN"; then
    test_passed "plan council - the progress file lists the job orders"
else
    test_failed "plan council - no progress file" "plan=$PLAN"
fi

# Three seats each added one order in each of two rounds (duplicates are
# skipped), and prioritizing reversed the board: the last order comes first.
if [ -n "$PLAN" ] && [ "$(grep -c 'COUNCILJOB' "$PLAN")" -eq 3 ] && head -8 "$PLAN" | grep -q '1\. \[ \] #3 '; then
    test_passed "plan council - the append-only board is sorted by the council's ranking"
else
    test_failed "plan council - unexpected board" "$(cat "$PLAN" 2>/dev/null | head -12)"
fi

before=$(wc -c < "$E2E_MOCK_LLM_LOG")
OUT=$(run_agent '/plan off' 'Please review the build scripts and summarize what each one does for a newcomer')
SENT=$(tail -c +"$((before + 1))" "$E2E_MOCK_LLM_LOG")
if [[ "$OUT" == *"planning council disabled"* ]] && [[ "$OUT" != *"[plan] council"* ]] && [[ "$SENT" != *"planning council for an AI agent"* ]]; then
    test_passed "plan council - /plan off skips the council"
else
    test_failed "plan council - /plan off did not skip it" "$(tail -10 <<<"$OUT")"
fi

echo ""
echo "plan council E2E: done"
