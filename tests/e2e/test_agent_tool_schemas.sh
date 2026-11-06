# shellcheck shell=bash
# E2E: the agent loop sends only the core tool definitions; load_tools names
# the rest, and a tool it loads is defined on the next request. Against the
# shared mock LLM, whose request log holds each request's tools.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing on-demand tool definitions (mock LLM)..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "tool schemas - the shared mock LLM is not running (python3 missing or KDEPS_E2E_REAL_LLM set)"
    return 0 2>/dev/null || exit 0
fi

TS_DIR=$(cd "$(mktemp -d)" && pwd -P)
trap 'rm -rf "$TS_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
mkdir -p "$TS_DIR/home/.kdeps" "$TS_DIR/work"
printf 'knowledge:\n  enabled: false\n' > "$TS_DIR/home/.kdeps/config.yaml"

before=$(wc -c < "$E2E_MOCK_LLM_LOG")
(cd "$TS_DIR/work" && printf 'CALL load_tools {"names":"git_ops"}\n/quit\n' \
    | env HOME="$TS_DIR/home" USERPROFILE="$(native "$TS_DIR/home")" \
        KDEPS_CONFIG_PATH="$TS_DIR/home/.kdeps/config.yaml" KDEPS_TUI=0 \
        timeout 180 "$KDEPS_BIN" --new --backend openai --model fake >/dev/null 2>&1) || true
SENT=$(tail -c +"$((before + 1))" "$E2E_MOCK_LLM_LOG")

# One request per line: report each chat request's tool names.
TOOLS=$(python3 -c '
import json, sys
for line in sys.stdin:
    try:
        d = json.loads(line)
    except ValueError:
        continue
    b = d.get("body", d)
    if isinstance(b, str):
        b = json.loads(b)
    names = [t["function"]["name"] for t in b.get("tools") or []]
    if names:
        desc = next((t["function"]["description"] for t in b["tools"] if t["function"]["name"] == "load_tools"), "")
        print(len(names), "git_ops" in names, "- git_ops:" in desc)
' <<<"$SENT")
FIRST=$(sed -n 1p <<<"$TOOLS")
LAST=$(tail -1 <<<"$TOOLS")

if [ -n "$FIRST" ] && [ "${FIRST%% *}" -lt 30 ]; then
    test_passed "tool schemas - the first request sends the core definitions only (${FIRST%% *})"
else
    test_failed "tool schemas - the first request sent every definition" "$FIRST"
fi
if [[ "$FIRST" == *"False True" ]]; then
    test_passed "tool schemas - load_tools names a deferred tool (git_ops)"
else
    test_failed "tool schemas - git_ops is not listed in load_tools" "$FIRST"
fi
if [[ "$LAST" == *" True False" ]]; then
    test_passed "tool schemas - a loaded tool is defined on the next request"
else
    test_failed "tool schemas - git_ops was not defined after load_tools" "$TOOLS"
fi

echo ""
echo "on-demand tool definitions E2E: done"
