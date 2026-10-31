# shellcheck shell=bash
# E2E: memory's caps are events. A ~/.kdeps/events override of
# memory-search-limit and memory-value-limit changes what the model sees,
# and one of memory-tool-results-limit what is captured, with no Go change.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
without_llm_override

echo "Testing memory limit events..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "memory limits - python3 not available for the fake model server"
    restore_llm_override
    return 0 2>/dev/null || exit 0
fi

ML_DIR=$(mktemp -d)
ML_SRV=""
trap '[ -n "$ML_SRV" ] && kill "$ML_SRV" 2>/dev/null; rm -rf "$ML_DIR"' EXIT

cat > "$ML_DIR/fake.py" << 'PY'
import json, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

LOG = sys.argv[2]

# The model saves three facts, searches for them, then answers.
STEPS = [
    ("memory_save", {"key": "e2e-one", "value": "e2eterm first fact with a long tail of words"}),
    ("memory_save", {"key": "e2e-two", "value": "e2eterm second fact with a long tail of words"}),
    ("memory_save", {"key": "e2e-three", "value": "e2eterm third fact with a long tail of words"}),
    ("memory_search", {"query": "e2eterm"}),
]
# In "tools" mode the model runs three commands, then searches memory for
# their captures.
if len(sys.argv) > 3 and sys.argv[3] == "tools":
    STEPS = [("bash_exec", {"command": "echo e2ecap%d" % i}) for i in range(3)]
    STEPS.append(("memory_search", {"query": "e2ecap"}))

def log(line):
    with open(LOG, "a", encoding="utf-8") as fh:
        fh.write(line + "\n")

def call(name, args, cid):
    return {"role": "assistant", "content": None, "tool_calls": [{
        "id": cid, "type": "function",
        "function": {"name": name, "arguments": json.dumps(args)},
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
        n = len(results)
        if n < len(STEPS) and body.get("tools"):
            msg = call(STEPS[n][0], STEPS[n][1], "c%d" % n)
        else:
            if results:
                log("SEARCH " + str(results[-1].get("content")))
            msg = {"role": "assistant", "content": "MEMORY_DONE"}
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

ML_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')

python3 "$ML_DIR/fake.py" "$ML_PORT" "$ML_DIR/server.log" >/dev/null 2>&1 &
ML_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import socket;socket.create_connection(('127.0.0.1',$ML_PORT),1)" 2>/dev/null && break
    sleep 0.1
done

ML_HOME="$ML_DIR/home"
mkdir -p "$ML_HOME/.kdeps/events" "$ML_DIR/work"
cat > "$ML_HOME/.kdeps/events/memory-search-limit.yaml" << 'EOF'
name: memory-search-limit
items: 2        # memory_search shows two matches
run: cap
EOF
cat > "$ML_HOME/.kdeps/events/memory-value-limit.yaml" << 'EOF'
name: memory-value-limit
on:
  bytes: 20     # and 20 bytes of each value
run: truncate
EOF
OUT=$(cd "$ML_DIR/work" && printf '/refine off\n/turo off\n/judges auto off\nsave three facts about e2eterm, then search memory for e2eterm\n/exit\n' \
    | HOME="$ML_HOME" USERPROFILE="$(to_native_path "$ML_HOME")" KDEPS_RTK=off OPENAI_API_KEY=e2e \
        timeout 180 "$KDEPS_BIN" --new --backend openai \
        --base-url "http://127.0.0.1:$ML_PORT/v1" --model fake 2>&1 || true)
LOG=$(cat "$ML_DIR/server.log" 2>/dev/null)

if output_grep_fixed "MEMORY_DONE" "$OUT" && output_grep_fixed "Found 3 memory entries (showing 2)" "$LOG" \
    && output_grep_fixed "1 more" "$LOG" && ! output_grep_fixed "long tail of words" "$LOG"; then
    test_passed "memory limits - memory-search-limit and memory-value-limit overrides cap memory_search"
else
    test_failed "memory limits - event overrides" "out=$(printf '%s' "$OUT" | tail -c 400) log=$LOG"
fi

# memory-tool-results-limit: one capture per tool is kept.
kill "$ML_SRV" 2>/dev/null
: > "$ML_DIR/server.log"
python3 "$ML_DIR/fake.py" "$ML_PORT" "$ML_DIR/server.log" tools >/dev/null 2>&1 &
ML_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import socket;socket.create_connection(('127.0.0.1',$ML_PORT),1)" 2>/dev/null && break
    sleep 0.1
done
ML_HOME2="$ML_DIR/home2"
mkdir -p "$ML_HOME2/.kdeps/events" "$ML_DIR/work2"
cat > "$ML_HOME2/.kdeps/events/memory-tool-results-limit.yaml" << 'EOF'
name: memory-tool-results-limit
items: 1        # keep one capture per tool
run: cap
EOF
OUT=$(cd "$ML_DIR/work2" && printf '/refine off\n/turo off\n/judges auto off\nrun three echo commands, then search memory for e2ecap\n/exit\n' \
    | HOME="$ML_HOME2" USERPROFILE="$(to_native_path "$ML_HOME2")" KDEPS_RTK=off OPENAI_API_KEY=e2e \
        timeout 180 "$KDEPS_BIN" --new --backend openai \
        --base-url "http://127.0.0.1:$ML_PORT/v1" --model fake 2>&1 || true)
LOG=$(cat "$ML_DIR/server.log" 2>/dev/null)
if output_grep_fixed "MEMORY_DONE" "$OUT" && output_grep_fixed "tool:bash_exec:e2ecap2" "$LOG" && ! output_grep_fixed "tool:bash_exec:e2ecap0" "$LOG"; then
    test_passed "memory limits - a memory-tool-results-limit override keeps only the newest capture"
else
    test_failed "memory limits - tool results limit" "out=$(printf '%s' "$OUT" | tail -c 600) log=$LOG"
fi

restore_llm_override
