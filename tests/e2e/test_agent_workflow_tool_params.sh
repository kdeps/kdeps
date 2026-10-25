# shellcheck shell=bash
# E2E: a workflow registered as an agent tool takes the keys its resources
# read with input(): the greeter reads input('name'), so the model is offered
# a "name" parameter, and calling it with {"name": "Ada"} greets Ada. A
# judge_verdict sent with approved as the string "true" approves.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
without_llm_override

echo "Testing workflow tool parameters..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "workflow tool params - python3 not available for the fake model server"
    restore_llm_override
    return 0 2>/dev/null || exit 0
fi

WT_DIR=$(mktemp -d)
WT_SRV=""
trap '[ -n "$WT_SRV" ] && kill "$WT_SRV" 2>/dev/null; rm -rf "$WT_DIR"' EXIT

cat > "$WT_DIR/fake.py" << 'PY'
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
        tools = body.get("tools") or []
        for t in tools:
            fn = t.get("function") or {}
            if fn.get("name") == "greeter" and "name" in ((fn.get("parameters") or {}).get("properties") or {}):
                log("PARAM_NAME")
        results = [m for m in msgs if m.get("role") == "tool"]
        if "panel of reviewer personas" in text:
            msg = {"role": "assistant", "content": json.dumps({"judges": [
                {"name": "e2e-greeting", "criteria": "The answer quotes the greeting."}]})}
        elif "independent reviewer" in text:
            if results:
                msg = {"role": "assistant", "content": "approved"}
            else:
                log("JUDGE")
                msg = {"role": "assistant", "content": None, "tool_calls": [{
                    "id": "j1", "type": "function",
                    "function": {"name": "judge_verdict", "arguments": json.dumps({"approved": "true", "feedback": ""})},
                }]}
        elif not tools:
            msg = {"role": "assistant", "content": "ok"}
        elif results:
            log("RESULT " + str(results[-1].get("content"))[:200])
            msg = {"role": "assistant", "content": "The greeter said: Hello, Ada! WT_OK"}
        else:
            msg = {"role": "assistant", "content": None, "tool_calls": [{
                "id": "c1", "type": "function",
                "function": {"name": "greeter", "arguments": json.dumps({"name": "Ada"})},
            }]}
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

WT_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
: > "$WT_DIR/server.log"
python3 "$WT_DIR/fake.py" "$WT_PORT" "$WT_DIR/server.log" >/dev/null 2>&1 &
WT_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import socket;socket.create_connection(('127.0.0.1',$WT_PORT),1)" 2>/dev/null && break
    sleep 0.1
done

WT_HOME="$WT_DIR/home"
mkdir -p "$WT_HOME" "$WT_DIR/greeter/resources"
cat > "$WT_DIR/greeter/workflow.yaml" << 'EOF'
apiVersion: kdeps.io/v1
kind: Workflow
metadata:
  name: greeter
  description: Greets a person by name
  version: "1.0.0"
  targetActionId: greet
settings:
  agentSettings:
    pythonVersion: "3.12"
EOF
cat > "$WT_DIR/greeter/resources/greet.yaml" << 'EOF'
actionId: greet
name: Greet
component:
  name: exec
  with:
    command: echo "Hello, {{ input('name') }}! (from greeter)"
EOF
OUT=$(cd "$WT_DIR" && printf '/refine off\n/turo off\n/judges auto on\nuse the greeter tool to greet Ada please\n/exit\n' \
    | HOME="$WT_HOME" USERPROFILE="$(to_native_path "$WT_HOME")" KDEPS_RTK=off OPENAI_API_KEY=e2e \
        timeout 180 "$KDEPS_BIN" --new --backend openai \
        --base-url "http://127.0.0.1:$WT_PORT/v1" --model fake greeter 2>&1 || true)
LOG=$(cat "$WT_DIR/server.log" 2>/dev/null)

if output_grep_fixed "PARAM_NAME" "$LOG" && output_grep_fixed "Hello, Ada!" "$LOG" && output_grep_fixed "WT_OK" "$OUT"; then
    test_passed "workflow tool params - input('name') is a name parameter the call fills"
else
    test_failed "workflow tool params - name parameter" "out=$(printf '%s' "$OUT" | tail -c 600) log=$LOG"
fi
if [ "$(printf '%s\n' "$LOG" | grep -c '^JUDGE')" = "1" ]; then
    test_passed "workflow tool params - a judge verdict with approved \"true\" approves the first time"
else
    test_failed "workflow tool params - string verdict" "log=$LOG"
fi

restore_llm_override
