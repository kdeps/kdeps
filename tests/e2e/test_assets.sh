# shellcheck shell=bash
# E2E: kdeps ships no assets of its own.
#
# Covers what test_update.sh does not:
#   - asset-free commands run offline before the first download
#   - KDEPS_ASSETS_SRC serves a checkout's assets/ to a dev binary
#   - published bundles leave hidden source files out
#   - the first download retries a flaky server
#   - a missing published item fails the first run and is named
#   - an installed file that fails its checksum is ignored
#   - an asset that does not load is skipped with a warning
#   - a prepackaged binary runs offline from the assets it bundles
#   - the Dockerfile copies the build host's assets into the image
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing kdeps assets (no embedded copies)..."

AS_DIR=$(mktemp -d)
AS_SRV=""
trap '[ -n "$AS_SRV" ] && kill "$AS_SRV" 2>/dev/null; rm -rf "$AS_DIR"' EXIT

# as_run runs kdeps like a release binary: no source dir, its own HOME and store.
as_run() {
    local home="$1"
    shift
    HOME="$home" KDEPS_ASSETS_SRC='' KDEPS_ASSETS_DIR="$home/store" "$@" 2>&1 </dev/null
}

# --- asset-free commands work offline with nothing installed -----------------
AF_HOME="$AS_DIR/free"
mkdir -p "$AF_HOME"
af_ok=1
af_out=""
for args in "validate $PROJECT_ROOT/examples/chatbot" "help" "completion bash" "update --list" "registry --help" "--version"; do
    # shellcheck disable=SC2086 # args is a word list
    if ! OUT=$(as_run "$AF_HOME" env KDEPS_ASSETS_URL=off "$KDEPS_BIN" $args) || output_grep "Downloading kdeps assets" "$OUT"; then
        af_ok=0
        af_out="$af_out [$args] $OUT"
    fi
done
if [ "$af_ok" = 1 ] && [ ! -e "$AF_HOME/store/lock.json" ]; then
    test_passed "assets - validate/help/completion/update/registry run offline before the first download"
else
    test_failed "assets - asset-free commands offline" "$af_out"
fi

# --- KDEPS_ASSETS_SRC: a dev binary reads a checkout's assets/ ---------------
AS_SRC="$AS_DIR/src"
mkdir -p "$AS_SRC"
cp -R "$PROJECT_ROOT/assets/." "$AS_SRC/"
printf '  - alias: "e2e-src-model"\n    url: "https://example.com/e2e-src.gguf"\n' >> "$AS_SRC/models/gguf.yaml"
DEV_HOME="$AS_DIR/dev"
mkdir -p "$DEV_HOME"
OUT=$(HOME="$DEV_HOME" KDEPS_ASSETS_SRC="$AS_SRC" KDEPS_ASSETS_DIR="$DEV_HOME/store" KDEPS_ASSETS_URL=off \
    "$KDEPS_BIN" llamafile list 2>&1 </dev/null)
if output_grep "GGUF +e2e-src-model" "$OUT" && ! output_grep "Downloading" "$OUT"; then
    test_passed "assets - KDEPS_ASSETS_SRC edits load without a download"
else
    test_failed "assets - KDEPS_ASSETS_SRC" "$OUT"
fi
OUT=$(HOME="$DEV_HOME" KDEPS_ASSETS_SRC="$AS_SRC" KDEPS_ASSETS_DIR="$DEV_HOME/store" KDEPS_ASSETS_URL=off \
    "$KDEPS_BIN" update --list 2>&1 </dev/null)
if output_grep "models/gguf +[0-9.]+ +source" "$OUT"; then
    test_passed "assets - update --list marks source items"
else
    test_failed "assets - update --list source state" "$OUT"
fi

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "assets - python3 not available for the local asset server"
    return 0 2>/dev/null || exit 0
fi

# --- publishing leaves hidden files out of bundles ---------------------------
: > "$AS_SRC/templates/api-service/.gitkeep"
echo "draft" > "$AS_SRC/templates/api-service/_draft.j2"
AS_PUB="$AS_DIR/pub"
if ! (cd "$PROJECT_ROOT" && go run ./tools/publish-assets -src "$AS_SRC" -out "$AS_PUB" -formulas "$AS_DIR/formulas" >/dev/null 2>&1); then
    test_failed "assets - publish source" "go run ./tools/publish-assets failed"
    return 0 2>/dev/null || exit 0
fi
bundle=$(find "$AS_PUB/templates/api-service" -name '*.yaml' | head -1)
if [ -n "$bundle" ] && grep -q "workflow.yaml" "$bundle" && ! grep -qE "\.gitkeep|_draft" "$bundle"; then
    test_passed "assets - published bundles skip .gitkeep and _ files"
else
    test_failed "assets - hidden files in bundle" "$(head -c 600 "$bundle" 2>/dev/null)"
fi

# serve starts the published dir on a free port. MODE:
#   flaky         - the first request for each path gets a 503
#   missing=PATH  - every request under PATH gets a 404
# The backlog is 64: kdeps downloads in parallel, and Windows refuses
# connections past python's default of 5.
serve() {
    [ -n "$AS_SRV" ] && kill "$AS_SRV" 2>/dev/null && wait "$AS_SRV" 2>/dev/null
    AS_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
    (cd "$AS_PUB" && exec python3 -c 'import functools,http.server as h,sys,threading
mode, seen, lock = sys.argv[2], set(), threading.Lock()
class H(h.SimpleHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        if mode.startswith("missing=") and self.path.lstrip("/").startswith(mode[8:]):
            return self.send_error(404)
        if mode == "flaky":
            with lock:
                first = self.path not in seen
                seen.add(self.path)
            if first:
                return self.send_error(503)
        return super().do_GET()
h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), functools.partial(H, directory=".")).serve_forever()' "$AS_PORT" "$1" >/dev/null 2>&1) &
    AS_SRV=$!
    for _ in $(seq 1 50); do
        python3 -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:$AS_PORT/index.json')" 2>/dev/null && break
        sleep 0.1
    done
}

# --- the first download retries a flaky server --------------------------------
serve flaky
FL_HOME="$AS_DIR/flaky"
mkdir -p "$FL_HOME"
OUT=$(as_run "$FL_HOME" env KDEPS_ASSETS_URL="http://127.0.0.1:$AS_PORT" "$KDEPS_BIN" llamafile list)
if output_grep "Downloading kdeps assets" "$OUT" && output_grep "GGUF +e2e-src-model" "$OUT" && [ -f "$FL_HOME/store/lock.json" ]; then
    test_passed "assets - first run retries 503s and installs every asset"
else
    test_failed "assets - first run against a flaky server" "$OUT"
fi

# --- a tampered installed asset is ignored, not fatal -------------------------
preset=$(find "$FL_HOME/store/presets" -name '*.yaml' | head -1)
if [ -n "$preset" ]; then
    printf 'version: 1.0.0\nname: x\n  bad: [\n' > "$preset"
    if OUT=$(as_run "$FL_HOME" env KDEPS_ASSETS_URL=off "$KDEPS_BIN" llamafile list) && output_grep "GGUF +e2e-src-model" "$OUT"; then
        test_passed "assets - a file that fails its lock checksum is ignored"
    else
        test_failed "assets - tampered installed asset" "$OUT"
    fi
else
    test_failed "assets - tampered installed asset" "no preset installed in $FL_HOME/store"
fi

# --- a missing published item fails the first run and is named --------------
serve "missing=themes/"
MS_HOME="$AS_DIR/missing"
mkdir -p "$MS_HOME"
if OUT=$(as_run "$MS_HOME" env KDEPS_ASSETS_URL="http://127.0.0.1:$AS_PORT" "$KDEPS_BIN" llamafile list); then
    test_failed "assets - first run with a missing item fails" "$OUT"
elif output_grep "run kdeps update" "$OUT" && output_grep "themes/" "$OUT"; then
    test_passed "assets - first run names the item it could not download"
else
    test_failed "assets - first run missing item message" "$OUT"
fi
kill "$AS_SRV" 2>/dev/null
wait "$AS_SRV" 2>/dev/null
AS_SRV=""

# --- an asset that does not load is skipped with a warning -------------------
printf 'version: 1.0.0\nname: x\n  bad: [\n' > "$AS_SRC/presets/e2e-broken.yaml"
if OUT=$(HOME="$DEV_HOME" KDEPS_ASSETS_SRC="$AS_SRC" KDEPS_ASSETS_DIR="$DEV_HOME/store" KDEPS_ASSETS_URL=off \
    "$KDEPS_BIN" llamafile list 2>&1 </dev/null) &&
    output_grep "kdeps: skipping asset: presets e2e-broken\.yaml" "$OUT" && output_grep "GGUF +e2e-src-model" "$OUT"; then
    test_passed "assets - an asset that does not load is skipped with a warning"
else
    test_failed "assets - broken asset warning" "$OUT"
fi

# --- a prepackaged binary carries its assets ---------------------------------
PK_WF="$AS_DIR/wf"
mkdir -p "$PK_WF/resources" "$AS_DIR/pkg"
cat > "$PK_WF/workflow.yaml" <<'YAML'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: e2e-assets
  version: "1.0.0"
  targetActionId: respond
settings:
  agentSettings:
    timezone: Etc/UTC
YAML
cat > "$PK_WF/resources/respond.yaml" <<'YAML'
actionId: respond
name: Respond
apiResponse:
  success: true
  response: bundled-assets-ok
YAML
goos=$(go env GOOS 2>/dev/null || uname -s | tr '[:upper:]' '[:lower:]')
goarch=$(go env GOARCH 2>/dev/null || uname -m | sed 's/x86_64/amd64/')
pkg=""
if "$KDEPS_BIN" bundle package "$PK_WF" --output "$AS_DIR/pkg" >/dev/null 2>&1; then
    pkg=$(find "$AS_DIR/pkg" -name '*.kdeps' -type f | head -1)
fi
if [ -n "$pkg" ] && "$KDEPS_BIN" bundle prepackage "$pkg" --arch "$goos-$goarch" --output "$AS_DIR/pkg/bin" >/dev/null 2>&1; then
    bin=$(find "$AS_DIR/pkg/bin" -type f | head -1)
    PK_HOME="$AS_DIR/pkg-home"
    mkdir -p "$PK_HOME"
    # No KDEPS_ASSETS_DIR: the binary must point the store at what it bundles.
    OUT=$(env -u KDEPS_ASSETS_DIR HOME="$PK_HOME" KDEPS_ASSETS_SRC='' KDEPS_ASSETS_URL=off timeout 120 "$bin" 2>&1 </dev/null)
    if output_grep "bundled-assets-ok" "$OUT" && [ ! -e "$PK_HOME/.kdeps/assets/lock.json" ]; then
        test_passed "assets - prepackaged binary runs offline from its bundled assets"
    else
        test_failed "assets - prepackaged binary offline" "$OUT"
    fi
    # Control: the same package run by a kdeps with no assets cannot start.
    if OUT=$(as_run "$AS_DIR/pkg-control" env KDEPS_ASSETS_URL=off "$KDEPS_BIN" run "$pkg"); then
        test_failed "assets - plain run without assets fails" "$OUT"
    elif output_grep "run kdeps update" "$OUT"; then
        test_passed "assets - the same package needs assets without the bundle"
    else
        test_failed "assets - plain run without assets message" "$OUT"
    fi
else
    test_failed "assets - prepackage" "bundle package/prepackage failed"
fi

# --- Docker images get the build host's assets -------------------------------
OUT=$("$KDEPS_BIN" bundle build "$PROJECT_ROOT/examples/chatbot" --show-dockerfile 2>&1 </dev/null)
if output_grep_fixed "COPY kdeps-assets/ /app/.kdeps/assets/" "$OUT" && output_grep "KDEPS_ASSETS_DIR=/app/\.kdeps/assets" "$OUT"; then
    test_passed "assets - Dockerfile copies kdeps-assets into the image store"
else
    test_failed "assets - Dockerfile assets" "$OUT"
fi

rm -rf "$AS_DIR"
trap - EXIT

echo ""
echo "kdeps assets E2E: done"
