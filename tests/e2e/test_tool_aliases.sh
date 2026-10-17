# shellcheck shell=bash
# E2E: a model that calls a tool by another name (bash for bash_exec, cat
# for read_file, grep for search_local) gets the tool, with its argument
# names mapped too, instead of "tool not found". Against the shared mock
# LLM, whose request log shows the tool results sent back.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing tool aliases (mock LLM)..."

if [ -z "${E2E_MOCK_LLM_LOG:-}" ] || [ ! -f "${E2E_MOCK_LLM_LOG:-}" ]; then
    test_skipped "tool aliases - the shared mock LLM is not running (python3 missing or KDEPS_E2E_REAL_LLM set)"
    return 0 2>/dev/null || exit 0
fi

TA_DIR=$(cd "$(mktemp -d)" && pwd -P)
trap '[ -n "${TA_KEEP:-}" ] || rm -rf "$TA_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
mkdir -p "$TA_DIR/home/.kdeps" "$TA_DIR/work"
printf 'knowledge:\n  enabled: false\n' > "$TA_DIR/home/.kdeps/config.yaml"
printf 'the alias-file-marker line\n' > "$TA_DIR/work/notes.txt"

# ta_call starts a session in which the mock calls $1 with JSON args $2, and
# prints what the session sent the mock afterwards (the tool's result).
ta_call() {
    local before
    before=$(wc -c < "$E2E_MOCK_LLM_LOG")
    (cd "$TA_DIR/work" && printf 'CALL %s %s\n/quit\n' "$1" "$2" | env HOME="$TA_DIR/home" \
        USERPROFILE="$(native "$TA_DIR/home")" KDEPS_CONFIG_PATH="$TA_DIR/home/.kdeps/config.yaml" \
        timeout 180 "$KDEPS_BIN" --new --backend openai --model fake >/dev/null 2>&1)
    tail -c +"$((before + 1))" "$E2E_MOCK_LLM_LOG"
}

# The command's output (42 then alias) is not in its text, so only a real run sends it back.
SENT=$(ta_call bash '{"cmd":"echo $((6*7))alias"}')
if grep -q "42alias" <<<"$SENT" && ! grep -qF 'not found\"' <<<"$SENT"; then
    test_passed "tool aliases - bash runs bash_exec (cmd mapped to command)"
else
    test_failed "tool aliases - bash" "$(tail -c 600 <<<"$SENT")"
fi

SENT=$(ta_call cat '{"path":"'"$(native "$TA_DIR/work/notes.txt" | sed 's/\\/\\\\\\\\/g')"'"}')
if grep -q "alias-file-marker" <<<"$SENT" && ! grep -qF 'not found\"' <<<"$SENT"; then
    test_passed "tool aliases - cat runs read_file (path mapped to file_path)"
else
    test_failed "tool aliases - cat" "$(tail -c 600 <<<"$SENT")"
fi

SENT=$(ta_call grep '{"pattern":"alias-file-marker","path":"'"$(native "$TA_DIR/work" | sed 's/\\/\\\\\\\\/g')"'"}')
# notes.txt is in no prompt: only search_local's result names it.
if grep -q "notes.txt" <<<"$SENT" && ! grep -qF 'not found\"' <<<"$SENT"; then
    test_passed "tool aliases - grep runs search_local (pattern mapped to query)"
else
    test_failed "tool aliases - grep" "$(tail -c 600 <<<"$SENT")"
fi

SENT=$(ta_call no_such_tool_name '{"x":"y"}')
# The tool result says so: {"error":"tool \"no_such_tool_name\" not found"}, JSON-escaped in the log.
if grep -qF 'no_such_tool_name\\\" not found' <<<"$SENT"; then
    test_passed "tool aliases - a name that is no tool and no alias is still reported as not found"
else
    test_failed "tool aliases - unknown name" "$(tail -c 600 <<<"$SENT")"
fi

[ -n "${TA_KEEP:-}" ] || rm -rf "$TA_DIR"
trap - EXIT

echo ""
echo "Tool aliases E2E: done"
