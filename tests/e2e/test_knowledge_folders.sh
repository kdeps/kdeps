# shellcheck shell=bash
# E2E: search folders - folders outside a project searched with it - against
# a mock LLM. Covers kdeps knowledge folders, kdeps --index, /search
# folders|add|remove|index, the knowledge_card_update parameters, recall's
# <knowledge-files> and search_local with no path:
#   - an added folder is a "| search |" row in the card and indexed under ~/.kdeps
#   - key files (id_rsa) are never indexed, so never reach a prompt
#   - home, a filesystem root, the project's own folder, a nested folder,
#     a missing folder and a file are refused; the card is unchanged
#   - a session started inside a search folder is not claimed by the project
#   - removing a folder drops its row and its index row; check flags a gone one
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
without_llm_override # this script serves its own LLM and reads what it was sent

echo "Testing knowledge search folders (mock LLM)..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "knowledge folders - python3 not available for the mock LLM"
    return 0 2>/dev/null || exit 0
fi

KF_DIR=$(cd "$(mktemp -d)" && pwd -P)
KF_SRV=""
trap '[ -n "$KF_SRV" ] && kill "$KF_SRV" 2>/dev/null; [ -n "${KF_KEEP:-}" ] || rm -rf "$KF_DIR"' EXIT
native() { cygpath -w "$1" 2>/dev/null || printf '%s' "$1"; }
KF_LOG="$KF_DIR/requests.jsonl"
: > "$KF_LOG"
KF_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
python3 "$SCRIPT_DIR/helpers/mock_llm.py" "$KF_PORT" "$KF_LOG" >/dev/null 2>"$KF_DIR/mock.err" </dev/null &
KF_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:$KF_PORT/v1/models')" 2>/dev/null && break
    sleep 0.1
done

KF_SRC="$KF_DIR/src"
mkdir -p "$KF_SRC"
cp -R "$PROJECT_ROOT/assets/." "$KF_SRC/"
cat > "$KF_SRC/providers/mockllm.yaml" <<YAML
version: 1.0.0
name: mockllm
label: Mock LLM
order: 999
configKey: mockllm_api_key
envVar: MOCKLLM_API_KEY
url: http://127.0.0.1:$KF_PORT
chatPath: /v1/chat/completions
YAML

KF_HOME="$KF_DIR/home"
V="$KF_DIR/vault"
P="$KF_DIR/app"
DOCS="$KF_DIR/docs"
mkdir -p "$KF_HOME/.kdeps" "$P" "$DOCS/specs"
printf "llm:\n  mockllm_api_key: sk-mock\nknowledge:\n  vault: '%s'\n" "$(native "$V")" > "$KF_HOME/.kdeps/config.yaml"
printf 'The zorbulator reconciles invoices every night.\n' > "$DOCS/specs/billing.md"
printf 'zorbulator invoices private key material\n' > "$DOCS/specs/id_rsa"
export GIT_CEILING_DIRECTORIES="$KF_DIR"

kf_env() {
    env HOME="$KF_HOME" USERPROFILE="$(native "$KF_HOME")" KDEPS_CONFIG_PATH="$KF_HOME/.kdeps/config.yaml" \
        KDEPS_ASSETS_SRC="$KF_SRC" KDEPS_ASSETS_DIR="$KF_HOME/store" KDEPS_ASSETS_URL=off "$@"
}
# kf runs a kdeps command in folder $1.
kf() {
    local dir="$1"
    shift
    (cd "$dir" && kf_env "$KDEPS_BIN" "$@" 2>&1 </dev/null)
}
# kf_session runs a piped agent session in folder $1 with input $2 and flags $3...
kf_session() {
    local dir="$1" input="$2"
    shift 2
    (cd "$dir" && printf '%b' "$input" | kf_env timeout 180 "$KDEPS_BIN" --new --backend mockllm --model fake "$@" 2>&1) || true
}
card() { cat "$V/projects/app/app.md" 2>/dev/null; }
nopanic() { ! output_grep "panic:|goroutine [0-9]+ \[" "$1"; }

# --- kdeps knowledge folders add ---------------------------------------------------
kf "$P" knowledge init "$(native "$V")" >/dev/null
OUT=$(kf "$P" knowledge folders add "$(native "$DOCS")")
if nopanic "$OUT" && card | grep -q '^| search | .*docs |$' && card | grep -q '^| index | .*index.db |$' &&
    output_grep "indexed 1 files" "$OUT"; then
    test_passed "knowledge folders - add records a search row and an index row, and indexes the folder"
else
    test_failed "knowledge folders - add" "$OUT $(card)"
fi

# --- refusals leave the card unchanged ----------------------------------------------
BEFORE=$(card)
printf 'x' > "$KF_DIR/afile"
bad=""
for f in "$KF_HOME" "$P" "$DOCS/specs" "$KF_DIR/missing" "$KF_DIR/afile" "$(native "$V")"; do
    OUT=$(kf "$P" knowledge folders add "$(native "$f")")
    if ! nopanic "$OUT" || ! output_grep "home folder|own folder|already searches|no such folder|is a file|vault" "$OUT"; then
        bad="$bad [$f] $OUT"
    fi
done
if [ -z "$bad" ] && [ "$(card)" = "$BEFORE" ]; then
    test_passed "knowledge folders - home, own, nested, missing, file and vault folders are refused; card unchanged"
else
    test_failed "knowledge folders - refusals" "$bad"
fi
OUT=$(kf "$P" knowledge folders add "$(native "$DOCS")")
if nopanic "$OUT" && [ "$(card | grep -c '^| search |')" = "1" ]; then
    test_passed "knowledge folders - adding a listed folder again re-indexes it, no duplicate row"
else
    test_failed "knowledge folders - re-add" "$OUT $(card)"
fi

# --- recall and search_local without a path cover the folder ---------------------------
: > "$KF_LOG"
OUT=$(kf_session "$P" 'how does the zorbulator handle invoices\nCALL search_local {"query":"zorbulator"}\n/quit\n')
if nopanic "$OUT" && grep -q 'knowledge-files' "$KF_LOG" && grep -q 'billing.md' "$KF_LOG"; then
    test_passed "knowledge folders - recall names the search folder's file in <knowledge-files>"
else
    test_failed "knowledge folders - recall" "$OUT $(grep -o 'knowledge-files[^<]*' "$KF_LOG" | head -2)"
fi
if grep -q 'folders' "$KF_LOG" && grep -q 'billing.md' "$KF_LOG" && ! grep -q 'id_rsa' "$KF_LOG"; then
    test_passed "knowledge folders - search_local without a path searches every project folder; key files never appear"
else
    test_failed "knowledge folders - search_local no path" "$(grep -o '.\{80\}billing.md.\{80\}' "$KF_LOG" | head -2) id_rsa=$(grep -c id_rsa "$KF_LOG")"
fi

# --- a session inside the search folder is its own project ------------------------------
OUT=$(kf_session "$DOCS" 'hello from the docs\n/quit\n')
if nopanic "$OUT" && [ -f "$V/projects/docs/docs.md" ] && ! grep -q 'hello from the docs' "$V"/projects/app/sessions/*.md 2>/dev/null; then
    test_passed "knowledge folders - a session started in a search folder is not claimed by the project"
else
    test_failed "knowledge folders - search folder claimed" "$OUT $(ls "$V/projects")"
fi

# --- kdeps --index and /search in a session ------------------------------------------------
LIB="$KF_DIR/lib"
mkdir -p "$LIB"
printf 'The frobnicator lives in the shared library.\n' > "$LIB/frob.md"
OUT=$(kf_session "$P" '/search folders\n/quit\n' --index "$(native "$LIB")")
OUT2=$(kf_session "$P" '/quit\n' --index "$(native "$LIB")")
if nopanic "$OUT" && card | grep -q '^| search | .*lib |$' && output_grep "lib" "$OUT" && nopanic "$OUT2" &&
    [ "$(card | grep -c '^| search | .*lib |$')" = "1" ]; then
    test_passed "knowledge folders - kdeps --index adds and indexes a folder; passing it again is fine"
else
    test_failed "knowledge folders - --index" "$OUT | $OUT2 | $(card)"
fi
# kf_session's printf %b reads escapes: a Windows path's backslashes are doubled.
OUT=$(kf_session "$P" '/search remove '"$(native "$LIB" | sed 's/\\/\\\\/g')"'\n/search folders\n/quit\n')
if nopanic "$OUT" && ! card | grep -q 'lib |$' && output_grep "no longer searches" "$OUT"; then
    test_passed "knowledge folders - /search remove drops the folder's search and index rows"
else
    test_failed "knowledge folders - /search remove" "$OUT $(card)"
fi
# Escaped twice: once for JSON, once for kf_session's printf %b.
OUT=$(kf_session "$P" 'CALL knowledge_card_update {"add_search_folder":"'"$(native "$LIB" | sed 's/\\/\\\\\\\\/g')"'"}\n/quit\n')
for _ in $(seq 1 30); do card | grep -q '^| search | .*lib |$' && break; sleep 0.5; done
if nopanic "$OUT" && card | grep -q '^| search | .*lib |$'; then
    test_passed "knowledge folders - the model adds a search folder with knowledge_card_update"
else
    test_failed "knowledge folders - card tool add" "$OUT $(card)"
fi

# --- remove, and a folder that is gone -------------------------------------------------------
OUT=$(kf "$P" knowledge folders remove "$(native "$DOCS")")
if nopanic "$OUT" && ! card | grep -q '^| search | .*docs |$' && ! card | grep -q '^| index | .*docs-[0-9a-f]*[/\\]index.db |$'; then
    test_passed "knowledge folders - remove drops the search row and its index row"
else
    test_failed "knowledge folders - remove" "$OUT $(card)"
fi
rm -rf "$LIB"
OUT=$(kf "$P" knowledge check --vault "$(native "$V")")
if nopanic "$OUT" && output_grep "search folder .*lib is gone" "$OUT" && ! output_grep "which has no note" "$OUT"; then
    test_passed "knowledge folders - kdeps knowledge check flags a search folder that is gone"
else
    test_failed "knowledge folders - check gone folder" "$OUT"
fi

kill "$KF_SRV" 2>/dev/null
wait "$KF_SRV" 2>/dev/null
KF_SRV=""
[ -n "${KF_KEEP:-}" ] || rm -rf "$KF_DIR"
trap - EXIT
unset GIT_CEILING_DIRECTORIES

echo ""
echo "Knowledge folders E2E: done"
restore_llm_override
