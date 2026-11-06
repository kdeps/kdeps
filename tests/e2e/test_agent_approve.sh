# shellcheck shell=bash
# E2E: an approval token is granted only by the user. The model can request
# one but is never offered approval_grant; /approve lists and grants it.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing /approve (mock LLM)..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "approve - the shared mock LLM is not running (python3 missing or KDEPS_E2E_REAL_LLM set)"
    return 0 2>/dev/null || exit 0
fi

AP_DIR=$(cd "$(mktemp -d)" && pwd -P)
trap 'rm -rf "$AP_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
mkdir -p "$AP_DIR/home/.kdeps" "$AP_DIR/work"
printf 'knowledge:\n  enabled: false\n' > "$AP_DIR/home/.kdeps/config.yaml"

OUT=$(cd "$AP_DIR/work" && printf '%s\n' \
    'CALL approval_request {"tool_name":"bash_exec","action":"e2e-approve-marker"}' \
    'CALL approval_grant {"token_id":"apt-1"}' \
    '/approve' '/approve apt-1' '/quit' \
    | env HOME="$AP_DIR/home" USERPROFILE="$(native "$AP_DIR/home")" \
        KDEPS_CONFIG_PATH="$AP_DIR/home/.kdeps/config.yaml" KDEPS_TUI=0 NO_COLOR=1 \
        timeout 180 "$KDEPS_BIN" --new --backend openai --model fake 2>&1) || true

if [[ "$OUT" == *"e2e-approve-marker"* ]] && [[ "$OUT" == *"pending"* ]]; then
    test_passed "approve - /approve lists the token the agent requested"
else
    test_failed "approve - /approve did not list the pending token" "$(tail -20 <<<"$OUT")"
fi
if [[ "$OUT" == *"approval_grant\" not found"* ]] || [[ "$OUT" == *"approval_grant not found"* ]] || [[ "$OUT" == *'tool "approval_grant"'* ]]; then
    test_passed "approve - the model cannot grant its own token"
else
    test_failed "approve - the model's approval_grant call was not refused" "$(tail -20 <<<"$OUT")"
fi
if [[ "$OUT" == *"Granted approval token apt-1"* ]]; then
    test_passed "approve - /approve <token_id> grants it"
else
    test_failed "approve - /approve apt-1 did not grant" "$(tail -20 <<<"$OUT")"
fi

echo ""
echo "approve E2E: done"
