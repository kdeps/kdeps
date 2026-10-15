# shellcheck shell=bash
# E2E: kdeps registry update against a local copy of the published assets/ in kdeps/packages.
#
# Publishes one newer tool definition and one newer harness section, then
# checks --check, a bare update, --list, a pinned downgrade, --remove (and the
# refusal for a required item) and restoring a removed item by name.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing kdeps registry update..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "update - python3 not available for the local asset server"
    return 0 2>/dev/null || exit 0
fi

UPD_DIR=$(mktemp -d)
UPD_PUB="$UPD_DIR/published"
UPD_ROOT="$UPD_DIR/assets"
mkdir -p "$UPD_PUB/tools/md5_file" "$UPD_PUB/harness/safety" "$UPD_PUB/models/gguf"

sha() { python3 -c 'import hashlib,sys;print(hashlib.sha256(open(sys.argv[1],"rb").read()).hexdigest())' "$1"; }

printf 'version: 1.1.0\ncategory: file\ndescription: Updated md5 description.\n' > "$UPD_PUB/tools/md5_file/1.1.0.yaml"
printf 'version: 1.0.0\nname: safety\nkind: preamble-section\norder: 50\nbody: published 1.0.0\n' > "$UPD_PUB/harness/safety/1.0.0.yaml"
printf 'version: 1.2.0\nname: safety\nkind: preamble-section\norder: 50\nbody: published 1.2.0\n' > "$UPD_PUB/harness/safety/1.2.0.yaml"
printf 'version: 9.0.0\nggufs:\n  - alias: e2e-fresh-model\n    url: https://example.com/e2e-fresh.gguf\n' > "$UPD_PUB/models/gguf/9.0.0.yaml"
cat > "$UPD_PUB/index.json" <<JSON
{"items": {
  "models/gguf": {"latest": "9.0.0", "versions": [
    {"version": "9.0.0", "sha256": "$(sha "$UPD_PUB/models/gguf/9.0.0.yaml")", "date": "2026-10-09T00:00:00Z"}]},
  "tools/md5_file": {"latest": "1.1.0", "versions": [
    {"version": "1.1.0", "sha256": "$(sha "$UPD_PUB/tools/md5_file/1.1.0.yaml")", "date": "2026-10-09T00:00:00Z"}]},
  "harness/safety": {"latest": "1.2.0", "versions": [
    {"version": "1.0.0", "sha256": "$(sha "$UPD_PUB/harness/safety/1.0.0.yaml")", "date": "2026-10-09T00:00:00Z"},
    {"version": "1.2.0", "sha256": "$(sha "$UPD_PUB/harness/safety/1.2.0.yaml")", "date": "2026-10-09T00:00:00Z"}]}
}}
JSON

UPD_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
(cd "$UPD_PUB" && exec python3 -m http.server "$UPD_PORT" --bind 127.0.0.1 >/dev/null 2>&1) &
UPD_SRV=$!
trap 'kill "$UPD_SRV" 2>/dev/null; rm -rf "$UPD_DIR"' EXIT
for _ in $(seq 1 50); do
    python3 -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:$UPD_PORT/index.json')" 2>/dev/null && break
    sleep 0.1
done

# upd runs the kdeps registry asset subcommand for: --list, --remove, else update.
upd() {
    local sub=(update)
    case "${1:-}" in
    --list) sub=(list --assets); shift ;;
    --remove) sub=(uninstall); shift ;;
    esac
    KDEPS_ASSETS_URL="http://127.0.0.1:$UPD_PORT" KDEPS_ASSETS_DIR="$UPD_ROOT" "$KDEPS_BIN" registry "${sub[@]}" "$@" 2>&1
}

OUT=$(upd --check)
if output_grep "tools/md5_file +available +1\.0\.0 -> 1\.1\.0" "$OUT" && output_grep "harness/safety +available" "$OUT" && [ ! -f "$UPD_ROOT/lock.json" ]; then
    test_passed "update --check lists updates without installing"
else
    test_failed "update --check" "$OUT"
fi

OUT=$(upd)
if output_grep "tools/md5_file +install" "$OUT" && grep -q "Updated md5 description" "$UPD_ROOT/tools/md5_file.yaml" 2>/dev/null; then
    test_passed "update installs newer versions"
else
    test_failed "update" "$OUT"
fi

OUT=$(HOME="$UPD_DIR" KDEPS_ASSETS_URL=off KDEPS_ASSETS_DIR="$UPD_ROOT" "$KDEPS_BIN" llamafile list 2>&1)
if output_grep "GGUF +e2e-fresh-model" "$OUT" && ! output_grep "GGUF +qwen3\.6 " "$OUT"; then
    test_passed "update models/gguf replaces the model registry"
else
    test_failed "update models/gguf" "$OUT"
fi

if OUT=$(upd --remove models/gguf); then
    test_failed "update --remove refuses a model registry" "$OUT"
else
    test_passed "update --remove refuses a model registry"
fi

OUT=$(upd --list)
if output_grep "harness/safety +1\.2\.0 +downloaded" "$OUT" && output_grep "harness/accuracy +1\.0\.0 +source" "$OUT"; then
    test_passed "update --list shows versions and state"
else
    test_failed "update --list" "$OUT"
fi

OUT=$(upd harness/safety@1.0.0)
if output_grep "harness/safety +pin +1\.2\.0 -> 1\.0\.0" "$OUT" && output_grep "harness/safety +1\.0\.0 +pinned" "$(upd --list)"; then
    test_passed "update item@version pins (downgrade)"
else
    test_failed "update item@version" "$OUT"
fi

OUT=$(upd --remove themes/vim)
if output_grep "themes/vim +remove" "$OUT" && output_grep "themes/vim +1\.0\.0 +removed" "$(upd --list)"; then
    test_passed "update --remove removes an item"
else
    test_failed "update --remove" "$OUT"
fi

if OUT=$(upd --remove themes/normal); then
    test_failed "update --remove refuses a required item" "$OUT"
elif output_grep "required" "$OUT"; then
    test_passed "update --remove refuses a required item"
else
    test_failed "update --remove required message" "$OUT"
fi

OUT=$(upd themes/vim)
if output_grep "themes/vim +1\.0\.0 +source" "$(upd --list)"; then
    test_passed "update <item> restores a removed item"
else
    test_failed "update <item> restore" "$OUT"
fi

kill "$UPD_SRV" 2>/dev/null
wait "$UPD_SRV" 2>/dev/null
rm -rf "$UPD_DIR"
trap - EXIT

# First run of a release binary: no source directory and an empty store.
FR_DIR=$(mktemp -d)
if (cd "$PROJECT_ROOT" && go run ./tools/publish-assets -src assets -out "$FR_DIR/pub" -formulas "$FR_DIR/formulas" >/dev/null 2>&1); then
    FR_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
    # kdeps downloads in parallel; the default listen backlog (5) makes Windows
    # refuse connections beyond it.
    (cd "$FR_DIR/pub" && exec python3 -c 'import functools,http.server as h,sys
h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), functools.partial(h.SimpleHTTPRequestHandler, directory=".")).serve_forever()' "$FR_PORT" >/dev/null 2>&1) &
    FR_SRV=$!
    trap 'kill "$FR_SRV" 2>/dev/null; rm -rf "$FR_DIR"' EXIT
    for _ in $(seq 1 50); do
        python3 -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:$FR_PORT/index.json')" 2>/dev/null && break
        sleep 0.1
    done
    fr() { HOME="$FR_DIR" KDEPS_ASSETS_SRC= KDEPS_ASSETS_DIR="$FR_DIR/store" "$@" 2>&1 </dev/null; }

    if OUT=$(fr env KDEPS_ASSETS_URL=off "$KDEPS_BIN" llamafile list); then
        test_failed "first run offline refuses to start" "$OUT"
    elif output_grep "run kdeps registry update" "$OUT"; then
        test_passed "first run offline refuses to start"
    else
        test_failed "first run offline message" "$OUT"
    fi

    OUT=$(fr env KDEPS_ASSETS_URL="http://127.0.0.1:$FR_PORT" KDEPS_PROFILE= "$KDEPS_BIN" llamafile list)
    if output_grep "Installing models" "$OUT" && output_grep "GGUF" "$OUT" && [ ! -e "$FR_DIR/store/themes" ]; then
        test_passed "first run without a terminal installs only what the command needs"
    else
        test_failed "first run download" "$OUT"
    fi

    OUT=$(fr env KDEPS_ASSETS_URL="http://127.0.0.1:$FR_PORT" "$KDEPS_BIN" registry profile agent)
    if [ -f "$FR_DIR/store/themes/normal.yaml" ] && [ -f "$FR_DIR/store/tools/bash_exec.yaml" ] &&
        [ ! -e "$FR_DIR/store/tools/zapier_run_action.yaml" ] && [ ! -e "$FR_DIR/store/docker" ] &&
        grep -q '"profile": "agent"' "$FR_DIR/store/lock.json"; then
        test_passed "registry profile agent installs the agent profile and records it"
    else
        test_failed "registry profile agent" "$OUT"
    fi

    if OUT=$(fr env KDEPS_ASSETS_URL="http://127.0.0.1:$FR_PORT" "$KDEPS_BIN" registry profile); then
        test_failed "registry profile without a terminal fails" "$OUT"
    elif output_grep "needs a terminal" "$OUT"; then
        test_passed "registry profile without a terminal asks for a profile name"
    else
        test_failed "registry profile message" "$OUT"
    fi

    MIN_HOME="$FR_DIR/min"
    mkdir -p "$MIN_HOME"
    OUT=$(HOME="$MIN_HOME" KDEPS_ASSETS_SRC= KDEPS_ASSETS_DIR="$MIN_HOME/store" KDEPS_PROFILE=minimum \
        KDEPS_ASSETS_URL="http://127.0.0.1:$FR_PORT" "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot" 2>&1 </dev/null)
    if output_grep "Installing the minimum profile" "$OUT" && output_grep "Schema validation passed" "$OUT" &&
        [ -f "$MIN_HOME/store/schemas/v1.yaml" ] && [ ! -e "$MIN_HOME/store/harness" ] && [ ! -e "$MIN_HOME/store/components" ]; then
        test_passed "KDEPS_PROFILE=minimum installs only the base sets; validate works"
    else
        test_failed "KDEPS_PROFILE=minimum first run" "$OUT"
    fi

    ITEM_HOME="$FR_DIR/items"
    mkdir -p "$ITEM_HOME"
    OUT=$(HOME="$ITEM_HOME" KDEPS_ASSETS_SRC= KDEPS_ASSETS_DIR="$ITEM_HOME/store" KDEPS_PROFILE=tools/bash_exec,themes/normal \
        KDEPS_ASSETS_URL="http://127.0.0.1:$FR_PORT" "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot" 2>&1 </dev/null)
    if output_grep "Installing the custom profile" "$OUT" && [ -f "$ITEM_HOME/store/tools/bash_exec.yaml" ] &&
        [ -f "$ITEM_HOME/store/themes/normal.yaml" ] && [ ! -e "$ITEM_HOME/store/tools/read_file.yaml" ] &&
        [ ! -e "$ITEM_HOME/store/themes/vim.yaml" ] && [ -f "$ITEM_HOME/store/schemas/v1.yaml" ]; then
        test_passed "KDEPS_PROFILE item list installs those items and the base, nothing else"
    else
        test_failed "KDEPS_PROFILE item list" "$OUT"
    fi

    OUT=$(fr env KDEPS_ASSETS_URL=off "$KDEPS_BIN" llamafile list)
    if ! output_grep "Installing" "$OUT" && output_grep "GGUF" "$OUT"; then
        test_passed "installed assets work offline"
    else
        test_failed "installed assets offline" "$OUT"
    fi

    kill "$FR_SRV" 2>/dev/null
    wait "$FR_SRV" 2>/dev/null
    trap - EXIT
else
    test_failed "first run: publish assets" "go run ./tools/publish-assets failed"
fi
rm -rf "$FR_DIR"

echo ""
echo "kdeps registry update E2E: done"
