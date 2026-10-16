# shellcheck shell=bash
# E2E: a tool that exists only as a YAML asset with an http: block.
#
# A fake OpenAI-compatible server plays the model: it asks for e2e_echo, a
# tool no Go code defines, and answers with what the tool returned. The same
# server is the HTTP API the tool calls, so the run checks the whole path:
# asset -> registration -> agent loop -> templated request -> templated result.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"

echo "Testing declared HTTP tools (tool assets with an http: block)..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "declared tools - python3 not available for the fake model/API server"
    return 0 2>/dev/null || exit 0
fi

DT_DIR=$(mktemp -d)
DT_SRV=""
trap '[ -n "$DT_SRV" ] && kill "$DT_SRV" 2>/dev/null; rm -rf "$DT_DIR"' EXIT

cat > "$DT_DIR/fake.py" <<'PY'
import functools, http.server as h, json, sys, time, urllib.parse

class H(h.BaseHTTPRequestHandler):
    def log_message(self, *a): pass
    def send_json(self, obj, code=200):
        b = json.dumps(obj).encode()
        self.send_response(code); self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b))); self.end_headers(); self.wfile.write(b)
    def do_GET(self):
        u = urllib.parse.urlparse(self.path)
        if u.path == "/echo":
            q = urllib.parse.parse_qs(u.query)
            self.send_json({"echo": q.get("text", [""])[0], "key": self.headers.get("X-Key", "")})
        else:
            self.send_json({"data": [{"id": "fake"}]})
    def do_POST(self):
        body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}")
        msgs = body.get("messages", [])
        tools = [t.get("function", {}).get("name") for t in body.get("tools", []) or []]
        print("REQ has_e2e=%s" % ("e2e_echo" in tools), file=sys.stderr, flush=True)
        tool_msgs = [m for m in msgs if m.get("role") == "tool"]
        if tool_msgs:
            msg = {"role": "assistant", "content": "TOOL SAID " + str(tool_msgs[-1].get("content"))}
        elif "e2e_echo" in tools:
            msg = {"role": "assistant", "content": None, "tool_calls": [{"id": "c1", "type": "function",
                   "function": {"name": "e2e_echo", "arguments": json.dumps({"text": "ping pong"})}}]}
        else:
            last = [m for m in msgs if m.get("role") == "user"]
            c = last[-1]["content"] if last else ""
            msg = {"role": "assistant", "content": c if isinstance(c, str) else "ok"}
        if body.get("stream"):
            self.send_response(200); self.send_header("Content-Type", "text/event-stream"); self.end_headers()
            delta = dict(msg)
            if "tool_calls" in delta:
                delta["tool_calls"] = [dict(tc, index=0) for tc in delta["tool_calls"]]
            ch = {"id": "x", "object": "chat.completion.chunk", "created": int(time.time()), "model": "fake",
                  "choices": [{"index": 0, "delta": delta, "finish_reason": None}]}
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\n")
            fin = "tool_calls" if "tool_calls" in msg else "stop"
            ch["choices"] = [{"index": 0, "delta": {}, "finish_reason": fin}]
            ch["usage"] = {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\ndata: [DONE]\n\n")
            return
        self.send_json({"id": "x", "object": "chat.completion", "created": int(time.time()), "model": "fake",
                        "choices": [{"index": 0, "message": msg, "finish_reason": "tool_calls" if "tool_calls" in msg else "stop"}],
                        "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}})

h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
PY

DT_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
python3 "$DT_DIR/fake.py" "$DT_PORT" 2>"$DT_DIR/server.log" &
DT_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import urllib.request;urllib.request.urlopen('http://127.0.0.1:$DT_PORT/v1/models')" 2>/dev/null && break
    sleep 0.1
done

# A source dir: the repo's assets plus one tool with no Go code behind it.
DT_SRC="$DT_DIR/src"
mkdir -p "$DT_SRC"
cp -R "$PROJECT_ROOT/assets/." "$DT_SRC/"
cat > "$DT_SRC/tools/e2e_echo.yaml" <<YAML
version: 1.0.0
description: Echo text through a local HTTP API (e2e test tool).
parameters:
  text:
    type: string
    description: Text to echo
    required: true
http:
  env:
    KEY: [E2E_ECHO_KEY]
  method: GET
  url: "http://127.0.0.1:$DT_PORT/echo?text={{urlquery .args.text}}"
  headers:
    X-Key: "{{.env.KEY}}"
  result: "echoed={{.response.echo}} key={{.response.key}}"
YAML

# dt_run sends one prompt to the agent loop, with E2E_ECHO_KEY set to $1.
dt_run() {
    local home="$DT_DIR/home-$RANDOM"
    mkdir -p "$home"
    printf 'use the e2e_echo tool\n/quit\n' | HOME="$home" KDEPS_ASSETS_SRC="$DT_SRC" \
        KDEPS_ASSETS_DIR="$home/store" KDEPS_ASSETS_URL=off E2E_ECHO_KEY="$1" OPENAI_API_KEY=e2e \
        timeout 120 "$KDEPS_BIN" --new --backend openai --base-url "http://127.0.0.1:$DT_PORT/v1" --model fake 2>&1 || true
}

OUT=$(dt_run "sekret")
if output_grep_fixed "TOOL SAID echoed=ping pong key=sekret" "$OUT"; then
    test_passed "declared tools - the agent loop calls a YAML-only tool and gets its templated result"
else
    test_failed "declared tools - YAML-only tool call" "$OUT $(cat "$DT_DIR/server.log")"
fi
if grep -q "has_e2e=True" "$DT_DIR/server.log"; then
    test_passed "declared tools - the tool is offered to the model when its env resolves"
else
    test_failed "declared tools - tool not offered" "$(cat "$DT_DIR/server.log")"
fi

: > "$DT_DIR/server.log"
OUT=$(dt_run "")
if grep -q "has_e2e=False" "$DT_DIR/server.log" && ! grep -q "has_e2e=True" "$DT_DIR/server.log"; then
    test_passed "declared tools - the tool is not offered when its env var is unset"
else
    test_failed "declared tools - tool offered without its env" "$(cat "$DT_DIR/server.log")"
fi

kill "$DT_SRV" 2>/dev/null
wait "$DT_SRV" 2>/dev/null
DT_SRV=""
rm -rf "$DT_DIR"
trap - EXIT

echo ""
echo "Declared HTTP tools E2E: done"
