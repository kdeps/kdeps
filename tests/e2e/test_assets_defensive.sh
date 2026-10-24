# shellcheck shell=bash
# E2E: asset installs and profiles under broken and hostile inputs. Each
# case checks kdeps never panics, names what went wrong and what to run,
# and leaves an installed store usable:
#   - a corrupt lock.json, and an asset store that is a file
#   - a server that returns invalid JSON, or 500s
#   - path-traversal ids, unknown sets, profiles and items
#   - installing a set the profile left empty
#   - kdeps info for an unknown item
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing assets against broken and hostile inputs..."

if ! command -v python3 >/dev/null 2>&1 || ! command -v go >/dev/null 2>&1; then
    test_skipped "assets defensive - python3 and go are needed to publish and serve assets"
    return 0 2>/dev/null || exit 0
fi

AD_DIR=$(cd "$(mktemp -d)" && pwd -P)
AD_SRV=""
trap '[ -n "$AD_SRV" ] && kill "$AD_SRV" 2>/dev/null; [ -n "${AD_KEEP:-}" ] || rm -rf "$AD_DIR"' EXIT
AD_PUB="$AD_DIR/pub"
if ! (cd "$PROJECT_ROOT" && go run ./tools/publish-assets -src "$PROJECT_ROOT/assets" -out "$AD_PUB" \
    -formulas "$AD_DIR/formulas" >/dev/null 2>&1); then
    test_failed "assets defensive - publish the repo's assets" "go run ./tools/publish-assets failed"
    return 0 2>/dev/null || exit 0
fi
# serve starts the published dir; MODE ok, badjson (index.json is garbage) or 500.
serve() {
    [ -n "$AD_SRV" ] && kill "$AD_SRV" 2>/dev/null && wait "$AD_SRV" 2>/dev/null
    AD_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
    (cd "$AD_PUB" && exec python3 -c 'import functools,http.server as h,sys
mode = sys.argv[2]
class H(h.SimpleHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        if mode == "500":
            return self.send_error(500)
        if mode == "badjson" and self.path.endswith("index.json"):
            b = b"{not json"
            self.send_response(200); self.send_header("Content-Length", str(len(b))); self.end_headers()
            return self.wfile.write(b)
        return super().do_GET()
h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), functools.partial(H, directory=".")).serve_forever()' \
        "$AD_PORT" "$1" >/dev/null 2>&1) &
    AD_SRV=$!
    for _ in $(seq 1 50); do
        python3 -c "import socket;socket.create_connection(('127.0.0.1',$AD_PORT),1)" 2>/dev/null && break
        sleep 0.1
    done
}
# ad runs kdeps in home $1 (store at $1/store) against the server.
ad() {
    local home="$1"
    shift
    mkdir -p "$home/work"
    (cd "$home/work" && env HOME="$home" USERPROFILE="$home" KDEPS_ASSETS_SRC='' KDEPS_ASSETS_DIR="$home/store" \
        KDEPS_ASSETS_URL="http://127.0.0.1:$AD_PORT" KDEPS_CONFIG_PATH="$home/config.yaml" "$@" 2>&1 </dev/null)
}
nopanic() { ! output_grep "panic:|goroutine [0-9]+ \[" "$1"; }

serve ok
H="$AD_DIR/base"
ad "$H" env KDEPS_PROFILE=minimum "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot" >/dev/null

# --- a corrupt lock.json ---------------------------------------------------------
cp "$H/store/lock.json" "$AD_DIR/lock.good"
printf '{ broken' > "$H/store/lock.json"
OUT=$(ad "$H" "$KDEPS_BIN" list --assets)
OUT2=$(ad "$H" "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
if nopanic "$OUT" && nopanic "$OUT2" && output_grep_fixed "lock.json is corrupt" "$OUT$OUT2" && output_grep_fixed "kdeps profile" "$OUT$OUT2"; then
    test_passed "assets defensive - a corrupt lock.json is named with the fix, not a crash"
else
    test_failed "assets defensive - corrupt lock" "$OUT | $OUT2"
fi
cp "$AD_DIR/lock.good" "$H/store/lock.json"

# --- a server returning garbage or errors ----------------------------------------------
for mode in badjson 500; do
    serve "$mode"
    OUT=$(ad "$H" "$KDEPS_BIN" install themes/vim)
    OUT2=$(ad "$H" "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
    if nopanic "$OUT" && ! output_grep "Installed|install +- -> " "$OUT" && output_grep_i "index|500|server|download|fetch" "$OUT" &&
        output_grep "Schema validation passed" "$OUT2"; then
        test_passed "assets defensive - a server answering $mode: install fails with the reason, installed assets keep working"
    else
        test_failed "assets defensive - server $mode" "$OUT | $OUT2"
    fi
done
serve ok

# --- hostile and unknown ids ---------------------------------------------------------
bad=""
for id in "tools/../escape" "../tools/x" "tools/a/b" "nosuchset/x" "tools/" "/etc/passwd"; do
    OUT=$(ad "$H" "$KDEPS_BIN" install "$id")
    if ! nopanic "$OUT" || [ -n "$(find "$AD_DIR" -name 'escape*' 2>/dev/null)" ]; then
        bad="$bad [$id] $OUT"
    fi
done
if [ -z "$bad" ] && [ ! -e "$H/escape.yaml" ]; then
    test_passed "assets defensive - traversal and malformed ids are refused without writing anything"
else
    test_failed "assets defensive - hostile ids" "$bad"
fi
OUT=$(ad "$H" "$KDEPS_BIN" install tools/definitely_not_published)
if nopanic "$OUT" && output_grep_i "not published|no such|unknown" "$OUT"; then
    test_passed "assets defensive - an item that is not published is named as such"
else
    test_failed "assets defensive - unpublished item" "$OUT"
fi

# --- unknown profiles ---------------------------------------------------------------------
H2="$AD_DIR/badprofile"
OUT=$(ad "$H2" env KDEPS_PROFILE=bogusprofile "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
OUT2=$(ad "$H" "$KDEPS_BIN" profile bogusprofile)
if nopanic "$OUT" && nopanic "$OUT2" && output_grep "minimum" "$OUT2" && output_grep_i "unknown|profile" "$OUT$OUT2"; then
    test_passed "assets defensive - an unknown profile is refused, listing the real ones"
else
    test_failed "assets defensive - unknown profile" "$OUT | $OUT2"
fi

# --- installing a set the profile left empty ---------------------------------------
ad "$H" "$KDEPS_BIN" profile minimum >/dev/null
mkdir -p "$H/work"
printf '/quit\n' | (cd "$H/work" && env HOME="$H" USERPROFILE="$H" KDEPS_ASSETS_SRC='' KDEPS_ASSETS_DIR="$H/store" \
    KDEPS_ASSETS_URL="http://127.0.0.1:$AD_PORT" KDEPS_CONFIG_PATH="$H/config.yaml" timeout 120 "$KDEPS_BIN" --new >/dev/null 2>&1)
OUT=$(ad "$H" "$KDEPS_BIN" install events)
OUT2=$(printf '/harness events\n/quit\n' | (cd "$H/work" && env HOME="$H" USERPROFILE="$H" KDEPS_ASSETS_SRC='' \
    KDEPS_ASSETS_DIR="$H/store" KDEPS_ASSETS_URL=off KDEPS_CONFIG_PATH="$H/config.yaml" timeout 120 "$KDEPS_BIN" --new 2>&1))
if nopanic "$OUT" && [ -n "$(find "$H/store/events" -name '*.yaml' 2>/dev/null)" ] && output_grep "auto-compact" "$OUT2"; then
    test_passed "assets defensive - a set the profile left empty installs on request and loads"
else
    test_failed "assets defensive - install an empty-on-purpose set" "$OUT | $OUT2"
fi

# Former built-ins that are now YAML commands stay in the minimum profile: their
# files are required.
OUT3=$(printf '/fold\n/memory\n/skills\n/prompts\n/reload\n/judges\n/harness\n/efficiency\n/quit\n' | (cd "$H/work" && env HOME="$H" USERPROFILE="$H" KDEPS_ASSETS_SRC='' \
    KDEPS_ASSETS_DIR="$H/store" KDEPS_ASSETS_URL=off KDEPS_CONFIG_PATH="$H/config.yaml" timeout 120 "$KDEPS_BIN" --new 2>&1))
if nopanic "$OUT3" && ! output_grep_fixed "Unknown command" "$OUT3" && output_grep_fixed "threshold:" "$OUT3" \
    && output_grep_fixed "Reloaded skills" "$OUT3"; then
    test_passed "assets defensive - the minimum profile keeps the former built-in commands"
else
    test_failed "assets defensive - former built-ins in the minimum profile" "$OUT3"
fi

# --- kdeps info for unknown assets ------------------------------------------------------
OUT=$(ad "$H" "$KDEPS_BIN" info tools/definitely_not_a_tool)
if nopanic "$OUT" && output_grep "Installed: no" "$OUT"; then
    test_passed "assets defensive - kdeps info on an item that does not exist says it is not installed"
else
    test_failed "assets defensive - info unknown item" "$OUT"
fi

# --- an asset store that is a file -----------------------------------------------------
H3="$AD_DIR/filestore"
mkdir -p "$H3"
printf 'not a dir\n' > "$H3/store"
OUT=$(ad "$H3" env KDEPS_PROFILE=minimum "$KDEPS_BIN" validate "$PROJECT_ROOT/examples/chatbot")
if nopanic "$OUT" && [ "$(cat "$H3/store")" = "not a dir" ] && output_grep_fixed "is a file, not a folder" "$OUT"; then
    test_passed "assets defensive - an asset store path that is a file is reported, the file untouched"
else
    test_failed "assets defensive - store is a file" "$OUT"
fi

kill "$AD_SRV" 2>/dev/null
wait "$AD_SRV" 2>/dev/null
AD_SRV=""
[ -n "${AD_KEEP:-}" ] || rm -rf "$AD_DIR"
trap - EXIT

echo ""
echo "Assets defensive E2E: done"
