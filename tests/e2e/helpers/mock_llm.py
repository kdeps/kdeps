"""A mock OpenAI-compatible LLM for E2E tests.

    python3 mock_llm.py <port> <request log> [parent pid]

With a parent pid, the mock exits once that process is gone, so a test run
that starts it once never leaves it behind.

Every request body is appended to the log. The reply to the last user
message:
  - prompt refinement: the request itself
  - a compaction request (<conversation>): a fixed summary
  - "CALL <tool> <json args>": that tool call, written as an <invoke> block
  - anything after a CALL request (the tool's result): "TOOL DONE"
  - anything else: "MOCK REPLY"
"""
import http.server as h, json, os, re, sys, threading, time

LOG = sys.argv[2]
SUMMARY = ("## Goal\nShip the thing\n\n## Progress\n### Done\n- [x] it\n\n## Key Decisions\n"
           "- **Defensive**: test the edges\n\n## Next Steps\n1. keep going\n")


def text(m):
    c = m.get("content", "")
    if isinstance(c, list):
        return " ".join(p.get("text", "") for p in c if isinstance(p, dict))
    return c or ""


def reply_for(msgs):
    everything = "\n".join(text(m) for m in msgs)
    last = text(msgs[-1]) if msgs else ""
    if "You rewrite a user's request" in everything:
        return last.split("Request:\n", 1)[-1]
    if "<conversation>" in everything:
        return SUMMARY
    calls = [i for i, m in enumerate(msgs) if m.get("role") == "user" and "CALL " in text(m)]
    if calls and calls[-1] != len(msgs) - 1:
        return "TOOL DONE"  # the tool ran: its result follows the CALL request
    m = re.search(r"CALL (\w+) (\{.*\})", last, re.S)
    if m:
        name, args = m.group(1), json.loads(m.group(2))
        params = "".join('<parameter name="%s">%s</parameter>\n' % (k, v) for k, v in args.items())
        return '<invoke name="%s">\n%s</invoke>' % (name, params)
    return "MOCK REPLY"


class H(h.BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def do_GET(self):
        b = json.dumps({"data": [{"id": "fake"}]}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)

    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length", 0))) or b"{}"
        with open(LOG, "a", encoding="utf-8") as f:
            f.write(raw.decode("utf-8") + "\n")
        body = json.loads(raw)
        msg = {"role": "assistant", "content": reply_for(body.get("messages", []))}
        usage = {"prompt_tokens": 1, "completion_tokens": 1, "total_tokens": 2}
        if body.get("stream"):
            self.send_response(200)
            self.send_header("Content-Type", "text/event-stream")
            self.end_headers()
            ch = {"id": "x", "object": "chat.completion.chunk", "created": int(time.time()), "model": "fake",
                  "choices": [{"index": 0, "delta": msg, "finish_reason": None}]}
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\n")
            ch["choices"] = [{"index": 0, "delta": {}, "finish_reason": "stop"}]
            ch["usage"] = usage
            self.wfile.write(b"data: " + json.dumps(ch).encode() + b"\n\ndata: [DONE]\n\n")
            return
        b = json.dumps({"id": "x", "object": "chat.completion", "created": int(time.time()), "model": "fake",
                        "choices": [{"index": 0, "message": msg, "finish_reason": "stop"}], "usage": usage}).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(b)))
        self.end_headers()
        self.wfile.write(b)


def watch_parent(pid):
    while True:
        time.sleep(2)
        try:
            os.kill(pid, 0)
        except OSError:
            os._exit(0)

if len(sys.argv) > 3:
    threading.Thread(target=watch_parent, args=(int(sys.argv[3]),), daemon=True).start()
h.ThreadingHTTPServer.request_queue_size = 64
h.ThreadingHTTPServer(("127.0.0.1", int(sys.argv[1])), H).serve_forever()
