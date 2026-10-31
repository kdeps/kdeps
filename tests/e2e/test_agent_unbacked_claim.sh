# shellcheck shell=bash
# E2E: a reply that reports an action no tool performed ("Created notes.md")
# is pushed back, and the model's real write_file call then creates the file.
# The caught claim is saved in the model's record: the next session sends it
# to the model (<your-record>) and /model record lists it.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
without_llm_override

echo "Testing unbacked action claim nudges..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "unbacked claim - python3 not available for the fake model server"
    restore_llm_override
    return 0 2>/dev/null || exit 0
fi

UC_DIR=$(mktemp -d)
UC_SRV=""
trap '[ -n "$UC_SRV" ] && kill "$UC_SRV" 2>/dev/null; rm -rf "$UC_DIR"' EXIT

cat > "$UC_DIR/fake.py" << 'PY'
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
        if any("<your-record>" in str(m.get("content") or "") for m in msgs):
            log("RECORD_SENT")
        if not body.get("tools"):
            msg = {"role": "assistant", "content": "ok"}
        elif any(m.get("role") == "tool" for m in msgs):
            log("TOOLRESULT")
            msg = {"role": "assistant", "content": "Wrote notes.md. CLAIM_OK"}
        elif any(NUDGE in str(m.get("content") or "") for m in msgs):
            log("NUDGED")
            msg = {"role": "assistant", "content": None, "tool_calls": [{
                "id": "c1", "type": "function",
                "function": {"name": "write_file",
                             "arguments": json.dumps({"path": "notes.md", "content": "hello kdeps\n"})},
            }]}
        else:
            log("CLAIM")
            msg = {"role": "assistant", "content": "Created [notes.md](https://example.invalid/notes.md) containing hello kdeps."}
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

UC_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
: > "$UC_DIR/server.log"
python3 "$UC_DIR/fake.py" "$UC_PORT" "$UC_DIR/server.log" >/dev/null 2>&1 &
UC_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import socket;socket.create_connection(('127.0.0.1',$UC_PORT),1)" 2>/dev/null && break
    sleep 0.1
done

UC_HOME="$UC_DIR/home"
mkdir -p "$UC_HOME" "$UC_DIR/work"
OUT=$(cd "$UC_DIR/work" && printf '/refine off\n/turo off\n/judges auto off\ncreate notes.md containing hello kdeps\n/exit\n' \
    | HOME="$UC_HOME" USERPROFILE="$(to_native_path "$UC_HOME")" KDEPS_RTK=off OPENAI_API_KEY=e2e \
        timeout 180 "$KDEPS_BIN" --new --backend openai \
        --base-url "http://127.0.0.1:$UC_PORT/v1" --model fake 2>&1 || true)
LOG=$(cat "$UC_DIR/server.log" 2>/dev/null)

if output_grep_fixed "CLAIM_OK" "$OUT" && output_grep_fixed "NUDGED" "$LOG" \
    && [ "$(cat "$UC_DIR/work/notes.md" 2>/dev/null)" = "hello kdeps" ]; then
    test_passed "unbacked claim - a claimed action with no tool call is pushed back and then really happens"
else
    test_failed "unbacked claim - nudge" "out=$(printf '%s' "$OUT" | tail -c 500) log=$LOG"
fi

# Second session, same home: the record reaches the model and /model record.
: > "$UC_DIR/server.log"
OUT=$(cd "$UC_DIR/work" && printf '/refine off\n/turo off\n/judges auto off\nhello\n/model record\n/exit\n' \
    | HOME="$UC_HOME" USERPROFILE="$(to_native_path "$UC_HOME")" KDEPS_RTK=off OPENAI_API_KEY=e2e \
        timeout 180 "$KDEPS_BIN" --new --backend openai \
        --base-url "http://127.0.0.1:$UC_PORT/v1" --model fake 2>&1 || true)
LOG=$(cat "$UC_DIR/server.log" 2>/dev/null)
if output_grep_fixed "RECORD_SENT" "$LOG" && output_grep_fixed "caught refusals" "$OUT" \
    && output_grep_fixed "no tool ran" "$OUT"; then
    test_passed "unbacked claim - the caught claim is the model's record in the next session and in /model record"
else
    test_failed "unbacked claim - model record" "out=$(printf '%s' "$OUT" | tail -c 600) log=$LOG"
fi

restore_llm_override
