# shellcheck shell=bash
# E2E: ask_question ends the turn with the question and its numbered options;
# the user's next line is the answer. Against the shared mock LLM.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing ask_question (mock LLM)..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "ask_question - the shared mock LLM is not running (python3 missing or KDEPS_E2E_REAL_LLM set)"
    return 0 2>/dev/null || exit 0
fi

AQ_DIR=$(cd "$(mktemp -d)" && pwd -P)
trap 'rm -rf "$AQ_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
mkdir -p "$AQ_DIR/home/.kdeps" "$AQ_DIR/work"
printf 'knowledge:\n  enabled: false\n' > "$AQ_DIR/home/.kdeps/config.yaml"

before=$(wc -c < "$E2E_MOCK_LLM_LOG")
OUT=$(cd "$AQ_DIR/work" && printf '%s\n' \
    'CALL ask_question {"question":"Pick a review mode","options":"Interactive\nConsolidated"}' \
    'Consolidated' '/quit' \
    | env HOME="$AQ_DIR/home" USERPROFILE="$(native "$AQ_DIR/home")" \
        KDEPS_CONFIG_PATH="$AQ_DIR/home/.kdeps/config.yaml" KDEPS_TUI=0 NO_COLOR=1 \
        timeout 180 "$KDEPS_BIN" --new --backend openai --model fake 2>&1) || true
SENT=$(tail -c +"$((before + 1))" "$E2E_MOCK_LLM_LOG")

if [[ "$OUT" == *"Pick a review mode"* ]] && [[ "$OUT" == *"1. Interactive"* ]] && [[ "$OUT" == *"2. Consolidated"* ]]; then
    test_passed "ask_question - the turn ends with the question and numbered options"
else
    test_failed "ask_question - the question was not shown" "$(tail -20 <<<"$OUT")"
fi
# The answer is the next user message, and the question is in the history
# the model sees with it.
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
    msgs = b.get("messages") or []
    if msgs and "Consolidated" in json.dumps(msgs[-1]) and "Pick a review mode" in json.dumps(msgs):
        sys.exit(0)
sys.exit(1)
' <<<"$SENT"; then
    test_passed "ask_question - the next line answers it, with the question in the history"
else
    test_failed "ask_question - the answer did not reach the model with the question" ""
fi

echo ""
echo "ask_question E2E: done"
