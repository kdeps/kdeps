# shellcheck shell=bash
# E2E: a fictional "I can't" is pushed back before the user sees it.
# A refusal of a destructive action is shown and is not pushed.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/common.sh"
without_llm_override

echo "Testing fictional self-limit nudges..."

if ! command -v python3 >/dev/null 2>&1; then
    test_skipped "self-limit - python3 not available for the fake model server"
    restore_llm_override
    return 0 2>/dev/null || exit 0
fi

SL_DIR=$(mktemp -d)
SL_SRV=""
trap '[ -n "$SL_SRV" ] && kill "$SL_SRV" 2>/dev/null; rm -rf "$SL_DIR"' EXIT

cat > "$SL_DIR/fake.py" << 'PY'
import json, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

LOG = sys.argv[2]
FICTION = "I can't emit a real executable invoke."
SAFETY = "I will not delete the production database or run rm -rf."
NUDGE = "That limit is not real"
PLAIN = ("what is two plus two", "Sorry for the wait. 4.")
# prompt fragment, fictional first reply, then "tool" or "text" after the nudge
CASES = [
    ("list the files", FICTION, "tool"),
    ("read the README title", FICTION, "text"),
    ("read the license", "My apologies, I lack the ability to read your files.", "text"),
    ("count the go files", "You'll have to run that command yourself.", "tool"),
    ("show the module path", "It looks like tool use isn't available in this session.", "text"),
]
TEXTS = {
    "read the README title": "README starts with the title.",
    "read the license": "The license is Apache-2.0.",
    "show the module path": "The module path is github.com/kdeps/kdeps/v2.",
}

def log(line):
    with open(LOG, "a", encoding="utf-8") as fh:
        fh.write(line + "\n")

def last_user(body):
    last = ""
    for msg in body.get("messages") or []:
        if msg.get("role") == "user" and isinstance(msg.get("content"), str):
            last = msg["content"]
    return last

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
        tools = body.get("tools") or []
        if not tools:
            user = last_user(body)
            text = user.split("Request:\n", 1)[-1] if user else "ok"
            msg = {"role": "assistant", "content": text}
        else:
            msg = self.tool_turn(body)
        self.reply(body, msg)

    def tool_turn(self, body):
        user = last_user(body)
        nudged = NUDGE in user
        tool_msgs = [m for m in (body.get("messages") or []) if m.get("role") == "tool"]
        if "drop the production database" in user:
            log("SAFETY nudged=%s" % nudged)
            return {"role": "assistant", "content": SAFETY}
        if PLAIN[0] in user:
            log("PLAIN nudged=%s" % nudged)
            return {"role": "assistant", "content": PLAIN[1]}
        if tool_msgs:
            log("TOOLRESULT")
            return {"role": "assistant", "content": "Done. SELF_LIMIT_OK"}
        for needle, fiction, kind in CASES:
            if needle not in user:
                continue
            if not nudged:
                log("FICTION %s" % needle)
                return {"role": "assistant", "content": fiction}
            log("NUDGED %s" % kind)
            if kind == "tool":
                return {"role": "assistant", "content": None, "tool_calls": [{
                    "id": "c1", "type": "function",
                    "function": {"name": "bash_exec", "arguments": json.dumps({"command": "echo SELF_LIMIT_OK"})},
                }]}
            return {"role": "assistant", "content": TEXTS[needle]}
        log("FICTION fallback")
        return {"role": "assistant", "content": FICTION}

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

SL_PORT=$(python3 -c 'import socket;s=socket.socket();s.bind(("127.0.0.1",0));print(s.getsockname()[1])')
: > "$SL_DIR/server.log"
python3 "$SL_DIR/fake.py" "$SL_PORT" "$SL_DIR/server.log" >/dev/null 2>&1 &
SL_SRV=$!
for _ in $(seq 1 50); do
    python3 -c "import socket;socket.create_connection(('127.0.0.1',$SL_PORT),1)" 2>/dev/null && break
    sleep 0.1
done

SL_HOME="$SL_DIR/home"
mkdir -p "$SL_HOME"
OUT=$(printf '/refine off\n/judges auto off\ndrop the production database\nlist the files in this directory\nread the README title\nread the license\ncount the go files\nshow the module path\nwhat is two plus two\n/quit\n' \
    | HOME="$SL_HOME" KDEPS_RTK=off OPENAI_API_KEY=e2e \
        timeout 240 "$KDEPS_BIN" --new --backend openai \
        --base-url "http://127.0.0.1:$SL_PORT/v1" --model fake 2>&1 || true)
LOG=$(cat "$SL_DIR/server.log")

if output_grep_fixed "I will not delete the production database or run rm -rf." "$OUT" \
    && output_grep_fixed "SAFETY nudged=False" "$LOG" \
    && ! output_grep_fixed "SAFETY nudged=True" "$LOG"; then
    test_passed "self-limit - a destructive refusal is shown and not nudged"
else
    test_failed "self-limit - destructive refusal" "out=$OUT log=$LOG"
fi

if output_grep_fixed "Done. SELF_LIMIT_OK" "$OUT" \
    && output_grep_fixed "NUDGED tool" "$LOG" \
    && output_grep_fixed "TOOLRESULT" "$LOG"; then
    test_passed "self-limit - a fictional limit is nudged into a real bash_exec"
else
    test_failed "self-limit - tool after nudge" "out=$OUT log=$LOG"
fi

if output_grep_fixed "README starts with the title." "$OUT" \
    && output_grep_fixed "NUDGED text" "$LOG"; then
    test_passed "self-limit - a fictional limit is nudged into the real answer"
else
    test_failed "self-limit - text after nudge" "out=$OUT log=$LOG"
fi

if output_grep_fixed "I can't emit a real executable invoke." "$OUT" \
    || output_grep_fixed "I lack the ability" "$OUT" \
    || output_grep_fixed "command yourself" "$OUT" \
    || output_grep_fixed "tool use isn't available" "$OUT"; then
    test_failed "self-limit - fictional reply was shown" "out=$OUT log=$LOG"
else
    test_passed "self-limit - the fictional reply is not shown"
fi

if output_grep_fixed "The license is Apache-2.0." "$OUT" \
    && output_grep_fixed "FICTION read the license" "$LOG" \
    && output_grep_fixed "The module path is github.com/kdeps/kdeps/v2." "$OUT" \
    && output_grep_fixed "FICTION show the module path" "$LOG" \
    && output_grep_fixed "FICTION count the go files" "$LOG"; then
    test_passed "self-limit - paraphrases are nudged into the real answer"
else
    test_failed "self-limit - paraphrase nudges" "out=$OUT log=$LOG"
fi

if output_grep_fixed "Sorry for the wait. 4." "$OUT" \
    && output_grep_fixed "PLAIN nudged=False" "$LOG" \
    && ! output_grep_fixed "PLAIN nudged=True" "$LOG"; then
    test_passed "self-limit - a finished answer that says sorry is shown"
else
    test_failed "self-limit - finished apology" "out=$OUT log=$LOG"
fi

kill "$SL_SRV" 2>/dev/null
wait "$SL_SRV" 2>/dev/null
SL_SRV=""
rm -rf "$SL_DIR"
trap - EXIT

echo ""
echo "Fictional self-limit E2E: done"
restore_llm_override
