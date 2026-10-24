# shellcheck shell=bash
# E2E: what each asset profile installs, and what the agent loop then shows.
#
# Against a local server publishing the repo's assets:
#   - minimum installs the base sets only; the agent loop then adds just the
#     items its loaders require (8 core commands, the normal theme, the
#     required tools and harness), never whole sets
#   - /help under minimum lists only the core commands; other built-ins
#     still run; a second start downloads nothing and works offline
#   - kdeps profile agent installs the whole commands set: /help lists them
#   - kdeps profile minimum prunes back what agent added (required stay)
#   - installing one command lists it; uninstalling hides it but it runs;
#     a required command cannot be uninstalled
#   - workflows installs templates and components but no agent sets
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
# ap_loop runs one piped agent session in home $1 with input $2.
ap_loop() {
    local home="$1" input="$2"
    shift 2
    (cd "$home/work" && printf '%b' "$input" | env HOME="$home" USERPROFILE="$home" KDEPS_ASSETS_SRC='' \
        KDEPS_COMPONENT_DIR='' \
        KDEPS_ASSETS_DIR="$home/store" KDEPS_ASSETS_URL="${AP_URL_OVERRIDE:-$AP_URL}" \
        KDEPS_CONFIG_PATH="$home/.kdeps/config.yaml" KDEPS_DATA_DIR="$home/data" "$@" \
        timeout 180 "$KDEPS_BIN" --new 2>&1) || true
}
ap_home() {
    local h="$AP_DIR/$1"
    mkdir -p "$h/work" "$h/.kdeps"
    printf 'llm:\n  backend: ollama\n' > "$h/.kdeps/config.yaml"
    printf '%s' "$h"
}
# set_count prints how many items of set $2 are installed in home $1.
set_count() { find "$1/store/$2" -maxdepth 1 -name '*.yaml' 2>/dev/null | wc -l | tr -d ' '; }
# help_commands prints the /help command names, one per line, sorted and unique.
help_commands() {
    printf '%s\n' "$1" | sed 's/\x1b\[[0-9;?]*[a-zA-Z]//g' | sed -n 's/^  \(\/[a-z][a-z!-]*\).*/\1/p' | sort -u
}
lock_field() { python3 -c "import json,sys;print(json.load(open(sys.argv[1])).get(sys.argv[2]))" "$1/store/lock.json" "$2"; }

CORE="/clear /compact /exit /help /model /session /settings /update"
TOTAL_COMMANDS=$(find "$PROJECT_ROOT/assets/commands" -name '*.yaml' | wc -l | tr -d ' ')

# --- minimum: base sets only ------------------------------------------------
MN=$(ap_home minimum)
OUT=$(ap "$MN" env KDEPS_PROFILE=minimum "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
if output_grep "Installing the minimum profile" "$OUT" && output_grep "Schema validation passed" "$OUT" &&
    [ "$(lock_field "$MN" profile)" = "minimum" ] && [ ! -d "$MN/store/commands" ] && [ ! -d "$MN/store/tools" ] &&
    [ ! -d "$MN/store/themes" ] && [ ! -d "$MN/store/templates" ]; then
    test_passed "asset profiles - minimum installs the base sets only; validate works"
else
    test_failed "asset profiles - minimum install" "$OUT $(ls "$MN/store" 2>&1)"
fi

OUT=$(ap_loop "$MN" '/help\n/context\n/quit\n')
INSTALL_LINE=$(printf '%s\n' "$OUT" | grep "needed by this command" | head -1)
if [ -n "$INSTALL_LINE" ] && ! output_grep "Installing [a-z]+, " "$INSTALL_LINE" &&
    printf '%s' "$INSTALL_LINE" | grep -q "commands/help" && ! printf '%s' "$INSTALL_LINE" | grep -q "commands/context"; then
    test_passed "asset profiles - the agent loop under minimum adds required items, never whole sets"
else
    test_failed "asset profiles - agent loop install under minimum" "$INSTALL_LINE"
fi
if [ "$(set_count "$MN" commands)" = "8" ] && [ "$(set_count "$MN" themes)" = "1" ] && [ -f "$MN/store/themes/normal.yaml" ] &&
    [ ! -d "$MN/store/events" ] && [ ! -d "$MN/store/actions" ] && [ ! -d "$MN/store/presets" ] &&
    [ ! -f "$MN/store/tools/web_search.yaml" ] && [ -f "$MN/store/tools/session_handshake.yaml" ]; then
    test_passed "asset profiles - minimum keeps 8 core commands, the normal theme, required tools; no events/actions/presets"
else
    test_failed "asset profiles - minimum store contents" "commands=$(set_count "$MN" commands) themes=$(set_count "$MN" themes) $(ls "$MN/store" "$MN/store/tools" 2>&1 | head -40)"
fi
GOT=$(help_commands "$OUT" | tr '\n' ' ' | sed 's/ $//')
if [ "$GOT" = "$CORE" ]; then
    test_passed "asset profiles - /help under minimum lists only the core commands"
else
    test_failed "asset profiles - /help under minimum" "got: $GOT"
fi
if output_grep "Context window:" "$OUT" &&
    ! output_grep "Unknown command: /context" "$OUT"; then
    test_passed "asset profiles - built-ins without their help file still run"
else
    test_failed "asset profiles - unlisted built-ins" "$OUT"
fi
if [ "$(python3 -c "import json,sys;print(','.join(sorted(json.load(open(sys.argv[1])).get('empty') or [])))" "$MN/store/lock.json")" = "actions,events,presets" ]; then
    test_passed "asset profiles - sets minimum leaves empty are recorded in the lock"
else
    test_failed "asset profiles - lock empty sets" "$(cat "$MN/store/lock.json" | head -c 400)"
fi

OUT=$(AP_URL_OVERRIDE=off ap_loop "$MN" '/help\n/quit\n')
if ! output_grep "Installing |kdeps assets .* not installed" "$OUT" && [ "$(help_commands "$OUT" | tr '\n' ' ' | sed 's/ $//')" = "$CORE" ]; then
    test_passed "asset profiles - a second start under minimum downloads nothing and works offline"
else
    test_failed "asset profiles - offline restart under minimum" "$OUT"
fi

# --- the knowledge graph page under minimum: its template comes on first use --
OUT=$(ap "$MN" "$KDEPS_BIN" knowledge init "$MN/vault")
OUT="$OUT $(ap "$MN" "$KDEPS_BIN" knowledge build --vault "$MN/vault")"
if [ -f "$MN/store/templates/knowledge.yaml" ] && [ "$(set_count "$MN" templates)" = "1" ] &&
    output_grep "Explore: .*graph.html" "$OUT" && grep -q 'id="legend"' "$MN/vault/graph.html" 2>/dev/null; then
    test_passed "asset profiles - under minimum, only templates/knowledge is installed and graph.html is written"
else
    test_failed "asset profiles - graph page under minimum" "$OUT $(ls "$MN/store/templates" 2>&1)"
fi

# --- agent: the whole commands set ------------------------------------------
OUT=$(ap "$MN" "$KDEPS_BIN" profile agent)
HELP=$(ap_loop "$MN" '/help\n/quit\n')
N=$(help_commands "$HELP" | wc -l | tr -d ' ')
if [ "$(lock_field "$MN" profile)" = "agent" ] && [ "$(set_count "$MN" commands)" = "$TOTAL_COMMANDS" ] &&
    help_commands "$HELP" | grep -qx "/recall" && help_commands "$HELP" | grep -qx "/refine" && [ "$N" -gt 20 ] &&
    [ -f "$MN/store/tools/web_search.yaml" ] && [ "$(lock_field "$MN" empty)" = "None" ]; then
    test_passed "asset profiles - kdeps profile agent installs every command; /help lists them ($N)"
else
    test_failed "asset profiles - switch to agent" "profile=$(lock_field "$MN" profile) commands=$(set_count "$MN" commands)/$TOTAL_COMMANDS help=$N $OUT"
fi

# --- back to minimum: pruned -------------------------------------------------
OUT=$(ap "$MN" "$KDEPS_BIN" profile minimum)
if output_grep "commands/refine" "$OUT" && output_grep "not in the profile" "$OUT" &&
    [ "$(set_count "$MN" commands)" = "8" ] && [ ! -f "$MN/store/tools/web_search.yaml" ] &&
    [ -f "$MN/store/tools/session_handshake.yaml" ] && [ ! -d "$MN/store/events" ] || [ "$(set_count "$MN" events)" = "0" ]; then
    test_passed "asset profiles - kdeps profile minimum prunes what agent added, keeps required items"
else
    test_failed "asset profiles - prune to minimum" "commands=$(set_count "$MN" commands) $OUT"
fi
HELP=$(ap_loop "$MN" '/help\n/quit\n')
if [ "$(help_commands "$HELP" | tr '\n' ' ' | sed 's/ $//')" = "$CORE" ]; then
    test_passed "asset profiles - /help is back to the core commands after pruning"
else
    test_failed "asset profiles - /help after pruning" "$(help_commands "$HELP" | tr '\n' ' ')"
fi
if ! grep -q '"commands/refine"' "$MN/store/lock.json"; then
    test_passed "asset profiles - pruned items are forgotten in the lock"
else
    test_failed "asset profiles - lock after prune" "$(head -c 600 "$MN/store/lock.json")"
fi

# --- single items -------------------------------------------------------------
OUT=$(ap "$MN" "$KDEPS_BIN" install commands/refine)
HELP=$(ap_loop "$MN" '/help\n/quit\n')
if [ -f "$MN/store/commands/refine.yaml" ] && help_commands "$HELP" | grep -qx "/refine" &&
    [ "$(help_commands "$HELP" | wc -l | tr -d ' ')" = "9" ]; then
    test_passed "asset profiles - installing one command file lists it in /help"
else
    test_failed "asset profiles - install one command" "$OUT $(help_commands "$HELP" | tr '\n' ' ')"
fi
OUT=$(ap "$MN" "$KDEPS_BIN" update)
if [ -f "$MN/store/commands/refine.yaml" ] && [ "$(set_count "$MN" commands)" = "9" ]; then
    test_passed "asset profiles - a bare update keeps an item installed by name, adds nothing else"
else
    test_failed "asset profiles - bare update" "commands=$(set_count "$MN" commands) $OUT"
fi
OUT=$(ap "$MN" "$KDEPS_BIN" uninstall commands/refine)
HELP=$(ap_loop "$MN" '/help\n/refine\n/quit\n')
# /refine is a YAML setting: command, so without its file it does not exist.
if ! help_commands "$HELP" | grep -qx "/refine" && output_grep "Unknown command: /refine" "$HELP"; then
    test_passed "asset profiles - uninstalling a YAML command file removes the command"
else
    test_failed "asset profiles - uninstall one command" "$OUT $HELP"
fi
if OUT=$(ap "$MN" "$KDEPS_BIN" uninstall commands/model); then
    test_failed "asset profiles - a required command is refused" "$OUT"
elif output_grep "required" "$OUT" && [ -f "$MN/store/commands/model.yaml" ]; then
    test_passed "asset profiles - a required command file cannot be uninstalled"
else
    test_failed "asset profiles - uninstall required" "$OUT"
fi

# --- workflows: templates and components, no agent sets ------------------------
WF=$(ap_home workflows)
OUT=$(ap "$WF" env KDEPS_PROFILE=workflows "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
if [ "$(lock_field "$WF" profile)" = "workflows" ] && [ "$(set_count "$WF" templates)" -gt 0 ] &&
    [ "$(set_count "$WF" components)" -gt 0 ] && [ ! -d "$WF/store/commands" ] && [ ! -d "$WF/store/harness" ]; then
    test_passed "asset profiles - workflows installs templates and components, no agent sets"
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
if [ -z "$fu_missing" ] && [ "$(set_count "$FU" commands)" = "$TOTAL_COMMANDS" ]; then
    test_passed "asset profiles - full installs every published set"
else
    test_failed "asset profiles - full install" "missing:$fu_missing"
fi
OUT=$(ap "$FU" "$KDEPS_BIN" profile minimum)
fu_left=""
for set in templates components docker k8s recipes events actions presets; do
    [ "$(set_count "$FU" "$set")" = "0" ] || fu_left="$fu_left $set=$(set_count "$FU" "$set")"
done
if [ -z "$fu_left" ] && [ "$(set_count "$FU" schemas)" -gt 0 ] && [ "$(set_count "$FU" commands)" = "8" ]; then
    test_passed "asset profiles - minimum after full prunes every non-base set, keeping required items"
else
    test_failed "asset profiles - full to minimum" "left:$fu_left commands=$(set_count "$FU" commands)"
fi
if OUT=$(ap "$FU" env KDEPS_ASSETS_URL=off "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot") &&
    output_grep "Schema validation passed" "$OUT"; then
    test_passed "asset profiles - after pruning, validate still works offline"
else
    test_failed "asset profiles - offline validate after prune" "$OUT"
fi

# --- custom: sets and single items, exactly ------------------------------------
CU=$(ap_home custom)
OUT=$(ap "$CU" env KDEPS_PROFILE="themes,commands/recall,tools/calculator" "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
if [ "$(lock_field "$CU" profile)" = "custom" ] && [ "$(set_count "$CU" themes)" -gt 1 ] &&
    [ "$(set_count "$CU" commands)" = "1" ] && [ -f "$CU/store/commands/recall.yaml" ] &&
    [ "$(set_count "$CU" tools)" = "1" ] && [ -f "$CU/store/tools/calculator.yaml" ]; then
    test_passed "asset profiles - a custom selection installs exactly its sets and items"
else
    test_failed "asset profiles - custom selection" "$OUT $(ls "$CU/store" "$CU/store/commands" "$CU/store/tools" 2>&1)"
fi
HELP=$(ap_loop "$CU" '/help\n/quit\n')
GOT=$(help_commands "$HELP" | tr '\n' ' ' | sed 's/ $//')
if [ "$GOT" = "/clear /compact /exit /help /model /recall /session /settings /update" ]; then
    test_passed "asset profiles - /help under a custom selection: the core plus the chosen command"
else
    test_failed "asset profiles - /help under custom" "got: $GOT"
fi

kill "$AP_SRV" 2>/dev/null
wait "$AP_SRV" 2>/dev/null
AP_SRV=""
rm -rf "$AP_DIR"
trap - EXIT

echo ""
echo "Asset profiles E2E: done"
