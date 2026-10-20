# shellcheck shell=bash
# E2E: a model that writes DeepSeek's DSML-wrapped <invoke> markup as text
# (deepseek-v4-flash does this on its OpenAI-compatible backend) gets its tool
# call parsed and executed instead of dropped and repeated. Against the shared
# mock LLM, whose request log shows the tool result sent back.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing DSML invoke recovery (mock LLM)..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "DSML invoke - the shared mock LLM is not running (python3 missing or KDEPS_E2E_REAL_LLM set)"
    return 0 2>/dev/null || exit 0
fi

DS_DIR=$(cd "$(mktemp -d)" && pwd -P)
trap '[ -n "${DS_KEEP:-}" ] || rm -rf "$DS_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
mkdir -p "$DS_DIR/home/.kdeps" "$DS_DIR/work"
printf 'knowledge:\n  enabled: false\n' > "$DS_DIR/home/.kdeps/config.yaml"

before=$(wc -c < "$E2E_MOCK_LLM_LOG")
OUT=$( (cd "$DS_DIR/work" && printf 'CALLDSML bash_exec {"command":"echo $((6*7))dsmlout"}\n/quit\n' \
    | env HOME="$DS_DIR/home" USERPROFILE="$(native "$DS_DIR/home")" \
        KDEPS_CONFIG_PATH="$DS_DIR/home/.kdeps/config.yaml" \
        timeout 180 "$KDEPS_BIN" --new --backend openai --model fake 2>&1) || true)
SENT=$(tail -c +"$((before + 1))" "$E2E_MOCK_LLM_LOG")

# The command's output is not in the model's text, so only a real run sends it
# back in the next request's tool message.
if grep -q "42dsmlout" <<<"$SENT"; then
    test_passed "DSML invoke - the DSML-wrapped call runs for real"
else
    test_failed "DSML invoke - the DSML-wrapped call was not executed" "$(tail -c 600 <<<"$SENT")"
fi

if grep -q "TOOL DONE" <<<"$OUT"; then
    test_passed "DSML invoke - the tool result reaches the model"
else
    test_failed "DSML invoke - no follow-up answer" "out=$OUT"
fi

# The control-token bars are never part of a command or an answer: finding one
# means the raw markup was shown instead of parsed.
if grep -q "｜" <<<"$OUT"; then
    test_failed "DSML invoke - the markup leaked into the answer" "out=$OUT"
else
    test_passed "DSML invoke - the markup is stripped from the answer"
fi

[ -n "${DS_KEEP:-}" ] || rm -rf "$DS_DIR"
trap - EXIT

echo ""
echo "DSML invoke E2E: done"
