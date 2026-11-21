# shellcheck shell=bash
# E2E: what each asset profile installs.
#
# Against a local server publishing the repo's assets:
#   - minimum installs the base sets and the chat sets (harness, events,
#     actions) that a workflow chat reads
#   - kdeps profile agent installs the agent tools and presets
#   - kdeps profile minimum prunes back what agent added
#   - an item installed by name stays on a bare update; uninstall removes it
#   - workflows installs templates, components and the chat sets
#   - full installs every set; minimum after full prunes to the base
#   - a custom selection of sets and single items is honoured exactly
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing asset profiles and what they install..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "asset profiles - python3 not available for the local asset server"
    return 0 2>/dev/null || exit 0
fi
if ! command -v go >/dev/null 2>&1; then
    test_skipped "asset profiles - go not available to publish the assets"
    return 0 2>/dev/null || exit 0
fi

AP_DIR=$(cd "$(mktemp -d)" && pwd -P)
AP_SRV=""
trap '[ -n "$AP_SRV" ] && kill "$AP_SRV" 2>/dev/null; rm -rf "$AP_DIR"' EXIT

AP_PUB="$AP_DIR/pub"
if ! (cd "$PROJECT_ROOT" && go run ./tools/publish-assets -src "$PROJECT_ROOT/assets" -out "$AP_PUB" \
    -formulas "$AP_DIR/formulas" >/dev/null 2>&1); then
    test_failed "asset profiles - publish the repo's assets" "go run ./tools/publish-assets failed"
    return 0 2>/dev/null || exit 0
fi
AP_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
(cd "$AP_PUB" && exec python3 -c 'import functools,http.server as h,sys
class H(h.SimpleHTTPRequestHandler):
    def log_message(self, *a): pass
h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), functools.partial(H, directory=".")).serve_forever()' \
    "$AP_PORT" >/dev/null 2>&1) &
AP_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:$AP_PORT/index.json')" 2>/dev/null && break
    sleep 0.1
done
AP_URL="http://127.0.0.1:$AP_PORT"

# ap runs kdeps as a release binary in home $1: no source dir, the local
# server, its own store and config. Extra env goes before the command.
ap() {
    local home="$1"
    shift
    (cd "$home/work" && env HOME="$home" USERPROFILE="$home" KDEPS_ASSETS_SRC='' KDEPS_COMPONENT_DIR='' \
        KDEPS_ASSETS_DIR="$home/store" \
        KDEPS_ASSETS_URL="${AP_URL_OVERRIDE:-$AP_URL}" KDEPS_CONFIG_PATH="$home/.kdeps/config.yaml" \
        KDEPS_DATA_DIR="$home/data" "$@" 2>&1 </dev/null)
}
ap_home() {
    local h="$AP_DIR/$1"
    mkdir -p "$h/work" "$h/.kdeps"
    printf 'llm:\n  backend: ollama\n' > "$h/.kdeps/config.yaml"
    printf '%s' "$h"
}
# set_count prints how many items of set $2 are installed in home $1.
set_count() { find "$1/store/$2" -maxdepth 1 -name '*.yaml' 2>/dev/null | wc -l | tr -d ' '; }
lock_field() { python3 -c "import json,sys;print(json.load(open(sys.argv[1])).get(sys.argv[2]))" "$1/store/lock.json" "$2"; }

# --- minimum: base sets and chat sets ---------------------------------------
MN=$(ap_home minimum)
OUT=$(ap "$MN" env KDEPS_PROFILE=minimum "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
if output_grep "Installing the minimum profile" "$OUT" && output_grep "Schema validation passed" "$OUT" &&
    [ "$(lock_field "$MN" profile)" = "minimum" ] && [ ! -d "$MN/store/commands" ] && [ ! -d "$MN/store/tools" ] &&
    [ ! -d "$MN/store/templates" ] && [ -d "$MN/store/harness" ] && [ -d "$MN/store/events" ] &&
    [ -f "$MN/store/harness/judge-system.yaml" ] && [ -f "$MN/store/events/chat-max-rounds.yaml" ]; then
    test_passed "asset profiles - minimum installs the base sets and the workflow chat harness and events; validate works"
else
    test_failed "asset profiles - minimum install" "$OUT $(ls "$MN/store" 2>&1)"
fi

# --- the knowledge graph page under minimum: its template comes on first use --
OUT=$(ap "$MN" "$KDEPS_BIN" knowledge init "$MN/vault")
OUT="$OUT $(ap "$MN" "$KDEPS_BIN" knowledge build --vault "$MN/vault")"
EXPLORE=$(printf '%s\n' "$OUT" | sed -n 's/^Explore: //p' | head -1)
if [ -f "$MN/store/templates/knowledge.yaml" ] && [ "$(set_count "$MN" templates)" = "1" ] &&
    output_grep "Explore: .*graph.html" "$OUT" && [ -n "$EXPLORE" ] && grep -q 'id="legend"' "$EXPLORE" 2>/dev/null &&
    [ ! -e "$MN/vault/graph.html" ]; then
    test_passed "asset profiles - under minimum, only templates/knowledge is installed and graph.html is written beside the graph"
else
    test_failed "asset profiles - graph page under minimum" "$OUT $(ls "$MN/store/templates" 2>&1)"
fi

# --- agent: the agent tools and presets --------------------------------------
OUT=$(ap "$MN" "$KDEPS_BIN" profile agent)
if [ "$(lock_field "$MN" profile)" = "agent" ] && [ -f "$MN/store/tools/web_search.yaml" ] &&
    [ "$(set_count "$MN" presets)" -gt 0 ] && [ ! -d "$MN/store/commands" ]; then
    test_passed "asset profiles - kdeps profile agent installs the agent tools and presets"
else
    test_failed "asset profiles - switch to agent" "profile=$(lock_field "$MN" profile) $OUT $(ls "$MN/store" 2>&1)"
fi

# --- back to minimum: pruned -------------------------------------------------
OUT=$(ap "$MN" "$KDEPS_BIN" profile minimum)
if output_grep "tools/web_search" "$OUT" && output_grep "not in the profile" "$OUT" &&
    [ ! -f "$MN/store/tools/web_search.yaml" ] && [ "$(set_count "$MN" events)" -gt 0 ]; then
    test_passed "asset profiles - kdeps profile minimum prunes what agent added"
else
    test_failed "asset profiles - prune to minimum" "$OUT"
fi
if ! grep -q '"tools/web_search"' "$MN/store/lock.json"; then
    test_passed "asset profiles - pruned items are forgotten in the lock"
else
    test_failed "asset profiles - lock after prune" "$(head -c 600 "$MN/store/lock.json")"
fi

# --- single items -------------------------------------------------------------
OUT=$(ap "$MN" "$KDEPS_BIN" install tools/calculator)
if [ -f "$MN/store/tools/calculator.yaml" ]; then
    test_passed "asset profiles - installing one item by name"
else
    test_failed "asset profiles - install one item" "$OUT"
fi
OUT=$(ap "$MN" "$KDEPS_BIN" update)
if [ -f "$MN/store/tools/calculator.yaml" ]; then
    test_passed "asset profiles - a bare update keeps an item installed by name"
else
    test_failed "asset profiles - bare update" "$OUT"
fi
OUT=$(ap "$MN" "$KDEPS_BIN" uninstall tools/calculator)
if [ ! -f "$MN/store/tools/calculator.yaml" ]; then
    test_passed "asset profiles - uninstall removes an item"
else
    test_failed "asset profiles - uninstall one item" "$OUT"
fi

# --- workflows: templates and components, no agent sets ------------------------
WF=$(ap_home workflows)
OUT=$(ap "$WF" env KDEPS_PROFILE=workflows "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
if [ "$(lock_field "$WF" profile)" = "workflows" ] && [ "$(set_count "$WF" templates)" -gt 0 ] &&
    [ "$(set_count "$WF" components)" -gt 0 ] && [ -f "$WF/store/harness/judge-system.yaml" ] &&
    [ -f "$WF/store/events/chat-max-rounds.yaml" ]; then
    test_passed "asset profiles - workflows installs templates, components and the chat harness and events"
else
    test_failed "asset profiles - workflows install" "$OUT $(ls "$WF/store" 2>&1)"
fi

# --- full, then minimum ---------------------------------------------------------
FU=$(ap_home full)
OUT=$(ap "$FU" env KDEPS_PROFILE=full "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
fu_missing=""
for set in $(find "$AP_PUB" -mindepth 1 -maxdepth 1 -type d -exec basename {} \;); do
    [ "$(set_count "$FU" "$set")" -gt 0 ] || fu_missing="$fu_missing $set"
done
if [ -z "$fu_missing" ]; then
    test_passed "asset profiles - full installs every published set"
else
    test_failed "asset profiles - full install" "missing:$fu_missing"
fi
OUT=$(ap "$FU" "$KDEPS_BIN" profile minimum)
fu_left=""
for set in templates components docker k8s recipes presets; do
    [ "$(set_count "$FU" "$set")" = "0" ] || fu_left="$fu_left $set=$(set_count "$FU" "$set")"
done
if [ -z "$fu_left" ] && [ "$(set_count "$FU" schemas)" -gt 0 ]; then
    test_passed "asset profiles - minimum after full prunes every set it leaves out"
else
    test_failed "asset profiles - full to minimum" "left:$fu_left"
fi
if OUT=$(ap "$FU" env KDEPS_ASSETS_URL=off "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot") &&
    output_grep "Schema validation passed" "$OUT"; then
    test_passed "asset profiles - after pruning, validate still works offline"
else
    test_failed "asset profiles - offline validate after prune" "$OUT"
fi

# --- custom: sets and single items, exactly ------------------------------------
CU=$(ap_home custom)
OUT=$(ap "$CU" env KDEPS_PROFILE="policy,tools/calculator" "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
if [ "$(lock_field "$CU" profile)" = "custom" ] && [ "$(set_count "$CU" policy)" = "2" ] &&
    [ "$(set_count "$CU" tools)" = "1" ] && [ -f "$CU/store/tools/calculator.yaml" ]; then
    test_passed "asset profiles - a custom selection installs exactly its sets and items"
else
    test_failed "asset profiles - custom selection" "$OUT $(ls "$CU/store" "$CU/store/tools" 2>&1)"
fi
kill "$AP_SRV" 2>/dev/null
wait "$AP_SRV" 2>/dev/null
AP_SRV=""
rm -rf "$AP_DIR"
trap - EXIT

echo ""
echo "Asset profiles E2E: done"
