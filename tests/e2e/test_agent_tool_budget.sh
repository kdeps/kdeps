# shellcheck shell=bash
# E2E: an event YAML with a tools: list, written by hand in ~/.kdeps/events,
# limits a real agent turn with no Go code that names it.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
without_llm_override

echo "Testing generic tool budget events..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "tool budget - python3 not available for the fake model server"
    restore_llm_override
    return 0 2>/dev/null || exit 0
fi

TB_DIR=$(mktemp -d)
TB_SRV=""
trap '[ -n "$TB_SRV" ] && kill "$TB_SRV" 2>/dev/null; rm -rf "$TB_DIR"' EXIT

cat > "$TB_DIR/fake.py" << 'PY'
import json, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

LOG = sys.argv[2]

def log(line):
    with open(LOG, "a", encoding="utf-8") as fh:
        fh.write(line + "\n")

def call(expr, cid):
    return {"role": "assistant", "content": None, "tool_calls": [{
        "id": cid, "type": "function",
        "function": {"name": "calculator", "arguments": json.dumps({"expression": expr})},
    }]}

class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def send_json(self, obj, code=200):
        raw = json.dumps(obj).encode()
        self.send_response(code)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self):
        self.send_json({"data": [{"id": "fake"}]})

    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", 0) or 0)) or b"{}"
        body = json.loads(raw or b"{}")
        results = [m for m in (body.get("messages") or []) if m.get("role") == "tool"]
        if not body.get("tools"):
            # The block forces an answer: the loop drops the tools for this call.
            if len(results) >= 2:
                log("SECOND " + str(results[-1].get("content")))
                self.reply(body, {"role": "assistant", "content": "BUDGET_DONE"})
            else:
                self.reply(body, {"role": "assistant", "content": "ok"})
            return
        if len(results) == 0:
            msg = call("1+1", "c1")
        elif len(results) == 1:
            msg = call("2+2", "c2")
        else:
            log("SECOND " + str(results[-1].get("content")))
            msg = {"role": "assistant", "content": "BUDGET_DONE"}
        self.reply(body, msg)

    def reply(self, body, msg):
        if body.get("stream"):
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            delta = dict(msg)
            if delta.get("content") is None:
                delta.pop("content", None)
            if delta.get("tool_calls"):
                delta["tool_calls"] = [dict(tc, index=0) for tc in delta["tool_calls"]]
            chunk = {"id": "x", "object": "chat.completion.chunk", "created": int(time.time()),
                     "model": "fake", "choices": [{"index": 0, "delta": delta, "finish_reason": None}]}
            self.wfile.write(b"data: " + json.dumps(chunk).encode() + b"\n\n")
            fin = "tool_calls" if msg.get("tool_calls") else "stop"
            chunk["choices"] = [{"index": 0, "delta": {}, "finish_reason": fin}]
            chunk["usage"] = {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
            self.wfile.write(b"data: " + json.dumps(chunk).encode() + b"\n\ndata: [DONE]\n\n")
            return
        self.send_json({"id": "x", "object": "chat.completion", "created": int(time.time()),
                        "model": "fake",
                        "choices": [{"index": 0, "message": msg,
                                     "finish_reason": "tool_calls" if msg.get("tool_calls") else "stop"}],
                        "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}})

ThreadingHTTPServer.request_queue_size = 64
ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
PY

TB_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
: > "$TB_DIR/server.log"
python3 "$TB_DIR/fake.py" "$TB_PORT" "$TB_DIR/server.log" >/dev/null 2>&1 &
TB_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import socket;socket.create_connection(('127.0.0.1',$TB_PORT),1)" 2>/dev/null && break
    sleep 0.1
done

TB_HOME="$TB_DIR/home"
mkdir -p "$TB_HOME/.kdeps/events"
cat > "$TB_HOME/.kdeps/events/calc-budget.yaml" << 'EOF'
name: calc-budget
tools: [calculator]   # the tools this budget limits
on:
  distinctCalls: 1    # one distinct calculator call per turn
run: block
EOF

OUT=$(printf '/refine off\n/judges auto off\nuse the calculator tool to add one plus one and then two plus two, and report both results\n/exit\n' \
    | HOME="$TB_HOME" USERPROFILE="$(to_native_path "$TB_HOME")" KDEPS_RTK=off OPENAI_API_KEY=e2e \
        timeout 180 "$KDEPS_BIN" --new --backend openai \
        --base-url "http://127.0.0.1:$TB_PORT/v1" --model fake 2>&1 || true)
LOG=$(cat "$TB_DIR/server.log")

if output_grep_fixed "BUDGET_DONE" "$OUT" \
    && output_grep_fixed "SECOND" "$LOG" && output_grep_fixed "calc-budget" "$LOG"; then
    test_passed "tool budget - a ~/.kdeps/events tools: event blocks the second distinct call"
else
    test_failed "tool budget - generic event" "out=$(printf '%s' "$OUT" | tail -c 400) log=$LOG"
fi

restore_llm_override
