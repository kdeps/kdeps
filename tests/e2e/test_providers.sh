# shellcheck shell=bash
# E2E: an LLM provider that exists only as a providers/ asset.
#
# The provider file points kdeps at a fake OpenAI-compatible server; its key
# lives in config.yaml under the provider's configKey. The agent loop must
# call the server at the declared chat path with that key as the bearer token.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing LLM providers declared in YAML (providers asset set)..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "providers - python3 not available for the fake provider"
    return 0 2>/dev/null || exit 0
fi

PV_DIR=$(mktemp -d)
PV_SRV=""
trap '[ -n "$PV_SRV" ] && kill "$PV_SRV" 2>/dev/null; rm -rf "$PV_DIR"' EXIT

cat > "$PV_DIR/fake.py" <<'PY'
import http.server as h, json, sys, time
class H(h.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def do_GET(self):
        b = json.dumps({"data": [{"id": "fake"}]}).encode()
        self.send_response(200); self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}")
        print("REQ path=%s auth=%s" % (self.path, self.headers.get("Authorization", "")), file=sys.stderr, flush=True)
        msg = {"role": "assistant", "content": "PROVIDER OK"}
        if body.get("stream"):
            self.send_response(200); self.send_header("Content-Type", "text/event-stream"); self.end_headers()
            ch = {"id": "x", "object": "chat.completion.chunk", "created": int(time.time()), "model": "fake",
                  "choices": [{"index": 0, "delta": msg, "finish_reason": None}]}
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\n")
            ch["choices"] = [{"index": 0, "delta": {}, "finish_reason": "stop"}]
            ch["usage"] = {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\ndata: [DONE]\n\n")
            return
        b = json.dumps({"id": "x", "object": "chat.completion", "created": int(time.time()), "model": "fake",
                        "choices": [{"index": 0, "message": msg, "finish_reason": "stop"}],
                        "usage": {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}}).encode()
        self.send_response(200); self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
PY

PV_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
python3 "$PV_DIR/fake.py" "$PV_PORT" 2>"$PV_DIR/server.log" >/dev/null </dev/null &
PV_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:$PV_PORT/v1/models')" 2>/dev/null && break
    sleep 0.1
done

PV_SRC="$PV_DIR/src"
mkdir -p "$PV_SRC"
cp -R "$PROJECT_ROOT/assets/." "$PV_SRC/"
cat > "$PV_SRC/providers/e2eprov.yaml" <<YAML
version: 1.0.0
name: e2eprov
label: E2E Provider
order: 999
configKey: e2eprov_api_key
envVar: E2EPROV_API_KEY
url: http://127.0.0.1:$PV_PORT
chatPath: /e2e/v1/chat/completions
YAML

PV_HOME="$PV_DIR/home"
mkdir -p "$PV_HOME/.kdeps"
printf 'llm:\n  e2eprov_api_key: sk-e2e-provider\n' > "$PV_HOME/.kdeps/config.yaml"

OUT=$(printf 'say hi\n/quit\n' | env -u E2EPROV_API_KEY HOME="$PV_HOME" KDEPS_CONFIG_PATH="$PV_HOME/.kdeps/config.yaml" \
    KDEPS_ASSETS_SRC="$PV_SRC" KDEPS_ASSETS_DIR="$PV_HOME/store" KDEPS_ASSETS_URL=off \
    timeout 120 "$KDEPS_BIN" --new --backend e2eprov --model fake 2>&1 || true)

if output_grep_fixed "PROVIDER OK" "$OUT"; then
    test_passed "providers - the agent loop talks to a provider declared only in YAML"
else
    test_failed "providers - YAML-only provider reply" "$OUT $(cat "$PV_DIR/server.log")"
fi
if grep -q "path=/e2e/v1/chat/completions auth=Bearer sk-e2e-provider" "$PV_DIR/server.log"; then
    test_passed "providers - its chatPath and config.yaml key (llm.<configKey>) are used"
else
    test_failed "providers - chat path / key" "$(cat "$PV_DIR/server.log")"
fi

OUT=$(HOME="$PV_HOME" KDEPS_CONFIG_PATH="$PV_HOME/.kdeps/config.yaml" KDEPS_ASSETS_SRC="$PV_SRC" \
    KDEPS_ASSETS_DIR="$PV_HOME/store" KDEPS_ASSETS_URL=off "$KDEPS_BIN" update --list 2>&1 </dev/null)
if output_grep "providers/e2eprov +1\.0\.0 +source" "$OUT" && output_grep "providers/groq" "$OUT"; then
    test_passed "providers - providers are an updatable asset set"
else
    test_failed "providers - update --list" "$OUT"
fi

kill "$PV_SRV" 2>/dev/null
wait "$PV_SRV" 2>/dev/null
PV_SRV=""
rm -rf "$PV_DIR"
trap - EXIT

echo ""
echo "LLM providers E2E: done"
