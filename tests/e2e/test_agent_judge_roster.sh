# shellcheck shell=bash
# E2E: with auto judges on, a model that answers the roster request with a
# clean JSON object gets its panel: the engine hands JSON-mode replies back
# decoded, and the roster is read from that object.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
without_llm_override

echo "Testing the auto judge roster..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "judge roster - python3 not available for the fake model server"
    restore_llm_override
    return 0 2>/dev/null || exit 0
fi

JR_DIR=$(mktemp -d)
JR_SRV=""
trap '[ -n "$JR_SRV" ] && kill "$JR_SRV" 2>/dev/null; rm -rf "$JR_DIR"' EXIT

cat > "$JR_DIR/fake.py" << 'PY'
import json, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

LOG = sys.argv[2]
NUDGE = "You made no tool call this turn"

def log(line):
    with open(LOG, "a", encoding="utf-8") as fh:
        fh.write(line + "\n")

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
        msgs = body.get("messages") or []
        text = " ".join(str(m.get("content") or "") for m in msgs)
        if "panel of reviewer personas" in text:
            log("ROSTER")
            msg = {"role": "assistant", "content": json.dumps({"judges": [
                {"name": "e2e-exactness", "criteria": "The answer states the sum exactly."}]})}
        elif "independent reviewer" in text:
            if any(m.get("role") == "tool" for m in msgs):
                msg = {"role": "assistant", "content": "approved"}
            else:
                log("JUDGE")
                msg = {"role": "assistant", "content": None, "tool_calls": [{
                    "id": "j1", "type": "function",
                    "function": {"name": "judge_verdict", "arguments": json.dumps({"approved": True, "feedback": ""})},
                }]}
        else:
            msg = {"role": "assistant", "content": "The sum is 4. ROSTER_OK"}
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

JR_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
: > "$JR_DIR/server.log"
python3 "$JR_DIR/fake.py" "$JR_PORT" "$JR_DIR/server.log" >/dev/null 2>&1 &
JR_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import socket;socket.create_connection(('127.0.0.1',$JR_PORT),1)" 2>/dev/null && break
    sleep 0.1
done

JR_HOME="$JR_DIR/home"
mkdir -p "$JR_HOME" "$JR_DIR/work"
OUT=$(cd "$JR_DIR/work" && printf '/refine off\n/turo off\n/judges auto on\nplease add two and two and tell me the exact sum\n/exit\n' \
    | HOME="$JR_HOME" USERPROFILE="$(to_native_path "$JR_HOME")" KDEPS_RTK=off OPENAI_API_KEY=e2e \
        timeout 180 "$KDEPS_BIN" --new --backend openai \
        --base-url "http://127.0.0.1:$JR_PORT/v1" --model fake 2>&1 || true)
LOG=$(cat "$JR_DIR/server.log" 2>/dev/null)

if output_grep_fixed "ROSTER_OK" "$OUT" && output_grep_fixed "ROSTER" "$LOG" \
    && output_grep_fixed "panel generated" "$OUT" && output_grep_fixed "e2e-exactness" "$OUT" \
    && ! output_grep_fixed "roster generation returned nothing" "$OUT"; then
    test_passed "judge roster - a JSON roster reply becomes the panel"
else
    test_failed "judge roster - panel" "out=$(printf '%s' "$OUT" | tail -c 800) log=$LOG"
fi

restore_llm_override
