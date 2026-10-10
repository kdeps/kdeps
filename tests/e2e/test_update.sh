# shellcheck shell=bash
# E2E: kdeps update against a local copy of the published assets/ in kdeps/packages.
#
# Publishes one newer tool definition and one newer harness section, then
# checks --check, a bare update, --list, a pinned downgrade, --remove (and the
# refusal for a required item) and restoring a removed item by name.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing kdeps update..."

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

upd() { KDEPS_ASSETS_URL="http://127.0.0.1:$UPD_PORT" KDEPS_ASSETS_DIR="$UPD_ROOT" "$KDEPS_BIN" update "$@" 2>&1; }

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
if output_grep "harness/safety +1\.2\.0 +downloaded" "$OUT" && output_grep "harness/accuracy +1\.0\.0 +built-in" "$OUT"; then
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
if output_grep "themes/vim +1\.0\.0 +built-in" "$(upd --list)"; then
    test_passed "update <item> restores a removed item"
else
    test_failed "update <item> restore" "$OUT"
fi

kill "$UPD_SRV" 2>/dev/null
wait "$UPD_SRV" 2>/dev/null
rm -rf "$UPD_DIR"
trap - EXIT

echo ""
echo "kdeps update E2E: done"
