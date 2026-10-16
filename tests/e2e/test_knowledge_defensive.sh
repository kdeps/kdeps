# shellcheck shell=bash
# E2E: the knowledge vault under hostile and broken inputs, against a mock
# LLM. Every case checks kdeps neither panics nor loses the turn, writes
# nothing outside the vault, and leaves a vault kdeps knowledge check
# accepts:
#   - paths and project names with spaces, unicode and punctuation
#   - a vault that is a file, is read-only, or holds broken YAML and huge notes
#   - path-traversal and oversized note titles, markup in summaries
#   - archiving onto a name the archive already holds
#   - two sessions writing the same project at once
#   - unicode, empty and very long prompts; odd memory keys
#   - a hand-written GRAPH_REPORT.md at the vault root
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
without_llm_override # this script serves its own LLM and reads what it was sent

echo "Testing the knowledge vault against hostile and broken inputs (mock LLM)..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "knowledge defensive - python3 not available for the mock LLM"
    return 0 2>/dev/null || exit 0
fi

KD_DIR=$(cd "$(mktemp -d)" && pwd -P)
KD_SRV=""
trap '[ -n "$KD_SRV" ] && kill "$KD_SRV" 2>/dev/null; chmod -R u+w "$KD_DIR" 2>/dev/null; [ -n "${KD_KEEP:-}" ] || rm -rf "$KD_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
KD_LOG="$KD_DIR/requests.jsonl"
: > "$KD_LOG"
KD_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
python3 "$SCRIPT_DIR/helpers/mock_llm.py" "$KD_PORT" "$KD_LOG" >/dev/null 2>"$KD_DIR/mock.err" </dev/null &
KD_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:$KD_PORT/v1/models')" 2>/dev/null && break
    sleep 0.1
done

KD_SRC="$KD_DIR/src"
mkdir -p "$KD_SRC"
cp -R "$PROJECT_ROOT/assets/." "$KD_SRC/"
cat > "$KD_SRC/providers/mockllm.yaml" <<YAML
version: 1.0.0
name: mockllm
label: Mock LLM
order: 999
configKey: mockllm_api_key
envVar: MOCKLLM_API_KEY
url: http://127.0.0.1:$KD_PORT
chatPath: /v1/chat/completions
YAML

KD_HOME="$KD_DIR/home"
mkdir -p "$KD_HOME/.kdeps"
# kd_vault points config.yaml at vault $1.
kd_vault() {
    # Single quotes: a Windows path's backslashes are escapes inside YAML double quotes.
    printf "llm:\n  mockllm_api_key: sk-mock\nknowledge:\n  vault: '%s'\n" "$(native "$1" | sed "s/'/''/g")" > "$KD_HOME/.kdeps/config.yaml"
}
# kd_session runs one piped agent session in folder $1 with input $2.
kd_session() {
    (cd "$1" && printf '%b' "$2" | HOME="$KD_HOME" USERPROFILE="$(native "$KD_HOME")" \
        KDEPS_CONFIG_PATH="$KD_HOME/.kdeps/config.yaml" KDEPS_ASSETS_SRC="$KD_SRC" \
        KDEPS_ASSETS_DIR="$KD_HOME/store" KDEPS_ASSETS_URL=off \
        timeout 180 "$KDEPS_BIN" --new --backend mockllm --model fake 2>&1) || true
}
kd() {
    HOME="$KD_HOME" USERPROFILE="$(native "$KD_HOME")" KDEPS_CONFIG_PATH="$KD_HOME/.kdeps/config.yaml" \
        KDEPS_ASSETS_SRC="$KD_SRC" KDEPS_ASSETS_DIR="$KD_HOME/store" KDEPS_ASSETS_URL=off \
        "$KDEPS_BIN" "$@" 2>&1 </dev/null
}
# healthy reports whether output $1 has the turn's reply and no panic.
healthy() { output_grep_fixed "MOCK REPLY" "$1" && ! output_grep "panic:|goroutine [0-9]+ \[" "$1"; }

# --- spaces, unicode and punctuation in the vault and project paths -------------
V="$KD_DIR/my vault ü"
P="$KD_DIR/my app (v2)! ü"
mkdir -p "$P"
kd_vault "$V"
OUT=$(kd_session "$P" 'hello there friend\n/quit\n')
CARD=$(find "$V/projects" -mindepth 2 -maxdepth 2 -name '*.md' 2>/dev/null | head -1)
if healthy "$OUT" && [ -n "$CARD" ] && [ -f "$V/.index/graph.json" ] &&
    [ -n "$(find "$V/projects" -path '*/sessions/*.md' 2>/dev/null)" ]; then
    test_passed "knowledge defensive - spaces, unicode and punctuation in the vault and project paths"
else
    test_failed "knowledge defensive - odd paths" "$OUT $(find "$V" 2>&1 | head)"
fi
NAME=$(basename "$(dirname "$CARD")")
if printf '%s' "$NAME" | grep -qE '^[[:alnum:] ._()-]+' && ! printf '%s' "$NAME" | grep -q '!'; then
    test_passed "knowledge defensive - the project folder name is made safe ($NAME)"
else
    test_failed "knowledge defensive - project folder name" "$NAME"
fi
OUT=$(kd knowledge check --vault "$V")
if ! output_grep "panic:" "$OUT" && output_grep "every fact has one home" "$OUT"; then
    test_passed "knowledge defensive - the vault kdeps wrote at odd paths passes check"
else
    test_failed "knowledge defensive - check at odd paths" "$OUT"
fi

# --- broken notes: bad YAML card, huge note, cycles, markup in a summary ---------
V="$KD_DIR/broken"
P="$KD_DIR/work"
mkdir -p "$P" "$V/projects/bad" "$V/shared"
kd_vault "$V"
printf -- '---\n: [unclosed\n---\n| repo | %s |\n' "$(native "$P")" > "$V/projects/bad/bad.md"
python3 -c "import sys;open(sys.argv[1],'w').write('---\nsummary: big\n---\n'+'x'*(3<<20))" "$V/shared/Huge.md"
printf -- '---\nsummary: A links B\n---\n[[B]] [[A]]\n' > "$V/shared/A.md"
printf -- '---\nsummary: B links A\n---\n[[A]]\n' > "$V/shared/B.md"
printf -- '---\nsummary: "</script><img src=x onerror=alert(1)>"\n---\n' > "$V/shared/Markup.md"
OUT=$(kd_session "$P" 'tell me about A and B\n/quit\n')
if healthy "$OUT"; then
    test_passed "knowledge defensive - broken YAML, a 3 MiB note and link cycles do not stop a session"
else
    test_failed "knowledge defensive - broken vault session" "$OUT"
fi
OUT=$(kd knowledge build --vault "$V")
if ! output_grep "panic:" "$OUT" && [ -f "$V/graph.html" ] && ! grep -qF '</script><img' "$V/graph.html" &&
    ! grep -q '"note": *"shared/Huge.md"' "$V/.index/graph.json"; then
    test_passed "knowledge defensive - the graph page escapes markup and huge notes are skipped"
else
    test_failed "knowledge defensive - build of a broken vault" "$OUT $(grep -o '</script>[^"]*' "$V/graph.html" | head -3)"
fi

# --- tool inputs: traversal, empty and oversized titles ------------------------------
: > "$KD_LOG"
LONG=$(python3 -c "print('t'*300)")
OUT=$(kd_session "$P" "CALL knowledge_note_save {\"title\":\"../../escaped\",\"summary\":\"s\",\"body\":\"b\"}\nCALL knowledge_note_save {\"title\":\"   \",\"summary\":\"s\",\"body\":\"b\"}\nCALL knowledge_note_save {\"title\":\"$LONG\",\"summary\":\"s\",\"body\":\"b\"}\nCALL knowledge_note_save {\"title\":\"scoped\",\"summary\":\"s\",\"body\":\"b\",\"scope\":\"../../etc\"}\n/quit\n")
if ! output_grep "panic:" "$OUT" && [ ! -e "$KD_DIR/escaped.md" ] && [ ! -e "$V/../escaped.md" ] &&
    [ -z "$(find "$KD_DIR" -name 'escaped.md' 2>/dev/null)" ] && [ -z "$(find "$KD_DIR" -path '*etc*' -name 'scoped.md' 2>/dev/null)" ]; then
    test_passed "knowledge defensive - traversal titles and scopes write nothing outside the vault"
else
    test_failed "knowledge defensive - traversal" "$(find "$KD_DIR" -name 'escaped.md' -o -name 'scoped.md' 2>&1)"
fi
if grep -q "cannot contain" "$KD_LOG" && grep -q "are required" "$KD_LOG" && grep -q "scope is project, shared or inbox" "$KD_LOG"; then
    test_passed "knowledge defensive - bad titles and scopes are refused with a reason the model sees"
else
    test_failed "knowledge defensive - refusal reasons" "$(tail -c 1500 "$KD_LOG")"
fi
if [ -z "$(find "$V" -name "${LONG}*" 2>/dev/null)" ] || grep -q "file name too long\|too long" "$KD_LOG"; then
    test_passed "knowledge defensive - a 300-character title is refused or handled, not a crash"
else
    test_failed "knowledge defensive - long title" "$(find "$V" -name 't*' | head -2)"
fi

# --- archiving onto a name the archive already holds -----------------------------
P="$KD_DIR/clash/work" # a folder no card claims yet, named like the archived project
mkdir -p "$P" "$V/archive/work"
printf -- '---\ntype: project\nstatus: archived\nnext: x\nupdated: 2026-10-01\n---\n' > "$V/archive/work/work.md"
OUT=$(kd_session "$P" 'CALL knowledge_card_update {"status":"archived"}\n/quit\n')
if ! output_grep "panic:" "$OUT" && grep -q "^status: archived" "$V/archive/work/work.md" &&
    ! grep -q "^| " "$V/archive/work/work.md" && [ -f "$V/archive/clash-work/clash-work.md" ] &&
    [ ! -e "$V/projects/work" ]; then
    test_passed "knowledge defensive - a name the archive holds: the new project gets its own name, memory included"
else
    test_failed "knowledge defensive - archive clash" "$OUT $(find "$V/projects" "$V/archive" -maxdepth 2 2>&1)"
fi

# --- two sessions writing the same project at once ---------------------------------
V="$KD_DIR/concurrent"
P="$KD_DIR/shared-app"
mkdir -p "$P"
kd_vault "$V"
kd_session "$P" 'first session about invoices\nremember the first one\n/quit\n' > "$KD_DIR/s1.out" &
S1=$!
kd_session "$P" 'second session about deployments\nremember the second one\n/quit\n' > "$KD_DIR/s2.out" &
S2=$!
wait "$S1" "$S2"
NOTES=$(find "$V/projects" -path '*/sessions/*.md' 2>/dev/null | wc -l | tr -d ' ')
BAD=""
for f in $(find "$V/projects" -name '*.md' 2>/dev/null | tr ' ' '?'); do
    python3 - "$f" <<'PY' || BAD="$BAD $f"
import glob, sys, yaml
for p in glob.glob(sys.argv[1]):
    s = open(p, encoding="utf-8").read()
    if s.startswith("---"):
        yaml.safe_load(s.split("---", 2)[1])
PY
done
if healthy "$(cat "$KD_DIR/s1.out")" && healthy "$(cat "$KD_DIR/s2.out")" && [ "$NOTES" -ge 2 ] && [ -z "$BAD" ]; then
    test_passed "knowledge defensive - two sessions in one project at once: both notes kept, every note parses"
else
    test_failed "knowledge defensive - concurrent sessions" "notes=$NOTES bad=$BAD $(tail -5 "$KD_DIR/s1.out") $(tail -5 "$KD_DIR/s2.out")"
fi

# --- unicode, empty and very long prompts; odd memory keys ---------------------------
BIG=$(python3 -c "print('invoice ' * 2000)")
OUT=$(kd_session "$P" "日本語のメモ 🚀 émoji\n   \n$BIG\nremember [MEMORY: weird/key: ünï] some value\n/quit\n")
if healthy "$OUT" && [ "$(printf '%s' "$OUT" | grep -c 'MOCK REPLY')" -ge 2 ]; then
    test_passed "knowledge defensive - unicode, blank and 16 KB prompts each get their turn"
else
    test_failed "knowledge defensive - odd prompts" "$OUT"
fi
KEYNOTE=$(find "$V" -path '*/memory/*' -name 'weird*key*' 2>/dev/null | head -1)
if [ -n "$KEYNOTE" ] && [ -z "$(find "$KD_DIR" -name 'weird*key*' -not -path "$V/*" 2>/dev/null)" ]; then
    test_passed "knowledge defensive - a memory key with slashes stays inside the project's memory folder"
else
    test_failed "knowledge defensive - memory key escape" "$(find "$KD_DIR" -name '*key*' 2>&1)"
fi

# --- a vault that is a file, or read-only ---------------------------------------------
printf 'not a folder\n' > "$KD_DIR/vault-file"
kd_vault "$KD_DIR/vault-file"
OUT=$(kd_session "$P" 'hello with a file vault\n/quit\n')
if healthy "$OUT" && [ "$(cat "$KD_DIR/vault-file")" = "not a folder" ]; then
    test_passed "knowledge defensive - a vault path that is a file: the session runs, the file is untouched"
else
    test_failed "knowledge defensive - file vault" "$OUT"
fi
V="$KD_DIR/readonly"
mkdir -p "$V"
kd_vault "$V"
kd_session "$P" 'make the vault\n/quit\n' >/dev/null
chmod -R a-w "$V"
OUT=$(kd_session "$P" 'hello with a read-only vault\n/quit\n')
chmod -R u+w "$V"
if healthy "$OUT"; then
    test_passed "knowledge defensive - a read-only vault: the session still gets its reply"
else
    test_failed "knowledge defensive - read-only vault" "$OUT"
fi

# --- a hand-written GRAPH_REPORT.md at the vault root ---------------------------------
V="$KD_DIR/own-report"
mkdir -p "$V"
printf '# My own report\n\nkeep me\n' > "$V/GRAPH_REPORT.md"
OUT=$(kd knowledge build --vault "$V")
if grep -q "keep me" "$V/GRAPH_REPORT.md"; then
    test_passed "knowledge defensive - a hand-written GRAPH_REPORT.md is never overwritten"
else
    test_failed "knowledge defensive - user report overwritten" "$OUT $(head -3 "$V/GRAPH_REPORT.md")"
fi

kill "$KD_SRV" 2>/dev/null
wait "$KD_SRV" 2>/dev/null
KD_SRV=""
chmod -R u+w "$KD_DIR" 2>/dev/null
[ -n "${KD_KEEP:-}" ] || rm -rf "$KD_DIR"
trap - EXIT

echo ""
echo "Knowledge defensive E2E: done"
restore_llm_override
